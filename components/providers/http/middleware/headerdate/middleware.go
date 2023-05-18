package headerdate

import (
	"context"
	"net/http"
	"time"

	"go.uber.org/zap"
)

const defaultHeaderName = "Date"

type Config struct {
	Location   string
	HeaderName string
}

func NewMiddleware(cfg Config) (*Middleware, error) {
	m := &Middleware{location: time.UTC, header: defaultHeaderName}

	if cfg.Location != "" {
		loc, err := time.LoadLocation(cfg.Location)
		if err != nil {
			return nil, err
		}
		m.location = loc
	}
	if cfg.HeaderName != "" {
		m.header = cfg.HeaderName
	}

	return m, nil
}

type Middleware struct {
	location *time.Location
	header   string
}

func (m *Middleware) InitMiddleware(ctx context.Context, log *zap.Logger) error {
	return nil
}

func (m *Middleware) UpdateRequest(req *http.Request) error {
	req.Header.Add(m.header, time.Now().In(m.location).Format(http.TimeFormat))
	return nil
}
