package jsonline

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	"github.com/spf13/afero"

	"github.com/yandex/pandora/ammo"
)

func TestJsonline(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Jsonline Suite")
}

const testFile = "./ammo.jsonline"

// testData holds jsonline.data that contains in testFile
var testData = []data{
	{
		Host:    "example.com",
		Method:  "GET",
		Uri:     "/00",
		Headers: map[string]string{"Accept": "*/*", "Accept-Encoding": "gzip, deflate", "Host": "example.org", "User-Agent": "Pandora/0.0.1"},
	},
	{
		Host:    "ya.ru",
		Method:  "HEAD",
		Uri:     "/01",
		Headers: map[string]string{"Accept": "*/*", "Accept-Encoding": "gzip, brotli", "Host": "ya.ru", "User-Agent": "YaBro/0.1"},
		Tag:     "head",
	},
}

var testFs = newTestFs()

func newTestFs() afero.Fs {
	fs := afero.NewMemMapFs()
	file, err := fs.Create(testFile)
	if err != nil {
		panic(err)
	}
	encoder := json.NewEncoder(file)
	for _, d := range testData {
		err := encoder.Encode(d)
		if err != nil {
			panic(err)
		}
	}
	return afero.NewReadOnlyFs(fs)
}

func newTestProvider(conf Config) ammo.Provider {
	return NewProvider(testFs, conf)
}

var _ = Describe("data", func() {
	It("decoded well", func() {
		data := data{
			Host:    "ya.ru",
			Method:  "GET",
			Uri:     "/00",
			Headers: map[string]string{"A": "a", "B": "b"},
			Tag:     "tag",
		}
		req, err := data.ToRequest()
		Expect(err).To(BeNil())
		Expect(*req).To(MatchFields(IgnoreExtras, Fields{
			"Proto":      Equal("HTTP/1.1"),
			"ProtoMajor": Equal(1),
			"ProtoMinor": Equal(1),
			"Body":       BeNil(),
			"URL": PointTo(MatchFields(IgnoreExtras, Fields{
				"Scheme": Equal("http"),
				"Host":   Equal(data.Host),
				"Path":   Equal(data.Uri),
			})),
			"Header": Equal(http.Header{
				"A": []string{"a"},
				"B": []string{"b"},
			}),
			"Method": Equal(data.Method),
		}))
	})
})

var _ = Describe("provider start", func() {
	It("ok", func() {
		p := newTestProvider(Config{File: testFile})
		ctx, cancel := context.WithCancel(context.Background())
		errch := make(chan error)
		go func() { errch <- p.Start(ctx) }()
		Expect(errch).NotTo(Receive())
		cancel()
		var err error
		Eventually(errch).Should(Receive(&err))
		Expect(err).To(Equal(ctx.Err()))
	})

	It("fail", func() {
		p := newTestProvider(Config{File: "no_such_file"})
		Expect(p.Start(context.Background())).NotTo(BeNil())
	})
})

var _ = Describe("provider decode", func() {
	var (
		// Configured in BeforeEach.
		conf             Config // File always is testFile.
		expectedStartErr error
		successReceives  int // How much ammo should be received.

		provider ammo.Provider
		cancel   context.CancelFunc
		errch    chan error
		ammos    []ammo.Ammo
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
		go func() { errch <- provider.Start(ctx) }()
		Expect(errch).NotTo(Receive())

		for i := 0; i < successReceives; i++ {
			var am ammo.Ammo
			Eventually(provider.Source()).Should(Receive(&am))
			ammos = append(ammos, am)
		}
	})

	AfterEach(func() {
		Eventually(provider.Source()).Should(BeClosed(), "All ammo should be readed.")
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
			expectedReq, err := expectedData.ToRequest()
			Expect(err).To(BeNil())
			ha := ammos[i].(ammo.HTTP)
			req, ss := ha.Request()
			Expect(req).To(Equal(expectedReq))
			Expect(ss.Tags()).To(Equal(expectedData.Tag))
		}
	})

	Context("unlimited", func() {
		BeforeEach(func() {
			successReceives = 5 * len(testData)
		})
		It("ok", func() {
			cancel()
			expectedStartErr = context.Canceled
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

func Benchmark(b *testing.B) {
	RegisterTestingT(b)
	jsonDoc, err := json.Marshal(testData[0])
	Expect(err).To(BeNil())
	pool := sync.Pool{
		New: func() interface{} { return &ammo.SimpleHTTP{} },
	}
	b.Run("Decode", func(b *testing.B) {
		for n := 0; n < b.N; n++ {
			decoder{}.Decode(jsonDoc, &ammo.SimpleHTTP{})
		}
	})
	b.Run("DecodeWithPool", func(b *testing.B) {
		for n := 0; n < b.N; n++ {
			h := pool.Get().(*ammo.SimpleHTTP)
			decoder{}.Decode(jsonDoc, h)
			pool.Put(h)
		}
	})
}
