package httpscenario

import (
	"io"
	"net/http"
	"time"
)

type Preprocessor interface {
	// Process is called before request is sent
	// templateVars - variables from template. Can be modified
	// sourceVars - variables from sources. Must NOT be modified
	Process(templateVars map[string]any) (map[string]any, error)
}

type Postprocessor interface {
	Process(requestVars map[string]any, resp *http.Response, body io.Reader) error
}

type VariableStorage interface {
	Variables() map[string]any
}

type Step interface {
	GetName() string
	GetURL() string
	GetMethod() string
	GetBody() []byte
	GetHeaders() map[string]string
	GetTag() string
	GetTemplater() string
	GetPostProcessors() []Postprocessor
	Preprocessor() Preprocessor
	GetSleep() time.Duration
}

type requestParts struct {
	URL     string
	Method  string
	Body    []byte
	Headers map[string]string
}

type Ammo interface {
	Steps() []Step
	ID() uint64
	Sources() VariableStorage
	Name() string
	GetMinWaitingTime() time.Duration
}
