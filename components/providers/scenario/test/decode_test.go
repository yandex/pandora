package test

import (
	"sync"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yandex/pandora/components/providers/scenario/config"
	_import "github.com/yandex/pandora/components/providers/scenario/import"
	"github.com/yandex/pandora/core/plugin/pluginconfig"
)

var testOnce = &sync.Once{}
var testFS = afero.NewOsFs()

func Test_ReadConfig_YamlAndHclSameResult(t *testing.T) {
	_import.Import(testFS)
	testOnce.Do(func() {
		pluginconfig.AddHooks()
	})

	t.Run("http", func(t *testing.T) {
		fromHCL, err := config.ReadAmmoConfig(testFS, "../testdata/http_payload.hcl")
		require.NoError(t, err)
		fromHCL.Locals = nil

		fromYaml, err := config.ReadAmmoConfig(testFS, "../testdata/http_payload.yaml")
		require.NoError(t, err)
		fromYaml.Locals = nil

		require.Equal(t, fromHCL, fromYaml)
	})
}

func Test_DecodeMap(t *testing.T) {
	_import.Import(testFS)
	testOnce.Do(func() {
		pluginconfig.AddHooks()
	})
	tests := []struct {
		name    string
		bytes   []byte
		want    *config.AmmoConfig
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "http",
			bytes: []byte(`variable_sources:
- name: users
  type: file/csv
  file: testdata/users.csv
  fields:
  - user_id
  - name
  - pass
  ignore_first_line: true
  delimiter: ','
- name: filter_src
  type: file/json
  file: testdata/filter.json
requests:
- name: auth_req
  method: POST
  uri: /auth
  headers:
    Content-Type: application/json
    Useragent: Yandex
  tag: auth
  body: |
    {"user_id":  {{.request.auth_req.preprocessor.user_id}}}
  preprocessor:
    mapping:
      user_id: source.users[next].user_id
  postprocessors:
  - type: var/header
    mapping:
      Content-Type: Content-Type|upper
      httpAuthorization: Http-Authorization
  - type: var/jsonpath
    mapping:
      token: $.auth_key
  - type: assert/response
    headers:
      Content-Type: json
    body:
    - key
    size:
      val: 40
      op: '>'
  - type: assert/response
    body:
    - auth
  templater:
    type: html
- name: list_req
  method: GET
  uri: /list
  headers:
    Authorization: Bearer {{.request.auth_req.postprocessor.token}}
    Content-Type: application/json
    Useragent: Yandex
  tag: list
  postprocessors:
  - type: var/jsonpath
    mapping:
      item_id: $.items[0]
      items: $.items
- name: order_req
  method: POST
  uri: /order
  headers:
    Authorization: Bearer {{.request.auth_req.postprocessor.token}}
    Content-Type: application/json
    Useragent: Yandex
  tag: order_req
  body: |
    {"item_id": {{.request.order_req.preprocessor.item}}}
  preprocessor:
    mapping:
      item: request.list_req.postprocessor.items[next]
- name: order_req2
  method: POST
  uri: /order
  headers:
    Authorization: Bearer {{.request.auth_req.postprocessor.token}}
    Content-Type: application/json
    Useragent: Yandex
  tag: order_req
  body: |
    {"item_id": {{.request.order_req2.preprocessor.item}}  }
  preprocessor:
    mapping:
      item: request.list_req.postprocessor.items[next]
calls: []
scenarios:
- name: scenario_name
  weight: 50
  min_waiting_time: 10
  requests:
  - auth_req(1)
  - sleep(100)
  - list_req(1)
  - sleep(100)
  - order_req(3)
`),
			want:    &config.AmmoConfig{},
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := config.DecodeMap(tt.bytes)
			if !tt.wantErr(t, err) {
				return
			}
		})
	}
}
