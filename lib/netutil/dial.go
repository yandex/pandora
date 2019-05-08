// Copyright (c) 2017 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package netutil

import (
	"context"
	"net"
	"sync"

	"github.com/pkg/errors"
)

//go:generate mockery -name=Dialer -case=underscore -outpkg=netmock

type Dialer interface {
	DialContext(ctx context.Context, net, addr string) (net.Conn, error)
}

var _ Dialer = &net.Dialer{}

type DialerFunc func(ctx context.Context, network, address string) (net.Conn, error)

func (f DialerFunc) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	return f(ctx, network, address)
}

// NewDNSCachingDialer returns dialer with primitive DNS caching logic
// that remembers remote address on first try, and use it in future.
func NewDNSCachingDialer(dialer Dialer, cache DNSCache) DialerFunc {
	return func(ctx context.Context, network, addr string) (conn net.Conn, err error) {
		resolved, ok := cache.Get(addr)
		if ok {
			return dialer.DialContext(ctx, network, resolved)
		}
		conn, err = dialer.DialContext(ctx, network, addr)
		if err != nil {
			return
		}
		remoteAddr := conn.RemoteAddr().(*net.TCPAddr)
		_, port, err := net.SplitHostPort(addr)
		if err != nil {
			conn.Close()
			return nil, errors.Wrap(err, "invalid address, but successful dial - should not happen")
		}
		cache.Add(addr, net.JoinHostPort(remoteAddr.IP.String(), port))
		return
	}
}

var DefaultDNSCache = &SimpleDNSCache{}

// LookupReachable tries to resolve addr via connecting to it.
// This method has much more overhead, but get guaranteed reachable resolved addr.
// Example: host is resolved to IPv4 and IPv6, but IPv4 is not working on machine.
// LookupReachable will return IPv6 in that case.
func LookupReachable(addr string) (string, error) {
	d := net.Dialer{DualStack: true}
	conn, err := d.Dial("tcp", addr)
	if err != nil {
		return "", err
	}
	defer conn.Close()
	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		return "", err
	}
	remoteAddr := conn.RemoteAddr().(*net.TCPAddr)
	return net.JoinHostPort(remoteAddr.IP.String(), port), nil
}

// WarmDNSCache tries connect to addr, and adds conn remote ip + addr port to cache.
func WarmDNSCache(c DNSCache, addr string) error {
	var d net.Dialer
	conn, err := NewDNSCachingDialer(&d, c).DialContext(context.Background(), "tcp", addr)
	if err != nil {
		return err
	}
	conn.Close()
	return nil
}

//go:generate mockery -name=DNSCache -case=underscore -outpkg=netmock

type DNSCache interface {
	Get(addr string) (string, bool)
	Add(addr, resolved string)
}

type SimpleDNSCache struct {
	rw         sync.RWMutex
	hostToAddr map[string]string
}

func (c *SimpleDNSCache) Get(addr string) (resolved string, ok bool) {
	c.rw.RLock()
	if c.hostToAddr == nil {
		c.rw.RUnlock()
		return
	}
	resolved, ok = c.hostToAddr[addr]
	c.rw.RUnlock()
	return
}

func (c *SimpleDNSCache) Add(addr, resolved string) {
	c.rw.Lock()
	if c.hostToAddr == nil {
		c.hostToAddr = make(map[string]string)
	}
	c.hostToAddr[addr] = resolved
	c.rw.Unlock()
}
