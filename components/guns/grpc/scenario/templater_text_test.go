package scenario

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTextTemplater_Apply(t *testing.T) {
	tests := []struct {
		name             string
		scenarioName     string
		stepName         string
		payload          []byte
		metadata         map[string]string
		vs               map[string]interface{}
		expectedMetadata map[string]string
		expectedPayload  string
		expectError      bool
	}{
		{
			name:         "Test Scenario 1",
			scenarioName: "TestScenario",
			stepName:     "TestStep",
			payload:      []byte(`{"name": "{{.name}}", "age": {{.age}}}`),
			metadata: map[string]string{
				"Authorization": "Bearer {{.token}}",
				"Content-Type":  "application/json",
			},
			vs: map[string]interface{}{
				"endpoint": "users",
				"token":    "abc123",
				"name":     "John",
				"age":      30,
			},
			expectedMetadata: map[string]string{
				"Authorization": "Bearer abc123",
				"Content-Type":  "application/json",
			},
			expectedPayload: `{"name": "John", "age": 30}`,
			expectError:     false,
		},
		{
			name:         "Test Scenario 2 (Invalid Template)",
			scenarioName: "TestScenario",
			stepName:     "TestStep",
			payload: []byte(`{
				URL: "http://example.com/{{.endpoint",
			}`),
			vs: map[string]interface{}{
				"endpoint": "users",
			},
			expectedMetadata: nil,
			expectedPayload:  "",
			expectError:      true,
		},
		{
			name:             "Test Scenario 3 (Empty Payload)",
			scenarioName:     "EmptyScenario",
			stepName:         "EmptyStep",
			payload:          []byte(`{}`),
			vs:               map[string]interface{}{},
			expectedMetadata: nil,
			expectedPayload:  "{}",
			expectError:      false,
		},
		{
			name:         "Test Scenario 4 (No Variables)",
			scenarioName: "NoVarsScenario",
			stepName:     "NoVarsStep",
			payload:      []byte(`{"name": "John", "age": 30}`),
			metadata: map[string]string{
				"Authorization": "Bearer abc123",
			},
			vs: map[string]interface{}{},
			expectedMetadata: map[string]string{
				"Authorization": "Bearer abc123",
			},
			expectedPayload: `{"name": "John", "age": 30}`,
			expectError:     false,
		},
		{
			name:         "Test Scenario 5 (Headers Only)",
			scenarioName: "HeaderScenario",
			stepName:     "HeaderStep",
			payload:      nil,
			metadata: map[string]string{
				"Authorization": "Bearer {{.token}}",
				"Content-Type":  "application/json",
			},
			vs: map[string]interface{}{
				"token": "xyz456",
			},
			expectedMetadata: map[string]string{
				"Authorization": "Bearer xyz456",
				"Content-Type":  "application/json",
			},
			expectedPayload: "",
			expectError:     false,
		},
		{
			name:         "Test Scenario 6 (Body Only)",
			scenarioName: "BodyScenario",
			stepName:     "BodyStep",
			payload:      []byte(`{"name": "{{.name}}", "age": {{.age}}}`),
			vs: map[string]interface{}{
				"name": "Alice",
				"age":  25,
			},
			expectedMetadata: nil,
			expectedPayload:  `{"name": "Alice", "age": 25}`,
			expectError:      false,
		},
		{
			name:             "Test Scenario 8 (Invalid Template in Headers)",
			scenarioName:     "InvalidHeaderScenario",
			stepName:         "InvalidHeaderStep",
			payload:          []byte(`{Headers: map[string]string{"Authorization": "Bearer {{.token",	},}`),
			vs:               map[string]interface{}{},
			expectedMetadata: nil,
			expectedPayload:  "",
			expectError:      true,
		},
		{
			name:         "Test Scenario 9 (Invalid Template in URL)",
			scenarioName: "InvalidURLScenario",
			stepName:     "InvalidURLStep",
			payload: []byte(`{
		URL: "http://example.com/{{.endpoint",
	}`),
			vs:               map[string]interface{}{},
			expectedMetadata: nil,
			expectedPayload:  "",
			expectError:      true,
		},
		{
			name:         "Test Scenario 10 (Invalid Template in Body)",
			scenarioName: "InvalidBodyScenario",
			stepName:     "InvalidBodyStep",
			payload: []byte(`{
	{"name": "{{.name}"}),
		}`),
			vs:               map[string]interface{}{},
			expectedMetadata: nil,
			expectedPayload:  "",
			expectError:      true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			templater := &TextTemplater{}
			payload, err := templater.Apply(test.payload, test.metadata, test.vs, test.scenarioName, test.stepName)

			if test.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, test.expectedMetadata, test.metadata)
				assert.Equal(t, test.expectedPayload, string(payload))
			}
		})
	}
}
