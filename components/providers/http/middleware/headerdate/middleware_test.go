package headerdate

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestMiddleware_InitMiddleware(t *testing.T) {
	cfg := Config{Location: ""}
	middleware, err := NewMiddleware(cfg)
	assert.NoError(t, err)

	err = middleware.InitMiddleware(context.Background(), zap.NewNop())
	assert.NoError(t, err)
}

func TestMiddleware_UpdateRequest(t *testing.T) {
	t.Run("empty config", func(t *testing.T) {
		cfg := Config{}
		middleware, err := NewMiddleware(cfg)
		assert.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/", nil)

		err = middleware.UpdateRequest(req)
		assert.NoError(t, err)

		dateHeader := req.Header.Get("Date")
		assert.NotEmpty(t, dateHeader)

		expectedDate := time.Now().In(time.UTC).Format(http.TimeFormat)
		assert.Equal(t, expectedDate, dateHeader)
	})
	t.Run("America/New_York", func(t *testing.T) {
		loc, err := time.LoadLocation("America/New_York")
		assert.NoError(t, err)

		cfg := Config{Location: "America/New_York"}
		middleware, err := NewMiddleware(cfg)
		assert.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/", nil)

		err = middleware.UpdateRequest(req)
		assert.NoError(t, err)

		dateHeader := req.Header.Get("Date")
		assert.NotEmpty(t, dateHeader)

		expectedDate := time.Now().In(loc).Format(http.TimeFormat)
		assert.Equal(t, expectedDate, dateHeader)
	})
	t.Run("custom header name", func(t *testing.T) {
		loc, err := time.LoadLocation("America/New_York")
		assert.NoError(t, err)

		cfg := Config{Location: "America/New_York", HeaderName: "CreatedDate"}
		middleware, err := NewMiddleware(cfg)
		assert.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/", nil)

		err = middleware.UpdateRequest(req)
		assert.NoError(t, err)

		dateHeader := req.Header.Get("CreatedDate")
		assert.NotEmpty(t, dateHeader)

		expectedDate := time.Now().In(loc).Format(http.TimeFormat)
		assert.Equal(t, expectedDate, dateHeader)
	})
}

func TestMiddleware_UpdateRequest_InvalidLocation(t *testing.T) {
	cfg := Config{Location: "Invalid/Location"}
	_, err := NewMiddleware(cfg)
	assert.Error(t, err)
}
