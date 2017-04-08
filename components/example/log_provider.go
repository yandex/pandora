package example

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/yandex/pandora/core"
)

type Ammo struct {
	Message string
}

type LogProviderConfig struct {
	AmmoLimit int
}

func NewLogProvider(conf LogProviderConfig) core.Provider {
	ap := &logProvider{
		size: conf.AmmoLimit,
		BaseProvider: NewBaseProvider(
			128,
			newLogJSONDecoder(),
			func() interface{} { return &Ammo{} },
		),
	}
	return ap
}

// logJSONDecoder implements ammo.Decoder interface
type logJSONDecoder struct{}

func (*logJSONDecoder) Decode(jsonDoc []byte, a *Ammo) (*Ammo, error) {
	err := json.Unmarshal(jsonDoc, a)
	return a, err
}

func newLogJSONDecoder() Decoder {
	return &logJSONDecoder{}
}

type logProvider struct {
	*BaseProvider

	size int
}

func (ap *logProvider) Start(ctx context.Context) error {
	defer close(ap.Sink)
loop:
	for i := 0; i < ap.size; i++ {
		if a, err := ap.Decode([]byte(fmt.Sprintf(`{"message": "Job #%d"}`, i))); err == nil {
			select {
			case ap.Sink <- a:
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
