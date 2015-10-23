package main

import (
	"bufio"
	"encoding/json"
	"log"
	"net/http"
	"os"
)

type HttpAmmo struct {
	Host    string
	Method  string
	Uri     string
	Headers map[string]string
	Tag     string
}

type HttpAmmoJsonDecoder struct{}

func (ha *HttpAmmoJsonDecoder) FromString(jsonDoc string) (a Ammo, err error) {
	a = &HttpAmmo{}
	err = json.Unmarshal([]byte(jsonDoc), a)
	return
}

func (ha *HttpAmmo) Request() (req *http.Request, err error) {
	//make a request
	req, err = http.NewRequest(ha.Method, "https://"+ha.Host+ha.Uri, nil)
	for k, v := range ha.Headers {
		req.Header.Set(k, v)
	}
	return
}

type HttpAmmoProvider struct {
	ammoProvider
	ammoFile  *os.File
	ammoLimit int
	loopLimit int
}

func (ap *HttpAmmoProvider) Start() {
	go func() { // requests reader/generator
		ammoNumber := 0
		loops := 0
		for {
			scanner := bufio.NewScanner(ap.ammoFile)
			scanner.Split(bufio.ScanLines)
			for scanner.Scan() && (ap.ammoLimit == 0 || ammoNumber < ap.ammoLimit) {
				txt := scanner.Text()
				if a, err := ap.decoder.FromString(txt); err != nil {
					log.Fatal("Failed to decode ammo: ", err)
				} else {
					ammoNumber += 1
					ap.source <- a
				}
			}
			if loops > ap.loopLimit {
				break
			}
			ap.ammoFile.Seek(0, 0)
			log.Printf("Restarted ammo the beginning. Loops left: %d\n", ap.loopLimit-loops)
			loops++
		}
		close(ap.source)
		log.Println("Ran out of ammo")
	}()
}

func NewHttpAmmoProvider(filename string, ammoLimit int, loopLimit int) (ap AmmoProvider, err error) {
	file, err := os.Open(filename)
	if err == nil {
		ap = &HttpAmmoProvider{
			ammoLimit: ammoLimit,
			loopLimit: loopLimit,
			ammoFile:  file,
			ammoProvider: ammoProvider{
				decoder: &HttpAmmoJsonDecoder{},
				source:  make(chan Ammo, 128),
			},
		}
	}
	return
}
