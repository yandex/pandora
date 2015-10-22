package main

import (
	"io/ioutil"
	"testing"
)

func TestPandora(t *testing.T) {
	jsonDoc, err := ioutil.ReadFile("./example/config/example.json")
	if err != nil {
		t.Errorf("Could not read config from file: %s", err)
		return
	}
	c, err := NewConfigFromJson(jsonDoc)
	if err != nil {
		t.Errorf("Could not unmarshal config from json: %s", err)
		return
	}
	PandoraRunConfig(c)
}
