package ammo

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/yandex/pandora/config"
	"golang.org/x/net/context"
)

type Log struct {
	Message string
}

// LogJSONDecoder implements ammo.Decoder interface
type LogJSONDecoder struct{}

func (*LogJSONDecoder) Decode(jsonDoc []byte, a Ammo) (Ammo, error) {
	err := json.Unmarshal(jsonDoc, a)
	return a, err
}

func NewLogJSONDecoder() Decoder {
	return &LogJSONDecoder{}
}

type LogProvider struct {
	*BaseProvider

	sink chan<- Ammo
	size int
}

func (ap *LogProvider) Start(ctx context.Context) error {
	defer close(ap.sink)
loop:
	for i := 0; i < ap.size; i++ {
		if a, err := ap.decode([]byte(fmt.Sprintf(`{"message": "Job #%d"}`, i))); err == nil {
			select {
			case ap.sink <- a:
			case <-ctx.Done():
				break loop
			}
		} else {
			return fmt.Errorf("Error decoding log ammo: %s", err)
		}
	}
	log.Println("Ran out of ammo")
	return nil
}

func NewLogAmmoProvider(c *config.AmmoProvider) (Provider, error) {
	ammoCh := make(chan Ammo, 128)
	ap := &LogProvider{
		size: c.AmmoLimit,
		sink: ammoCh,
		BaseProvider: NewBaseProvider(
			ammoCh,
			NewLogJSONDecoder(),
			func() interface{} { return &Log{} },
		),
	}
	return ap, nil
}
