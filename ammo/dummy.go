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

// LogJSONDecode implements ammo.Decoder interface
func LogJSONDecode(jsonDoc []byte) (Ammo, error) {
	a := &Log{}
	err := json.Unmarshal(jsonDoc, a)
	return a, err
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
		if a, err := ap.Decode([]byte(fmt.Sprintf(`{"message": "Job #%d"}`, i))); err == nil {
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
			LogJSONDecode,
		),
	}
	return ap, nil
}
