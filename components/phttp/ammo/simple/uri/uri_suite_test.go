package uri

import (
	"testing"
)

func TestUri(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Uri Suite")
}
