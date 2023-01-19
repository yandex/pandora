package simple

import (
	"reflect"
	"testing"
)

func TestDecodeHTTPConfigHeaders(t *testing.T) {
	headers := []string{
		"[Host: some.host]",
		"[User-Agent: Tank]",
	}

	header := []Header{{key: "Host", value: "some.host"}, {key: "User-Agent", value: "Tank"}}
	configHeaders, err := DecodeHTTPConfigHeaders(headers)
	if err == nil && !reflect.DeepEqual(configHeaders, header) {
		t.Errorf("Got: %v, expected: %v", configHeaders, header)
	}

}
