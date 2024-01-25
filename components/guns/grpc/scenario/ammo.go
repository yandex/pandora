package scenario

import (
	"time"

	"github.com/golang/protobuf/proto"
)

type SourceStorage interface {
	Variables() map[string]any
}

type Scenario struct {
	id              uint64
	Calls           []Call
	Name            string
	MinWaitingTime  time.Duration
	VariableStorage SourceStorage
}

func (a *Scenario) SetID(id uint64) {
	a.id = id
}

type Call struct {
	Name           string
	Preprocessors  []Preprocessor
	Postprocessors []Postprocessor

	Tag      string            `json:"tag"`
	Call     string            `json:"call"`
	Metadata map[string]string `json:"metadata"`
	Payload  []byte            `json:"payload"`

	Sleep time.Duration `json:"sleep"`
}

type Postprocessor interface {
	Process(out proto.Message, code int) (map[string]any, error)
}

type Preprocessor interface {
	Process(call *Call, templateVars map[string]any) (newVars map[string]any, err error)
}
