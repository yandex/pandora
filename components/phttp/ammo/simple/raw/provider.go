package raw

import (
	"bufio"
	"context"
	"io"

	"github.com/pkg/errors"
	"github.com/spf13/afero"

	"github.com/yandex/pandora/components/phttp/ammo/simple"
)

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

func filePosition(file afero.File) (position int64) {
	position, _ = file.Seek(0, io.SeekCurrent)
	return
}

type Config struct {
	File string `validate:"required"`
	// Limit limits total num of ammo. Unlimited if zero.
	Limit int `validate:"min=0"`
	// Redefine HTTP headers
	Headers []string
	// Passes limits ammo file passes. Unlimited if zero.
	Passes int `validate:"min=0"`
}

func NewProvider(fs afero.Fs, conf Config) *Provider {
	var p Provider
	p = Provider{
		Provider: simple.NewProvider(fs, conf.File, p.start),
		Config:   conf,
	}
	return &p
}

type Provider struct {
	simple.Provider
	Config
}

func (p *Provider) start(ctx context.Context, ammoFile afero.File) error {
	var passNum int
	var ammoNum int
	// parse and prepare Headers from config
	decodedConfigHeaders, err := decodeHTTPConfigHeaders(p.Config.Headers)
	if err != nil {
		return err
	}
	for {
		passNum++
		reader := bufio.NewReader(ammoFile)
		for p.Limit == 0 || ammoNum < p.Limit {
			data, isPrefix, err := reader.ReadLine()
			if isPrefix {
				return errors.Errorf("too long header in ammo at position %v", filePosition(ammoFile))
			}
			if err == io.EOF {
				break // start over from the beginning
			}
			if err != nil {
				return errors.Wrapf(err, "reading ammo failed at position: %v", filePosition(ammoFile))
			}
			if len(data) == 0 {
				continue // skip empty lines
			}
			reqSize, tag, err := decodeHeader(data)
			if reqSize == 0 {
				break // start over from the beginning of file if ammo size is 0
			}
			buff := make([]byte, reqSize)
			if n, err := io.ReadFull(reader, buff); err != nil {
				return errors.Wrapf(err, "failed to read ammo at position: %v; tried to read: %v; have read: %v", filePosition(ammoFile), reqSize, n)
			}
			req, err := decodeRequest(buff)
			if err != nil {
				return errors.Wrapf(err, "failed to decode ammo at position: %v; data: %q", filePosition(ammoFile), buff)
			}

			// redefine request Headers from config
			for _, header := range decodedConfigHeaders {
				// special behavior for `Host` header
				if header.key == "Host" {
					req.URL.Host = header.value
				} else {
					req.Header.Set(header.key, header.value)
				}
			}

			sh := p.Pool.Get().(*simple.Ammo)
			sh.Reset(req, tag)

			select {
			case p.Sink <- sh:
				ammoNum++
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		if ammoNum == 0 {
			return errors.New("no ammo in file")
		}
		if p.Passes != 0 && passNum >= p.Passes {
			break
		}
		ammoFile.Seek(0, 0)
	}
	return nil
}
