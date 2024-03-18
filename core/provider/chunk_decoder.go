package provider

import (
	"bufio"
	"fmt"

	"github.com/pkg/errors"
	"github.com/yandex/pandora/core"
)

var ErrNoAmmoDecoded = fmt.Errorf("no ammo has been decoded from chunk")

// ChunkAmmoDecoder accept data chunks that can contain encoded ammo or some meta information that
// changes ChunkAmmoDecoder state and affects next ammo decoding.
// For example, chunks are lines that contains HTTP URI to be transformed to http.Request or
// HTTP header to be added to next decoded http.Requests.
type ChunkAmmoDecoder interface {
	// DecodeChunk accepts chunk of data, than decode it to ammo or change ChunkAmmoDecoder internal
	// state.
	// Returns nil on when ammo was successfully decoded.
	// ErrNoAmmoDecoded MAY be returned, to indicate that chunk was accepted, but ammo were not
	// decoded.
	// Returns other non nil error, on chunk decode fail.
	// Panics if ammo type is not supported.
	DecodeChunk(chunk []byte, ammo core.Ammo) error
}

func NewScanDecoder(scanner Scanner, decoder ChunkAmmoDecoder) *ScanAmmoDecoder {
	return &ScanAmmoDecoder{scanner: scanner, decoder: decoder}
}

// Scanner is interface of bufio.Scanner like scanners.
type Scanner interface {
	Scan() bool
	Bytes() []byte
	Err() error
}

var _ Scanner = &bufio.Scanner{}

type ScanAmmoDecoder struct {
	chunkCounter int
	scanner      Scanner
	decoder      ChunkAmmoDecoder
}

var _ AmmoDecoder = &ScanAmmoDecoder{}

func (d *ScanAmmoDecoder) Decode(ammo core.Ammo) error {
	for {
		if !d.scanner.Scan() {
			return d.scanner.Err()
		}
		chunk := d.scanner.Bytes()
		err := d.decoder.DecodeChunk(chunk, ammo)
		if err == ErrNoAmmoDecoded {
			continue
		}
		if err != nil {
			return errors.Wrapf(err, "chunk %v decode failed", d.chunkCounter)
		}
		d.chunkCounter++
		return nil
	}

}
