package postprocessor

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/html"
)

func TestVarXpathPostprocessor_Process(t *testing.T) {
	// Define test cases with different bodies and mappings.
	testCases := []struct {
		name           string
		body           []byte
		mappings       map[string]string
		expectedReqMap map[string]interface{}
	}{
		{
			name: "Test Case 1",
			body: []byte(`
				<html>
					<body>
						<div class="data">Value1</div>
						<div class="data">Value2</div>
					</body>
				</html>
			`),
			mappings: map[string]string{
				"key1": "//div[@class='data']",
			},
			expectedReqMap: map[string]interface{}{
				"key1": []string{"Value1", "Value2"},
			},
		},
		{
			name: "Test Case 2",
			body: []byte(`
				<html>
					<body>
						<span class="span-data">ValueX</span>
						<span class="span-data">ValueY</span>
					</body>
				</html>
			`),
			mappings: map[string]string{
				"keyAlpha": "//span[@class='span-data']",
			},
			expectedReqMap: map[string]interface{}{
				"keyAlpha": []string{"ValueX", "ValueY"},
			},
		},
		{
			name: "Test Case 3",
			body: []byte(`
				<html>
					<body>
						<p class="paragraph">This is a paragraph</p>
					</body>
				</html>
			`),
			mappings: map[string]string{
				"keyParagraph": "//p[@class='paragraph']",
			},
			expectedReqMap: map[string]interface{}{
				"keyParagraph": "This is a paragraph",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			postprocessor := &VarXpathPostprocessor{
				Mapping: tc.mappings,
			}

			buf := bytes.NewBuffer(tc.body)
			reqMap := make(map[string]interface{})
			err := postprocessor.Process(reqMap, nil, buf)

			assert.NoError(t, err)
			assert.Equal(t, tc.expectedReqMap, reqMap)
		})
	}
}

func Test_getValuesFromDOM(t *testing.T) {
	data := []byte(`
		<html>
			<head>
				<title>Example</title>
			</head>
			<body>
				<ul>
					<li>Order 1</li>
					<li>Order 2</li>
					<li>Order 3</li>
				</ul>
			</body>
		</html>
	`)

	doc, err := html.Parse(bytes.NewReader(data))
	require.NoError(t, err)

	xpathQuery := "//li"
	p := &VarXpathPostprocessor{}
	results, err := p.getValuesFromDOM(doc, xpathQuery)
	require.NoError(t, err)

	require.Equal(t, []string{"Order 1", "Order 2", "Order 3"}, results)
}
