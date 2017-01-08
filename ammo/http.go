//go:generate ffjson $GOFILE

package ammo

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
)

func NewHttpProvider(conf HttpProviderConfig) Provider {
	ammoCh := make(chan Ammo)
	return &httpProvider{
		HttpProviderConfig: conf,
		sink:               ammoCh,
		BaseProvider: NewBaseProvider(
			ammoCh,
			&httpJSONDecoder{},
			func() interface{} { return &Http{} },
		),
	}
}

// ffjson: skip
type httpProvider struct {
	*BaseProvider
	sink chan<- Ammo
	HttpProviderConfig
}

// ffjson: skip
type HttpProviderConfig struct {
	AmmoFileName string
	AmmoLimit    int
	Passes       int
}

// ffjson: noencoder
type Http struct {
	Host    string
	Method  string
	Uri     string
	Headers map[string]string
	Tag     string
}

// httpJSONDecoder implements ammo.Decoder interface
type httpJSONDecoder struct{}

func (d *httpJSONDecoder) Decode(jsonDoc []byte, a Ammo) (Ammo, error) {
	err := a.(*Http).UnmarshalJSON(jsonDoc)
	return a, err
}

func (ap *httpProvider) Start(ctx context.Context) error {
	defer close(ap.sink)
	ammoFile, err := os.Open(ap.AmmoFileName)
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
		for scanner.Scan() && (ap.AmmoLimit == 0 || ammoNumber < ap.AmmoLimit) {
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
		if ap.Passes != 0 && passNum >= ap.Passes {
			break
		}
		ammoFile.Seek(0, 0)
		if ap.Passes == 0 {
			evPassesLeft.Set(-1)
			//log.Printf("Restarted ammo from the beginning. Infinite passes.\n")
		} else {
			evPassesLeft.Set(int64(ap.Passes - passNum))
			//log.Printf("Restarted ammo from the beginning. Passes left: %d\n", ap.passes-passNum)
		}
	}
	log.Println("Ran out of ammo")
	return nil
}
