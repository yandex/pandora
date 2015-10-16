package main

import (
	"http"
)

type HttpAmmo struct {
	Host    string
	Method  string
	Uri     string
	Headers map[string]string
}

func NewAmmoFromJson(jsonDoc string) (a *Ammo, err error) {
	err = json.Unmarshal([]byte(txt), &a)
	return
}

func (ha *HttpAmmo) Request() (req *http.Request, err error) {
	//make a request
	req, err = http.NewRequest(a.Method, "https://"+a.Host+a.Uri, nil)
	for k, v := range a.Headers {
		req.Header.Set(k, v)
	}
	return
}
