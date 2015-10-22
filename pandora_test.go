package main

import (
	"testing"
)

func TestPandora(t *testing.T) {
	c, err := NewConfigFromJson([]byte(exampleConfig))
	if err != nil {
		t.Errorf("Could not unmarshal config from json: %s", err)
		return
	}
	PandoraRunConfig(c)
}
