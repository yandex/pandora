package templater

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gun "github.com/yandex/pandora/components/guns/http_scenario"
)

func TestTextTemplater_Apply(t *testing.T) {
	tests := []struct {
		name            string
		scenarioName    string
		stepName        string
		parts           *gun.RequestParts
		vs              map[string]interface{}
		expectedURL     string
		expectedHeaders map[string]string
		expectedBody    string
		expectError     bool
	}{
		{
			name:         "Test Scenario 1",
			scenarioName: "TestScenario",
			stepName:     "TestStep",
			parts: &gun.RequestParts{
				URL: "http://example.com/{{.endpoint}}",
				Headers: map[string]string{
					"Authorization": "Bearer {{.token}}",
					"Content-Type":  "application/json",
				},
				Body: []byte(`{"name": "{{.name}}", "age": {{.age}}}`),
			},
			vs: map[string]interface{}{
				"endpoint": "users",
				"token":    "abc123",
				"name":     "John",
				"age":      30,
			},
			expectedURL: "http://example.com/users",
			expectedHeaders: map[string]string{
				"Authorization": "Bearer abc123",
				"Content-Type":  "application/json",
			},
			expectedBody: `{"name": "John", "age": 30}`,
			expectError:  false,
		},
		{
			name:         "Test Scenario 2 (Invalid Template)",
			scenarioName: "TestScenario",
			stepName:     "TestStep",
			parts: &gun.RequestParts{
				URL: "http://example.com/{{.endpoint",
			},
			vs: map[string]interface{}{
				"endpoint": "users",
			},
			expectedURL:     "",
			expectedHeaders: nil,
			expectedBody:    "",
			expectError:     true,
		},
		{
			name:            "Test Scenario 3 (Empty RequestParts)",
			scenarioName:    "EmptyScenario",
			stepName:        "EmptyStep",
			parts:           &gun.RequestParts{},
			vs:              map[string]interface{}{},
			expectedURL:     "",
			expectedHeaders: nil,
			expectedBody:    "",
			expectError:     false,
		},
		{
			name:         "Test Scenario 4 (No Variables)",
			scenarioName: "NoVarsScenario",
			stepName:     "NoVarsStep",
			parts: &gun.RequestParts{
				URL: "http://example.com",
				Headers: map[string]string{
					"Authorization": "Bearer abc123",
				},
				Body: []byte(`{"name": "John", "age": 30}`),
			},
			vs:          map[string]interface{}{},
			expectedURL: "http://example.com",
			expectedHeaders: map[string]string{
				"Authorization": "Bearer abc123",
			},
			expectedBody: `{"name": "John", "age": 30}`,
			expectError:  false,
		},
		{
			name:         "Test Scenario 5 (URL Only)",
			scenarioName: "URLScenario",
			stepName:     "URLStep",
			parts: &gun.RequestParts{
				URL: "http://example.com/{{.endpoint}}",
			},
			vs: map[string]interface{}{
				"endpoint": "users",
			},
			expectedURL:     "http://example.com/users",
			expectedHeaders: nil,
			expectedBody:    "",
			expectError:     false,
		},
		{
			name:         "Test Scenario 6 (Headers Only)",
			scenarioName: "HeaderScenario",
			stepName:     "HeaderStep",
			parts: &gun.RequestParts{
				Headers: map[string]string{
					"Authorization": "Bearer {{.token}}",
					"Content-Type":  "application/json",
				},
			},
			vs: map[string]interface{}{
				"token": "xyz456",
			},
			expectedURL: "",
			expectedHeaders: map[string]string{
				"Authorization": "Bearer xyz456",
				"Content-Type":  "application/json",
			},
			expectedBody: "",
			expectError:  false,
		},
		{
			name:         "Test Scenario 7 (Body Only)",
			scenarioName: "BodyScenario",
			stepName:     "BodyStep",
			parts: &gun.RequestParts{
				Body: []byte(`{"name": "{{.name}}", "age": {{.age}}}`),
			},
			vs: map[string]interface{}{
				"name": "Alice",
				"age":  25,
			},
			expectedURL:     "",
			expectedHeaders: nil,
			expectedBody:    `{"name": "Alice", "age": 25}`,
			expectError:     false,
		},
		{
			name:         "Test Scenario 8 (Invalid Template in Headers)",
			scenarioName: "InvalidHeaderScenario",
			stepName:     "InvalidHeaderStep",
			parts: &gun.RequestParts{
				Headers: map[string]string{
					"Authorization": "Bearer {{.token",
				},
			},
			vs:              map[string]interface{}{},
			expectedURL:     "",
			expectedHeaders: nil,
			expectedBody:    "",
			expectError:     true,
		},
		{
			name:         "Test Scenario 9 (Invalid Template in URL)",
			scenarioName: "InvalidURLScenario",
			stepName:     "InvalidURLStep",
			parts: &gun.RequestParts{
				URL: "http://example.com/{{.endpoint",
			},
			vs:              map[string]interface{}{},
			expectedURL:     "",
			expectedHeaders: nil,
			expectedBody:    "",
			expectError:     true,
		},
		{
			name:         "Test Scenario 10 (Invalid Template in Body)",
			scenarioName: "InvalidBodyScenario",
			stepName:     "InvalidBodyStep",
			parts: &gun.RequestParts{
				Body: []byte(`{"name": "{{.name}"}`),
			},
			vs:              map[string]interface{}{},
			expectedURL:     "",
			expectedHeaders: nil,
			expectedBody:    "",
			expectError:     true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			templater := &TextTemplater{}
			err := templater.Apply(test.parts, test.vs, test.scenarioName, test.stepName)

			if test.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, test.expectedURL, test.parts.URL)
				assert.Equal(t, test.expectedHeaders, test.parts.Headers)
				assert.Equal(t, test.expectedBody, string(test.parts.Body))
			}
		})
	}
}

func TestTextTemplater_Apply_WithRandFunct(t *testing.T) {
	tests := []struct {
		name        string
		parts       *gun.RequestParts
		vs          map[string]interface{}
		assertBody  func(t *testing.T, body string)
		expectError bool
	}{
		{
			name:  "randInt with vars",
			parts: &gun.RequestParts{Body: []byte(`{{ randInt .from .to }}`)},
			vs: map[string]interface{}{
				"from": int64(10),
				"to":   30,
			},
			assertBody: func(t *testing.T, body string) {
				v, err := strconv.ParseInt(body, 10, 64)
				require.NoError(t, err)
				require.InDelta(t, 20, v, 10)
			},
			expectError: false,
		},
		{
			name:  "randInt with literals",
			parts: &gun.RequestParts{Body: []byte(`{{ randInt 10 30 }}`)},
			vs:    map[string]interface{}{},
			assertBody: func(t *testing.T, body string) {
				v, err := strconv.ParseInt(body, 10, 64)
				require.NoError(t, err)
				require.InDelta(t, 20, v, 10)
			},
			expectError: false,
		},
		{
			name:  "randInt with no args",
			parts: &gun.RequestParts{Body: []byte(`{{ randInt }}`)},
			vs:    map[string]interface{}{},
			assertBody: func(t *testing.T, body string) {
				v, err := strconv.ParseInt(body, 10, 64)
				require.NoError(t, err)
				require.InDelta(t, 5, v, 5)
			},
			expectError: false,
		},
		{
			name:  "randInt with literals",
			parts: &gun.RequestParts{Body: []byte(`{{ randInt -10 }}`)},
			vs:    map[string]interface{}{},
			assertBody: func(t *testing.T, body string) {
				v, err := strconv.ParseInt(body, 10, 64)
				require.NoError(t, err)
				require.InDelta(t, -5, v, 5)
			},
			expectError: false,
		},
		{
			name:        "randInt with invalid args",
			parts:       &gun.RequestParts{Body: []byte(`{{ randInt 10 "asdf" }}`)},
			vs:          map[string]interface{}{},
			assertBody:  nil,
			expectError: true,
		},
		{
			name:  "randString with 2 arg",
			parts: &gun.RequestParts{Body: []byte(`{{ randString 10 "asdfgzxcv" }}`)},
			vs:    map[string]interface{}{},
			assertBody: func(t *testing.T, body string) {
				require.Len(t, body, 10)
			},
			expectError: false,
		},
		{
			name:  "randString with 1 arg",
			parts: &gun.RequestParts{Body: []byte(`{{ randString 10 }}`)},
			vs:    map[string]interface{}{},
			assertBody: func(t *testing.T, body string) {
				require.Len(t, body, 10)
			},
			expectError: false,
		},
		{
			name:  "randString with 0 arg",
			parts: &gun.RequestParts{Body: []byte(`{{ randString }}`)},
			vs:    map[string]interface{}{},
			assertBody: func(t *testing.T, body string) {
				require.Len(t, body, 1)
			},
			expectError: false,
		},
		{
			name:        "randString with invalid arg",
			parts:       &gun.RequestParts{Body: []byte(`{{ randString "asdf" }}`)},
			vs:          map[string]interface{}{},
			assertBody:  nil,
			expectError: true,
		},
		{
			name:  "uuid",
			parts: &gun.RequestParts{Body: []byte(`{{ uuid }}`)},
			vs:    map[string]interface{}{},
			assertBody: func(t *testing.T, body string) {
				require.Len(t, body, 36)
			},
			expectError: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			templater := &TextTemplater{}
			err := templater.Apply(tt.parts, tt.vs, "scenarioName", "stepName")

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				tt.assertBody(t, string(tt.parts.Body))
			}
		})
	}
}
