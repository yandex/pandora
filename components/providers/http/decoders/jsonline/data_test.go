package jsonline

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yandex/pandora/components/providers/base"
)

func TestToRequest(t *testing.T) {
	var tests = []struct {
		name       string
		json       []byte
		confHeader http.Header
		want       *base.Ammo
		wantErr    bool
	}{
		{
			name:       "GET request",
			json:       []byte(`{"host": "ya.ru", "method": "GET", "uri": "/00", "tag": "tag", "headers": {"A": "a", "B": "b"}}`),
			confHeader: http.Header{"Default": []string{"def"}},
			want:       MustNewAmmo(t, "GET", "http://ya.ru/00", nil, http.Header{"Default": []string{"def"}, "A": []string{"a"}, "B": []string{"b"}}, "tag"),
			wantErr:    false,
		},
		{
			name:       "POST request",
			json:       []byte(`{"host": "ya.ru", "method": "POST", "uri": "/01?sleep=10", "tag": "tag", "headers": {"A": "a", "B": "b"}, "body": "body"}`),
			confHeader: http.Header{"Default": []string{"def"}},
			want:       MustNewAmmo(t, "POST", "http://ya.ru/01?sleep=10", []byte(`body`), http.Header{"Default": []string{"def"}, "A": []string{"a"}, "B": []string{"b"}}, "tag"),
			wantErr:    false,
		},
		{
			name:       "POST request with json",
			json:       []byte(`{"host": "ya.ru", "method": "POST", "uri": "/01?sleep=10", "tag": "tag", "headers": {"A": "a", "B": "b"}, "body": "{\"field\":\"value\"}"}`),
			confHeader: http.Header{"Default": []string{"def"}},
			want:       MustNewAmmo(t, "POST", "http://ya.ru/01?sleep=10", []byte(`{"field":"value"}`), http.Header{"Default": []string{"def"}, "A": []string{"a"}, "B": []string{"b"}}, "tag"),
			wantErr:    false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			ans, err := DecodeAmmo(tt.json, tt.confHeader)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			assert.NoError(err)
			assert.Equal(tt.want, ans)
		})
	}
}

func MustNewAmmo(t *testing.T, method string, url string, body []byte, header http.Header, tag string) *base.Ammo {
	ammo, err := base.NewAmmo(method, url, body, header, tag)
	require.NoError(t, err)
	return ammo
}
