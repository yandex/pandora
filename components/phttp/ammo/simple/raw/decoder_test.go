package raw

import (
	"net/http"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
)

var _ = Describe("Decoder", func() {
	header := http.Header{"Connection": []string{"close"}}
	It("should parse GET request", func() {
		raw := "GET /some/path HTTP/1.0\r\n" +
			"Host: www.ya.ru\r\n" +
			"Connection: close\r\n\r\n"
		req, err := Decode([]byte(raw))
		Expect(err).To(BeNil())
		Expect(*req.URL).To(MatchFields(IgnoreExtras, Fields{
			"Path":   Equal("/some/path"),
			"Scheme": BeEmpty(),
		}))
		Expect(req.Host).To(Equal("www.ya.ru"))
		Expect(req.Header).To(Equal(header))
	})
})
