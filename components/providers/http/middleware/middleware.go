package middleware

import (
	"context"
	"net/http"

	"go.uber.org/zap"
)

type Middleware interface {
	InitMiddleware(ctx context.Context, log *zap.Logger) error
	UpdateRequest(req *http.Request) error
}
