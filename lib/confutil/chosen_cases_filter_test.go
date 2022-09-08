package confutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type testCase struct {
	ammoTag  string
	expected bool
}

func TestChosenCases(t *testing.T) {
	cases := []string{"tag1", "tag3"}

	tests := []testCase{
		{"tag1", true},
		{"tag2", false},
	}

	for _, tc := range tests {
		assert.Equal(t, tc.expected, IsChosenCase(tc.ammoTag, cases))
	}
}
