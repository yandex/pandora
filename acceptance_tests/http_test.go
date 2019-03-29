package acceptance

import (
	"net/http"

	"golang.org/x/net/http2"

	"net/http/httptest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"go.uber.org/atomic"
)

var _ = Describe("http", func() {
	var (
		server *httptest.Server
		conf   *TestConfig
		tester *PandoraTester
	)
	ServerEndpoint := func() string {
		Expect(server).NotTo(BeNil())
		return server.Listener.Addr().String()
	}
	BeforeEach(func() {
		conf = NewTestConfig()
		tester = nil
	})
	JustBeforeEach(func() {
		if server != nil {
			Expect(server.URL).NotTo(BeEmpty(), "Please start server manually")
		}
		tester = NewTester(conf)
	})
	AfterEach(func() {
		tester.Close()
		if server != nil {
			server.Close()
			server = nil
		}
	})

	Context("uri ammo", func() {
		var (
			requetsCount atomic.Int64           // Request served by test server.
			gunConfig    map[string]interface{} // gunConfig config section.
		)
		const (
			Requests  = 4
			Instances = 2
			OutFile   = "out.log"
		)
		BeforeEach(func() {
			requetsCount.Store(0)
			server = httptest.NewUnstartedServer(http.HandlerFunc(
				func(rw http.ResponseWriter, req *http.Request) {
					requetsCount.Inc()
					rw.WriteHeader(http.StatusOK)
				}))

			conf.Pool[0].Gun = map[string]interface{}{
				// Set type in test.
				"target": ServerEndpoint(),
			}
			const ammoFile = "ammo.uri"
			conf.Pool[0].Provider = map[string]interface{}{
				"type": "uri",
				"file": ammoFile,
			}
			conf.Files[ammoFile] = "/"
			conf.Pool[0].Aggregator = map[string]interface{}{
				"type":        "phout",
				"destination": OutFile,
			}
			conf.Pool[0].RPSSchedule = []map[string]interface{}{
				{"type": "once", "times": Requests / 2},
				{"type": "const", "ops": Requests, "duration": "0.5s"},
			}
			conf.Pool[0].StartupSchedule = []map[string]interface{}{
				{"type": "once", "times": Instances},
			}
			gunConfig = conf.Pool[0].Gun
			conf.Level = "debug"
		})
		itOk := func() {
			It("ok", func() {
				exitCode := tester.ExitCode()
				Expect(exitCode).To(BeZero(), "Pandora finish execution with non zero code")
				Expect(requetsCount.Load()).To(BeEquivalentTo(Requests))
				// TODO(skipor): parse and check phout output
			})
		}

		Context("http", func() {
			BeforeEach(func() {
				server.Start()
				gunConfig["type"] = "http"
			})
			itOk()
		})

		Context("https", func() {
			BeforeEach(func() {
				server.StartTLS()
				gunConfig["type"] = "http"
				gunConfig["ssl"] = true
			})
			itOk()
		})

		Context("http2", func() {
			Context("target support HTTP/2", func() {
				BeforeEach(func() {
					startHTTP2(server)
					gunConfig["type"] = "http2"
				})
				itOk()
			})
			Context("target DOESN'T support HTTP/2", func() {
				BeforeEach(func() {
					server.StartTLS()
					gunConfig["type"] = "http2"
				})
				It("ok", func() {
					exitCode := tester.ExitCode()
					Expect(exitCode).NotTo(BeZero(), "Pandora should fail")
				})

			})
		})

	})
})

func startHTTP2(server *httptest.Server) {
	http2.ConfigureServer(server.Config, nil)
	server.TLS = server.Config.TLSConfig
	server.StartTLS()
}
