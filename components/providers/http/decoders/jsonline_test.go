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

func Test_jsonlineDecoder_Scan(t *testing.T) {
	input := `{"host": "4bs65mu2kdulxmir.myt.yp-c.yandex.net", "method": "GET", "uri": "/?sleep=100", "tag": "sleep1", "headers": {"User-agent": "Tank", "Connection": "close"}}
{"host": "4bs65mu2kdulxmir.myt.yp-c.yandex.net", "method": "POST", "uri": "/?sleep=200", "tag": "sleep2", "headers": {"User-agent": "Tank", "Connection": "close"}, "body": "body_data"}


`

	decoder := newJsonlineDecoder(strings.NewReader(input), config.Config{
		Limit: 4,
	}, http.Header{"Content-Type": []string{"application/json"}})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	tests := []struct {
		wantTag  string
		wantErr  bool
		wantBody string
	}{
		{
			wantTag:  "sleep1",
			wantErr:  false,
			wantBody: "GET /?sleep=100 HTTP/1.1\r\nHost: 4bs65mu2kdulxmir.myt.yp-c.yandex.net\r\nConnection: close\r\nContent-Type: application/json\r\nUser-Agent: Tank\r\n\r\n",
		},
		{
			wantTag:  "sleep2",
			wantErr:  false,
			wantBody: "POST /?sleep=200 HTTP/1.1\r\nHost: 4bs65mu2kdulxmir.myt.yp-c.yandex.net\r\nConnection: close\r\nContent-Type: application/json\r\nUser-Agent: Tank\r\n\r\nbody_data",
		},
	}
	for j := 0; j < 2; j++ {
		for i, tt := range tests {
			req, tag, err := decoder.Scan(ctx)
			if tt.wantErr {
				assert.Error(t, err, "iteration %d-%d", j, i)
				continue
			} else {
				assert.NoError(t, err, "iteration %d-%d", j, i)
			}
			assert.Equal(t, tt.wantTag, tag, "iteration %d-%d", j, i)

			req.Close = false
			body, _ := httputil.DumpRequest(req, true)
			assert.Equal(t, tt.wantBody, string(body), "iteration %d-%d", j, i)
		}
	}

	_, _, err := decoder.Scan(ctx)
	assert.Equal(t, err, ErrAmmoLimit)
	assert.Equal(t, decoder.ammoNum, uint(len(tests)*2))
	assert.Equal(t, decoder.passNum, uint(1))
}
