package preprocessor

import (
	"github.com/yandex/pandora/components/guns/grpc/scenario"
)

type Preprocessor interface {
	Process(call *scenario.Call, templateVars map[string]any) (newVars map[string]any, err error)
}
