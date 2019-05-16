package uri

import (
	"context"
	"net/http"
	"sync"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"

	"github.com/yandex/pandora/components/phttp/ammo/simple"
	"github.com/yandex/pandora/core"
)

func newAmmoPool() *sync.Pool {
	return &sync.Pool{New: func() interface{} { return &simple.Ammo{} }}

}

var _ = Describe("Decoder", func() {
	It("uri decode ctx cancel", func() {
		ctx, cancel := context.WithCancel(context.Background())
		decoder := newDecoder(ctx, make(chan *simple.Ammo), newAmmoPool())
		cancel()
		err := decoder.Decode([]byte("/some/path"))
		Expect(err).To(Equal(context.Canceled))
	})
	var (
		ammoCh  chan *simple.Ammo
		decoder *decoder
	)
	BeforeEach(func() {
		ammoCh = make(chan *simple.Ammo, 10)
		decoder = newDecoder(context.Background(), ammoCh, newAmmoPool())
	})
	DescribeTable("invalid input",
		func(line string) {
			err := decoder.Decode([]byte(line))
			Expect(err).NotTo(BeNil())
			Expect(ammoCh).NotTo(Receive())
			Expect(decoder.header).To(BeEmpty())
		},
		Entry("empty line", ""),
		Entry("line start", "test"),
		Entry("empty header", "[  ]"),
		Entry("no closing brace", "[key: val "),
		Entry("no header key", "[ : val ]"),
		Entry("no colon", "[ key  val ]"),
	)

	Decode := func(line string) {
		err := decoder.Decode([]byte(line))
		Expect(err).To(BeNil())
	}
	It("uri", func() {
		header := http.Header{"a": []string{"b"}, "c": []string{"d"}}
		for k, v := range header {
			decoder.header[k] = v
		}
		const host = "example.com"
		decoder.header.Set("Host", host)
		line := "/some/path"
		Decode(line)
		var am core.Ammo
		Expect(ammoCh).To(Receive(&am))
		sh, ok := am.(*simple.Ammo)
		Expect(ok).To(BeTrue())
		req, sample := sh.Request()
		Expect(*req.URL).To(MatchFields(IgnoreExtras, Fields{
			"Path":   Equal(line),
			"Host":   Equal(host),
			"Scheme": BeEmpty(),
		}))
		Expect(req.Host).To(Equal(host))
		Expect(req.Header).To(Equal(header))
		header.Set("Host", host)
		Expect(decoder.header).To(Equal(header))
		Expect(decoder.ammoNum).To(Equal(1))
		Expect(sample.Tags()).To(Equal(""))
	})
	It("uri and tag", func() {
		header := http.Header{"a": []string{"b"}, "c": []string{"d"}}
		for k, v := range header {
			decoder.header[k] = v
		}
		const host = "example.com"
		decoder.header.Set("Host", host)
		line := "/some/path some tag"
		Decode(line)
		var am core.Ammo
		Expect(ammoCh).To(Receive(&am))
		sh, ok := am.(*simple.Ammo)
		Expect(ok).To(BeTrue())
		req, sample := sh.Request()
		Expect(*req.URL).To(MatchFields(IgnoreExtras, Fields{
			"Path":   Equal("/some/path"),
			"Host":   Equal(host),
			"Scheme": BeEmpty(),
		}))
		Expect(req.Host).To(Equal(host))
		Expect(req.Header).To(Equal(header))
		header.Set("Host", host)
		Expect(decoder.header).To(Equal(header))
		Expect(decoder.ammoNum).To(Equal(1))
		Expect(sample.Tags()).To(Equal("some tag"))
	})
	Context("header", func() {
		AfterEach(func() {
			Expect(decoder.ammoNum).To(BeZero())
		})
		It("overwrite", func() {
			decoder.header.Set("A", "b")
			Decode("[A: c]")
			Expect(decoder.header).To(Equal(http.Header{
				"A": []string{"c"},
			}))
		})
		It("add", func() {
			decoder.header.Set("A", "b")
			Decode("[C: d]")
			Expect(decoder.header).To(Equal(http.Header{
				"A": []string{"b"},
				"C": []string{"d"},
			}))
		})
		It("spaces", func() {
			Decode(" [ C :   d   ] 			")
			Expect(decoder.header).To(Equal(http.Header{
				"C": []string{"d"},
			}))
		})
		It("value colons", func() {
			Decode("[C:c:d]")
			Expect(decoder.header).To(Equal(http.Header{
				"C": []string{"c:d"},
			}))
		})
		It("empty value", func() {
			Decode("[C:]")
			Expect(decoder.header).To(Equal(http.Header{
				"C": []string{""},
			}))
		})
		It("overwrite by config", func() {
			decodedConfigHeaders, _ := decodeHTTPConfigHeaders([]string{
				"[Host: youhost.tld]",
				"[SomeHeader: somevalue]",
			})
			decoder.configHeaders = decodedConfigHeaders
			cfgHeaders := []ConfigHeader{
				{"Host", "youhost.tld"},
				{"SomeHeader", "somevalue"},
			}
			Expect(decoder.configHeaders).To(Equal(cfgHeaders))
		})
	})
	It("Reset", func() {
		decoder.header.Set("a", "b")
		decoder.ammoNum = 10
		decoder.ResetHeader()
		Expect(decoder.header).To(BeEmpty())
		Expect(decoder.ammoNum).To(Equal(10))
	})

})
