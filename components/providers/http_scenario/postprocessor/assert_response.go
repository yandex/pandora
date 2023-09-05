package postprocessor

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type errAssert struct {
	pattern string
	t       string
}

func (e *errAssert) Error() string {
	return "assert failed: " + e.t + " does not contain " + e.pattern
}

type AssertSize struct {
	Val int
	Op  string
}

type AssertResponse struct {
	Headers    map[string]string
	Body       []string
	StatusCode int `config:"status_code"`
	Size       *AssertSize
}

func (a AssertResponse) Process(resp *http.Response, body io.Reader) (map[string]any, error) {
	var b []byte
	var err error
	if len(a.Body) > 0 && body != nil {
		b, err = io.ReadAll(body)
		if err != nil {
			return nil, fmt.Errorf("cant read body: %w", err)
		}
	}
	for _, v := range a.Body {
		if !bytes.Contains(b, []byte(v)) {
			return nil, &errAssert{pattern: v, t: "body"}
		}
	}
	for k, v := range a.Headers {
		if !(strings.Contains(resp.Header.Get(k), v)) {
			return nil, &errAssert{pattern: v, t: "header " + k}
		}
	}
	if a.StatusCode != 0 && a.StatusCode != resp.StatusCode {
		return nil, &errAssert{
			pattern: fmt.Sprintf("expect code %d, recieve code %d", a.StatusCode, resp.StatusCode),
			t:       "code",
		}
	}
	if a.Size != nil {
		pattern := fmt.Sprintf("expect size %d %s %d", a.Size.Val, a.Size.Op, len(b))
		switch a.Size.Op {
		case "eq", "=":
			if a.Size.Val != len(b) {
				return nil, &errAssert{
					pattern: pattern,
					t:       "size",
				}
			}
		case "lt", "<":
			if a.Size.Val < len(b) {
				return nil, &errAssert{
					pattern: pattern,
					t:       "size",
				}
			}
		case "gt", ">":
			if a.Size.Val > len(b) {
				return nil, &errAssert{
					pattern: pattern,
					t:       "size",
				}
			}
		default:
			return nil, fmt.Errorf("unknown op %s", a.Size.Op)
		}
	}

	return nil, nil
}

func (a AssertResponse) Validate() error {
	if a.Size != nil {
		if a.Size.Val < 0 {
			return fmt.Errorf("size must be positive")
		}
		if a.Size.Op != "eq" && a.Size.Op != "=" && a.Size.Op != "lt" && a.Size.Op != "<" && a.Size.Op != "gt" && a.Size.Op != ">" {
			return fmt.Errorf("assert/response validation unknown op %s", a.Size.Op)
		}
	}
	return nil
}

func NewAssertResponsePostprocessor(cfg AssertResponse) (Postprocessor, error) {
	err := cfg.Validate()
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}
