package raw

import "testing"

var (
	benchTestConfigHeaders = []string{
		"[Host: yourhost.tld]",
		"[Sometest: someval]",
	}
)

const (
	benchTestRequest = "GET / HTTP/1.1\r\n" +
		"Host: yourhost.tld" +
		"Content-Length: 0\r\n" +
		"\r\n"
)

// BenchmarkRawDecoder-4              	  500000	      2040 ns/op	    5152 B/op	      11 allocs/op
// BenchmarkRawDecoderWithHeaders-4   	 1000000	      1944 ns/op	    5168 B/op	      12 allocs/op

func BenchmarkRawDecoder(b *testing.B) {
	for i := 0; i < b.N; i++ {
		decodeRequest([]byte(benchTestRequest))
	}
}

func BenchmarkRawDecoderWithHeaders(b *testing.B) {
	b.StopTimer()
	decodedHTTPConfigHeaders, _ := decodeHTTPConfigHeaders(benchTestConfigHeaders)
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		req, _ := decodeRequest([]byte(benchTestRequest))
		for _, header := range decodedHTTPConfigHeaders {
			if header.key == "Host" {
				req.URL.Host = header.value
			} else {
				req.Header.Set(header.key, header.value)
			}
		}
	}
}
