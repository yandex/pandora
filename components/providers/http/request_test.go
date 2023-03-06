package simple

import (
	"net/http"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("HTTPHeaders", func() {
	It("decode http config headers", func() {
		decodedConfigHeaders, err := DecodeHTTPConfigHeaders([]string{
			"[Host: youhost.tld]",
			"[SomeHeader: somevalue]",
		})
		expectHeaders := []Header{
			{"Host", "youhost.tld"},
			{"SomeHeader", "somevalue"},
		}
		Expect(err).To(BeNil())
		Expect(decodedConfigHeaders).To(Equal(expectHeaders))
	})

	It("add new http headers", func() {
		const origHost = "example.com"
		req, _ := http.NewRequest("GET", origHost, nil)
		req.Host = origHost
		req.Header.Set("SomeHeader", "oldvalue")
		headers := []Header{
			{"Host", "youhost.tld"},
			{"SomeHeader", "newvalue"},
			{"SecondHeader", "new_second_value"},
		}
		EnrichRequestWithHeaders(req, headers)
		Expect(req.Host).To(Equal(origHost))
		Expect(req.Header).To(Equal(http.Header{
			"Someheader":   []string{"oldvalue"},
			"Secondheader": []string{"new_second_value"},
		}))
	})
})

func TestExample(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Example Suite")
}
