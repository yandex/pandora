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
		wantMap     map[string]any
		wantErr     bool
	}{
		{
			name: "No Headers",
			mappings: map[string]string{
				"key1": "header1",
				"key2": "header2",
			},
			respHeaders: map[string]string{},
			wantMap:     map[string]any{},
		},
		{
			name:     "No Fields",
			mappings: map[string]string{},
			respHeaders: map[string]string{
				"key1": "header1",
				"key2": "header2"},
			wantMap: nil,
		},
		{
			name: "Error in Fields",
			mappings: map[string]string{
				"key1": "header1||",
			},
			respHeaders: map[string]string{},
			wantMap:     map[string]any{},
			wantErr:     true,
		},
		{
			name: "Headers Exist",
			mappings: map[string]string{
				"key1": "header1",
				"key2": "header2|lower",
				"key3": "header3|upper",
				"key4": "header4|substr(1,3)",
				"key5": "header5|lower|replace(s,x)|substr(1,3)",
				"auth": "Authorization|lower|replace(=,)|substr(6)",
			},
			respHeaders: map[string]string{
				"header1":       "Value1",
				"header2":       "Value2",
				"header3":       "Value3",
				"header4":       "Value4",
				"header5":       "aSdFgHjKl",
				"Authorization": "Basic Ym9zY236Ym9zY28=",
			},
			wantMap: map[string]any{
				"key1": "Value1",
				"key2": "value2",
				"key3": "VALUE3",
				"key4": "al",
				"key5": "xd",
				"auth": "ym9zy236ym9zy28",
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
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.wantMap, reqMap)
		})
	}
}

func TestVarHeaderPostprocessor_ParseValue(t *testing.T) {
	tests := []struct {
		name                 string
		input                string
		wantVal              string
		wantValAfterModifier string
		wantErr              error
	}{
		{
			name:                 "No Modifier",
			input:                "hello",
			wantVal:              "hello",
			wantValAfterModifier: "hello",
			wantErr:              nil,
		},
		{
			name:                 "Lowercase Modifier",
			input:                "fOOt|lower",
			wantVal:              "fOOt",
			wantValAfterModifier: "foot",
			wantErr:              nil,
		},
		{
			name:                 "Uppercase Modifier",
			input:                "bar|upper",
			wantVal:              "bar",
			wantValAfterModifier: "BAR",
			wantErr:              nil,
		},
		{
			name:                 "Substring Modifier",
			input:                "asdfghjkl|substr(1,3)",
			wantVal:              "asdfghjkl",
			wantValAfterModifier: "sd",
			wantErr:              nil,
		},
		{
			name:                 "Multiple Modifiers",
			input:                "aSdFgHjKl|lower|replace(s,x)|substr(1,3)",
			wantVal:              "aSdFgHjKl",
			wantValAfterModifier: "xd",
			wantErr:              nil,
		},
		{
			name:                 "Multiple Modifiers 2",
			input:                "aPPliCation-JSONbro|lower|replace(-, /)|substr(0, 16)",
			wantVal:              "aPPliCation-JSONbro",
			wantValAfterModifier: "application/json",
			wantErr:              nil,
		},
		{
			name:                 "Invalid Modifier",
			input:                "invalid|unknown",
			wantVal:              "", // The method should return an empty string when the modifier is unknown.
			wantValAfterModifier: "",
			wantErr:              fmt.Errorf("failed to parse modifier unknown: unknown modifier unknown"),
		},
		{
			name:                 "Invalid Modifier Arguments",
			input:                "invalid|substr(abc)",
			wantVal:              "", // The method should return an empty string when the modifier arguments are invalid.
			wantValAfterModifier: "",
			wantErr:              fmt.Errorf("failed to parse modifier substr(abc): substr modifier requires integer as first argument, got abc"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &VarHeaderPostprocessor{}
			value, modifier, err := p.parseValue(tt.input)
			if err != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
				return
			}
			assert.NoError(t, err)

			assert.Equal(t, tt.wantVal, value)

			gotModifierVal := modifier(value)
			assert.Equal(t, tt.wantValAfterModifier, gotModifierVal)

		})
	}
}

func TestVarHeaderPostprocessor_ParseModifier(t *testing.T) {
	p := &VarHeaderPostprocessor{}

	tests := []struct {
		name    string
		input   string
		value   string
		want    string
		wantErr error
	}{
		{
			name:    "Lowercase Modifier",
			input:   "lower",
			value:   "HELLO",
			want:    "hello",
			wantErr: nil,
		},
		{
			name:    "Uppercase Modifier",
			input:   "upper",
			value:   "world",
			want:    "WORLD",
			wantErr: nil,
		},
		{
			name:    "Substring Modifier - Normal Case",
			input:   "substr(1,4)",
			value:   "abcdefgh",
			want:    "bcd",
			wantErr: nil,
		},
		{
			name:    "Substring Modifier - Start Index Out of Range (Negative)",
			input:   "substr(-2,4)",
			value:   "abcdefgh",
			want:    "ef",
			wantErr: nil,
		},
		{
			name:    "Substring Modifier - Start Index Greater Than End Index",
			input:   "substr(5,3)",
			value:   "abcdefgh",
			want:    "de",
			wantErr: nil,
		},
		{
			name:    "Substring Modifier - End Index Beyond Length",
			input:   "substr(2,100)",
			value:   "abcdefgh",
			want:    "cdefgh", // End index is beyond the length of the input value, so the modifier should return the substring from index 2 to the end.
			wantErr: nil,
		},
		{
			name:    "Replace Modifier",
			input:   "replace(a,x)",
			value:   "banana",
			want:    "bxnxnx",
			wantErr: nil,
		},
		{
			name:    "Invalid Modifier",
			input:   "invalid",
			value:   "test",
			want:    "", // The modFunc will be nil, so want should be an empty string.
			wantErr: fmt.Errorf("unknown modifier invalid"),
		},
		{
			name:    "Substring Modifier with Invalid Arguments",
			input:   "substr(2)",
			value:   "abc",
			want:    "c",
			wantErr: nil,
		},
		{
			name:    "Replace Modifier with Invalid Arguments",
			input:   "replace(x)",
			value:   "abc",
			want:    "", // The modFunc will be nil, so want should be an empty string.
			wantErr: fmt.Errorf("replace modifier requires 2 arguments"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			modFunc, err := p.parseModifier(tt.input)

			// If there is an error, modFunc should be nil, and the result should be an empty string.
			if err != nil {
				assert.Nil(t, modFunc)
				assert.EqualError(t, err, tt.wantErr.Error())
				return
			}

			// If there is no error, apply the modFunc and check the result.
			res := modFunc(tt.value)
			assert.Equal(t, tt.want, res)
			assert.NoError(t, err)
		})
	}
}

func TestVarHeaderPostprocessor_Substr(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		value   string
		want    string
		wantErr error
	}{
		{
			name:    "Substring Modifier - Normal Case",
			args:    []string{"1", "4"},
			value:   "abcdefgh",
			want:    "bcd",
			wantErr: nil,
		},
		{
			name:    "Substring Modifier - Start Index Out of Range (Negative)",
			args:    []string{"-2", "4"},
			value:   "abcdefgh",
			want:    "ef", // Start index is negative, so it should count from the end of the string.
			wantErr: nil,
		},
		{
			name:    "Substring Modifier - End Index Out of Range (Negative)",
			args:    []string{"1", "-2"},
			value:   "abcdefgh",
			want:    "bcdef", // End index is negative, so it should count from the end of the string.
			wantErr: nil,
		},
		{
			name:    "Substring Modifier - Start Index Greater Than End Index",
			args:    []string{"5", "3"},
			value:   "abcdefgh",
			want:    "de", // Start index is greater than end index, so the modifier should return the substring from index 3 to 5.
			wantErr: nil,
		},
		{
			name:    "Substring Modifier - End Index Beyond Length",
			args:    []string{"2", "100"},
			value:   "abcdefgh",
			want:    "cdefgh", // End index is beyond the length of the input value, so the modifier should return the substring from index 2 to the end.
			wantErr: nil,
		},
		{
			name:    "Substring Modifier with Invalid Arguments",
			args:    []string{"2"},
			value:   "abc",
			want:    "c",
			wantErr: nil,
		},
		{
			name:    "Substring Modifier with Empty Arguments",
			args:    []string{},
			value:   "abc",
			want:    "", // The modFunc will be nil, so want should be an empty string.
			wantErr: fmt.Errorf("substr modifier requires one or two arguments"),
		},
		{
			name:    "Substring Modifier with Non-Integer Arguments",
			args:    []string{"abc", "xyz"},
			value:   "abc",
			want:    "", // The modFunc will be nil, so want should be an empty string.
			wantErr: fmt.Errorf("substr modifier requires integer as first argument, got abc"),
		},
		{
			name:    "Substring Modifier with Non-Integer second Argument",
			args:    []string{"1", "xyz"},
			value:   "abc",
			want:    "", // The modFunc will be nil, so want should be an empty string.
			wantErr: fmt.Errorf("substr modifier requires integer as second argument, got xyz"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &VarHeaderPostprocessor{}
			modFunc, err := p.substr(tt.args)
			if err != nil {
				assert.Nil(t, modFunc)
				assert.EqualError(t, err, tt.wantErr.Error())
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want, modFunc(tt.value))
		})
	}
}
