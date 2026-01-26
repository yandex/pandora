package confutil

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPropertyTokenResolver(t *testing.T) {
	fileContent := []byte(`name=John Doe
age=25
email=johndoe@example.com`)
	tmpFile, err := os.CreateTemp("", "testfile*.txt")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()
	_, err = tmpFile.Write(fileContent)
	assert.NoError(t, err)

	testCases := []struct {
		input          string
		expectedResult string
		expectedError  string
	}{
		{
			input:          tmpFile.Name() + "#name",
			expectedResult: "John Doe",
			expectedError:  "",
		},
		{
			input:          tmpFile.Name() + "#age",
			expectedResult: "25",
			expectedError:  "",
		},
		{
			input:          tmpFile.Name() + "#email",
			expectedResult: "johndoe@example.com",
			expectedError:  "",
		},
		{
			input:          tmpFile.Name() + "#address",
			expectedResult: "",
			expectedError:  "no such property 'address', in file '" + tmpFile.Name() + "'",
		},
		{
			input:          "nonexistent.txt#property",
			expectedResult: "",
			expectedError:  "cannot open file: 'nonexistent.txt'",
		},
	}

	for _, testCase := range testCases {
		result, err := propertyTokenResolver(testCase.input)

		assert.Equal(t, testCase.expectedResult, result)
		if testCase.expectedError != "" {
			assert.EqualError(t, err, testCase.expectedError)
		} else {
			assert.NoError(t, err)
		}
	}
}
