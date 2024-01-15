package vs

import (
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVariableSourceCsv_Init(t *testing.T) {
	initFs := func(t *testing.T) afero.Fs {
		fs := afero.NewMemMapFs()
		file, err := fs.Create("users.csv")
		require.NoError(t, err)
		_, err = file.WriteString("USER_ID,NAME\n1,John\n2,Jack\n3,Jim\n")
		require.NoError(t, err)
		return fs
	}
	deferFs := func(t *testing.T, fs afero.Fs) {
		err := fs.Remove("users.csv")
		require.NoError(t, err)
	}

	tests := []struct {
		name      string
		initFs    func(t *testing.T) afero.Fs
		deferFs   func(t *testing.T, fs afero.Fs)
		vs        *VariableSourceCsv
		wantErr   bool
		wantStore []map[string]string
	}{
		{
			name:    "default",
			initFs:  initFs,
			deferFs: deferFs,
			vs: &VariableSourceCsv{
				Name:            "users",
				File:            "users.csv",
				Fields:          []string{"user_id", "name"},
				IgnoreFirstLine: false,
				Delimiter:       ",",
			},
			wantErr:   false,
			wantStore: []map[string]string{{"name": "NAME", "user_id": "USER_ID"}, {"name": "John", "user_id": "1"}, {"name": "Jack", "user_id": "2"}, {"name": "Jim", "user_id": "3"}},
		},
		{
			name:    "replace spaces in field names",
			initFs:  initFs,
			deferFs: deferFs,
			vs: &VariableSourceCsv{
				Name:            "users",
				File:            "users.csv",
				Fields:          []string{"user id", "name name"},
				IgnoreFirstLine: false,
				Delimiter:       ",",
			},
			wantErr:   false,
			wantStore: []map[string]string{{"name_name": "NAME", "user_id": "USER_ID"}, {"name_name": "John", "user_id": "1"}, {"name_name": "Jack", "user_id": "2"}, {"name_name": "Jim", "user_id": "3"}},
		},
		{
			name:    "skip header",
			initFs:  initFs,
			deferFs: deferFs,
			vs: &VariableSourceCsv{
				Name:            "users",
				File:            "users.csv",
				Fields:          []string{"user_id", "name"},
				IgnoreFirstLine: true,
				Delimiter:       ",",
			},
			wantErr:   false,
			wantStore: []map[string]string{{"name": "John", "user_id": "1"}, {"name": "Jack", "user_id": "2"}, {"name": "Jim", "user_id": "3"}},
		},
		{
			name:    "empty fields and not skip header and not header as fields",
			initFs:  initFs,
			deferFs: deferFs,
			vs: &VariableSourceCsv{
				Name:            "users",
				File:            "users.csv",
				Fields:          nil,
				IgnoreFirstLine: false,
				Delimiter:       ",",
			},
			wantErr:   false,
			wantStore: []map[string]string{{"NAME": "NAME", "USER_ID": "USER_ID"}, {"NAME": "John", "USER_ID": "1"}, {"NAME": "Jack", "USER_ID": "2"}, {"NAME": "Jim", "USER_ID": "3"}},
		},
		{
			name: "replace spaces in field names in first line",
			initFs: func(t *testing.T) afero.Fs {
				fs := afero.NewMemMapFs()
				file, err := fs.Create("users.csv")
				require.NoError(t, err)
				_, err = file.WriteString("USER ID,NAME NAME\n1,John\n2,Jack\n3,Jim\n")
				require.NoError(t, err)
				return fs
			},
			deferFs: deferFs,
			vs: &VariableSourceCsv{
				Name:            "users",
				File:            "users.csv",
				Fields:          nil,
				IgnoreFirstLine: false,
				Delimiter:       ",",
			},
			wantErr:   false,
			wantStore: []map[string]string{{"NAME_NAME": "NAME NAME", "USER_ID": "USER ID"}, {"NAME_NAME": "John", "USER_ID": "1"}, {"NAME_NAME": "Jack", "USER_ID": "2"}, {"NAME_NAME": "Jim", "USER_ID": "3"}},
		},
		{
			name:    "empty fields and skip header",
			initFs:  initFs,
			deferFs: deferFs,
			vs: &VariableSourceCsv{
				Name:            "users",
				File:            "users.csv",
				Fields:          nil,
				IgnoreFirstLine: true,
				Delimiter:       ",",
			},
			wantErr:   false,
			wantStore: []map[string]string{{"NAME": "John", "USER_ID": "1"}, {"NAME": "Jack", "USER_ID": "2"}, {"NAME": "Jim", "USER_ID": "3"}},
		},
		{
			name: "skipped header field",
			initFs: func(t *testing.T) afero.Fs {
				fs := afero.NewMemMapFs()
				file, err := fs.Create("users.csv")
				require.NoError(t, err)
				_, err = file.WriteString(",NAME\n1,John\n2,Jack\n3,Jim\n")
				require.NoError(t, err)
				return fs
			},
			deferFs: deferFs,
			vs: &VariableSourceCsv{
				Name:            "users",
				File:            "users.csv",
				Fields:          nil,
				IgnoreFirstLine: true,
				Delimiter:       ",",
			},
			wantErr:   false,
			wantStore: []map[string]string{{"NAME": "John", "0": "1"}, {"NAME": "Jack", "0": "2"}, {"NAME": "Jim", "0": "3"}},
		},
		{
			name:    "skipped header field",
			initFs:  initFs,
			deferFs: deferFs,
			vs: &VariableSourceCsv{
				Name:            "users",
				File:            "users.csv",
				Fields:          []string{"", "name"},
				IgnoreFirstLine: true,
				Delimiter:       ",",
			},
			wantErr:   false,
			wantStore: []map[string]string{{"name": "John", "0": "1"}, {"name": "Jack", "0": "2"}, {"name": "Jim", "0": "3"}},
		},
		{
			name: "delimiter ;",
			initFs: func(t *testing.T) afero.Fs {
				fs := afero.NewMemMapFs()
				file, err := fs.Create("users.csv")
				require.NoError(t, err)
				_, err = file.WriteString("USER_ID;NAME\n1;John\n2;Jack\n3;Jim\n")
				require.NoError(t, err)
				return fs
			},
			deferFs: deferFs,
			vs: &VariableSourceCsv{
				Name:            "users",
				File:            "users.csv",
				Fields:          []string{"", "name"},
				IgnoreFirstLine: true,
				Delimiter:       ";",
			},
			wantErr:   false,
			wantStore: []map[string]string{{"name": "John", "0": "1"}, {"name": "Jack", "0": "2"}, {"name": "Jim", "0": "3"}},
		},
		{
			name: "error when values more than fields",
			initFs: func(t *testing.T) afero.Fs {
				fs := afero.NewMemMapFs()
				file, err := fs.Create("users2.csv")
				require.NoError(t, err)
				_, err = file.WriteString("USER_ID,NAME\n1,John\n2,Jack,skipthisvalue\n3\n")
				require.NoError(t, err)
				return fs
			},
			deferFs: func(t *testing.T, fs afero.Fs) {
				err := fs.Remove("users2.csv")
				require.NoError(t, err)
			},
			vs: &VariableSourceCsv{
				Name:            "users",
				File:            "users2.csv",
				Fields:          nil,
				IgnoreFirstLine: true,
				Delimiter:       ",",
			},
			wantErr:   true,
			wantStore: nil,
		},
		{
			name: "error when values less than fields",
			initFs: func(t *testing.T) afero.Fs {
				fs := afero.NewMemMapFs()
				file, err := fs.Create("users2.csv")
				require.NoError(t, err)
				_, err = file.WriteString("USER_ID,NAME\n1,John\n2,Jack\n3\n")
				require.NoError(t, err)
				return fs
			},
			deferFs: func(t *testing.T, fs afero.Fs) {
				err := fs.Remove("users2.csv")
				require.NoError(t, err)
			},
			vs: &VariableSourceCsv{
				Name:            "users",
				File:            "users2.csv",
				Fields:          nil,
				IgnoreFirstLine: true,
				Delimiter:       ",",
			},
			wantErr:   true,
			wantStore: nil,
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
