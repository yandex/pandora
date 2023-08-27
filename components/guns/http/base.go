// Copyright (c) 2017 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package phttp

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptrace"
	"net/http/httputil"
	"net/url"

	"github.com/pkg/errors"
	"github.com/yandex/pandora/core"
	"github.com/yandex/pandora/core/aggregator/netsample"
	"go.uber.org/zap"
)

const (
	EmptyTag = "__EMPTY__"
)

type BaseGunConfig struct {
	AutoTag   AutoTagConfig   `config:"auto-tag"`
	AnswLog   AnswLogConfig   `config:"answlog"`
	HTTPTrace HTTPTraceConfig `config:"httptrace"`
}

// AutoTagConfig configure automatic tags generation based on ammo URI. First AutoTag URI path elements becomes tag.
// Example: /my/very/deep/page?id=23&param=33 -> /my/very when uri-elements: 2.
type AutoTagConfig struct {
	Enabled     bool `config:"enabled"`
	URIElements int  `config:"uri-elements" validate:"min=1"` // URI elements used to autotagging
	NoTagOnly   bool `config:"no-tag-only"`                   // When true, autotagged only ammo that has no tag before.
}

type AnswLogConfig struct {
	Enabled bool   `config:"enabled"`
	Path    string `config:"path"`
	Filter  string `config:"filter" valid:"oneof=all warning error"`
}

type HTTPTraceConfig struct {
	DumpEnabled  bool `config:"dump"`
	TraceEnabled bool `config:"trace"`
}

func DefaultBaseGunConfig() BaseGunConfig {
	return BaseGunConfig{
		AutoTagConfig{
			Enabled:     false,
			URIElements: 2,
			NoTagOnly:   true,
		},
		AnswLogConfig{
			Enabled: false,
			Path:    "answ.log",
			Filter:  "error",
		},
		HTTPTraceConfig{
			DumpEnabled:  false,
			TraceEnabled: false,
		},
	}
}

type BaseGun struct {
	DebugLog   bool // Automaticaly set in Bind if Log accepts debug messages.
	Config     BaseGunConfig
	Do         func(r *http.Request) (*http.Response, error) // Required.
	Connect    func(ctx context.Context) error               // Optional hook.
	OnClose    func() error                                  // Optional. Called on Close().
	Aggregator netsample.Aggregator                          // Lazy set via BindResultTo.
	AnswLog    *zap.Logger
	core.GunDeps
}

var _ Gun = (*BaseGun)(nil)
var _ io.Closer = (*BaseGun)(nil)

func (b *BaseGun) Bind(aggregator netsample.Aggregator, deps core.GunDeps) error {
	log := deps.Log
	if ent := log.Check(zap.DebugLevel, "Gun bind"); ent != nil {
		// Enable debug level logging during shooting. Creating log entries isn't free.
		b.DebugLog = true
	}

	if b.Aggregator != nil {
		log.Panic("already binded")
	}
	if aggregator == nil {
		log.Panic("nil aggregator")
	}
	b.Aggregator = aggregator
	b.GunDeps = deps

	return nil
}

// Shoot is thread safe iff Do and Connect hooks are thread safe.
func (b *BaseGun) Shoot(ammo Ammo) {
	var bodyBytes []byte
	if b.Aggregator == nil {
		zap.L().Panic("must bind before shoot")
	}
	if b.Connect != nil {
		err := b.Connect(b.Ctx)
		if err != nil {
			b.Log.Warn("Connect fail", zap.Error(err))
			return
		}
	}

	req, sample := ammo.Request()
	if ammo.IsInvalid() {
		sample.AddTag(EmptyTag)
		sample.SetProtoCode(0)
		b.Aggregator.Report(sample)
		b.Log.Warn("Invalid ammo", zap.Uint64("request", ammo.ID()))
		return
	}
	if b.DebugLog {
		b.Log.Debug("Prepared ammo to shoot", zap.Stringer("url", req.URL))
	}
	if b.Config.AutoTag.Enabled && (!b.Config.AutoTag.NoTagOnly || sample.Tags() == "") {
		sample.AddTag(autotag(b.Config.AutoTag.URIElements, req.URL))
	}
	if sample.Tags() == "" {
		sample.AddTag(EmptyTag)
	}
	if b.Config.AnswLog.Enabled {
		bodyBytes = GetBody(req)
	}

	var err error
	defer func() {
		if err != nil {
			sample.SetErr(err)
		}
		b.Aggregator.Report(sample)
		err = errors.WithStack(err)
	}()

	var timings *TraceTimings
	if b.Config.HTTPTrace.TraceEnabled {
		var clientTracer *httptrace.ClientTrace
		clientTracer, timings = CreateHTTPTrace()
		req = req.WithContext(httptrace.WithClientTrace(req.Context(), clientTracer))
	}
	if b.Config.HTTPTrace.DumpEnabled {
		requestDump, err := httputil.DumpRequest(req, true)
		if err != nil {
			b.Log.Error("DumpRequest error", zap.Error(err))
		} else {
			sample.SetRequestBytes(len(requestDump))
		}
	}
	var res *http.Response
	res, err = b.Do(req)
	if b.Config.HTTPTrace.TraceEnabled && timings != nil {
		sample.SetReceiveTime(timings.GetReceiveTime())
	}
	if b.Config.HTTPTrace.DumpEnabled && res != nil {
		responseDump, err := httputil.DumpResponse(res, true)
		if err != nil {
			b.Log.Error("DumpResponse error", zap.Error(err))
		} else {
			sample.SetResponseBytes(len(responseDump))
		}
	}
	if b.Config.HTTPTrace.TraceEnabled && timings != nil {
		sample.SetConnectTime(timings.GetConnectTime())
		sample.SetSendTime(timings.GetSendTime())
		sample.SetLatency(timings.GetLatency())
	}

	if err != nil {
		b.Log.Warn("Request fail", zap.Error(err))
		return
	}

	if b.DebugLog {
		b.verboseLogging(res)
	}
	if b.Config.AnswLog.Enabled {
		switch b.Config.AnswLog.Filter {
		case "all":
			b.answLogging(req, bodyBytes, res)

		case "warning":
			if res.StatusCode >= 400 {
				b.answLogging(req, bodyBytes, res)
			}

		case "error":
			if res.StatusCode >= 500 {
				b.answLogging(req, bodyBytes, res)
			}
		}
	}

	sample.SetProtoCode(res.StatusCode)
	defer res.Body.Close()
	// TODO: measure body read time
	_, err = io.Copy(ioutil.Discard, res.Body) // Buffers are pooled for ioutil.Discard
	if err != nil {
		b.Log.Warn("Body read fail", zap.Error(err))
		return
	}
}

func (b *BaseGun) Close() error {
	if b.OnClose != nil {
		return b.OnClose()
	}
	return nil
}

func (b *BaseGun) verboseLogging(res *http.Response) {
	if res.Request.Body != nil {
		reqBody, err := ioutil.ReadAll(res.Request.Body)
		if err != nil {
			b.Log.Debug("Body read failed for verbose logging of Request")
		} else {
			b.Log.Debug("Request body", zap.ByteString("Body", reqBody))
		}
	}
	b.Log.Debug(
		"Request debug info",
		zap.String("URL", res.Request.URL.String()),
		zap.String("Host", res.Request.Host),
		zap.Any("Headers", res.Request.Header),
	)

	if res.Body != nil {
		respBody, err := ioutil.ReadAll(res.Body)
		if err != nil {
			b.Log.Debug("Body read failed for verbose logging of Response")
		} else {
			b.Log.Debug("Response body", zap.ByteString("Body", respBody))
		}
	}
	b.Log.Debug(
		"Response debug info",
		zap.Int("Status Code", res.StatusCode),
		zap.String("Status", res.Status),
		zap.Any("Headers", res.Header),
	)
}

func (b *BaseGun) answLogging(req *http.Request, bodyBytes []byte, res *http.Response) {
	isBody := false
	if bodyBytes != nil {
		req.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))
		isBody = true
	}
	dump, err := httputil.DumpRequestOut(req, isBody)
	if err != nil {
		zap.L().Error("Error dumping request: %s", zap.Error(err))
	}
	msg := fmt.Sprintf("REQUEST:\n%s\n\n", string(dump))
	b.AnswLog.Debug(msg)

	dump, err = httputil.DumpResponse(res, true)
	if err != nil {
		zap.L().Error("Error dumping response: %s", zap.Error(err))
	}
	msg = fmt.Sprintf("RESPONSE:\n%s", string(dump))
	b.AnswLog.Debug(msg)
}

func autotag(depth int, URL *url.URL) string {
	path := URL.Path
	var ind int
	for ; ind < len(path); ind++ {
		if path[ind] == '/' {
			if depth == 0 {
				break
			}
			depth--
		}
	}
	return path[:ind]
}

func GetBody(req *http.Request) []byte {
	if req.Body != nil && req.Body != http.NoBody {
		bodyBytes, _ := ioutil.ReadAll(req.Body)
		req.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))
		return bodyBytes
	}

	return nil

}
