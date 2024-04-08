package netutil

import (
	"context"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	netmock "github.com/yandex/pandora/lib/netutil/mocks"
)

func Test_DNS(t *testing.T) {
	t.Run("lookup reachable", func(t *testing.T) {
		listener, err := net.ListenTCP("tcp4", nil)
		defer func() { _ = listener.Close() }()
		assert.NoError(t, err)

		port := strconv.Itoa(listener.Addr().(*net.TCPAddr).Port)
		addr := "localhost:" + port
		expectedResolved := "127.0.0.1:" + port

		resolved, err := LookupReachable(addr, time.Second)
		assert.NoError(t, err)
		assert.Equal(t, expectedResolved, resolved)
	})

	const (
		addr     = "localhost:8888"
		resolved = "[::1]:8888"
	)

	t.Run("cache", func(t *testing.T) {
		cache := &SimpleDNSCache{}
		got, ok := cache.Get(addr)
		assert.False(t, ok)
		assert.Equal(t, "", got)

		cache.Add(addr, resolved)
		got, ok = cache.Get(addr)
		assert.True(t, ok)
		assert.Equal(t, resolved, got)
	})

	t.Run("Dialer cache miss", func(t *testing.T) {
		ctx := context.Background()
		mockConn := &netmock.Conn{}
		mockConn.On("RemoteAddr").Return(&net.TCPAddr{
			IP:   net.IPv6loopback,
			Port: 8888,
		})
		cache := &netmock.DNSCache{}
		cache.On("Get", addr).Return("", false)
		cache.On("Add", addr, resolved)
		dialer := &netmock.Dialer{}
		dialer.On("DialContext", ctx, "tcp", addr).Return(mockConn, nil)

		testee := NewDNSCachingDialer(dialer, cache)
		conn, err := testee.DialContext(ctx, "tcp", addr)
		assert.NoError(t, err)
		assert.Equal(t, mockConn, conn)

		mockConn.AssertExpectations(t)
		cache.AssertExpectations(t)
		dialer.AssertExpectations(t)
	})

	t.Run("Dialer cache hit", func(t *testing.T) {
		ctx := context.Background()
		mockConn := &netmock.Conn{}
		cache := &netmock.DNSCache{}
		cache.On("Get", addr).Return(resolved, true)
		dialer := &netmock.Dialer{}
		dialer.On("DialContext", ctx, "tcp", resolved).Return(mockConn, nil)

		testee := NewDNSCachingDialer(dialer, cache)
		conn, err := testee.DialContext(ctx, "tcp", addr)
		assert.NoError(t, err)
		assert.Equal(t, mockConn, conn)

		mockConn.AssertExpectations(t)
		cache.AssertExpectations(t)
		dialer.AssertExpectations(t)
	})

	t.Run("Dialer cache miss err", func(t *testing.T) {
		ctx := context.Background()
		expectedErr := errors.New("dial failed")
		cache := &netmock.DNSCache{}
		cache.On("Get", addr).Return("", false)
		dialer := &netmock.Dialer{}
		dialer.On("DialContext", ctx, "tcp", addr).Return(nil, expectedErr)

		testee := NewDNSCachingDialer(dialer, cache)
		conn, err := testee.DialContext(ctx, "tcp", addr)
		assert.ErrorIs(t, err, expectedErr)
		assert.Nil(t, conn)

		cache.AssertExpectations(t)
		dialer.AssertExpectations(t)
	})

}
