package coremock

import (
	"fmt"
	"unsafe"
)

// Implement Stringer, so when Aggregator is passed as arg to another mock call,
// it not read and data races not created.
func (_m *Aggregator) String() string {
	return fmt.Sprintf("coremock.Aggregator{%v}", unsafe.Pointer(_m))
}
