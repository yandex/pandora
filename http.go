package main

import (
	"encoding/json"
	"net/http"
)

type HttpAmmo struct {
	Host    string
	Method  string
	Uri     string
	Headers map[string]string
}

func (ha *HttpAmmo) FromJson(jsonDoc string) (err error) {
	err = json.Unmarshal([]byte(jsonDoc), &ha)
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
