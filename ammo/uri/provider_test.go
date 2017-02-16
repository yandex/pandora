package uri

import (
	"context"
	"fmt"
	"net/http"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	"github.com/spf13/afero"
	"github.com/yandex/pandora/ammo"
)

const testFile = "./ammo.uri"
const testFileData = `/0
[A:b]
/1
[ C : d ]
/2
[A:]
/3`

var testData = []ammoData{
	{"/0", http.Header{}},
	{"/1", http.Header{"A": []string{"b"}}},
	{"/2", http.Header{
		"A": []string{"b"},
		"C": []string{"d"},
	}},
	{"/3", http.Header{
		"A": []string{""},
		"C": []string{"d"},
	}},
}

type ammoData struct {
	uri    string
	header http.Header
}

var testFs = func() afero.Fs {
	fs := afero.NewMemMapFs()
	err := afero.WriteFile(fs, testFile, []byte(testFileData), 0)
	if err != nil {
		panic(err)
	}
	return afero.NewReadOnlyFs(fs)
}()

func newTestProvider(conf Config) ammo.Provider {
	return NewProvider(testFs, conf)
}

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
			By(fmt.Sprint(i))
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
			ha := ammos[i].(ammo.HTTP)
			req, ss := ha.Request()
			Expect(*req).To(MatchFields(IgnoreExtras, Fields{
				"Method":     Equal("GET"),
				"Proto":      Equal("HTTP/1.1"),
				"ProtoMajor": Equal(1),
				"ProtoMinor": Equal(1),
				"Body":       BeNil(),
				"Host":       BeEmpty(),
				"URL": PointTo(MatchFields(IgnoreExtras, Fields{
					"Scheme": BeEmpty(),
					"Host":   BeEmpty(),
					"Path":   Equal(expectedData.uri),
				})),
				"Header": Equal(expectedData.header),
			}))
			Expect(ss.Tags()).To(Equal("REQUEST"))
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
