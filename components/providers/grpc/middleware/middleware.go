package middleware

import (
	"context"

	"github.com/yandex/pandora/components/providers/grpc/ammo"
	"go.uber.org/zap"
)

type Middleware interface {
	InitMiddleware(ctx context.Context, log *zap.Logger) error
	UpdateRequest(req *ammo.Ammo) error
}
