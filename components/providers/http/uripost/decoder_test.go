package uripost

import (
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
		line   []byte
		errMsg string
	}{
		{line: []byte("[Host some.host]"), errMsg: "missing colon"},
		{line: []byte("[User-agent: Tank"), errMsg: "header line should be like '[key: value]'"},
		{line: []byte("[: Tank]"), errMsg: "missing header key"},
	}

	for _, test := range tests {
		if _, _, err := decodeHeader(test.line); err.Error() != test.errMsg {
			t.Errorf("Got: %v, expected: %v", err.Error(), test.errMsg)
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
		dSize, dURI, dTag, _ := decodeURI(test.line)
		if dSize != test.size && dURI != test.uri && dTag != test.tag {
			t.Errorf("Got: %v %v %v, expected: %v %v %v", strconv.Itoa(dSize), dURI, dTag, test.size, test.uri, test.tag)
		}
	}

}

func TestDecodeBadURI(t *testing.T) {
	var tests = []struct {
		line   []byte
		errMsg string
	}{
		{line: []byte("3"), errMsg: "Wrong ammo format, should be like 'bodySize uri [tag]'"},
		{line: []byte("a"), errMsg: "Wrong ammo body size, should be in bytes"},
	}

	for _, test := range tests {

		_, _, _, err := decodeURI(test.line)
		if err.Error() != test.errMsg {
			t.Errorf("Got: %v, expected: %v", err.Error(), test.errMsg)
		}
	}

}
