package httpscenario

import (
	"bytes"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"net/http/httptrace"
	"net/http/httputil"
	"strconv"
	"strings"
	"time"

	phttp "github.com/yandex/pandora/components/guns/http"
	"github.com/yandex/pandora/core"
	"github.com/yandex/pandora/core/aggregator/netsample"
	"github.com/yandex/pandora/core/warmup"
	"go.uber.org/zap"
)

type Gun interface {
	Shoot(ammo *Scenario)
	Bind(sample netsample.Aggregator, deps core.GunDeps) error
	WarmUp(opts *warmup.Options) (any, error)
}

const (
	EmptyTag = "__EMPTY__"
)

type ScenarioGun struct {
	base *phttp.BaseGun
}

var _ Gun = (*ScenarioGun)(nil)
var _ io.Closer = (*ScenarioGun)(nil)

func (g *ScenarioGun) WarmUp(opts *warmup.Options) (any, error) {
	return g.base.WarmUp(opts)
}

func (g *ScenarioGun) Bind(aggregator netsample.Aggregator, deps core.GunDeps) error {
	return g.base.Bind(aggregator, deps)
}

// Shoot is thread safe if Do and Connect hooks are thread safe.
func (g *ScenarioGun) Shoot(ammo *Scenario) {
	if g.base.Aggregator == nil {
		zap.L().Panic("must bind before shoot")
	}
	if g.base.Connect != nil {
		err := g.base.Connect(g.base.Ctx)
		if err != nil {
			g.base.Log.Warn("Connect fail", zap.Error(err))
			return
		}
	}

	templateVars := map[string]any{
		"source": ammo.VariableStorage.Variables(),
	}

	err := g.shoot(ammo, templateVars)
	if err != nil {
		g.base.Log.Warn("Invalid ammo", zap.Uint64("request", ammo.ID), zap.Error(err))
		return
	}
}

func (g *ScenarioGun) Do(req *http.Request) (*http.Response, error) {
	return g.base.Client.Do(req)
}

func (g *ScenarioGun) Close() error {
	if g.base.OnClose != nil {
		return g.base.OnClose()
	}
	return nil
}

func (g *ScenarioGun) shoot(ammo *Scenario, templateVars map[string]any) error {
	if templateVars == nil {
		templateVars = map[string]any{}
	}

	requestVars := map[string]any{}
	templateVars["request"] = requestVars

	startAt := time.Now()
	var idBuilder strings.Builder
	rnd := strconv.Itoa(rand.Int())
	for _, req := range ammo.Requests {
		tag := ammo.Name + "." + req.Name
		g.buildLogID(&idBuilder, tag, ammo.ID, rnd)
		sample := netsample.Acquire(tag)

		err := g.shootStep(req, sample, ammo.Name, templateVars, requestVars, idBuilder.String())
		if err != nil {
			g.reportErr(sample, err)
			return err
		}
	}
	spent := time.Since(startAt)
	if ammo.MinWaitingTime > spent {
		time.Sleep(ammo.MinWaitingTime - spent)
	}
	return nil
}

func (g *ScenarioGun) shootStep(step Request, sample *netsample.Sample, ammoName string, templateVars map[string]any, requestVars map[string]any, stepLogID string) error {
	const op = "base_gun.shootStep"

	stepVars := map[string]any{}
	requestVars[step.Name] = stepVars

	// Preprocessor
	if step.Preprocessor != nil {
		preProcVars, err := step.Preprocessor.Process(templateVars)
		if err != nil {
			return fmt.Errorf("%s preProcessor %w", op, err)
		}
		stepVars["preprocessor"] = preProcVars
		if g.base.DebugLog {
			g.base.GunDeps.Log.Debug("Preprocessor variables", zap.Any(fmt.Sprintf(".request.%s.preprocessor", step.Name), preProcVars))
		}
	}

	// Entities
	reqParts := RequestParts{
		URL:     step.URI,
		Method:  step.Method,
		Body:    step.GetBody(),
		Headers: step.GetHeaders(),
	}

	// Template
	if err := step.Templater.Apply(&reqParts, templateVars, ammoName, step.Name); err != nil {
		return fmt.Errorf("%s templater.Apply %w", op, err)
	}

	// Prepare request
	req, err := g.prepareRequest(reqParts)
	if err != nil {
		return fmt.Errorf("%s prepareRequest %w", op, err)
	}

	var reqBytes []byte
	if g.base.Config.AnswLog.Enabled {
		var dumpErr error
		reqBytes, dumpErr = httputil.DumpRequestOut(req, true)
		if dumpErr != nil {
			g.base.Log.Error("Error dumping request:", zap.Error(dumpErr))
		}
	}

	timings, req := g.initTracing(req, sample)

	resp, err := g.base.Client.Do(req)

	g.saveTrace(timings, sample, resp)

	if err != nil {
		return fmt.Errorf("%s g.Do %w", op, err)
	}

	// Log
	processors := step.Postprocessors
	var respBody *bytes.Reader
	var respBodyBytes []byte
	if g.base.Config.AnswLog.Enabled || g.base.DebugLog || len(processors) > 0 {
		respBodyBytes, err = io.ReadAll(resp.Body)
		if err == nil {
			respBody = bytes.NewReader(respBodyBytes)
		}
	} else {
		_, err = io.Copy(io.Discard, resp.Body)
	}
	if err != nil {
		return fmt.Errorf("%s io.Copy %w", op, err)
	}
	defer func() {
		closeErr := resp.Body.Close()
		if closeErr != nil {
			g.base.GunDeps.Log.Error("resp.Body.Close", zap.Error(closeErr))
		}
	}()

	if g.base.DebugLog {
		g.verboseLogging(resp, reqBytes, respBodyBytes)
	}
	if g.base.Config.AnswLog.Enabled {
		g.answReqRespLogging(reqBytes, resp, respBodyBytes, stepLogID)
	}

	// Postprocessor
	postprocessorVars := map[string]any{}
	var vars map[string]any
	for _, postprocessor := range processors {
		vars, err = postprocessor.Process(resp, respBody)
		if err != nil {
			return fmt.Errorf("%s postprocessor.Postprocess %w", op, err)
		}
		for k, v := range vars {
			postprocessorVars[k] = v
		}
		_, err = respBody.Seek(0, io.SeekStart)
		if err != nil {
			return fmt.Errorf("%s postprocessor.Postprocess %w", op, err)
		}
	}
	stepVars["postprocessor"] = postprocessorVars

	sample.SetProtoCode(resp.StatusCode)
	g.base.Aggregator.Report(sample)

	if g.base.DebugLog {
		g.base.GunDeps.Log.Debug("Postprocessor variables", zap.Any(fmt.Sprintf(".request.%s.postprocessor", step.Name), postprocessorVars))
	}

	if step.Sleep > 0 {
		time.Sleep(step.Sleep)
	}
	return nil
}

func (g *ScenarioGun) buildLogID(idBuilder *strings.Builder, tag string, ammoID uint64, rnd string) {
	idBuilder.Reset()
	idBuilder.WriteString(tag)
	idBuilder.WriteByte('.')
	idBuilder.WriteString(rnd)
	idBuilder.WriteByte('.')
	idBuilder.WriteString(strconv.Itoa(int(ammoID)))
}

func (g *ScenarioGun) prepareRequest(reqParts RequestParts) (*http.Request, error) {
	const op = "base_gun.prepareRequest"

	var reader io.Reader
	if reqParts.Body != nil {
		reader = bytes.NewReader(reqParts.Body)
	}

	req, err := http.NewRequest(reqParts.Method, reqParts.URL, reader)
	if err != nil {
		return nil, fmt.Errorf("%s http.NewRequest %w", op, err)
	}
	for k, v := range reqParts.Headers {
		req.Header.Set(k, v)
	}

	if g.base.Config.SSL {
		req.URL.Scheme = "https"
	} else {
		req.URL.Scheme = "http"
	}
	if req.Host == "" {
		req.Host = getHostWithoutPort(g.base.Config.Target)
	}
	req.URL.Host = g.base.Config.TargetResolved

	return req, err
}

func (g *ScenarioGun) initTracing(req *http.Request, sample *netsample.Sample) (*phttp.TraceTimings, *http.Request) {
	var timings *phttp.TraceTimings
	if g.base.Config.HTTPTrace.TraceEnabled {
		var clientTracer *httptrace.ClientTrace
		clientTracer, timings = phttp.CreateHTTPTrace()
		req = req.WithContext(httptrace.WithClientTrace(req.Context(), clientTracer))
	}
	if g.base.Config.HTTPTrace.DumpEnabled {
		requestDump, err := httputil.DumpRequest(req, true)
		if err != nil {
			g.base.Log.Error("DumpRequest error", zap.Error(err))
		} else {
			sample.SetRequestBytes(len(requestDump))
		}
	}
	return timings, req
}

func (g *ScenarioGun) saveTrace(timings *phttp.TraceTimings, sample *netsample.Sample, resp *http.Response) {
	if g.base.Config.HTTPTrace.TraceEnabled && timings != nil {
		sample.SetReceiveTime(timings.GetReceiveTime())
	}
	if g.base.Config.HTTPTrace.DumpEnabled && resp != nil {
		responseDump, e := httputil.DumpResponse(resp, true)
		if e != nil {
			g.base.Log.Error("DumpResponse error", zap.Error(e))
		} else {
			sample.SetResponseBytes(len(responseDump))
		}
	}
	if g.base.Config.HTTPTrace.TraceEnabled && timings != nil {
		sample.SetConnectTime(timings.GetConnectTime())
		sample.SetSendTime(timings.GetSendTime())
		sample.SetLatency(timings.GetLatency())
	}
}

func (g *ScenarioGun) verboseLogging(resp *http.Response, reqBody, respBody []byte) {
	if resp == nil {
		g.base.Log.Error("Response is nil")
		return
	}
	fields := make([]zap.Field, 0, 4)
	fields = append(fields, zap.String("URL", resp.Request.URL.String()))
	fields = append(fields, zap.String("Host", resp.Request.Host))
	fields = append(fields, zap.Any("Headers", resp.Request.Header))
	if reqBody != nil {
		fields = append(fields, zap.ByteString("Body", reqBody))
	}
	g.base.Log.Debug("Request debug info", fields...)

	fields = fields[:0]
	fields = append(fields, zap.Int("Status Code", resp.StatusCode))
	fields = append(fields, zap.String("Status", resp.Status))
	fields = append(fields, zap.Any("Headers", resp.Header))
	if reqBody != nil {
		fields = append(fields, zap.ByteString("Body", respBody))
	}
	g.base.Log.Debug("Response debug info", fields...)
}

func (g *ScenarioGun) answLogging(bodyBytes []byte, resp *http.Response, respBytes []byte, stepName string) {
	msg := fmt.Sprintf("REQUEST[%s]:\n%s\n", stepName, string(bodyBytes))
	g.base.AnswLog.Debug(msg)

	headers := ""
	var writer bytes.Buffer
	err := resp.Header.Write(&writer)
	if err == nil {
		headers = writer.String()
	} else {
		g.base.AnswLog.Error("error writing header", zap.Error(err))
	}

	msg = fmt.Sprintf("RESPONSE[%s]:\n%s %s\n%s\n%s\n", stepName, resp.Proto, resp.Status, headers, string(respBytes))
	g.base.AnswLog.Debug(msg)
}

func (g *ScenarioGun) answReqRespLogging(reqBytes []byte, resp *http.Response, respBytes []byte, stepName string) {
	switch g.base.Config.AnswLog.Filter {
	case "all":
		g.answLogging(reqBytes, resp, respBytes, stepName)
	case "warning":
		if resp.StatusCode >= 400 {
			g.answLogging(reqBytes, resp, respBytes, stepName)
		}
	case "error":
		if resp.StatusCode >= 500 {
			g.answLogging(reqBytes, resp, respBytes, stepName)
		}
	}
}

func (g *ScenarioGun) reportErr(sample *netsample.Sample, err error) {
	if err == nil {
		return
	}
	sample.AddTag(EmptyTag)
	sample.SetProtoCode(0)
	sample.SetErr(err)
	g.base.Aggregator.Report(sample)
}

func getHostWithoutPort(target string) string {
	host, _, err := net.SplitHostPort(target)
	if err != nil {
		host = target
	}
	return host
}
