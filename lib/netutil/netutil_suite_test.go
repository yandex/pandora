package netutil

import (
	"context"
	"net"
	"strconv"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/pkg/errors"
	"github.com/yandex/pandora/lib/ginkgoutil"
	netmock "github.com/yandex/pandora/lib/netutil/mocks"
)

func TestNetutil(t *testing.T) {
	ginkgoutil.RunSuite(t, "Netutil Suite")
}

var _ = Describe("DNS", func() {

	It("lookup reachable", func() {
		listener, err := net.ListenTCP("tcp4", nil)
		defer listener.Close()
		Expect(err).NotTo(HaveOccurred())

		port := strconv.Itoa(listener.Addr().(*net.TCPAddr).Port)
		addr := "localhost:" + port
		expectedResolved := "127.0.0.1:" + port

		resolved, err := LookupReachable(addr)
		Expect(err).NotTo(HaveOccurred())
		Expect(resolved).To(Equal(expectedResolved))
	})

	const (
		addr     = "localhost:8888"
		resolved = "[::1]:8888"
	)

	It("cache", func() {
		cache := &SimpleDNSCache{}
		got, ok := cache.Get(addr)
		Expect(ok).To(BeFalse())
		Expect(got).To(BeEmpty())

		cache.Add(addr, resolved)
		got, ok = cache.Get(addr)
		Expect(ok).To(BeTrue())
		Expect(got).To(Equal(resolved))
	})

	It("Dialer cache miss", func() {
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
		Expect(err).NotTo(HaveOccurred())
		Expect(conn).To(Equal(mockConn))

		ginkgoutil.AssertExpectations(mockConn, cache, dialer)
	})

	It("Dialer cache hit", func() {
		ctx := context.Background()
		mockConn := &netmock.Conn{}
		cache := &netmock.DNSCache{}
		cache.On("Get", addr).Return(resolved, true)
		dialer := &netmock.Dialer{}
		dialer.On("DialContext", ctx, "tcp", resolved).Return(mockConn, nil)

		testee := NewDNSCachingDialer(dialer, cache)
		conn, err := testee.DialContext(ctx, "tcp", addr)
		Expect(err).NotTo(HaveOccurred())
		Expect(conn).To(Equal(mockConn))

		ginkgoutil.AssertExpectations(mockConn, cache, dialer)
	})

	It("Dialer cache miss err", func() {
		ctx := context.Background()
		expectedErr := errors.New("dial failed")
		cache := &netmock.DNSCache{}
		cache.On("Get", addr).Return("", false)
		dialer := &netmock.Dialer{}
		dialer.On("DialContext", ctx, "tcp", addr).Return(nil, expectedErr)

		testee := NewDNSCachingDialer(dialer, cache)
		conn, err := testee.DialContext(ctx, "tcp", addr)
		Expect(err).To(Equal(expectedErr))
		Expect(conn).To(BeNil())

		ginkgoutil.AssertExpectations(cache, dialer)
	})

})
