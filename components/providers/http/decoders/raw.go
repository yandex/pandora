package decoders

import (
	"bufio"
	"context"
	"io"
	"net/http"
	"strings"

	"github.com/yandex/pandora/components/providers/base"
	"github.com/yandex/pandora/components/providers/http/config"
	"github.com/yandex/pandora/components/providers/http/decoders/raw"
	"github.com/yandex/pandora/components/providers/http/util"
	"golang.org/x/xerrors"
)

func newRawDecoder(file io.ReadSeeker, cfg config.Config, decodedConfigHeaders http.Header) *rawDecoder {
	return &rawDecoder{
		protoDecoder: protoDecoder{
			file:                 file,
			config:               cfg,
			decodedConfigHeaders: decodedConfigHeaders,
		},
		reader: bufio.NewReader(file),
	}
}

/*
Parses size-prefixed HTTP ammo files. Each ammo is prefixed with a header line (delimited with \n), which consists of
two fields delimited by a space: ammo size and tag. Ammo size is in bytes (integer, including special characters like CR, LF).
Tag is a string. Example:

77 bad
GET /abra HTTP/1.0
Host: xxx.tanks.example.com
User-Agent: xxx (shell 1)

904
POST /upload/2 HTTP/1.0
Content-Length: 801
Host: xxxxxxxxx.dev.example.com
User-Agent: xxx (shell 1)

^.^........W.j^1^.^.^.²..^^.i.^B.P..-!(.l/Y..V^.      ...L?...S'NR.^^vm...3Gg@s...d'.\^.5N.$NF^,.Z^.aTE^.
._.[..k#L^ƨ`\RE.J.<.!,.q5.F^՚iΔĬq..^6..P..тH.`..i2
.".uuzs^^F2...Rh.&.U.^^..J.P@.A......x..lǝy^?.u.p{4..g...m.,..R^.^.^......].^^.^J...p.ifTF0<.s.9V.o5<..%!6ļS.ƐǢ..㱋....C^&.....^.^y...v]^YT.1.#K.ibc...^.26...   ..7.
b.$...j6.٨f...W.R7.^1.3....K`%.&^..d..{{      l0..^\..^X.g.^.r.(!.^^...4.1.$\ .%.8$(.n&..^^q.,.Q..^.D^.].^.R9.kE.^.$^.I..<..B^..^.h^^C.^E.|....3o^.@..Z.^.s.$[v.
527
POST /upload/3 HTTP/1.0
Content-Length: 424
Host: xxxxxxxxx.dev.example.com
User-Agent: xxx (shell 1)

^.^........QMO.0^.++^zJw.ر^$^.^Ѣ.^V.J....vM.8r&.T+...{@pk%~C.G../z顲^.7....l...-.^W"cR..... .&^?u.U^^.^.....{^.^..8.^.^.I.EĂ.p...'^.3.Tq..@R8....RAiBU..1.Bd*".7+.
.Ol.j=^.3..n....wp..,Wg.y^.T..~^..
*/
type rawDecoder struct {
	protoDecoder
	reader *bufio.Reader
}

func (d *rawDecoder) Scan(ctx context.Context) (*base.Ammo, error) {
	var data string
	var buff []byte
	var req *http.Request
	var err error

	if d.config.Limit != 0 && d.ammoNum >= d.config.Limit {
		return nil, ErrAmmoLimit
	}
	for {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		data, err = d.reader.ReadString('\n')
		if err == io.EOF {
			d.passNum++
			if d.config.Passes != 0 && d.passNum >= d.config.Passes {
				return nil, ErrPassLimit
			}
			if d.ammoNum == 0 {
				return nil, ErrNoAmmo
			}
			_, err := d.file.Seek(0, io.SeekStart)
			if err != nil {
				return nil, err
			}
			d.reader.Reset(d.file)
			continue
		}
		if err != nil {
			return nil, xerrors.Errorf("reading ammo failed with err: %w, at position: %v", err, filePosition(d.file))
		}
		data = strings.TrimSpace(data)
		if len(data) == 0 {
			continue // skip empty lines
		}
		d.ammoNum++
		reqSize, tag, err := raw.DecodeHeader(data)
		if err != nil {
			return nil, xerrors.Errorf("header decoding error: %w", err)
		}

		if reqSize != 0 {
			if cap(buff) < reqSize {
				buff = make([]byte, reqSize)
			}
			buff = buff[:reqSize]
			if n, err := io.ReadFull(d.reader, buff); err != nil {
				return nil, xerrors.Errorf("failed to read ammo with err: %w, at position: %v; tried to read: %v; have read: %v", err, filePosition(d.file), reqSize, n)
			}
			req, err = raw.DecodeRequest(buff)
			if err != nil {
				return nil, xerrors.Errorf("failed to decode ammo with err: %w, at position: %v; data: %q", err, filePosition(d.file), buff)
			}
		} else {
			req, _ = http.NewRequest("", "/", nil)
		}

		// add new Headers to request from config
		util.EnrichRequestWithHeaders(req, d.decodedConfigHeaders)
		ammo := &base.Ammo{Req: req}
		ammo.SetTag(tag)
		return ammo, nil
	}
}