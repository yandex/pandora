package ammo

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/yandex/pandora/config"
	"golang.org/x/net/context"
)

type Http struct {
	Host    string
	Method  string
	Uri     string
	Headers map[string]string
	Tag     string
}

func (ha *Http) Request() (*http.Request, error) {
	// FIXME: something wrong here with https
	req, err := http.NewRequest(ha.Method, "https://"+ha.Host+ha.Uri, nil)
	if err == nil {
		for k, v := range ha.Headers {
			req.Header.Set(k, v)
		}
	}
	return req, err
}

// HttpJSONDecode implements ammo.Decoder interface
func HttpJSONDecode(jsonDoc []byte) (Ammo, error) {
	a := &Http{}
	err := json.Unmarshal(jsonDoc, a)
	return a, err
}

type HttpProvider struct {
	*BaseProvider

	sink      chan<- Ammo
	ammoFile  io.ReadSeeker
	ammoLimit int
	passes    int
}

func (ap *HttpProvider) Start(ctx context.Context) error {
	defer close(ap.sink)
	ammoNumber := 0
	passNum := 0
	for {
		passNum++
		scanner := bufio.NewScanner(ap.ammoFile)
		scanner.Split(bufio.ScanLines)
		for scanner.Scan() && (ap.ammoLimit == 0 || ammoNumber < ap.ammoLimit) {
			data := scanner.Bytes()
			if a, err := ap.Decode(data); err != nil {
				return fmt.Errorf("failed to decode ammo: %v", err)
			} else {
				ammoNumber++
				ap.sink <- a
			}
		}
		if ap.passes != 0 && passNum >= ap.passes {
			break
		}
		ap.ammoFile.Seek(0, 0)
		if ap.passes == 0 {
			log.Printf("Restarted ammo from the beginning. Infinite passes.\n")
		} else {
			log.Printf("Restarted ammo from the beginning. Passes left: %d\n", ap.passes-passNum)
		}
	}
	log.Println("Ran out of ammo")
	return nil
}

func NewHttpProvider(c *config.AmmoProvider) (Provider, error) {
	// When we should close a file?
	// Also I'm not sure that we should open a file here but in Start method
	file, err := os.Open(c.AmmoSource)
	if err != nil {
		return nil, err
	}
	ammoCh := make(chan Ammo)
	ap := &HttpProvider{
		ammoLimit: c.AmmoLimit,
		passes:    c.Passes,
		ammoFile:  file,
		sink:      ammoCh,
		BaseProvider: NewBaseProvider(
			ammoCh,
			HttpJSONDecode,
		),
	}
	return ap, nil
}
