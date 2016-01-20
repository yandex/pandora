//go:generate ffjson $GOFILE

package ammo

import (
	"bufio"
	"fmt"
	"log"
	"os"

	"github.com/yandex/pandora/config"
	"golang.org/x/net/context"
)

// ffjson: noencoder
type Http struct {
	Host    string
	Method  string
	Uri     string
	Headers map[string]string
	Tag     string
}

// HttpJSONDecoder implements ammo.Decoder interface
type HttpJSONDecoder struct{}

func (d *HttpJSONDecoder) Decode(jsonDoc []byte, a Ammo) (Ammo, error) {
	err := a.(*Http).UnmarshalJSON(jsonDoc)
	return a, err
}

// ffjson: skip
type HttpProvider struct {
	*BaseProvider

	sink         chan<- Ammo
	ammoFileName string
	ammoLimit    int
	passes       int
}

func (ap *HttpProvider) Start(ctx context.Context) error {
	defer close(ap.sink)
	ammoFile, err := os.Open(ap.ammoFileName)
	if err != nil {
		return fmt.Errorf("failed to open ammo source: %v", err)
	}
	defer ammoFile.Close()
	ammoNumber := 0
	passNum := 0
loop:
	for {
		passNum++
		scanner := bufio.NewScanner(ammoFile)
		scanner.Split(bufio.ScanLines)
		for scanner.Scan() && (ap.ammoLimit == 0 || ammoNumber < ap.ammoLimit) {
			data := scanner.Bytes()
			if a, err := ap.decode(data); err != nil {
				return fmt.Errorf("failed to decode ammo: %v", err)
			} else {
				ammoNumber++
				select {
				case ap.sink <- a:
				case <-ctx.Done():
					break loop
				}
			}
		}
		if ap.passes != 0 && passNum >= ap.passes {
			break
		}
		ammoFile.Seek(0, 0)
		if ap.passes == 0 {
			evPassesLeft.Set(-1)
			//log.Printf("Restarted ammo from the beginning. Infinite passes.\n")
		} else {
			evPassesLeft.Set(int64(ap.passes - passNum))
			//log.Printf("Restarted ammo from the beginning. Passes left: %d\n", ap.passes-passNum)
		}
	}
	log.Println("Ran out of ammo")
	return nil
}

func NewHttpProvider(c *config.AmmoProvider) (Provider, error) {
	ammoCh := make(chan Ammo)
	ap := &HttpProvider{
		ammoLimit:    c.AmmoLimit,
		passes:       c.Passes,
		ammoFileName: c.AmmoSource,
		sink:         ammoCh,
		BaseProvider: NewBaseProvider(
			ammoCh,
			&HttpJSONDecoder{},
			func() interface{} { return &Http{} },
		),
	}
	return ap, nil
}
