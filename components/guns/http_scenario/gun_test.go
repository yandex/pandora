package httpscenario

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	phttp "github.com/yandex/pandora/components/guns/http"
	"github.com/yandex/pandora/core"
	"github.com/yandex/pandora/core/aggregator/netsample"
	"go.uber.org/zap"
)

func TestBaseGun_shoot(t *testing.T) {
	type fields struct {
		DebugLog       bool
		Config         phttp.BaseGunConfig
		Connect        func(ctx context.Context) error
		OnClose        func() error
		Aggregator     netsample.Aggregator
		AnswLog        *zap.Logger
		GunDeps        core.GunDeps
		scheme         string
		hostname       string
		targetResolved string
		client         Client
	}

	tests := []struct {
		name            string
		templateVars    map[string]any
		wantTempateVars map[string]any
		ammoMock        func(t *testing.T, m *MockAmmo)
		stepMocks       []func(t *testing.T, m *MockStep)
		clientMock      func(t *testing.T, m *MockClient)
		fields          fields
		wantErr         assert.ErrorAssertionFunc
	}{
		{
			name:         "default",
			templateVars: map[string]any{"source": map[string]any{"users": []map[string]any{{"id": 1, "name": "test1"}, {"id": 2, "name": "test2"}}}},
			wantTempateVars: map[string]any{
				"request": map[string]any{
					"step 1": map[string]any{"postprocessor": map[string]any{}},
					"step 2": map[string]any{"postprocessor": map[string]any{}},
				},
				"source": map[string]any{"users": []map[string]any{{"id": 1, "name": "test1"}, {"id": 2, "name": "test2"}}},
			},
			stepMocks: []func(t *testing.T, m *MockStep){
				func(t *testing.T, step *MockStep) {
					templater := NewMockTemplater(t)
					templater.On("Apply", mock.Anything, mock.Anything, "testAmmo", "step 1").Return(nil)
					step.On("Preprocessor").Return(nil).Times(1)

					commonStepMocks(t, step, "step 1", "tag1", "http://localhost:8080", "GET", nil, map[string]string{"Content-Type": "application/json"}, templater)

					step.On("GetPostProcessors").Return(nil).Times(1)
				},
				func(t *testing.T, step *MockStep) {
					templater := NewMockTemplater(t)
					templater.On("Apply", mock.Anything, mock.Anything, "testAmmo", "step 2").Return(nil)
					step.On("Preprocessor").Return(nil).Times(1)

					commonStepMocks(t, step, "step 2", "tag2", "http://localhost:8080", "GET", nil, map[string]string{"Content-Type": "application/json"}, templater)

					step.On("GetPostProcessors").Return(nil).Times(1)
				},
			},
			ammoMock: func(t *testing.T, ammo *MockAmmo) {
				ammo.On("ID").Return(uint64(0)).Times(2)
				ammo.On("Name").Return("testAmmo").Times(4)
				ammo.On("GetMinWaitingTime").Return(time.Duration(0))
			},
			clientMock: func(t *testing.T, client *MockClient) {
				body := io.NopCloser(strings.NewReader("test response body"))
				resp := &http.Response{Body: body}
				client.On("Do", mock.Anything).Return(resp, nil) //TODO: check response after template
			},
			wantErr: assert.NoError,
		},
		{
			name:         "check preprocessor",
			templateVars: map[string]any{"source": map[string]any{"users": []map[string]any{{"id": 1, "name": "test1"}, {"id": 2, "name": "test2"}}}},
			wantTempateVars: map[string]any{
				"request": map[string]any{
					"step 3": map[string]any{
						"preprocessor": map[string]any{
							"preprocessor_var": "preprocessor_test",
						},
						"postprocessor": map[string]any{},
					},
				},
				"source": map[string]any{"users": []map[string]any{{"id": 1, "name": "test1"}, {"id": 2, "name": "test2"}}},
			},
			stepMocks: []func(t *testing.T, m *MockStep){
				func(t *testing.T, step *MockStep) {
					templater := NewMockTemplater(t)
					templater.On("Apply", mock.Anything, mock.Anything, "testAmmo", "step 3").Return(nil)
					preprocessor := NewMockPreprocessor(t)
					preprocessor.On("Process", mock.Anything).Return(func(templVars map[string]any) map[string]any {
						return map[string]any{"preprocessor_var": "preprocessor_test"}
					}, nil).Times(1)
					step.On("Preprocessor").Return(preprocessor).Times(1)
					commonStepMocks(t, step, "step 3", "tag3", "http://localhost:8080", "GET", nil, map[string]string{"Content-Type": "application/json"}, templater)

					step.On("GetPostProcessors").Return(nil).Times(1)
				},
			},
			ammoMock: func(t *testing.T, ammo *MockAmmo) {
				ammo.On("ID").Return(uint64(0)).Times(1)
				ammo.On("Name").Return("testAmmo").Times(2)
				ammo.On("GetMinWaitingTime").Return(time.Duration(0))
			},
			clientMock: func(t *testing.T, client *MockClient) {
				body := io.NopCloser(strings.NewReader("test response body"))
				resp := &http.Response{Body: body}
				client.On("Do", mock.Anything).Return(resp, nil) //TODO: check response after template
			},
			wantErr: assert.NoError,
		},
		{
			name:         "check postprocessor",
			templateVars: map[string]any{"source": map[string]any{"users": []map[string]any{{"id": 1, "name": "test1"}, {"id": 2, "name": "test2"}}}},
			wantTempateVars: map[string]any{
				"request": map[string]any{
					"step 4": map[string]any{
						"postprocessor": map[string]any{
							"token":         "body_token",
							"Conteant-Type": "application/json",
						},
					},
				},
				"source": map[string]any{"users": []map[string]any{{"id": 1, "name": "test1"}, {"id": 2, "name": "test2"}}},
			},
			stepMocks: []func(t *testing.T, m *MockStep){
				func(t *testing.T, step *MockStep) {
					templater := NewMockTemplater(t)
					templater.On("Apply", mock.Anything, mock.Anything, "testAmmo", "step 4").Return(nil)
					step.On("Preprocessor").Return(nil).Times(1)
					commonStepMocks(t, step, "step 4", "tag3", "http://localhost:8080", "GET", nil, map[string]string{"Content-Type": "application/json"}, templater)

					postprocessor1 := NewMockPostprocessor(t)
					postprocessor1.On("Process", mock.Anything, mock.Anything).Return(func(resp *http.Response, body io.Reader) map[string]any {
						return map[string]any{"token": "body_token"}
					}, nil)
					postprocessor2 := NewMockPostprocessor(t)
					postprocessor2.On("Process", mock.Anything, mock.Anything).Return(func(resp *http.Response, body io.Reader) map[string]any {
						return map[string]any{"Conteant-Type": "application/json"}
					}, nil)
					postprocessors := []Postprocessor{postprocessor1, postprocessor2}
					step.On("GetPostProcessors").Return(postprocessors).Times(1)
				},
			},
			ammoMock: func(t *testing.T, ammo *MockAmmo) {
				ammo.On("ID").Return(uint64(0)).Times(1)
				ammo.On("Name").Return("testAmmo").Times(2)
				ammo.On("GetMinWaitingTime").Return(time.Duration(0))
			},
			clientMock: func(t *testing.T, client *MockClient) {
				body := io.NopCloser(strings.NewReader("test response body"))
				resp := &http.Response{Body: body}
				client.On("Do", mock.Anything).Return(resp, nil) //TODO: check response after template
			},
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			steps := make([]Step, 0, len(tt.stepMocks))
			for _, step := range tt.stepMocks {
				st := NewMockStep(t)
				step(t, st)
				steps = append(steps, st)
			}

			ammo := NewMockAmmo(t)
			ammo.On("Steps").Return(steps)
			tt.ammoMock(t, ammo)

			client := NewMockClient(t)
			tt.clientMock(t, client)

			aggregator := netsample.NewMockAggregator(t)
			aggregator.On("Report", mock.Anything)

			g := &BaseGun{Aggregator: aggregator, client: client}
			tt.wantErr(t, g.shoot(ammo, tt.templateVars), fmt.Sprintf("shoot(%v)", ammo))
			require.Equal(t, tt.wantTempateVars, tt.templateVars)
		})
	}
}

func commonStepMocks(t *testing.T, step *MockStep, name, tag, url, method string, body []byte, headers map[string]string, tmpl Templater) {
	t.Helper()

	step.On("GetURL").Return(url).Times(1)
	step.On("GetMethod").Return(method).Times(1)
	step.On("GetBody").Return(body).Times(1)
	step.On("GetHeaders").Return(headers).Times(1)
	step.On("GetTag").Return(tag).Times(1)
	step.On("GetTemplater").Return(tmpl).Times(1)
	step.On("GetName").Return(name).Times(2)
	step.On("GetSleep").Return(time.Duration(0)).Times(1)
}
