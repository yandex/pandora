package acceptance

import (
	"net/http"
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
		tester = NewTester(conf)
	})
	AfterEach(func() {
		tester.Close()
		if server != nil {
			server.Close()
			server = nil
		}
	})

	Context("jsonline", func() {
		var (
			requetsCount atomic.Int64
		)
		const (
			Requests  = 2
			Instances = 2
			OutFile   = "out.log"
		)
		BeforeEach(func() {
			server = httptest.NewServer(http.HandlerFunc(
				func(rw http.ResponseWriter, req *http.Request) {
					requetsCount.Inc()
					rw.WriteHeader(http.StatusOK)
				}))

			conf.Pool[0].Gun = map[string]interface{}{
				"type":   "http",
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
			conf.Pool[0].RPSSchedule = map[string]interface{}{
				"type":  "once",
				"times": Requests,
			}
			conf.Pool[0].StartupSchedule = map[string]interface{}{
				"type":  "once",
				"times": Instances,
			}
		})
		It("", func() {
			exitCode := tester.Session.Wait(5).ExitCode()
			Expect(exitCode).To(BeZero())
			Expect(requetsCount.Load()).To(BeEquivalentTo(Requests))
		})
	})

})
