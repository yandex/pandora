package preprocessor

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPreprocessor_Process(t *testing.T) {
	tests := []struct {
		name      string
		prep      Preprocessor
		templVars map[string]any
		wantMap   map[string]any
		wantErr   bool
	}{
		{
			name: "nil templateVars",
			prep: Preprocessor{
				Mapping: map[string]string{
					"var1": "source.items[0].id",
					"var2": "source.items[1]",
				},
			},
			wantErr: true,
		},
		{
			name: "simple processing",
			prep: Preprocessor{
				Mapping: map[string]string{
					"var1": "source.items[0].id",
					"var2": "source.items[1]",
					"var3": "request.auth.token",
				},
			},
			templVars: map[string]any{
				"request": map[string]any{
					"auth": map[string]any{"token": "Bearer token"},
				},
				"source": map[string]any{
					"items": []map[string]any{
						{"id": "1"},
						{"id": "2"},
					},
				},
			},
			wantMap: map[string]any{
				"var1": "1",
				"var2": map[string]any{"id": "2"},
				"var3": "Bearer token",
			},
			wantErr: false,
		},
		{
			name: "template funcs",
			prep: Preprocessor{
				Mapping: map[string]string{
					"var4": "randInt(.request.auth.min, 201)",
					"var5": "randString(source.items[last].id, .request.get.letters)",
				},
			},
			templVars: map[string]any{
				"request": map[string]any{
					"auth": map[string]any{"min": 200},
					"get":  map[string]any{"letters": "a"},
				},
				"source": map[string]any{
					"items": []map[string]any{
						{"id": "1"},
						{"id": "2"},
						{"id": "10"},
					},
				},
			},
			wantMap: map[string]any{
				"var4": "200",
				"var5": "aaaaaaaaaa",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.prep.Process(tt.templVars)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantMap, result)
			}
		})
	}
}
