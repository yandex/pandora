package phttp

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	ammomock "github.com/yandex/pandora/components/guns/http/mocks"
	"github.com/yandex/pandora/core/aggregator/netsample"
	"github.com/yandex/pandora/core/config"
	"go.uber.org/atomic"
	"go.uber.org/zap"
	"golang.org/x/net/http2"
)

func TestBaseGun_GunClientConfig_decode(t *testing.T) {
	conf := DefaultHTTPGunConfig()
	data := map[interface{}]interface{}{
		"target": "test-trbo01e.haze.yandex.net:3000",
	}
	err := config.DecodeAndValidate(data, &conf)
	require.NoError(t, err)
}

func TestBaseGun_integration(t *testing.T) {
	const host = "example.com"
	const path = "/smth"
	expectedReq, err := http.NewRequest("GET", "http://"+host+path, nil)
	expectedReq.Host = "" // Important. Ammo may have empty host.
	require.NoError(t, err)
	var actualReq *http.Request
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.WriteHeader(http.StatusOK)
		actualReq = req
	}))
	defer server.Close()
	log := zap.NewNop()
	conf := DefaultHTTPGunConfig()
	conf.Target = host + ":80"
	targetResolved := strings.TrimPrefix(server.URL, "http://")
	results := &netsample.TestAggregator{}
	conf.TargetResolved = targetResolved
	httpGun := NewHTTP1Gun(conf, log)
	_ = httpGun.Bind(results, testDeps())

	am := newAmmoReq(t, expectedReq)
	httpGun.Shoot(am)
	require.NoError(t, results.Samples[0].Err())
	require.NotNil(t, actualReq)

	require.Equal(t, actualReq.Method, "GET")
	require.Equal(t, actualReq.Proto, "HTTP/1.1")
	require.Equal(t, actualReq.Host, host)
	require.NotNil(t, actualReq.URL)
	require.Empty(t, actualReq.URL.Host)
	require.Equal(t, actualReq.URL.Path, path)
}

func TestHTTP(t *testing.T) {
	tests := []struct {
		name  string
		https bool
	}{
		{
			name:  "http ok",
			https: false,
		},
		{
			name:  "https ok",
			https: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var isServed atomic.Bool
			server := httptest.NewUnstartedServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
				require.Empty(t, req.Header.Get("Accept-Encoding"))
				rw.WriteHeader(http.StatusOK)
				isServed.Store(true)
			}))
			if tt.https {
				server.StartTLS()
			} else {
				server.Start()
			}
			defer server.Close()
			log := zap.NewNop()
			conf := DefaultHTTPGunConfig()
			conf.Target = server.Listener.Addr().String()
			conf.SSL = tt.https
			conf.TargetResolved = conf.Target
			gun := NewHTTP1Gun(conf, log)
			var aggr netsample.TestAggregator
			_ = gun.Bind(&aggr, testDeps())
			gun.Shoot(newAmmoURL(t, "/"))

			require.Equal(t, len(aggr.Samples), 1)
			require.Equal(t, aggr.Samples[0].ProtoCode(), http.StatusOK)
			require.True(t, isServed.Load())
		})
	}
}

func TestHTTP_Redirect(t *testing.T) {
	tests := []struct {
		name     string
		redirect bool
	}{
		{
			name:     "not follow redirects by default",
			redirect: false,
		},
		{
			name:     "follow redirects if option set",
			redirect: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
				if req.URL.Path == "/redirect" {
					rw.Header().Add("Location", "/")
					rw.WriteHeader(http.StatusMovedPermanently)
				} else {
					rw.WriteHeader(http.StatusOK)
				}
			}))
			defer server.Close()
			log := zap.NewNop()
			conf := DefaultHTTPGunConfig()
			conf.Target = server.Listener.Addr().String()
			conf.Client.Redirect = tt.redirect
			conf.TargetResolved = conf.Target
			gun := NewHTTP1Gun(conf, log)
			var aggr netsample.TestAggregator
			_ = gun.Bind(&aggr, testDeps())
			gun.Shoot(newAmmoURL(t, "/redirect"))

			require.Equal(t, len(aggr.Samples), 1)
			expectedCode := http.StatusMovedPermanently
			if tt.redirect {
				expectedCode = http.StatusOK
			}
			require.Equal(t, aggr.Samples[0].ProtoCode(), expectedCode)
		})
	}
}

func TestHTTP_notSupportHTTP2(t *testing.T) {
	server := newHTTP2TestServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if isHTTP2Request(req) {
			rw.WriteHeader(http.StatusForbidden)
		} else {
			rw.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	// Test, that configured server serves HTTP2 well.
	http2OnlyClient := http.Client{
		Transport: &http2.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}}
	res, err := http2OnlyClient.Get(server.URL)
	require.NoError(t, err)
	require.Equal(t, res.StatusCode, http.StatusForbidden)

	log := zap.NewNop()
	conf := DefaultHTTPGunConfig()
	conf.Target = server.Listener.Addr().String()
	conf.SSL = true
	conf.TargetResolved = conf.Target
	gun := NewHTTP1Gun(conf, log)
	var results netsample.TestAggregator
	_ = gun.Bind(&results, testDeps())
	gun.Shoot(newAmmoURL(t, "/"))

	require.Equal(t, len(results.Samples), 1)
	require.Equal(t, results.Samples[0].ProtoCode(), http.StatusOK)
}

func TestHTTP2(t *testing.T) {
	t.Run("HTTP/2 ok", func(t *testing.T) {
		server := newHTTP2TestServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			if isHTTP2Request(req) {
				rw.WriteHeader(http.StatusOK)
			} else {
				rw.WriteHeader(http.StatusForbidden)
			}
		}))
		defer server.Close()
		log := zap.NewNop()
		conf := DefaultHTTP2GunConfig()
		conf.Target = server.Listener.Addr().String()
		conf.TargetResolved = conf.Target
		gun, _ := NewHTTP2Gun(conf, log)
		var results netsample.TestAggregator
		_ = gun.Bind(&results, testDeps())
		gun.Shoot(newAmmoURL(t, "/"))
		require.Equal(t, results.Samples[0].ProtoCode(), http.StatusOK)
	})

	t.Run("HTTP/1.1 panic", func(t *testing.T) {
		server := httptest.NewTLSServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			zap.S().Info("Served")
		}))
		defer server.Close()
		log := zap.NewNop()
		conf := DefaultHTTP2GunConfig()
		conf.Target = server.Listener.Addr().String()
		conf.TargetResolved = conf.Target
		gun, _ := NewHTTP2Gun(conf, log)
		var results netsample.TestAggregator
		_ = gun.Bind(&results, testDeps())
		var r interface{}
		func() {
			defer func() {
				r = recover()
			}()
			gun.Shoot(newAmmoURL(t, "/"))
		}()
		require.NotNil(t, r)
		require.Contains(t, r, notHTTP2PanicMsg)
	})

	t.Run("no SSL construction fails", func(t *testing.T) {
		server := httptest.NewTLSServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			zap.S().Info("Served")
		}))
		defer server.Close()
		log := zap.NewNop()
		conf := DefaultHTTP2GunConfig()
		conf.Target = server.Listener.Addr().String()
		conf.SSL = false
		conf.TargetResolved = conf.Target
		_, err := NewHTTP2Gun(conf, log)
		require.Error(t, err)
	})
}

func newAmmoURL(t *testing.T, url string) Ammo {
	req, err := http.NewRequest("GET", url, nil)
	require.NoError(t, err)
	return newAmmoReq(t, req)
}

func newAmmoReq(t *testing.T, req *http.Request) Ammo {
	ammo := ammomock.NewAmmo(t)
	ammo.On("IsInvalid").Return(false).Once()
	ammo.On("Request").Return(req, netsample.Acquire("REQUEST")).Once()
	return ammo
}

func isHTTP2Request(req *http.Request) bool {
	return checkHTTP2(req.TLS) == nil
}

func newHTTP2TestServer(handler http.Handler) *httptest.Server {
	server := httptest.NewUnstartedServer(handler)
	_ = http2.ConfigureServer(server.Config, nil)
	server.TLS = server.Config.TLSConfig // StartTLS takes TLS configuration from that field.
	server.StartTLS()
	return server
}
