package templater

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRandInt(t *testing.T) {
	tests := []struct {
		name      string
		args      []any
		want      int
		wantDelta float64
		wantErr   bool
	}{
		{
			name:      "No args",
			args:      nil,
			want:      15,
			wantDelta: 15,
		},
		{
			name:      "two args",
			args:      []any{10, 20},
			want:      15,
			wantDelta: 5,
		},
		{
			name:    "second arg is invalid",
			args:    []any{"26", "invalid"},
			wantErr: true,
		},
		{
			name:      "two string args can be converted",
			args:      []any{"200", "300"},
			want:      250,
			wantDelta: 50,
		},
		{
			name:    "second arg is invalid",
			args:    []any{20, "invalid"},
			wantErr: true,
		},
		{
			name:    "more than two args",
			args:    []any{100, 200, 30},
			wantErr: true,
		},
		{
			name:      "one args",
			args:      []any{50},
			want:      25,
			wantDelta: 25,
		},
		{
			name:      "one arg",
			args:      []any{10},
			want:      5,
			wantDelta: 5,
		},
		{
			name:      "two args",
			args:      []any{-10, 10},
			want:      0,
			wantDelta: 10,
		},
		{
			name:      "one negative arg",
			args:      []any{-5},
			want:      -3,
			wantDelta: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			get, err := RandInt(tt.args...)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			g, err := strconv.Atoi(get)
			require.NoError(t, err)
			require.InDelta(t, tt.want, g, tt.wantDelta)
		})
	}
}

func TestRandString(t *testing.T) {
	tests := []struct {
		name       string
		args       []any
		wantErr    bool
		wantLength int
	}{
		{
			name:       "No args, default length",
			args:       nil,
			wantLength: 1,
		},
		{
			name:       "Specific length",
			args:       []any{5},
			wantLength: 5,
		},
		{
			name:       "Specific length and characters",
			args:       []any{10, "abc"},
			wantLength: 10,
		},
		{
			name:    "Invalid length argument",
			args:    []any{"invalid"},
			wantErr: true,
		},
		{
			name:    "Invalid length, valid characters",
			args:    []any{"invalid", "def"},
			wantErr: true,
		},
		{
			name:    "More than two args",
			args:    []any{5, "gh", "extra"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := RandString(tt.args...)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Len(t, got, tt.wantLength)
		})
	}
}

func TestRandStringLetters(t *testing.T) {
	tests := []struct {
		name    string
		length  int
		letters string
		wantErr bool
	}{
		{
			name:    "Simple",
			length:  10,
			letters: "ab",
		},
		{
			name:    "Simple",
			length:  100,
			letters: "absdfave",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := RandString(tt.length, tt.letters)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Len(t, got, tt.length)

			l := map[rune]int{}
			for _, r := range got {
				l[r]++
			}
			gotCount := 0
			for _, c := range l {
				gotCount += c
			}
			require.Equal(t, tt.length, gotCount)
		})
	}
}

func TestParseFunc(t *testing.T) {
	tests := []struct {
		name     string
		arg      string
		wantF    any
		wantArgs []string
	}{
		{
			name:     "Simple",
			arg:      "randInt(10, 20)",
			wantF:    RandInt,
			wantArgs: []string{"10", "20"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotF, gotArgs := ParseFunc(tt.arg)
			f := gotF.(func(args ...any) (string, error))
			a := []any{}
			for _, arg := range gotArgs {
				a = append(a, arg)
			}
			_, _ = f(a...)
			require.Equal(t, tt.wantArgs, gotArgs)
		})
	}
}
