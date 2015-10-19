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
	decoder  AmmoDecoder
	ammoFile *os.File
	source   chan Ammo
}

func (ap *HttpAmmoProvider) Source() (s chan Ammo) {
	return ap.source
}

func (ap *HttpAmmoProvider) Start() {
	go func() { // requests reader/generator
		scanner := bufio.NewScanner(ap.ammoFile)
		scanner.Split(bufio.ScanLines)

		for scanner.Scan() {
			txt := scanner.Text()
			if a, err := ap.decoder.FromString(txt); err != nil {
				log.Println("Failed to decode ammo: %s", err)
			} else {
				ap.source <- a
			}
		}
		close(ap.source)
		log.Println("Ran out of ammo")
	}()
}

func NewHttpAmmoProvider(filename string) (ap AmmoProvider, err error) {
	if file, err := os.Open(filename); err == nil {
		ap = &HttpAmmoProvider{
			decoder:  &HttpAmmoJsonDecoder{},
			ammoFile: file,
			source:   make(chan Ammo, 128),
		}
	}
	return
}
