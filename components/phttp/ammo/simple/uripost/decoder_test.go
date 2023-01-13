package uripost

import (
	"reflect"
	"strconv"
	"testing"
)

func TestDecoderHeader(t *testing.T) {
	var tests = []struct {
		line     []byte
		key, val string
	}{
		{line: []byte("[Host: some.host]"), key: "Host", val: "some.host"},
		{line: []byte("[User-agent: Tank]"), key: "User-agent", val: "Tank"},
	}

	for _, test := range tests {
		if rkey, rval, _ := decodeHeader(test.line); rkey != test.key && rval != test.val {
			t.Errorf("(%v) = %v %v, expected %v %v", string(test.line), rkey, rval, test.key, test.val)
		}
	}

}

func TestDecoderBadHeader(t *testing.T) {
	var tests = []struct {
		line    []byte
		err_msg string
	}{
		{line: []byte("[Host some.host]"), err_msg: "missing colon"},
		{line: []byte("[User-agent: Tank"), err_msg: "header line should be like '[key: value]'"},
		{line: []byte("[: Tank]"), err_msg: "missing header key"},
	}

	for _, test := range tests {
		if _, _, err := decodeHeader(test.line); err.Error() != test.err_msg {
			t.Errorf("Got: %v, expected: %v", err.Error(), test.err_msg)
		}
	}

}

func TestDecodeURI(t *testing.T) {
	var tests = []struct {
		line     []byte
		size     int
		uri, tag string
	}{
		{line: []byte("7 /test tag1"), size: 7, uri: "/test", tag: "tag1"},
		{line: []byte("10 /test"), size: 10, uri: "/test", tag: ""},
	}

	for _, test := range tests {
		d_size, d_uri, d_tag, _ := decodeURI(test.line)
		if d_size != test.size && d_uri != test.uri && d_tag != test.tag {
			t.Errorf("Got: %v %v %v, expected: %v %v %v", strconv.Itoa(d_size), d_uri, d_tag, test.size, test.uri, test.tag)
		}
	}

}

func TestDecodeBadURI(t *testing.T) {
	var tests = []struct {
		line    []byte
		err_msg string
	}{
		{line: []byte("3"), err_msg: "Wrong ammo format, should be like 'bodySize uri [tag]'"},
		{line: []byte("a"), err_msg: "Wrong ammo body size, should be in bytes"},
	}

	for _, test := range tests {

		_, _, _, err := decodeURI(test.line)
		if err.Error() != test.err_msg {
			t.Errorf("Got: %v, expected: %v", err.Error(), test.err_msg)
		}
	}

}

func TestDecodeHTTPConfigHeaders(t *testing.T) {
	headers := []string{
		"[Host: some.host]",
		"[User-Agent: Tank]",
	}

	header := []Header{{key: "Host", value: "some.host"}, {key: "User-Agent", value: "Tank"}}
	configHeaders, err := decodeHTTPConfigHeaders(headers)
	if err == nil && !reflect.DeepEqual(configHeaders, header) {
		t.Errorf("Got: %v, expected: %v", configHeaders, header)
	}

}
