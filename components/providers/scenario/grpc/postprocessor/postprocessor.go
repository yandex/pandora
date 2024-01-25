package postprocessor

import (
	"github.com/golang/protobuf/proto"
)

type Config struct {
	Mapping map[string]string
}

type Postprocessor interface {
	Process(out proto.Message, code int) (map[string]any, error)
}
