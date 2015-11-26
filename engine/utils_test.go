package engine

import (
	"testing"

	"github.com/yandex/pandora/config"
)

func TestNotExistentLimiterConfig(t *testing.T) {
	lc := &config.Limiter{
		LimiterType: "NOT_EXISTENT",
		Parameters:  nil,
	}
	l, err := GetLimiter(lc)

	if err == nil {
		t.Errorf("No error on non existent limiter type")
	}
	if l != nil {
		t.Errorf("Returned non-nil limiter for non existent limiter type")
	}
}

func TestEmptyLimiterConfig(t *testing.T) {
	l, err := GetLimiter(nil)

	if err != nil {
		t.Errorf("Error on empty limiter config: %s", err)
	}
	if l != nil {
		t.Errorf("Returned non-nil limiter for empty config")
	}
}
