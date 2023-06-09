package raw

import (
	"testing"

	"github.com/yandex/pandora/components/providers/http/util"
	"github.com/yandex/pandora/lib/testutil"
)

var (
	benchTestConfigHeaders = []string{
		"[Host: yourhost.tld]",
		"[Sometest: someval]",
	}
	benchTestRequest = []byte("GET / HTTP/1.1\r\n" +
		"Host: yourhost.tld" +
		"Content-Length: 0\r\n" +
		"\r\n")
)

// $ go test -bench . -benchmem -benchtime=10s
// cpu: Intel(R) Core(TM) i7-10850H CPU @ 2.70GHz
// BenchmarkRawDecoder-12               	 9371816	      1426 ns/op	    5122 B/op	      10 allocs/op
// BenchmarkRawDecoderWithHeaders-12    	 6145189	      1886 ns/op	    5122 B/op	      10 allocs/op

func BenchmarkRawDecoder(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = DecodeRequest(benchTestRequest)
	}
}

func BenchmarkRawDecoderWithHeaders(b *testing.B) {
	decodedHTTPConfigHeaders := testutil.Must(util.DecodeHTTPConfigHeaders(benchTestConfigHeaders))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req, _ := DecodeRequest(benchTestRequest)
		util.EnrichRequestWithHeaders(req, decodedHTTPConfigHeaders)
	}
}
