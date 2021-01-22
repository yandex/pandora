package uri

import (
	"context"
	"fmt"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	"github.com/spf13/afero"

	"github.com/yandex/pandora/components/phttp/ammo/simple"
	"github.com/yandex/pandora/core"
	"github.com/pkg/errors"
)

const testFile = "./ammo.uri"
const testFileData = `/0
[A:b]
/1
[Host : example.com]
[ C : d ]
/2
[A:]
[Host : other.net]

/3
/4 some tag
`

var testData = []ammoData{
	{"", "/0", http.Header{}, ""},
	{"", "/1", http.Header{"A": []string{"b"}}, ""},
	{"example.com", "/2", http.Header{
		"A": []string{"b"},
		"C": []string{"d"},
	}, ""},
	{"other.net", "/3", http.Header{
		"A": []string{""},
		"C": []string{"d"},
	}, ""},
	{"other.net", "/4", http.Header{
		"A": []string{""},
		"C": []string{"d"},
	}, "some tag"},
}

type ammoData struct {
	host   string
	path   string
	header http.Header
	tag    string
}

var testFs = func() afero.Fs {
	fs := afero.NewMemMapFs()
	err := afero.WriteFile(fs, testFile, []byte(testFileData), 0)
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
		Expect(errors.Cause(err)).To(Equal(ctx.Err()))
	})

	It("fail", func() {
		p := newTestProvider(Config{File: "no_such_file"})
		Expect(p.Run(context.Background(), core.ProviderDeps{})).NotTo(BeNil())
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
			Expect(err).NotTo(HaveOccurred())
		} else {
			Expect(errors.Cause(err)).To(Equal(expectedStartErr))
		}
		for i := 0; i < len(ammos); i++ {
			expectedData := testData[i%len(testData)]
			ammo := ammos[i]
			req, ss := ammo.Request()
			By(fmt.Sprintf("%v", i))
			Expect(*req).To(MatchFields(IgnoreExtras, Fields{
				"Method":     Equal("GET"),
				"Proto":      Equal("HTTP/1.1"),
				"ProtoMajor": Equal(1),
				"ProtoMinor": Equal(1),
				"Body":       BeNil(),
				"Host":       Equal(expectedData.host),
				"URL": PointTo(MatchFields(IgnoreExtras, Fields{
					"Scheme": BeEmpty(),
					"Host":   Equal(expectedData.host),
					"Path":   Equal(expectedData.path),
				})),
				"Header": Equal(expectedData.header),
			}))
			Expect(ss.Tags()).To(Equal(expectedData.tag))
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
