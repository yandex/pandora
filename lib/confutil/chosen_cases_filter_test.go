package confutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestChosenCases(t *testing.T) {
	type testCase struct {
		ammoTag  string
		expected bool
	}

	cases := []string{"tag1", "tag3"}

	tests := []testCase{
		{"tag1", true},
		{"tag2", false},
	}

	for _, tc := range tests {
		assert.Equal(t, tc.expected, IsChosenCase(tc.ammoTag, cases))
	}
}
