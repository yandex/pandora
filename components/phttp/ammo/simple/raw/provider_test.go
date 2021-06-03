package raw

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	"github.com/spf13/afero"

	"a.yandex-team.ru/load/projects/pandora/components/phttp/ammo/simple"
	"a.yandex-team.ru/load/projects/pandora/core"
)

const testFile = "./ammo.stpd"
const testFileData = "../../../testdata/ammo.stpd"

var testData = []ammoData{
	{"GET", "www.ya.ru", "/", http.Header{"Connection": []string{"close"}}, "", ""},
	{"GET", "www.ya.ru", "/test", http.Header{"Connection": []string{"close"}}, "", ""},
	{"GET", "www.ya.ru", "/test2", http.Header{"Connection": []string{"close"}}, "tag", ""},
	{"POST", "www.ya.ru", "/test3", http.Header{"Connection": []string{"close"}, "Content-Length": []string{"5"}}, "tag", "hello"},
}

type ammoData struct {
	method string
	host   string
	path   string
	header http.Header
	tag    string
	body   string
}

var testFs = func() afero.Fs {
	testFs := afero.NewOsFs()
	testFileData, err := testFs.Open(testFileData)
	if err != nil {
		panic(err)
	}
	testDataBuffer, _ := ioutil.ReadAll(testFileData)
	testFileData.Read(testDataBuffer)
	fs := afero.NewMemMapFs()
	err = afero.WriteFile(fs, testFile, testDataBuffer, 0)
	if err != nil {
		panic(err)
	}
	return afero.NewReadOnlyFs(fs)
}()

func newTestProvider(conf Config) *Provider {
	return NewProvider(testFs, conf)
}

var _ = Describe("provider start", func() {
	It("ok", func() {
		p := newTestProvider(Config{File: testFile})
		ctx, cancel := context.WithCancel(context.Background())
		errch := make(chan error)
		go func() { errch <- p.Run(ctx, core.ProviderDeps{}) }()
		Expect(errch).NotTo(Receive())
		cancel()
		var err error
		Eventually(errch).Should(Receive(&err))
		Expect(err).To(Equal(ctx.Err()))
	})

	It("fail", func() {
		p := newTestProvider(Config{File: "no_such_file"})
		Expect(p.Run(context.Background(), core.ProviderDeps{}), core.ProviderDeps{}).NotTo(BeNil())
	})
})
var _ = Describe("provider decode", func() {
	var (
		// Configured in BeforeEach.
		conf             Config // File always is testFile.
		expectedStartErr error
		successReceives  int // How much ammo should be received.

		provider *Provider
		cancel   context.CancelFunc
		errch    chan error
		ammos    []*simple.Ammo
	)

	BeforeEach(func() {
		conf = Config{}
		expectedStartErr = nil
		ammos = nil
		successReceives = 0
	})

	JustBeforeEach(func() {
		conf.File = testFile
		provider = newTestProvider(conf)
		errch = make(chan error)
		var ctx context.Context
		ctx, cancel = context.WithCancel(context.Background())
		go func() { errch <- provider.Run(ctx, core.ProviderDeps{}) }()
		Expect(errch).NotTo(Receive())

		for i := 0; i < successReceives; i++ {
			By(fmt.Sprint(i))
			am, ok := provider.Acquire()
			Expect(ok).To(BeTrue())
			ammos = append(ammos, am.(*simple.Ammo))
		}
	})

	AfterEach(func() {
		_, ok := provider.Acquire()
		Expect(ok).To(BeFalse(), "All ammo should be readed.")
		defer cancel()
		var err error
		Eventually(errch).Should(Receive(&err))
		if expectedStartErr == nil {
			Expect(err).To(BeNil())
		} else {
			Expect(err).To(Equal(expectedStartErr))
		}
		for i := 0; i < len(ammos); i++ {
			expectedData := testData[i%len(testData)]
			ammo := ammos[i]
			req, ss := ammo.Request()
			By(fmt.Sprintf("%v", i))
			Expect(*req).To(MatchFields(IgnoreExtras, Fields{
				"Method":     Equal(expectedData.method),
				"Proto":      Equal("HTTP/1.1"),
				"ProtoMajor": Equal(1),
				"ProtoMinor": Equal(1),
				"Host":       Equal(expectedData.host),
				"URL": PointTo(MatchFields(IgnoreExtras, Fields{
					"Scheme": BeEmpty(),
					//"Host":   Equal(expectedData.host),
					"Path": Equal(expectedData.path),
				})),
				"Header":     Equal(expectedData.header),
				"RequestURI": Equal(""),
			}))
			Expect(ss.Tags()).To(Equal(expectedData.tag))
			var bout bytes.Buffer
			if req.Body != nil {
				_, err := io.Copy(&bout, req.Body)
				Expect(err).To(BeNil())
				req.Body.Close()
			}
			Expect(bout.String()).To(Equal(expectedData.body))
		}
	})

	Context("unlimited", func() {
		BeforeEach(func() {
			successReceives = 5 * len(testData)
		})
		It("ok", func() {
			cancel()
			expectedStartErr = context.Canceled
			Eventually(provider.Sink, time.Second, time.Millisecond).Should(BeClosed())
		})
	})

	Context("limit set", func() {
		BeforeEach(func() {
			conf.Passes = 2
			successReceives = len(testData) * conf.Passes
		})
		It("ok", func() {})
	})

	Context("passes set", func() {
		BeforeEach(func() {
			conf.Passes = 10
			conf.Limit = 5
			successReceives = conf.Limit
		})
		It("ok", func() {})
	})

})
