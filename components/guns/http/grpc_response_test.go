package phttp

import (
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/yandex/pandora/components/answ/filter"
	ammomock "github.com/yandex/pandora/components/guns/http/mocks"
	"github.com/yandex/pandora/core/aggregator/netsample"
	"github.com/yandex/pandora/core/coretest"
	"github.com/yandex/pandora/core/warmup"
	"github.com/yandex/pandora/lib/answlog"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/grpc/codes"
)

type grpcTrailersOnEOF struct {
	reader  *strings.Reader
	trailer http.Header
	status  string
	message string
	details string
}

func (b *grpcTrailersOnEOF) Read(p []byte) (int, error) {
	n, err := b.reader.Read(p)
	if err == io.EOF && b.status != "" {
		b.trailer.Set("Grpc-Status", b.status)
		if b.message != "" {
			b.trailer.Set("Grpc-Message", b.message)
		}
		if b.details != "" {
			b.trailer.Set("Grpc-Status-Details-Bin", b.details)
		}
	}
	return n, err
}

func TestHTTP2GRPCAnswerLogIncludesFinalMetadata(t *testing.T) {
	cfg := DefaultHTTP2GunConfig()
	cfg.Target = "localhost:8080"
	cfg.TargetResolved = cfg.Target
	cfg.SSL = false
	cfg.ResponseCode = "grpc"
	gun, err := NewHTTP2Gun(cfg)
	require.NoError(t, err)

	trailer := make(http.Header)
	response := &http.Response{
		StatusCode: 200,
		Header:     http.Header{"Content-Type": []string{"application/grpc"}},
		Trailer:    trailer,
		Body: &grpcTrailersOnEOF{
			reader:  strings.NewReader("response body"),
			trailer: trailer,
			status:  "14",
			message: "quota%20exceeded",
			details: "CA4=",
		},
	}
	gun.Client = &testDecoratedClient{returnRes: response}
	core, logs := observer.New(zap.DebugLevel)
	gun.AnswLog = answlog.NewLogger(
		zap.New(core),
		&answlog.PassAllSampler{},
		filter.NewGRPCStatusCodeFilter(filter.FilterError),
		&answlog.PassAllMasker{},
		true,
		false,
	)
	var results netsample.TestAggregator
	require.NoError(t, gun.Bind(&results, testDeps()))
	gun.Shoot(newAmmoURL(t, "/"))

	require.Len(t, results.Samples, 1)
	require.Equal(t, 503, results.Samples[0].ProtoCode())
	require.Len(t, logs.All(), 1)
	fields := logs.All()[0].ContextMap()
	require.Equal(t, int64(14), fields["status"])
	require.Equal(t, "quota exceeded", fields["grpc_message"])
	require.Equal(t, "CA4=", fields["grpc_status_details_bin"])
	require.Contains(t, fields["grpc_status_details"], "code:14")
	require.Equal(t, "trailers", fields["grpc_status_source"])
	require.Equal(t, int64(200), fields["http_status"])
	require.Contains(t, fields["req"], "GET /")
	require.Contains(t, fields["resp"], "200")
}

func TestHTTP2GRPCAnswerLogKeepsDiagnosticsWithoutStatus(t *testing.T) {
	cfg := DefaultHTTP2GunConfig()
	cfg.Target = "localhost:8080"
	cfg.TargetResolved = cfg.Target
	cfg.SSL = false
	cfg.ResponseCode = "grpc"
	gun, err := NewHTTP2Gun(cfg)
	require.NoError(t, err)
	gun.Client = &testDecoratedClient{returnRes: &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/grpc"}},
		Trailer: http.Header{
			"Grpc-Message":            []string{"quota%20exceeded"},
			"Grpc-Status-Details-Bin": []string{"CA4="},
		},
		Body: io.NopCloser(strings.NewReader("response body")),
	}}
	core, logs := observer.New(zap.DebugLevel)
	gun.AnswLog = answlog.NewLogger(
		zap.New(core),
		&answlog.PassAllSampler{},
		filter.NewGRPCStatusCodeFilter(filter.FilterError),
		&answlog.PassAllMasker{},
		true,
		false,
	)
	var results netsample.TestAggregator
	require.NoError(t, gun.Bind(&results, testDeps()))
	gun.Shoot(newAmmoURL(t, "/"))

	require.Len(t, results.Samples, 1)
	require.Equal(t, 500, results.Samples[0].ProtoCode()) // Missing grpc-status uses HTTP fallback.
	require.Len(t, logs.All(), 1)
	fields := logs.All()[0].ContextMap()
	require.Equal(t, "quota exceeded", fields["grpc_message"])
	require.Equal(t, "CA4=", fields["grpc_status_details_bin"])
	require.Contains(t, fields["grpc_status_details"], "code:14")
	require.Equal(t, "http-fallback", fields["grpc_status_source"])

	gun.Client = &testDecoratedClient{returnRes: &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/grpc"}},
		Trailer: http.Header{
			"Grpc-Message":            []string{"broken%20metadata"},
			"Grpc-Status-Details-Bin": []string{"!!!"},
		},
		Body: io.NopCloser(strings.NewReader("response body")),
	}}
	gun.Shoot(newAmmoURL(t, "/"))
	require.Len(t, results.Samples, 2)
	require.Equal(t, 500, results.Samples[1].ProtoCode())
	require.Len(t, logs.All(), 2)
	fields = logs.All()[1].ContextMap()
	require.Equal(t, int64(codes.Internal), fields["status"])
	require.Equal(t, "broken metadata", fields["grpc_message"])
	require.Equal(t, "!!!", fields["grpc_status_details_bin"])
	require.Equal(t, "invalid-status-details", fields["grpc_status_source"])
}

func TestHTTP2GRPCRejectsConflictingStatusDetails(t *testing.T) {
	cfg := DefaultHTTP2GunConfig()
	cfg.Target = "localhost:8080"
	cfg.TargetResolved = cfg.Target
	cfg.SSL = false
	cfg.ResponseCode = "grpc"
	gun, err := NewHTTP2Gun(cfg)
	require.NoError(t, err)

	trailer := make(http.Header)
	gun.Client = &testDecoratedClient{returnRes: &http.Response{
		StatusCode: 200,
		Header:     http.Header{"Content-Type": []string{"application/grpc"}},
		Trailer:    trailer,
		Body: &grpcTrailersOnEOF{
			reader:  strings.NewReader("response body"),
			trailer: trailer,
			status:  "14",
			details: "CAA=", // google.rpc.Status{code: 0} contradicts grpc-status: 14.
		},
	}}
	var results netsample.TestAggregator
	require.NoError(t, gun.Bind(&results, testDeps()))
	gun.Shoot(newAmmoURL(t, "/"))
	require.Len(t, results.Samples, 1)
	require.Equal(t, 500, results.Samples[0].ProtoCode()) // gRPC INTERNAL.
}

type failingGRPCBody struct {
	err  error
	data []byte
	sent bool
}

func (b *failingGRPCBody) Read(p []byte) (int, error) {
	if !b.sent {
		b.sent = true
		return copy(p, b.data), nil
	}
	return 0, b.err
}
func (b *failingGRPCBody) Close() error { return nil }

func TestHTTP2GRPCBodyFailureIsNotSuccess(t *testing.T) {
	cfg := DefaultHTTP2GunConfig()
	cfg.Target = "localhost:8080"
	cfg.TargetResolved = cfg.Target
	cfg.SSL = false
	cfg.ResponseCode = "grpc"
	cfg.HTTPTrace.DumpEnabled = true
	gun, err := NewHTTP2Gun(cfg)
	require.NoError(t, err)
	core, logs := observer.New(zap.DebugLevel)
	gun.AnswLog = answlog.NewLogger(
		zap.New(core),
		&answlog.PassAllSampler{},
		filter.NewGRPCStatusCodeFilter(filter.FilterError),
		&answlog.PassAllMasker{},
		true,
		false,
	)

	readErr := errors.New("body failed")
	gun.Client = &testDecoratedClient{returnRes: &http.Response{
		StatusCode: 200,
		Header:     http.Header{"Content-Type": []string{"application/grpc"}},
		Trailer:    make(http.Header),
		Body:       &failingGRPCBody{err: readErr, data: []byte("partial response")},
	}}
	var results netsample.TestAggregator
	require.NoError(t, gun.Bind(&results, testDeps()))
	gun.Shoot(newAmmoURL(t, "/"))
	require.Len(t, results.Samples, 1)
	require.Equal(t, 500, results.Samples[0].ProtoCode())
	require.ErrorIs(t, results.Samples[0].Err(), readErr)
	require.GreaterOrEqual(t, results.Samples[0].GetResponseBytes(), len("partial response"))
	require.Len(t, logs.All(), 1)
	fields := logs.All()[0].ContextMap()
	require.Equal(t, int64(200), fields["http_status"])
	require.Equal(t, "body-read-error", fields["grpc_status_source"])
	require.Contains(t, fields["read_error"], "body failed")
}

type checkHeaderRTTBody struct {
	t       *testing.T
	sample  *netsample.Sample
	trailer http.Header
}

func (b *checkHeaderRTTBody) Read([]byte) (int, error) {
	require.Positive(b.t, b.sample.GetUserDurationMicroseconds(), "RTT must be set before reading the body")
	b.trailer.Set("Grpc-Status", "14")
	return 0, io.EOF
}

func (b *checkHeaderRTTBody) Close() error { return nil }

func TestHTTP2GRPCKeepsHeaderRTT(t *testing.T) {
	cfg := DefaultHTTP2GunConfig()
	cfg.Target = "localhost:8080"
	cfg.TargetResolved = cfg.Target
	cfg.SSL = false
	cfg.ResponseCode = "grpc"
	gun, err := NewHTTP2Gun(cfg)
	require.NoError(t, err)

	sample := netsample.Acquire("request")
	trailer := make(http.Header)
	gun.Client = &testDecoratedClient{
		before: func(*http.Request) { time.Sleep(time.Millisecond) },
		returnRes: &http.Response{
			StatusCode: 200,
			Header:     http.Header{"Content-Type": []string{"application/grpc"}},
			Trailer:    trailer,
			Body:       &checkHeaderRTTBody{t: t, sample: sample, trailer: trailer},
		},
	}
	req := httptest.NewRequest(http.MethodPost, "http://example.com/test", nil)
	ammo := ammomock.NewAmmo(t)
	ammo.On("IsInvalid").Return(false).Once()
	ammo.On("Request").Return(req, sample).Once()
	var results netsample.TestAggregator
	require.NoError(t, gun.Bind(&results, testDeps()))
	gun.Shoot(ammo)
	require.Len(t, results.Samples, 1)
	require.Equal(t, 503, results.Samples[0].ProtoCode())
}

func TestHTTP2GRPCH2CWithRealTrailers(t *testing.T) {
	tests := []struct {
		name          string
		trailersOnly  bool
		headerTimeout bool
		wantCode      int
	}{
		{name: "status after response body", wantCode: 503},
		{name: "trailers-only", trailersOnly: true, wantCode: 403},
		{name: "trailers-only with response header timeout", trailersOnly: true, headerTimeout: true, wantCode: 403},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(h2c.NewHandler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/grpc")
				if tt.trailersOnly {
					w.Header().Set("Grpc-Status", "7")
					w.WriteHeader(http.StatusOK)
					return
				}
				w.Header().Set("Trailer", "Grpc-Status")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("response body"))
				w.Header().Set("Grpc-Status", "14")
			}), &http2.Server{}))
			defer server.Close()

			cfg := DefaultHTTP2GunConfig()
			cfg.Target = server.Listener.Addr().String()
			cfg.TargetResolved = cfg.Target
			cfg.SSL = false
			cfg.ResponseCode = "grpc"
			if tt.headerTimeout {
				cfg.Client.Transport.ResponseHeaderTimeout = time.Second
				cfg.HTTPTrace.DumpEnabled = true
			}
			gun, err := NewHTTP2Gun(cfg)
			require.NoError(t, err)
			var results netsample.TestAggregator
			require.NoError(t, gun.Bind(&results, testDeps()))
			gun.Shoot(newAmmoURL(t, "/service/method"))
			require.Len(t, results.Samples, 1)
			require.NoError(t, results.Samples[0].Err())
			require.Equal(t, tt.wantCode, results.Samples[0].ProtoCode())
			if tt.headerTimeout {
				require.Positive(t, results.Samples[0].GetResponseBytes(), "trailers-only response has headers")
			}
		})
	}
}

func TestHTTP2GRPCInvalidBinaryTrailerIsInternal(t *testing.T) {
	for _, status := range []string{"0", "3"} {
		t.Run("grpc-status="+status, func(t *testing.T) {
			server := httptest.NewServer(h2c.NewHandler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/grpc")
				w.Header().Set("Trailer", "Grpc-Status, Grpc-Status-Details-Bin")
				w.WriteHeader(http.StatusOK)
				w.Header().Set("Grpc-Status", status)
				w.Header().Set("Grpc-Status-Details-Bin", "!!!")
			}), &http2.Server{}))
			defer server.Close()

			cfg := DefaultHTTP2GunConfig()
			cfg.Target = server.Listener.Addr().String()
			cfg.TargetResolved = cfg.Target
			cfg.SSL = false
			cfg.ResponseCode = "grpc"
			gun, err := NewHTTP2Gun(cfg)
			require.NoError(t, err)
			core, logs := observer.New(zap.DebugLevel)
			gun.AnswLog = answlog.NewLogger(
				zap.New(core),
				&answlog.PassAllSampler{},
				filter.NewGRPCStatusCodeFilter(filter.FilterError),
				&answlog.PassAllMasker{},
				true,
				false,
			)
			var results netsample.TestAggregator
			require.NoError(t, gun.Bind(&results, testDeps()))
			gun.Shoot(newAmmoURL(t, "/service/method"))

			require.Len(t, results.Samples, 1)
			require.NoError(t, results.Samples[0].Err())
			require.Equal(t, 500, results.Samples[0].ProtoCode())
			require.Len(t, logs.All(), 1)
			fields := logs.All()[0].ContextMap()
			require.Equal(t, int64(codes.Internal), fields["status"])
			require.Equal(t, status, fields["grpc_status_raw"])
			require.Equal(t, "!!!", fields["grpc_status_details_bin"])
			require.Equal(t, "invalid-status-details", fields["grpc_status_source"])
		})
	}
}

func TestHTTP2GRPCTrailersOnlyTLS(t *testing.T) {
	server := newHTTP2TestServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/grpc")
		w.Header().Set("Grpc-Status", "7")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	cfg := DefaultHTTP2GunConfig()
	cfg.Target = server.Listener.Addr().String()
	cfg.TargetResolved = cfg.Target
	cfg.ResponseCode = "grpc"
	gun, err := NewHTTP2Gun(cfg)
	require.NoError(t, err)
	var results netsample.TestAggregator
	require.NoError(t, gun.Bind(&results, testDeps()))
	gun.Shoot(newAmmoURL(t, "/service/method"))
	require.Len(t, results.Samples, 1)
	require.NoError(t, results.Samples[0].Err())
	require.Equal(t, 403, results.Samples[0].ProtoCode())
}

func TestHTTP2GRPCConfigRejectsUnsupportedCombinations(t *testing.T) {
	cfg := DefaultHTTP2GunConfig()
	cfg.Target = "localhost:8080"
	cfg.ResponseCode = "unknown"
	_, err := NewHTTP2Gun(cfg)
	require.ErrorContains(t, err, "response-code")

	cfg.ResponseCode = "grpc"
	http1Gun := NewHTTP1Gun(cfg)
	err = http1Gun.Bind(&netsample.TestAggregator{}, testDeps())
	require.ErrorContains(t, err, "http2")

	cfg.ResponseCode = "http"
	_, err = NewHTTP2Gun(cfg)
	require.NoError(t, err)
	http1Gun = NewHTTP1Gun(cfg)
	require.NoError(t, http1Gun.Bind(&netsample.TestAggregator{}, testDeps()))
}

func TestGRPCResponseMetadataEdges(t *testing.T) {
	tests := []struct {
		name        string
		trailer     http.Header
		wantCode    codes.Code
		wantMessage string
	}{
		{
			name: "duplicate details do not override the wire status",
			trailer: http.Header{
				"Grpc-Status":             []string{"14"},
				"Grpc-Status-Details-Bin": []string{"CAA=", "CA4="},
			},
			wantCode: codes.Unavailable,
		},
		{
			name: "comma-joined binary details are equivalent to separate values",
			trailer: http.Header{
				"Grpc-Status":             []string{"14"},
				"Grpc-Status-Details-Bin": []string{"CAA=,CA4="},
			},
			wantCode: codes.Unavailable,
		},
		{
			name: "decode valid percent escapes around an invalid one",
			trailer: http.Header{
				"Grpc-Status":  []string{"14"},
				"Grpc-Message": []string{"quota%20full%ZZ"},
			},
			wantCode:    codes.Unavailable,
			wantMessage: "quota full%ZZ",
		},
		{
			name: "details without a code field preserve the wire status",
			trailer: http.Header{
				"Grpc-Status": []string{"14"},
				"Grpc-Status-Details-Bin": []string{base64.StdEncoding.EncodeToString(
					[]byte{0x12, 0x05, 'q', 'u', 'o', 't', 'a'}, // google.rpc.Status{message: "quota"}
				)},
			},
			wantCode: codes.Unavailable,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := readGRPCResponse(&http.Response{
				StatusCode: 200,
				Header:     http.Header{"Content-Type": []string{"application/grpc"}},
				Trailer:    tt.trailer,
			}, 1)
			require.Equal(t, tt.wantCode, got.code)
			require.Equal(t, tt.wantMessage, decodeGRPCMessage(got.messageRaw))
		})
	}
}

func TestGRPCStatusInNonterminalInitialHeadersIsIgnored(t *testing.T) {
	tests := []struct {
		name     string
		response *http.Response
	}{
		{
			name: "unknown content length",
			response: &http.Response{
				StatusCode:    200,
				Header:        http.Header{"Content-Type": []string{"application/grpc"}, "Grpc-Status": []string{"0"}},
				ContentLength: -1, // Initial HEADERS did not end the stream.
			},
		},
		{
			name: "declared empty body and a later trailer",
			response: &http.Response{
				StatusCode:    200,
				Header:        http.Header{"Content-Type": []string{"application/grpc"}, "Grpc-Status": []string{"0"}, "Content-Length": []string{"0"}},
				ContentLength: 0,
				Trailer:       http.Header{"Grpc-Message": []string{"late"}},
			},
		},
		{
			name: "declared empty body ending in DATA",
			response: &http.Response{
				StatusCode:    200,
				Header:        http.Header{"Content-Type": []string{"application/grpc"}, "Grpc-Status": []string{"0"}, "Content-Length": []string{"0"}},
				ContentLength: 0,
				Body:          io.NopCloser(strings.NewReader("")),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := readGRPCResponse(tt.response, 0)
			require.Equal(t, codes.Unknown, got.code)
			require.Equal(t, "http-fallback", got.source)
		})
	}
}

func TestHTTP2GRPCRejectsNonGRPCContentType(t *testing.T) {
	cfg := DefaultHTTP2GunConfig()
	cfg.Target = "localhost:8080"
	cfg.TargetResolved = cfg.Target
	cfg.SSL = false
	cfg.ResponseCode = "grpc"
	gun, err := NewHTTP2Gun(cfg)
	require.NoError(t, err)
	gun.Client = &testDecoratedClient{returnRes: &http.Response{
		StatusCode: 200,
		Header:     http.Header{"Content-Type": []string{"text/plain"}},
		Trailer:    http.Header{"Grpc-Status": []string{"0"}},
		Body:       http.NoBody,
	}}
	var results netsample.TestAggregator
	require.NoError(t, gun.Bind(&results, testDeps()))
	gun.Shoot(newAmmoURL(t, "/"))
	require.Len(t, results.Samples, 1)
	require.Equal(t, 500, results.Samples[0].ProtoCode())
}

func TestHTTP2GRPCResponseBytesIncludesTrailers(t *testing.T) {
	cfg := DefaultHTTP2GunConfig()
	cfg.Target = "localhost:8080"
	cfg.TargetResolved = cfg.Target
	cfg.SSL = false
	cfg.ResponseCode = "grpc"
	cfg.HTTPTrace.DumpEnabled = true
	gun, err := NewHTTP2Gun(cfg)
	require.NoError(t, err)

	response := &http.Response{
		StatusCode: 200,
		Status:     "200 OK",
		Proto:      "HTTP/2.0",
		ProtoMajor: 2,
		Header:     http.Header{"Content-Type": []string{"application/grpc"}},
		Trailer: http.Header{
			"Grpc-Status":             []string{"14"},
			"Grpc-Status-Details-Bin": []string{"CA4="},
		},
		Body: http.NoBody,
	}
	withoutTrailers, err := httputil.DumpResponse(response, false)
	require.NoError(t, err)
	gun.Client = &testDecoratedClient{returnRes: response}
	var results netsample.TestAggregator
	require.NoError(t, gun.Bind(&results, testDeps()))
	gun.Shoot(newAmmoURL(t, "/"))
	require.Len(t, results.Samples, 1)
	require.Greater(t, results.Samples[0].GetResponseBytes(), len(withoutTrailers))
}

func TestHTTP2ResponseCodeConfigDecode(t *testing.T) {
	tests := []struct {
		name string
		yaml string
		want string
	}{
		{name: "omitted uses HTTP", yaml: "target: localhost:8095\n", want: "http"},
		{name: "explicit HTTP", yaml: "target: localhost:8095\nresponse-code: http\n", want: "http"},
		{name: "gRPC", yaml: "target: localhost:8095\nresponse-code: grpc\n", want: "grpc"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultHTTP2GunConfig()
			coretest.DecodeAndValidateT(t, tt.yaml, &cfg)
			require.Equal(t, tt.want, cfg.ResponseCode)
		})
	}
}

func TestHTTP2GRPCAnswerLogFilterUsesOriginalStatus(t *testing.T) {
	tests := []struct {
		name       string
		grpcStatus string
		wantLog    bool
	}{
		{name: "unavailable is an error", grpcStatus: "14", wantLog: true},
		{name: "unknown numeric status is not in the native error filter", grpcStatus: "99"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultHTTP2GunConfig()
			cfg.Target = "localhost:8080"
			cfg.TargetResolved = cfg.Target
			cfg.SSL = false
			cfg.ResponseCode = "grpc"
			cfg.AnswLog.Enabled = true
			cfg.AnswLog.Path = filepath.Join(t.TempDir(), "answ.log")
			cfg.AnswLog.Filter = filter.FilterError
			cfg.AnswLog.Sampling.Enabled = false
			gun, err := NewHTTP2Gun(cfg)
			require.NoError(t, err)
			shared, err := gun.WarmUp(&warmup.Options{})
			require.NoError(t, err)
			deps := testDeps()
			deps.Shared = shared
			gun.Client = &testDecoratedClient{returnRes: &http.Response{
				StatusCode: 200,
				Header:     http.Header{"Content-Type": []string{"application/grpc"}},
				Trailer:    http.Header{"Grpc-Status": []string{tt.grpcStatus}},
				Body:       http.NoBody,
			}}
			var results netsample.TestAggregator
			require.NoError(t, gun.Bind(&results, deps))
			gun.Shoot(newAmmoURL(t, "/"))
			data, err := os.ReadFile(cfg.AnswLog.Path)
			require.NoError(t, err)
			if tt.wantLog {
				require.Contains(t, string(data), tt.grpcStatus)
			} else {
				require.Empty(t, data)
			}
		})
	}
}

func (b *grpcTrailersOnEOF) Close() error { return nil }

func TestHTTP2GRPCResponseCodes(t *testing.T) {
	tests := []struct {
		name         string
		httpCode     int
		grpcStatus   string
		trailersOnly bool
		wantCode     int
	}{
		{name: "OK from trailers", httpCode: 200, grpcStatus: "0", wantCode: 200},
		{name: "Unavailable from trailers", httpCode: 200, grpcStatus: "14", wantCode: 503},
		{name: "PermissionDenied from trailers-only", httpCode: 200, grpcStatus: "7", trailersOnly: true, wantCode: 403},
		{name: "HTTP 503 is not masked by gRPC OK", httpCode: 503, grpcStatus: "0", wantCode: 503},
		{name: "HTTP 503 without gRPC status", httpCode: 503, wantCode: 503},
		{name: "HTTP 404 without gRPC status", httpCode: 404, wantCode: 501},
		{name: "HTTP 200 without gRPC status", httpCode: 200, wantCode: 500},
		{name: "unknown numeric gRPC status", httpCode: 200, grpcStatus: "99", wantCode: 500},
		{name: "malformed gRPC status", httpCode: 200, grpcStatus: "broken", wantCode: 500},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultHTTP2GunConfig()
			cfg.Target = "localhost:8080"
			cfg.TargetResolved = cfg.Target
			cfg.SSL = false
			cfg.ResponseCode = "grpc"
			gun, err := NewHTTP2Gun(cfg)
			require.NoError(t, err)

			response := &http.Response{
				StatusCode: tt.httpCode,
				Header:     http.Header{"Content-Type": []string{"application/grpc"}},
				Trailer:    make(http.Header),
			}
			if tt.trailersOnly {
				response.Header.Set("Grpc-Status", tt.grpcStatus)
				response.Body = http.NoBody
			} else {
				response.Body = &grpcTrailersOnEOF{
					reader:  strings.NewReader("response body"),
					trailer: response.Trailer,
					status:  tt.grpcStatus,
				}
			}
			gun.Client = &testDecoratedClient{returnRes: response}

			var results netsample.TestAggregator
			require.NoError(t, gun.Bind(&results, testDeps()))
			gun.Shoot(newAmmoURL(t, "/"))
			require.Len(t, results.Samples, 1)
			require.Equal(t, tt.wantCode, results.Samples[0].ProtoCode())
		})
	}
}

func TestHTTP2DefaultStillReportsHTTPStatus(t *testing.T) {
	cfg := DefaultHTTP2GunConfig()
	cfg.Target = "localhost:8080"
	cfg.TargetResolved = cfg.Target
	cfg.SSL = false
	gun, err := NewHTTP2Gun(cfg)
	require.NoError(t, err)

	response := &http.Response{
		StatusCode: http.StatusOK,
		Body:       http.NoBody,
		Header:     http.Header{"Grpc-Status": []string{"14"}},
	}
	gun.Client = &testDecoratedClient{returnRes: response}
	var results netsample.TestAggregator
	require.NoError(t, gun.Bind(&results, testDeps()))
	gun.Shoot(newAmmoURL(t, "/"))
	require.Len(t, results.Samples, 1)
	require.Equal(t, http.StatusOK, results.Samples[0].ProtoCode())
}
