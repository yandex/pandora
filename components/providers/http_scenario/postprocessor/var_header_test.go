package postprocessor

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVarHeaderPostprocessor_Process(t *testing.T) {
	tests := []struct {
		name        string
		mappings    map[string]string
		respHeaders map[string]string
		expectedMap map[string]any
		expectErr   bool
	}{
		{
			name: "No Headers",
			mappings: map[string]string{
				"key1": "header1",
				"key2": "header2",
			},
			respHeaders: map[string]string{},
			expectedMap: map[string]any{},
		},
		{
			name:     "No Fields",
			mappings: map[string]string{},
			respHeaders: map[string]string{
				"key1": "header1",
				"key2": "header2"},
			expectedMap: nil,
		},
		{
			name: "Error in Fields",
			mappings: map[string]string{
				"key1": "header1||",
			},
			respHeaders: map[string]string{},
			expectedMap: map[string]any{},
			expectErr:   true,
		},
		{
			name: "Headers Exist",
			mappings: map[string]string{
				"key1": "header1",
				"key2": "header2|lower",
				"key3": "header3|upper",
				"key4": "header4|substr(1,3)",
			},
			respHeaders: map[string]string{
				"header1": "Value1",
				"header2": "Value2",
				"header3": "Value3",
				"header4": "Value4",
			},
			expectedMap: map[string]any{
				"key1": "Value1",
				"key2": "value2",
				"key3": "VALUE3",
				"key4": "al",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &VarHeaderPostprocessor{Mapping: tt.mappings}
			resp := &http.Response{
				Header: make(http.Header),
			}
			for k, v := range tt.respHeaders {
				resp.Header.Set(k, v)
			}

			reqMap, err := p.Process(resp, nil)
			if tt.expectErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedMap, reqMap)
		})
	}
}

func TestVarHeaderPostprocessor_ParseValue(t *testing.T) {

	tests := []struct {
		name                string
		input               string
		modifierVal         string
		expectedValue       string
		expectedModifierVal string
		expectedError       error
	}{
		{
			name:                "No Modifier",
			input:               "hello",
			modifierVal:         "asdf",
			expectedValue:       "hello",
			expectedModifierVal: "asdf",
			expectedError:       nil,
		},
		{
			name:                "Lowercase Modifier",
			input:               "foo|lower",
			modifierVal:         "ASDF",
			expectedValue:       "foo",
			expectedModifierVal: "asdf",
			expectedError:       nil,
		},
		{
			name:                "Uppercase Modifier",
			input:               "bar|upper",
			modifierVal:         "upper",
			expectedValue:       "bar",
			expectedModifierVal: "UPPER",
			expectedError:       nil,
		},
		{
			name:                "Substring Modifier",
			input:               "baz|substr(1,3)",
			modifierVal:         "asdfghjkl",
			expectedValue:       "baz",
			expectedModifierVal: "sd",
			expectedError:       nil,
		},
		{
			name:                "Multiple Modifiers",
			input:               "test|lower|upper",
			modifierVal:         "lower|upper",
			expectedValue:       "", // The method should return an empty string when multiple modifiers are provided.
			expectedModifierVal: "",
			expectedError:       fmt.Errorf("VarHeaderPostprocessor supports only one modifier yet"),
		},
		{
			name:                "Invalid Modifier",
			input:               "invalid|unknown",
			modifierVal:         "unknown",
			expectedValue:       "", // The method should return an empty string when the modifier is unknown.
			expectedModifierVal: "",
			expectedError:       fmt.Errorf("failed to parse modifier unknown: unknown modifier unknown"),
		},
		{
			name:                "Invalid Modifier Arguments",
			input:               "invalid|substr(abc)",
			modifierVal:         "substr(abc)",
			expectedValue:       "", // The method should return an empty string when the modifier arguments are invalid.
			expectedModifierVal: "",
			expectedError:       fmt.Errorf("failed to parse modifier substr(abc): substr modifier requires integer as first argument, got abc"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &VarHeaderPostprocessor{}
			value, modifier, err := p.parseValue(tt.input)
			if err != nil {
				assert.EqualError(t, err, tt.expectedError.Error())
				return
			}
			assert.NoError(t, err)

			assert.Equal(t, tt.expectedValue, value)

			gotModifierVal := modifier(tt.modifierVal)
			assert.Equal(t, tt.expectedModifierVal, gotModifierVal)

		})
	}
}

func TestVarHeaderPostprocessor_ParseModifier(t *testing.T) {
	p := &VarHeaderPostprocessor{}

	tests := []struct {
		name          string
		input         string
		value         string
		expectedRes   string
		expectedError error
	}{
		{
			name:          "Lowercase Modifier",
			input:         "lower",
			value:         "HELLO",
			expectedRes:   "hello",
			expectedError: nil,
		},
		{
			name:          "Uppercase Modifier",
			input:         "upper",
			value:         "world",
			expectedRes:   "WORLD",
			expectedError: nil,
		},
		{
			name:          "Substring Modifier - Normal Case",
			input:         "substr(1,4)",
			value:         "abcdefgh",
			expectedRes:   "bcd",
			expectedError: nil,
		},
		{
			name:          "Substring Modifier - Start Index Out of Range (Negative)",
			input:         "substr(-2,4)",
			value:         "abcdefgh",
			expectedRes:   "ef",
			expectedError: nil,
		},
		{
			name:          "Substring Modifier - Start Index Greater Than End Index",
			input:         "substr(5,3)",
			value:         "abcdefgh",
			expectedRes:   "de",
			expectedError: nil,
		},
		{
			name:          "Substring Modifier - End Index Beyond Length",
			input:         "substr(2,100)",
			value:         "abcdefgh",
			expectedRes:   "cdefgh", // End index is beyond the length of the input value, so the modifier should return the substring from index 2 to the end.
			expectedError: nil,
		},
		{
			name:          "Replace Modifier",
			input:         "replace(a,x)",
			value:         "banana",
			expectedRes:   "bxnxnx",
			expectedError: nil,
		},
		{
			name:          "Invalid Modifier",
			input:         "invalid",
			value:         "test",
			expectedRes:   "", // The modFunc will be nil, so expectedRes should be an empty string.
			expectedError: fmt.Errorf("unknown modifier invalid"),
		},
		{
			name:          "Substring Modifier with Invalid Arguments",
			input:         "substr(2)",
			value:         "abc",
			expectedRes:   "c",
			expectedError: nil,
		},
		{
			name:          "Replace Modifier with Invalid Arguments",
			input:         "replace(x)",
			value:         "abc",
			expectedRes:   "", // The modFunc will be nil, so expectedRes should be an empty string.
			expectedError: fmt.Errorf("replace modifier requires 2 arguments"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			modFunc, err := p.parseModifier(tt.input)

			// If there is an error, modFunc should be nil, and the result should be an empty string.
			if err != nil {
				assert.Nil(t, modFunc)
				assert.EqualError(t, err, tt.expectedError.Error())
				return
			}

			// If there is no error, apply the modFunc and check the result.
			res := modFunc(tt.value)
			assert.Equal(t, tt.expectedRes, res)
			assert.NoError(t, err)
		})
	}
}

func TestVarHeaderPostprocessor_Substr(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		value         string
		expectedRes   string
		expectedError error
	}{
		{
			name:          "Substring Modifier - Normal Case",
			args:          []string{"1", "4"},
			value:         "abcdefgh",
			expectedRes:   "bcd",
			expectedError: nil,
		},
		{
			name:          "Substring Modifier - Start Index Out of Range (Negative)",
			args:          []string{"-2", "4"},
			value:         "abcdefgh",
			expectedRes:   "ef", // Start index is negative, so it should count from the end of the string.
			expectedError: nil,
		},
		{
			name:          "Substring Modifier - End Index Out of Range (Negative)",
			args:          []string{"1", "-2"},
			value:         "abcdefgh",
			expectedRes:   "bcdef", // End index is negative, so it should count from the end of the string.
			expectedError: nil,
		},
		{
			name:          "Substring Modifier - Start Index Greater Than End Index",
			args:          []string{"5", "3"},
			value:         "abcdefgh",
			expectedRes:   "de", // Start index is greater than end index, so the modifier should return the substring from index 3 to 5.
			expectedError: nil,
		},
		{
			name:          "Substring Modifier - End Index Beyond Length",
			args:          []string{"2", "100"},
			value:         "abcdefgh",
			expectedRes:   "cdefgh", // End index is beyond the length of the input value, so the modifier should return the substring from index 2 to the end.
			expectedError: nil,
		},
		{
			name:          "Substring Modifier with Invalid Arguments",
			args:          []string{"2"},
			value:         "abc",
			expectedRes:   "c",
			expectedError: nil,
		},
		{
			name:          "Substring Modifier with Empty Arguments",
			args:          []string{},
			value:         "abc",
			expectedRes:   "", // The modFunc will be nil, so expectedRes should be an empty string.
			expectedError: fmt.Errorf("substr modifier requires one or two arguments"),
		},
		{
			name:          "Substring Modifier with Non-Integer Arguments",
			args:          []string{"abc", "xyz"},
			value:         "abc",
			expectedRes:   "", // The modFunc will be nil, so expectedRes should be an empty string.
			expectedError: fmt.Errorf("substr modifier requires integer as first argument, got abc"),
		},
		{
			name:          "Substring Modifier with Non-Integer second Argument",
			args:          []string{"1", "xyz"},
			value:         "abc",
			expectedRes:   "", // The modFunc will be nil, so expectedRes should be an empty string.
			expectedError: fmt.Errorf("substr modifier requires integer as second argument, got xyz"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &VarHeaderPostprocessor{}
			modFunc, err := p.substr(tt.args)

			// If there is an error, modFunc should be nil, and the result should be an empty string.
			if err != nil {
				assert.Nil(t, modFunc)
				assert.EqualError(t, err, tt.expectedError.Error())
				return
			}

			// If there is no error, apply the modFunc and check the result.
			res := modFunc(tt.value)
			assert.Equal(t, tt.expectedRes, res)
			assert.NoError(t, err)
		})
	}
}
