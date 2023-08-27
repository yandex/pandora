package math

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_GCD(t *testing.T) {
	tests := []struct {
		name string
		a    int64
		b    int64
		want int64
	}{
		{
			name: "",
			a:    40,
			b:    60,
			want: 20,
		},
		{
			name: "",
			a:    2,
			b:    3,
			want: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, GCD(tt.a, tt.b), "GCD(%v, %v)", tt.a, tt.b)
		})
	}
}

func Test_GCDM(t *testing.T) {
	tests := []struct {
		name    string
		weights []int64
		want    int64
	}{
		{
			name:    "",
			weights: []int64{20, 30, 60},
			want:    10,
		},
		{
			name:    "",
			weights: []int64{6, 6, 6},
			want:    6,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, GCDM(tt.weights...), "GCDM(%v)", tt.weights)
		})
	}
}
func Test_LCM(t *testing.T) {
	tests := []struct {
		name string
		a    int64
		b    int64
		want int64
	}{
		{
			name: "",
			a:    40,
			b:    60,
			want: 120,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, LCM(tt.a, tt.b), "LCM(%v, %v)", tt.a, tt.b)
		})
	}
}

func Test_lcmm(t *testing.T) {
	tests := []struct {
		name string
		a    []int64
		want int64
	}{
		{
			name: "",
			a:    []int64{3, 4, 6},
			want: 12,
		},
		{
			name: "",
			a:    []int64{3, 4, 5, 6, 7}, // 140,105,84,70,60
			want: 420,
		},
		{
			name: "",
			a:    []int64{2, 4, 5, 10},
			want: 20,
		},
		{
			name: "",
			a:    []int64{20, 20, 20, 20},
			want: 20,
		},
		{
			name: "",
			a:    []int64{40, 50, 70},
			want: 1400,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, LCMM(tt.a...), "LCMM(%v)", tt.a)
		})
	}
}
