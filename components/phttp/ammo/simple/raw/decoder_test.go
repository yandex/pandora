package raw

import (
	"bytes"
	"io"
	"net/http"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
)

var _ = Describe("Decoder", func() {
	It("should parse header with tag", func() {
		raw := "123 tag"
		reqSize, tag, err := decodeHeader([]byte(raw))
		Expect(err).To(BeNil())
		Expect(reqSize).To(Equal(123))
		Expect(tag).To(Equal("tag"))
	})
	It("should parse header without tag", func() {
		raw := "123"
		reqSize, tag, err := decodeHeader([]byte(raw))
		Expect(err).To(BeNil())
		Expect(reqSize).To(Equal(123))
		Expect(tag).To(Equal(""))
	})
	It("should parse GET request", func() {
		raw := "GET /some/path HTTP/1.0\r\n" +
			"Host: foo.com\r\n" +
			"Connection: close\r\n\r\n"
		req, err := decodeRequest([]byte(raw))
		Expect(err).To(BeNil())
		Expect(*req.URL).To(MatchFields(IgnoreExtras, Fields{
			"Path":   Equal("/some/path"),
			"Scheme": BeEmpty(),
		}))
		Expect(req.RequestURI).To(Equal(""))
		Expect(req.Host).To(Equal("foo.com"))
		Expect(req.Header).To(Equal(http.Header{"Connection": []string{"close"}}))
	})
	It("should parse POST request with body", func() {
		raw := "POST /some/path HTTP/1.1\r\n" +
			"Host: foo.com\r\n" +
			"Transfer-Encoding: chunked\r\n" +
			"Foo: bar\r\n" +
			"Content-Length: 9999\r\n\r\n" + // to be removed.
			"3\r\nfoo\r\n" +
			"3\r\nbar\r\n" +
			"0\r\n" +
			"\r\n"
		req, err := decodeRequest([]byte(raw))
		Expect(err).To(BeNil())
		Expect(*req.URL).To(MatchFields(IgnoreExtras, Fields{
			"Path":   Equal("/some/path"),
			"Scheme": BeEmpty(),
		}))
		Expect(req.RequestURI).To(Equal(""))
		Expect(req.Host).To(Equal("foo.com"))
		Expect(req.Header).To(Equal(http.Header{"Foo": []string{"bar"}}))
		var bout bytes.Buffer
		if req.Body != nil {
			_, err := io.Copy(&bout, req.Body)
			Expect(err).To(BeNil())
			req.Body.Close()
		}
		Expect(bout.String()).To(Equal("foobar"))
	})
	It("should return error on bad urls", func() {
		raw := "GET ../../../../etc/passwd HTTP/1.1\r\n" +
			"Host: foo.com\r\n" +
			"Content-Length: 0\r\n" +
			"\r\n"
		req, err := decodeRequest([]byte(raw))
		Expect(err).ToNot(BeNil())
		Expect(req).To(BeNil())
	})
	It("should replace header Host for URL if specified", func() {
		raw := "GET /etc/passwd HTTP/1.1\r\n" +
			"Host: hostname.tld\r\n" +
			"Content-Length: 0\r\n" +
			"\r\n"
		req, err := decodeRequest([]byte(raw))
		Expect(err).To(BeNil())
		Expect(req.Host).To(Equal("hostname.tld"))
		Expect(req.URL.Host).To(Equal("hostname.tld"))
	})
	It("should replace header Host from config", func() {
		const host = "hostname.tld"
		const newhost = "newhostname.tld"

		raw := "GET / HTTP/1.1\r\n" +
			"Host: " + host + "\r\n" +
			"Content-Length: 0\r\n" +
			"\r\n"
		configHeaders := []string{
			"[Host: " + newhost + "]",
			"[SomeTestKey: sometestvalue]",
		}
		req, err := decodeRequest([]byte(raw))
		Expect(err).To(BeNil())
		Expect(req.Host).To(Equal(host))
		Expect(req.URL.Host).To(Equal(host))
		decodedConfigHeaders, _ := decodeHTTPConfigHeaders(configHeaders)
		for _, header := range decodedConfigHeaders {
			// special behavior for `Host` header
			if header.key == "Host" {
				req.URL.Host = header.value
			} else {
				req.Header.Set(header.key, header.value)
			}
		}
		Expect(req.URL.Host).To(Equal(newhost))
		Expect(req.Header.Get("SomeTestKey")).To(Equal("sometestvalue"))
	})
})
