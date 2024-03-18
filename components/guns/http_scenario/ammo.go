package httpscenario

import (
	"io"
	"net/http"
	"time"

	"github.com/yandex/pandora/components/providers/scenario"
)

type SourceStorage interface {
	Variables() map[string]any
}

type Scenario struct {
	Requests        []Request
	ID              uint64
	Name            string
	MinWaitingTime  time.Duration
	VariableStorage SourceStorage
}

func (a *Scenario) SetID(id uint64) {
	a.ID = id
}

func (a *Scenario) Clone() scenario.ProvAmmo {
	return &Scenario{
		Requests:        a.Requests,
		Name:            a.Name,
		MinWaitingTime:  a.MinWaitingTime,
		VariableStorage: a.VariableStorage,
	}
}

type Request struct {
	Method         string
	Headers        map[string]string
	Tag            string
	Body           *string
	Name           string
	URI            string
	Preprocessor   Preprocessor
	Postprocessors []Postprocessor
	Templater      Templater
	Sleep          time.Duration
}

func (r *Request) GetBody() []byte {
	if r.Body == nil {
		return nil
	}
	return []byte(*r.Body)
}

func (r *Request) GetHeaders() map[string]string {
	result := make(map[string]string, len(r.Headers))
	for k, v := range r.Headers {
		result[k] = v
	}
	return result
}

type Preprocessor interface {
	// Process is called before request is sent
	// templateVars - variables from template. Can be modified
	// sourceVars - variables from sources. Must NOT be modified
	Process(templateVars map[string]any) (map[string]any, error)
}

type Postprocessor interface {
	Process(resp *http.Response, body io.Reader) (map[string]any, error)
}

type RequestParts struct {
	URL     string
	Method  string
	Body    []byte
	Headers map[string]string
}
