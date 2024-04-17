package vs

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVariableSourceVariables_Init(t *testing.T) {
	tests := []struct {
		name      string
		variables map[string]any
		want      map[string]any
		wantErr   assert.ErrorAssertionFunc
	}{
		{
			name: "default",
			variables: map[string]any{
				"random": "randInt(0,1)",
				"object": map[string]any{
					"strings": []string{
						"randString(2, a)",
						"randString(3, b)",
					},
				},
			},
			want: map[string]any{
				"random": "0",
				"object": map[string]any{
					"strings": []string{
						"aa",
						"bbb",
					},
				},
			},
			wantErr: assert.NoError,
		},
		{
			name: "invalid func name return string",
			variables: map[string]any{
				"random": "randInteger(0,1)",
			},
			want: map[string]any{
				"random": "randInteger(0,1)",
			},
			wantErr: assert.NoError,
		},
		{
			name: "invalid func arg",
			variables: map[string]any{
				"random": "randInt(asdf)",
			},
			want: map[string]any{
				"random": "",
			},
			wantErr: assert.Error,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &VariableSourceVariables{Variables: tt.variables}
			err := v.Init()
			if tt.wantErr(t, err) {
				assert.Equal(t, tt.want, v.Variables)
			}
		})
	}
}
