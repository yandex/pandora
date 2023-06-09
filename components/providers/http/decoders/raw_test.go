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

func Test_rawDecoder_Scan(t *testing.T) {
	input := `68 good50
GET /?sleep=50 HTTP/1.0
Host: 4bs65mu2kdulxmir.myt.yp-c.yandex.net


74 bad
GET /abra HTTP/1.0
Host: xxx.tanks.example.com
User-Agent: xxx (shell 1)

599
POST /upload/2 HTTP/1.0
Content-Length: 496
Host: xxxxxxxxx.dev.example.com
User-Agent: xxx (shell 1)

^.^.1......W.j^1^.^.^.²..^^.i.^B.P..-!(.l/Y..V^.      ...L?...S'NR.^^vm...3Gg@s...d'.\^.5N.$NF^,.Z^.aTE^.
._.[..k#L^ƨ'\RE.J.<.!,.q5.F^՚iΔĬq..^6..P..тH.'..i2
.".uuzs^^F2...Rh.&.U.^^.2J.P@.A......x..lǝy^?.u.p{4..g...m.,..R^.^.^.3....].^^.^J...p.ifTF0<.s.9V.o5<..%!6ļS.ƐǢ..㱋....C^&.....^.^y...v]^YT.1.#K.ibc...^.26...   ..7.
b.$...j6.٨f...W.R7.^1.3....K'%.&^.4d..{{      l0..^\..^X.g.^.r.(!.^^.5.4.1.$\ .%.8$(.n&..^^q.,.Q..^.D^.].^.R9.kE.^.$^.I..<..B^.6^.h^^C.^E.|....3o^.@..Z.^.s.$[v.
305
POST /upload/3 HTTP/1.0
Content-Length: 202
Host: xxxxxxxxx.dev.example.com
User-Agent: xxx (shell 1)

^.^.7......QMO.0^.++^zJw.ر^$^.^Ѣ.^V.J....vM.8r&.T+...{@pk%~C.G../z顲^.7....l...-.^W"cR..... .&^?u.U^^.^.8...{^.^.98.^.^.I.EĂ.p...'^.3.Tq..@R8....RAiBU..1.Bd*".7+.
.Ol.j=^.3..n....wp..,Wg.y^.T..~^.0
`

	decoder := newRawDecoder(strings.NewReader(input), config.Config{
		Limit: 8,
	}, http.Header{"Content-Type": []string{"application/json"}})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	tests := []struct {
		wantTag  string
		wantErr  bool
		wantBody string
	}{
		{
			wantTag:  "good50",
			wantErr:  false,
			wantBody: "GET /?sleep=50 HTTP/1.0\r\nHost: 4bs65mu2kdulxmir.myt.yp-c.yandex.net\r\nContent-Type: application/json\r\n\r\n",
		},
		{
			wantTag:  "bad",
			wantErr:  false,
			wantBody: "GET /abra HTTP/1.0\r\nHost: xxx.tanks.example.com\r\nContent-Type: application/json\r\nUser-Agent: xxx (shell 1)\r\n\r\n",
		},
		{
			wantTag:  "",
			wantErr:  false,
			wantBody: "POST /upload/2 HTTP/1.0\r\nHost: xxxxxxxxx.dev.example.com\r\nContent-Length: 496\r\nContent-Type: application/json\r\nUser-Agent: xxx (shell 1)\r\n\r\n^.^.1......W.j^1^.^.^.²..^^.i.^B.P..-!(.l/Y..V^.      ...L?...S'NR.^^vm...3Gg@s...d'.\\^.5N.$NF^,.Z^.aTE^.\n._.[..k#L^ƨ'\\RE.J.<.!,.q5.F^՚iΔĬq..^6..P..тH.'..i2\n.\".uuzs^^F2...Rh.&.U.^^.2J.P@.A......x..lǝy^?.u.p{4..g...m.,..R^.^.^.3....].^^.^J...p.ifTF0<.s.9V.o5<..%!6ļS.ƐǢ..㱋....C^&.....^.^y...v]^YT.1.#K.ibc...^.26...   ..7.\nb.$...j6.٨f...W.R7.^1.3....K'%.&^.4d..{{      l0..^\\..^X.g.^.r.(!.^^.5.4.1.$\\ .%.8$(.n&..^^q.,.Q..^.D^.].^.R9.kE.^.$^.I..<..B^.6^.h^^C.^E.|....3o^.@..Z.^.s.$[v.\n",
		},
		{
			wantTag:  "",
			wantErr:  false,
			wantBody: "POST /upload/3 HTTP/1.0\r\nHost: xxxxxxxxx.dev.example.com\r\nContent-Length: 202\r\nContent-Type: application/json\r\nUser-Agent: xxx (shell 1)\r\n\r\n^.^.7......QMO.0^.++^zJw.ر^$^.^Ѣ.^V.J....vM.8r&.T+...{@pk%~C.G../z顲^.7....l...-.^W\"cR..... .&^?u.U^^.^.8...{^.^.98.^.^.I.EĂ.p...'^.3.Tq..@R8....RAiBU..1.Bd*\".7+.\n.Ol.j=^.3..n....wp..,Wg.y^.T..~^.0\n",
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
