package decoders

import (
	"context"
	"io/ioutil"
	"net/http"
	"testing"

	// . "github.com/onsi/ginkgo"
	// . "github.com/onsi/gomega"

	// . "github.com/onsi/gomega/gstruct"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/yandex/pandora/components/providers/http/config"
)

const rawTestFile = "./ammo.stpd"
const testFileData = "../testdata/ammo.stpd"

var rawTestData = []ammoData{
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

var rawTestFs = func() afero.Fs {
	testFs := afero.NewOsFs()
	testFileData, err := testFs.Open(testFileData)
	if err != nil {
		panic(err)
	}
	testDataBuffer, _ := ioutil.ReadAll(testFileData)
	_, _ = testFileData.Read(testDataBuffer)
	fs := afero.NewMemMapFs()
	err = afero.WriteFile(fs, rawTestFile, testDataBuffer, 0)
	if err != nil {
		panic(err)
	}
	return afero.NewReadOnlyFs(fs)
}()

func TestRawDecode(t *testing.T) {
	var conf = config.Config{}
	tests := []struct {
		name  string
		input DecodeInput
		want  DecodeWant
	}{}
	// var ans DecodeWant
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			// sink := make(chan *base.Ammo[http.Request])
			decoder, err := NewDecoder(conf, nil)
			assert.NoError(err)
			decoder.Scan(context.Background())
			// ans.req = decoder.Next()
		})
	}
}

// var _ = Describe("provider start", func() {
// 	It("ok", func() {
// 		p := newTestProvider(config.Config{File: rawTestFile})
// 		ctx, cancel := context.WithCancel(context.Background())
// 		errch := make(chan error)
// 		go func() { errch <- p.Run(ctx, nil) }()
// 		Expect(errch).NotTo(Receive())
// 		cancel()
// 		var err error
// 		Eventually(errch).Should(Receive(&err))
// 		Expect(err).To(Equal(ctx.Err()))
// 	})

// 	It("fail", func() {
// 		p := newTestProvider(config.Config{File: "no_such_file"})
// 		Expect(p.Run(context.Background(), nil), core.ProviderDeps{}).NotTo(BeNil())
// 	})
// })

// var _ = Describe("provider decode", func() {
// 	var (
// 		// Configured in BeforeEach.
// 		conf             config.Config // File always is rawTestFile.
// 		expectedStartErr error
// 		successReceives  int // How much ammo should be received.

// 		provider *Decoder
// 		cancel   context.CancelFunc
// 		errch    chan error
// 		ammos    []*base.Ammo[http.Request]
// 	)

// 	BeforeEach(func() {
// 		conf = config.Config{}
// 		expectedStartErr = nil
// 		ammos = nil
// 		successReceives = 0
// 	})

// 	JustBeforeEach(func() {
// 		conf.File = rawTestFile
// 		provider = newTestProvider(conf)
// 		errch = make(chan error)
// 		var ctx context.Context
// 		ctx, cancel = context.WithCancel(context.Background())
// 		go func() { errch <- provider.Run(ctx) }()
// 		Expect(errch).NotTo(Receive())

// 		for i := 0; i < successReceives; i++ {
// 			By(fmt.Sprint(i))
// 			am, ok := provider.Acquire()
// 			Expect(ok).To(BeTrue())
// 			ammos = append(ammos, am.(*base.Ammo[http.Request]))
// 		}
// 	})

// 	AfterEach(func() {
// 		_, ok := provider.Acquire()
// 		Expect(ok).To(BeFalse(), "All ammo should be readed.")
// 		defer cancel()
// 		var err error
// 		Eventually(errch).Should(Receive(&err))
// 		if expectedStartErr == nil {
// 			Expect(err).To(BeNil())
// 		} else {
// 			Expect(err).To(Equal(expectedStartErr))
// 		}
// 		for i := 0; i < len(ammos); i++ {
// 			expectedData := rawTestData[i%len(rawTestData)]
// 			ammo := ammos[i]
// 			req, ss := ammo.Request()
// 			By(fmt.Sprintf("%v", i))
// 			Expect(*req).To(MatchFields(IgnoreExtras, Fields{
// 				"Method":     Equal(expectedData.method),
// 				"Proto":      Equal("HTTP/1.1"),
// 				"ProtoMajor": Equal(1),
// 				"ProtoMinor": Equal(1),
// 				"Host":       Equal(expectedData.host),
// 				"URL": PointTo(MatchFields(IgnoreExtras, Fields{
// 					"Scheme": BeEmpty(),
// 					//"Host":   Equal(expectedData.host),
// 					"Path": Equal(expectedData.path),
// 				})),
// 				"Header":     Equal(expectedData.header),
// 				"RequestURI": Equal(""),
// 			}))
// 			Expect(ss.Tags()).To(Equal(expectedData.tag))
// 			var bout bytes.Buffer
// 			if req.Body != nil {
// 				_, err := io.Copy(&bout, req.Body)
// 				Expect(err).To(BeNil())
// 				_ = req.Body.Close()
// 			}
// 			Expect(bout.String()).To(Equal(expectedData.body))
// 		}
// 	})

// 	Context("unlimited", func() {
// 		BeforeEach(func() {
// 			successReceives = 5 * len(rawTestData)
// 		})
// 		It("ok", func() {
// 			cancel()
// 			expectedStartErr = context.Canceled
// 			Eventually(provider.Sink, time.Second, time.Millisecond).Should(BeClosed())
// 		})
// 	})

// 	Context("limit set", func() {
// 		BeforeEach(func() {
// 			conf.Passes = 2
// 			successReceives = len(rawTestData) * int(conf.Passes)
// 		})
// 		It("ok", func() {})
// 	})

// 	Context("passes set", func() {
// 		BeforeEach(func() {
// 			conf.Passes = 10
// 			conf.Limit = 5
// 			successReceives = int(conf.Limit)
// 		})
// 		It("ok", func() {})
// 	})

// })
