package warmup

import (
	"context"

	"go.uber.org/zap"
)

type Options struct {
	Log *zap.Logger
	Ctx context.Context
}
