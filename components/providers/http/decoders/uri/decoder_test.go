package uri

import ( // "context"
	// "net/http"
	"net/http"
	"sync"

	// . "github.com/onsi/ginkgo"
	// . "github.com/onsi/ginkgo/extensions/table"
	// . "github.com/onsi/gomega"
	// . "github.com/onsi/gomega/gstruct"

	"github.com/yandex/pandora/components/providers/base"
	// "github.com/yandex/pandora/core"

)

func newAmmoPool() *sync.Pool {
	return &sync.Pool{New: func() interface{} { return &base.Ammo[http.Request]{} }}

}

// var _ = Describe("Decoder", func() {
// 	It("uri decode ctx cancel", func() {
// 		ctx, cancel := context.WithCancel(context.Background())
// 		decoder := newDecoderPlus(ctx, make(chan *base.Ammo[http.Request]), newAmmoPool(), nil)
// 		cancel()
// 		err := decoder.Decode([]byte("/some/path"))
// 		Expect(err).To(Equal(context.Canceled))
// 	})
// 	var (
// 		ammoCh  chan *base.Ammo[http.Request]
// 		decoder *decoderPlus
// 	)
// 	BeforeEach(func() {
// 		ammoCh = make(chan *base.Ammo[http.Request], 10)
// 		decoder = newDecoderPlus(context.Background(), ammoCh, newAmmoPool(), nil)
// 	})
// 	DescribeTable("invalid input",
// 		func(line string) {
// 			err := decoder.Decode([]byte(line))
// 			Expect(err).NotTo(BeNil())
// 			Expect(ammoCh).NotTo(Receive())
// 			Expect(decoder.header).To(BeEmpty())
// 		},
// 		Entry("empty line", ""),
// 		Entry("empty header", "[  ]"),
// 		Entry("no closing brace", "[key: val "),
// 		Entry("no header key", "[ : val ]"),
// 		Entry("no colon", "[ key  val ]"),
// 	)

// 	Decode := func(line string) {
// 		err := decoder.Decode([]byte(line))
// 		Expect(err).To(BeNil())
// 	}
// 	It("uri", func() {
// 		header := http.Header{"a": []string{"b"}, "c": []string{"d"}}
// 		for k, v := range header {
// 			decoder.header[k] = v
// 		}
// 		const host = "example.com"
// 		decoder.header.Set("Host", host)
// 		line := "/some/path"
// 		Decode(line)
// 		var am core.Ammo
// 		Expect(ammoCh).To(Receive(&am))
// 		sh, ok := am.(*base.Ammo[http.Request])
// 		Expect(ok).To(BeTrue())
// 		req, sample := sh.Request()
// 		Expect(*req.URL).To(MatchFields(IgnoreExtras, Fields{
// 			"Path":   Equal(line),
// 			"Scheme": BeEmpty(),
// 		}))
// 		Expect(req.Host).To(Equal(host))
// 		Expect(req.Header).To(Equal(header))
// 		header.Set("Host", host)
// 		Expect(decoder.header).To(Equal(header))
// 		Expect(decoder.ammoNum).To(Equal(uint(1)))
// 		Expect(sample.Tags()).To(Equal(""))
// 	})
// 	It("uri and tag", func() {
// 		header := http.Header{"a": []string{"b"}, "c": []string{"d"}}
// 		for k, v := range header {
// 			decoder.header[k] = v
// 		}
// 		const host = "example.com"
// 		decoder.header.Set("Host", host)
// 		line := "/some/path some tag"
// 		Decode(line)
// 		var am core.Ammo
// 		Expect(ammoCh).To(Receive(&am))
// 		sh, ok := am.(*base.Ammo[http.Request])
// 		Expect(ok).To(BeTrue())
// 		req, sample := sh.Request()
// 		Expect(*req.URL).To(MatchFields(IgnoreExtras, Fields{
// 			"Path":   Equal("/some/path"),
// 			"Scheme": BeEmpty(),
// 		}))
// 		Expect(req.Host).To(Equal(host))
// 		Expect(req.Header).To(Equal(header))
// 		header.Set("Host", host)
// 		Expect(decoder.header).To(Equal(header))
// 		Expect(decoder.ammoNum).To(Equal(uint(1)))
// 		Expect(sample.Tags()).To(Equal("some tag"))
// 	})
// 	Context("header", func() {
// 		AfterEach(func() {
// 			Expect(decoder.ammoNum).To(BeZero())
// 		})
// 		It("overwrite", func() {
// 			decoder.header.Set("A", "b")
// 			Decode("[A: c]")
// 			Expect(decoder.header).To(Equal(http.Header{
// 				"A": []string{"c"},
// 			}))
// 		})
// 		It("add", func() {
// 			decoder.header.Set("A", "b")
// 			Decode("[C: d]")
// 			Expect(decoder.header).To(Equal(http.Header{
// 				"A": []string{"b"},
// 				"C": []string{"d"},
// 			}))
// 		})
// 		It("spaces", func() {
// 			Decode(" [ C :   d   ] 			")
// 			Expect(decoder.header).To(Equal(http.Header{
// 				"C": []string{"d"},
// 			}))
// 		})
// 		It("value colons", func() {
// 			Decode("[C:c:d]")
// 			Expect(decoder.header).To(Equal(http.Header{
// 				"C": []string{"c:d"},
// 			}))
// 		})
// 		It("empty value", func() {
// 			Decode("[C:]")
// 			Expect(decoder.header).To(Equal(http.Header{
// 				"C": []string{""},
// 			}))
// 		})
// 	})
// 	It("Reset", func() {
// 		decoder.header.Set("a", "b")
// 		decoder.ammoNum = 10
// 		decoder.ResetHeader()
// 		Expect(decoder.header).To(BeEmpty())
// 		Expect(decoder.ammoNum).To(Equal(uint(10)))
// 	})

// })
