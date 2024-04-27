package str

import (
	"testing"
)

func BenchmarkFormatString(b *testing.B) {
	n := 1000
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		FormatString(n)
	}
}

func TestFormatString(t *testing.T) {
	n := 100.001
	s := FormatString(n)
	t.Log(s)
	if s != "100.001" {
		t.Errorf("%s", s)
	}
}

func TestMultiFormatString(t *testing.T) {
	list := map[string]interface{}{
		"10":              10,
		"100":             "100",
		"100.001":         100.001,
		"hello":           "hello",
		"[1 2 3]":         []int{1, 2, 3},
		"true":            true,
		"map[name:jason]": map[string]interface{}{"name": "jason"},
	}
	for k, v := range list {
		s := FormatString(v)
		if s != k {
			t.Errorf("Error: %v to %s,but %s", v, k, s)
		}
	}

}
