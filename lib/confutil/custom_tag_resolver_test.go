package confutil

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStringToExpectedCast(t *testing.T) {
	type testCase struct {
		val      string
		expected interface{}
		err      error
	}

	tests := []testCase{
		{"True", true, nil},
		{"T", true, nil},
		{"t", true, nil},
		{"TRUE", true, nil},
		{"true", true, nil},
		{"1", true, nil},
		{"False", false, nil},
		{"false", false, nil},
		{"0", false, nil},
		{"f", false, nil},
		{"", false, ErrCantCastVariableToTargetType},

		{"11", uint(11), nil},
		{"10", uint8(10), nil},
		{"10", uint16(10), nil},
		{"10", uint32(10), nil},
		{"10", uint64(10), nil},
		{"11", int(11), nil},
		{"10", int8(10), nil},
		{"10", int16(10), nil},
		{"10", int32(10), nil},
		{"10", int64(10), nil},
		{"", int(0), ErrCantCastVariableToTargetType},
		{"asdf", int(0), ErrCantCastVariableToTargetType},
		{" -14", int(0), ErrCantCastVariableToTargetType},

		{"-10", float32(-10), nil},
		{"10.23", float32(10.23), nil},
		{"-10", float64(-10), nil},
		{"10.23", float64(10.23), nil},
		{"", float64(0), ErrCantCastVariableToTargetType},
		{"asdf", float64(0), ErrCantCastVariableToTargetType},
		{" -14", float64(0), ErrCantCastVariableToTargetType},

		{"10", "10", nil},
		{"value-port", "value-port", nil},
		{"", "", nil},
	}

	for _, test := range tests {
		expectedType := reflect.TypeOf(test.expected)
		t.Run(fmt.Sprintf("Test string to %s cast", expectedType), func(t *testing.T) {
			actual, err := cast(test.val, expectedType)
			if test.err == nil {
				assert.NoError(t, err)
				assert.Exactly(t, test.expected, actual)
			} else {
				assert.ErrorIs(t, err, test.err)
			}

		})
	}
}

func TestFindTokens(t *testing.T) {
	type testCase struct {
		val      string
		expected []*tagEntry
		err      error
	}

	tests := []testCase{
		{
			"${token}",
			[]*tagEntry{{string: "${token}", tagType: "", varname: "token"}},
			nil,
		},
		{
			"${token}-${ second\t}",
			[]*tagEntry{
				{string: "${token}", tagType: "", varname: "token"},
				{string: "${ second\t}", tagType: "", varname: "second"},
			},
			nil,
		},
		{
			"asdf${ee:token}aa",
			[]*tagEntry{
				{string: "${ee:token}", tagType: "ee", varname: "token"},
			},
			nil,
		},
		{
			"asdf${ee: to:ken}aa-${ e2 :to  }ken}",
			[]*tagEntry{
				{string: "${ee: to:ken}", tagType: "ee", varname: "to:ken"},
				{string: "${ e2 :to  }", tagType: "e2", varname: "to"},
			},
			nil,
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("Test findTokens in %s", test.val), func(t *testing.T) {
			actual, err := findTags(test.val)
			if test.err == nil {
				assert.NoError(t, err)
				assert.EqualValues(t, test.expected, actual)
			} else {
				assert.ErrorIs(t, err, test.err)
			}
		})
	}
}
