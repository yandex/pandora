package ammo

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
)

type Log struct {
	Message string
}

type LogAmmoProviderConfig struct {
	AmmoLimit int
}

func NewLogAmmoProvider(conf LogAmmoProviderConfig) Provider {
	ammoCh := make(chan Ammo, 128)
	ap := &logProvider{
		size: conf.AmmoLimit,
		sink: ammoCh,
		BaseProvider: NewBaseProvider(
			ammoCh,
			newLogJSONDecoder(),
			func() interface{} { return &Log{} },
		),
	}
	return ap
}

// logJSONDecoder implements ammo.Decoder interface
type logJSONDecoder struct{}

func (*logJSONDecoder) Decode(jsonDoc []byte, a Ammo) (Ammo, error) {
	err := json.Unmarshal(jsonDoc, a)
	return a, err
}

func newLogJSONDecoder() Decoder {
	return &logJSONDecoder{}
}

type logProvider struct {
	*BaseProvider

	sink chan<- Ammo
	size int
}

func (ap *logProvider) Start(ctx context.Context) error {
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
