package uripost

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type DecodeURIWant struct {
	body     int
	uri, tag string
	err      error
}

func TestDecodeURI(t *testing.T) {
	var tests = []struct {
		name     string
		input    string
		want     DecodeURIWant
		body     int
		uri, tag string
	}{
		{
			name:  "Default uri",
			input: "7 /test tag1",
			want:  DecodeURIWant{body: 7, uri: "/test", tag: "tag1"},
		},
		{
			name:  "uri wothout tag",
			input: "10 /test",
			want:  DecodeURIWant{body: 10, uri: "/test", tag: ""},
		},
	}

	var ans DecodeURIWant
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			ans.body, ans.uri, ans.tag, ans.err = DecodeURI(tt.input)
			assert.Equal(tt.want, ans)
		})
	}

}

func TestDecodeBadURI(t *testing.T) {
	var tests = []struct {
		line   string
		errMsg string
	}{
		{line: "a a", errMsg: ErrWrongSize.Error()},
		{line: "3", errMsg: ErrAmmoFormat.Error()},
		{line: "a", errMsg: ErrAmmoFormat.Error()},
	}

	for _, test := range tests {

		_, _, _, err := DecodeURI(test.line)
		if err.Error() != test.errMsg {
			t.Errorf("Got: %v, expected: %v", err.Error(), test.errMsg)
		}
	}

}
