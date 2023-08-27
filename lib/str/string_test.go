package str

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseStringFunc(t *testing.T) {
	testCases := []struct {
		name          string
		input         string
		expectedName  string
		expectedArgs  []string
		expectedError error
	}{
		{
			name:          "TestValidInputNoArgs",
			input:         "functionName",
			expectedName:  "functionName",
			expectedArgs:  nil,
			expectedError: nil,
		},
		{
			name:          "TestValidInputWithArgs",
			input:         "functionName(arg1, arg2, arg3)",
			expectedName:  "functionName",
			expectedArgs:  []string{"arg1", "arg2", "arg3"},
			expectedError: nil,
		},
		{
			name:          "TestInvalidCloseBracket",
			input:         "functionName(arg1, arg2, arg3",
			expectedName:  "",
			expectedArgs:  nil,
			expectedError: errors.New("invalid close bracket position"),
		},
		{
			name:          "TestValidInputOneArg",
			input:         "functionName(arg1)",
			expectedName:  "functionName",
			expectedArgs:  []string{"arg1"},
			expectedError: nil,
		},
		{
			name:          "TestEmptyInput",
			input:         "",
			expectedName:  "",
			expectedArgs:  nil,
			expectedError: nil,
		},
		{
			name:          "TestOnlyOpenBracket",
			input:         "(",
			expectedName:  "",
			expectedArgs:  nil,
			expectedError: errors.New("invalid close bracket position"),
		},
		{
			name:          "TestOnlyCloseBracket",
			input:         ")",
			expectedName:  "",
			expectedArgs:  nil,
			expectedError: errors.New("invalid close bracket position"),
		},
		{
			name:          "TestSingleEmptyArgument",
			input:         "functionName()",
			expectedName:  "functionName",
			expectedArgs:  []string{""},
			expectedError: nil,
		},
		{
			name:          "TestBracketInFunctionName",
			input:         "functionName)arg1, arg2, arg3)",
			expectedName:  "",
			expectedArgs:  nil,
			expectedError: errors.New("invalid close bracket position"),
		},
		{
			name:          "TestExtraCloseBracket",
			input:         "functionName(arg1, arg2, arg3))",
			expectedName:  "",
			expectedArgs:  nil,
			expectedError: errors.New("invalid close bracket position"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			name, args, err := ParseStringFunc(tc.input)

			// Assert the values
			assert.Equal(t, tc.expectedName, name)
			assert.Equal(t, tc.expectedArgs, args)
			assert.Equal(t, tc.expectedError, err)
		})
	}
}
