package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yandex/pandora/components/providers/scenario/vs"
)

type mockVS struct {
	name     string
	vars     map[string]string
	initErr  error
	initCall int
}

func (m *mockVS) GetName() string {
	return m.name
}

func (m *mockVS) GetVariables() any {
	return m.vars
}

func (m *mockVS) Init() error {
	m.initCall--
	return m.initErr
}

func TestExtractVariableStorage(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *AmmoConfig
		want    map[string]any
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "default",
			cfg: &AmmoConfig{
				VariableSources: []vs.VariableSource{
					&mockVS{initCall: 1, name: "users", vars: map[string]string{"user_id": "1"}},
					&mockVS{initCall: 1, name: "filter_src", vars: map[string]string{"filter": "filter"}},
				},
			},
			want: map[string]any{
				"users":      map[string]string{"user_id": "1"},
				"filter_src": map[string]string{"filter": "filter"},
			},
			wantErr: assert.NoError,
		},
		{
			name: "init error",
			cfg: &AmmoConfig{
				VariableSources: []vs.VariableSource{
					&mockVS{initCall: 1, name: "users", vars: map[string]string{"user_id": "1"}},
					&mockVS{initCall: 1, name: "filter_src", vars: map[string]string{"filter": "filter"}, initErr: assert.AnError},
				},
			},
			wantErr: assert.Error,
			want:    map[string]any{"users": map[string]string{"user_id": "1"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExtractVariableStorage(tt.cfg)
			if !tt.wantErr(t, err) {
				return
			}

			vars := got.Variables()
			assert.Equal(t, tt.want, vars)
			for _, source := range tt.cfg.VariableSources {
				assert.Equal(t, 0, source.(*mockVS).initCall)
			}
		})
	}
}

func Test_SpreadNames(t *testing.T) {
	tests := []struct {
		name      string
		input     []ScenarioConfig
		want      map[string]int
		wantTotal int
	}{
		{
			name:      "",
			input:     []ScenarioConfig{{Name: "a", Weight: 20}, {Name: "b", Weight: 30}, {Name: "c", Weight: 60}},
			want:      map[string]int{"a": 2, "b": 3, "c": 6},
			wantTotal: 11,
		},
		{
			name:      "",
			input:     []ScenarioConfig{{Name: "a", Weight: 100}, {Name: "b", Weight: 100}, {Name: "c", Weight: 100}},
			want:      map[string]int{"a": 1, "b": 1, "c": 1},
			wantTotal: 3,
		},
		{
			name:      "",
			input:     []ScenarioConfig{{Name: "a", Weight: 100}},
			want:      map[string]int{"a": 1},
			wantTotal: 1,
		},
		{
			name:      "",
			input:     []ScenarioConfig{{Name: "a", Weight: 0}},
			want:      map[string]int{"a": 1},
			wantTotal: 1,
		},
		{
			name:      "",
			input:     []ScenarioConfig{{Name: "a", Weight: 0}, {Name: "b", Weight: 1}},
			want:      map[string]int{"a": 1, "b": 1},
			wantTotal: 2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, total := SpreadNames(tt.input)
			assert.Equalf(t, tt.want, got, "spreadNames(%v)", tt.input)
			assert.Equalf(t, tt.wantTotal, total, "spreadNames(%v)", tt.input)
		})
	}
}

func Test_ParseShootName(t *testing.T) {
	testCases := []struct {
		input     string
		wantName  string
		wantCnt   int
		wantSleep int
		wantErr   bool
	}{
		{"shoot", "shoot", 1, 0, false},
		{"shoot(5)", "shoot", 5, 0, false},
		{"shoot(3,4,5)", "shoot", 3, 4, false},
		{"shoot(5,6)", "shoot", 5, 6, false},
		{"space test(7)", "space test", 7, 0, false},
		{"symbol#(3)", "symbol#", 3, 0, false},
		{"shoot(  9  )", "shoot", 9, 0, false},
		{"shoot (6)", "shoot", 6, 0, false},
		{"shoot()", "shoot", 1, 0, false},
		{"shoot(abc)", "", 0, 0, true},
		{"shoot(6", "", 0, 0, true},
		{"shoot(6),", "", 0, 0, true},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			name, cnt, sleep, err := ParseShootName(tc.input)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tc.wantName, name, "Name does not match for input: %s", tc.input)
			assert.Equal(t, tc.wantSleep, sleep, "Name does not match for input: %s", tc.input)
			assert.Equal(t, tc.wantCnt, cnt, "Count does not match for input: %s", tc.input)
		})
	}
}
