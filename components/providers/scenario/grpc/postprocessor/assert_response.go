package postprocessor

import (
	"fmt"
	"strings"

	"github.com/golang/protobuf/proto"
)

type errAssert struct {
	pattern string
	t       string
}

func (e *errAssert) Error() string {
	return "assert failed: " + e.t + " does not contain " + e.pattern
}

type AssertResponse struct {
	Payload    []string
	StatusCode int `config:"status_code"`
}

func (a AssertResponse) Process(out proto.Message, code int) (map[string]any, error) {
	if a.StatusCode != 0 && a.StatusCode != code {
		return nil, &errAssert{
			pattern: fmt.Sprintf("expect code %d, recieve code %d", a.StatusCode, code),
			t:       "code",
		}
	}
	if len(a.Payload) == 0 {
		return nil, nil
	}
	if out == nil {
		return nil, &errAssert{pattern: "response is nil", t: "payload"}
	}

	o := out.String()
	for _, v := range a.Payload {
		if !strings.Contains(o, v) {
			return nil, &errAssert{pattern: v, t: "body"}
		}
	}

	return nil, nil
}

func NewAssertResponsePostprocessor(cfg AssertResponse) (Postprocessor, error) {
	return &cfg, nil
}
