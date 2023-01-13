package confutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEnvVarResolver(t *testing.T) {
	type testCase struct {
		varname string
		val     string
		err     error
	}

	tests := []testCase{
		{"SOME_BOOL", "True", nil},
		{"INT_VALUE", "10", nil},
		{"V_NAME", "10", nil},
	}

	for _, test := range tests {
		t.Setenv(test.varname, test.val)
	}

	tests = append(tests, testCase{"NOT_EXISTS", "", ErrEnvVariableNotProvided})

	for _, test := range tests {
		actual, err := envTokenResolver(test.varname)
		if test.err != nil {
			assert.ErrorIs(t, err, test.err)
		} else {
			assert.NoError(t, err)
			assert.Exactly(t, test.val, actual)
		}
	}
}
