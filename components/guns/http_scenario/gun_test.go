package httpscenario

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

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
		Config         phttp.GunConfig
		Connect        func(ctx context.Context) error
		OnClose        func() error
		Aggregator     netsample.Aggregator
		AnswLog        *zap.Logger
		GunDeps        core.GunDeps
		scheme         string
		hostname       string
		targetResolved string
		client         phttp.Client
	}

	tests := []struct {
		name            string
		templateVars    map[string]any
		wantTempateVars map[string]any
		ammoMock        *Scenario
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
			ammoMock: &Scenario{
				Requests: []Request{
					{
						Name:      "step 1",
						URI:       "http://localhost:8080",
						Method:    "GET",
						Headers:   map[string]string{"Content-Type": "application/json"},
						Tag:       "tag1",
						Templater: &MockTemplater{err: nil, applyCalls: 1, expectedArgs: [][2]string{{"testAmmo", "step 1"}}},
					},
					{
						Name:      "step 2",
						URI:       "http://localhost:8080",
						Method:    "GET",
						Headers:   map[string]string{"Content-Type": "application/json"},
						Tag:       "tag2",
						Templater: &MockTemplater{err: nil, applyCalls: 1, expectedArgs: [][2]string{{"testAmmo", "step 2"}}},
					},
				},
				ID:             2,
				Name:           "testAmmo",
				MinWaitingTime: 0,
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
			ammoMock: &Scenario{
				Requests: []Request{
					{
						Name:      "step 3",
						Tag:       "tag3",
						URI:       "http://localhost:8080",
						Method:    "GET",
						Headers:   map[string]string{"Content-Type": "application/json"},
						Templater: &MockTemplater{err: nil, applyCalls: 1, expectedArgs: [][2]string{{"testAmmo", "step 3"}}},
						Preprocessor: &mockPreprocessor{
							t:                  t,
							processExpectCalls: 1,
							processArgsReturns: []mockPreprocessorArgsReturns{{
								templateVars: map[string]any{"request": map[string]any{"step 3": map[string]any{}}, "source": map[string]any{"users": []map[string]any{{"id": 1, "name": "test1"}, {"id": 2, "name": "test2"}}}},
								returnVars:   map[string]any{"preprocessor_var": "preprocessor_test"},
								returnErr:    nil,
							}},
						},
					}},
				Name: "testAmmo",
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
			ammoMock: &Scenario{
				Requests: []Request{
					{
						Name:      "step 4",
						Tag:       "tag4",
						URI:       "http://localhost:8080",
						Method:    "GET",
						Headers:   map[string]string{"Content-Type": "application/json"},
						Templater: &MockTemplater{err: nil, applyCalls: 1, expectedArgs: [][2]string{{"testAmmo", "step 3"}}},
						Postprocessors: []Postprocessor{
							&mockPostprocessor{
								t:                  t,
								processExpectCalls: 1,
								processArgsReturns: []mockPostprocessorArgsReturns{
									{
										returnVars: map[string]any{"token": "body_token"},
										returnErr:  nil,
									},
								},
							},
							&mockPostprocessor{
								t:                  t,
								processExpectCalls: 1,
								processArgsReturns: []mockPostprocessorArgsReturns{
									{
										returnVars: map[string]any{"Conteant-Type": "application/json"},
										returnErr:  nil,
									},
								},
							},
						},
					}},
				Name: "testAmmo",
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

			client := NewMockClient(t)
			tt.clientMock(t, client)

			aggregator := netsample.NewMockAggregator(t)
			aggregator.On("Report", mock.Anything)

			g := &ScenarioGun{base: &phttp.BaseGun{Aggregator: aggregator, Client: client}}
			tt.wantErr(t, g.shoot(tt.ammoMock, tt.templateVars), fmt.Sprintf("shoot(%v)", tt.ammoMock))
			require.Equal(t, tt.wantTempateVars, tt.templateVars)

			for _, req := range tt.ammoMock.Requests {
				if req.Preprocessor != nil {
					req.Preprocessor.(*mockPreprocessor).validateCalls(t, req.Name)
				}
				if req.Templater != nil {
					req.Templater.(*MockTemplater).validateCalls(t, req.Name)
				}
				for _, postprocessor := range req.Postprocessors {
					postprocessor.(*mockPostprocessor).validateCalls(t, req.Name)
				}
			}
		})
	}
}

var _ Postprocessor = (*mockPostprocessor)(nil)

type mockPostprocessorArgsReturns struct {
	returnVars map[string]any
	returnErr  error
}

type mockPostprocessor struct {
	t                  *testing.T
	processExpectCalls int
	processArgsReturns []mockPostprocessorArgsReturns
	i                  int
}

func (m *mockPostprocessor) Process(resp *http.Response, body io.Reader) (map[string]any, error) {
	m.processExpectCalls--
	require.NotEqual(m.t, 0, len(m.processArgsReturns), "wrong postprocessor.Process calls")

	i := m.i % len(m.processArgsReturns)
	m.i++
	return m.processArgsReturns[i].returnVars, m.processArgsReturns[i].returnErr
}

func (m *mockPostprocessor) validateCalls(t *testing.T, stepName string) {
	if m == nil {
		return
	}
	assert.Equalf(t, 0, m.processExpectCalls, "wrong preprocessor.Process calls with step name `%s`", stepName)
}

var _ Preprocessor = (*mockPreprocessor)(nil)

type mockPreprocessorArgsReturns struct {
	templateVars map[string]any
	returnVars   map[string]any
	returnErr    error
}

type mockPreprocessor struct {
	t                  *testing.T
	processExpectCalls int
	processArgsReturns []mockPreprocessorArgsReturns
	invalidArgs        []error
	i                  int
}

func (m *mockPreprocessor) Process(templateVars map[string]any) (map[string]any, error) {
	m.processExpectCalls--
	if len(m.processArgsReturns) == 0 {
		err := fmt.Errorf("forgot init mockPreprocessor.processArgsReturns; call Process(%+v)", templateVars)
		m.invalidArgs = append(m.invalidArgs, err)
		return nil, err
	}

	i := m.i % len(m.processArgsReturns)
	m.i++
	args, returnVars, returnErr := m.processArgsReturns[i].templateVars, m.processArgsReturns[i].returnVars, m.processArgsReturns[i].returnErr

	if !assert.Equalf(m.t, args, templateVars, "unexpected arg templateVars; call#%d Process(%+v)", m.i-1, templateVars) {
		m.invalidArgs = append(m.invalidArgs, fmt.Errorf("unexpected arg templateVars; call Process(%+v)", templateVars))
	}
	return returnVars, returnErr
}

func (m *mockPreprocessor) validateCalls(t *testing.T, stepName string) {
	if m == nil {
		return
	}
	assert.Equalf(t, 0, m.processExpectCalls, "wrong preprocessor.Process calls with step name `%s`", stepName)
}

var _ Templater = (*MockTemplater)(nil)

type MockTemplater struct {
	err          error
	applyCalls   int
	expectedArgs [][2]string
	invalidArgs  []error
	i            int
}

func (m *MockTemplater) Apply(request *RequestParts, variables map[string]any, scenarioName, stepName string) error {
	if len(m.expectedArgs) == 0 {
		m.invalidArgs = append(m.invalidArgs, fmt.Errorf("forgot init mockTemplate.expectedArgs; call Apply(%+v, %+v, %s, %s)", request, variables, scenarioName, stepName))
	} else {
		i := m.i % len(m.expectedArgs)
		m.i++
		args := m.expectedArgs[i]
		if args[0] != scenarioName {
			m.invalidArgs = append(m.invalidArgs, fmt.Errorf("unexpected arg scenarioName `%s != %s`; call Apply(%+v, %+v, %s, %s)", args[0], scenarioName, request, variables, scenarioName, stepName))
		}
		if args[1] != stepName {
			m.invalidArgs = append(m.invalidArgs, fmt.Errorf("unexpected arg stepName `%s != %s`; call Apply(%+v, %+v, %s, %s)", args[1], stepName, request, variables, scenarioName, stepName))
		}
	}
	m.applyCalls--
	return m.err
}

func (m *MockTemplater) validateCalls(t *testing.T, stepName string) {
	assert.Equalf(t, 0, m.applyCalls, "wrong template.applyCalls calls with step name `%s`", stepName)
}
