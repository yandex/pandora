package decoders

import (
	"context"
	"net/http"
	"net/http/httputil"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yandex/pandora/components/providers/http/config"
)

func Test_uripostDecoder_Scan(t *testing.T) {
	input := `5 /0
class
[A:b]
5 /1
class
[Host : example.com]
[ C : d ]
10 /2
classclass
[A:]
[Host : other.net]

15 /3 wantTag
classclassclass
`

	decoder := newURIPostDecoder(strings.NewReader(input), config.Config{
		Limit: 8,
	}, http.Header{"Content-Type": []string{"application/json"}})

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
	defer cancel()

	tests := []struct {
		wantTag  string
		wantErr  bool
		wantBody string
	}{
		{
			wantTag:  "",
			wantErr:  false,
			wantBody: "POST /0 HTTP/1.1\r\nContent-Type: application/json\r\n\r\nclass",
		},
		{
			wantTag:  "",
			wantErr:  false,
			wantBody: "POST /1 HTTP/1.1\r\nA: b\r\nContent-Type: application/json\r\n\r\nclass",
		},
		{
			wantTag:  "",
			wantErr:  false,
			wantBody: "POST /2 HTTP/1.1\r\nHost: example.com\r\nA: b\r\nC: d\r\nContent-Type: application/json\r\n\r\nclassclass",
		},
		{
			wantTag:  "wantTag",
			wantErr:  false,
			wantBody: "POST /3 HTTP/1.1\r\nHost: other.net\r\nA: \r\nC: d\r\nContent-Type: application/json\r\n\r\nclassclassclass",
		},
	}
	for j := 0; j < 2; j++ {
		for i, tt := range tests {
			scan := decoder.Scan(ctx)
			assert.True(t, scan)
			if tt.wantErr {
				assert.Error(t, decoder.err, "iteration %d-%d", j, i)
				continue
			} else {
				assert.NoError(t, decoder.err, "iteration %d-%d", j, i)
			}
			assert.Equal(t, tt.wantTag, decoder.tag, "iteration %d-%d", j, i)

			decoder.req.Close = false
			body, _ := httputil.DumpRequest(decoder.req, true)
			assert.Equal(t, tt.wantBody, string(body), "iteration %d-%d", j, i)
		}
	}

	assert.False(t, decoder.Scan(ctx))

	assert.Equal(t, decoder.ammoNum, uint(len(tests)*2))
	assert.Equal(t, decoder.passNum, uint(1))
	assert.Equal(t, decoder.err, ErrAmmoLimit)
}
