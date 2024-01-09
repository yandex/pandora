package vs

import (
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVariableSourceJson_Init(t *testing.T) {
	initFs := func(t *testing.T) afero.Fs {
		fs := afero.NewMemMapFs()
		file, err := fs.Create("users.json")
		require.NoError(t, err)
		_, err = file.WriteString(`{"error": "timeout", "timeout": "3s", "isResult": true, "number": 1}`)
		require.NoError(t, err)
		return fs
	}
	deferFs := func(t *testing.T, fs afero.Fs) {
		err := fs.Remove("users.json")
		require.NoError(t, err)
	}

	tests := []struct {
		name      string
		initFs    func(t *testing.T) afero.Fs
		deferFs   func(t *testing.T, fs afero.Fs)
		vs        *VariableSourceJSON
		wantErr   bool
		wantStore any
	}{
		{
			name:    "default",
			initFs:  initFs,
			deferFs: deferFs,
			vs: &VariableSourceJSON{
				Name: "users",
				File: "users.json",
			},
			wantErr:   false,
			wantStore: map[string]any{"error": "timeout", "timeout": "3s", "isResult": true, "number": float64(1)},
		},
		{
			name: "slice",
			initFs: func(t *testing.T) afero.Fs {
				fs := afero.NewMemMapFs()
				file, err := fs.Create("users.json")
				require.NoError(t, err)
				_, err = file.WriteString(`[{"error": "timeout", "timeout": "3s", "isResult": true, "number": 1}]`)
				require.NoError(t, err)
				return fs
			},
			deferFs: func(t *testing.T, fs afero.Fs) {
				err := fs.Remove("users.json")
				require.NoError(t, err)
			},
			vs: &VariableSourceJSON{
				Name: "users",
				File: "users.json",
			},
			wantErr:   false,
			wantStore: []any{map[string]any{"error": "timeout", "timeout": "3s", "isResult": true, "number": float64(1)}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.vs.fs = tt.initFs(t)
			defer tt.deferFs(t, tt.vs.fs)

			err := tt.vs.Init()
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.wantStore, tt.vs.store)

		})
	}
}
