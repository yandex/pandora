package phttp

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptrace"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/yandex/pandora/lib/netutil"
	"go.uber.org/zap"
	"golang.org/x/net/http2"
)

//go:generate mockery --name=Client --case=underscore --inpackage --testonly

type Client interface {
	Do(req *http.Request) (*http.Response, error)
	CloseIdleConnections() // We should close idle conns after gun close.
}

type ClientConfig struct {
	Redirect   bool            // When true, follow HTTP redirects.
	Dialer     DialerConfig    `config:"dial"`
	Transport  TransportConfig `config:",squash"`
	ConnectSSL bool            `config:"connect-ssl"` // Defines if tunnel encrypted.
}

type ClientConstructor func(clientConfig ClientConfig, target string) Client

func DefaultClientConfig() ClientConfig {
	return ClientConfig{
		Transport: DefaultTransportConfig(),
		Dialer:    DefaultDialerConfig(),
		Redirect:  false,
	}
}

// DialerConfig can be mapped on net.Dialer.
// Set net.Dialer for details.
type DialerConfig struct {
	DNSCache bool `config:"dns-cache" map:"-"`

	Timeout   time.Duration `config:"timeout"`
	DualStack bool          `config:"dual-stack"`

	// IPv4/IPv6 settings should not matter really,
	// because target should be dialed using pre-resolved addr.
	FallbackDelay time.Duration `config:"fallback-delay"`
	KeepAlive     time.Duration `config:"keep-alive"`
}

func DefaultDialerConfig() DialerConfig {
	return DialerConfig{
		DNSCache:  true,
		DualStack: true,
		Timeout:   3 * time.Second,
		KeepAlive: 120 * time.Second,
	}
}

func NewDialer(conf DialerConfig) netutil.Dialer {
	d := &net.Dialer{
		Timeout:       conf.Timeout,
		DualStack:     conf.DualStack,
		FallbackDelay: conf.FallbackDelay,
		KeepAlive:     conf.KeepAlive,
	}
	if !conf.DNSCache {
		return d
	}
	return netutil.NewDNSCachingDialer(d, netutil.DefaultDNSCache)
}

// TransportConfig can be mapped on http.Transport.
// See http.Transport for details.
type TransportConfig struct {
	TLSHandshakeTimeout   time.Duration `config:"tls-handshake-timeout"`
	DisableKeepAlives     bool          `config:"disable-keep-alives"`
	DisableCompression    bool          `config:"disable-compression"`
	MaxIdleConns          int           `config:"max-idle-conns"`
	MaxIdleConnsPerHost   int           `config:"max-idle-conns-per-host"`
	IdleConnTimeout       time.Duration `config:"idle-conn-timeout"`
	ResponseHeaderTimeout time.Duration `config:"response-header-timeout"`
	ExpectContinueTimeout time.Duration `config:"expect-continue-timeout"`
}

func DefaultTransportConfig() TransportConfig {
	return TransportConfig{
		MaxIdleConns:          0, // No limit.
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   1 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		DisableCompression:    true,
	}
}

func NewTransport(conf TransportConfig, dial netutil.DialerFunc, target string) http.RoundTripper {
	tr := &http.Transport{
		TLSHandshakeTimeout:   conf.TLSHandshakeTimeout,
		DisableKeepAlives:     conf.DisableKeepAlives,
		DisableCompression:    conf.DisableCompression,
		MaxIdleConns:          conf.MaxIdleConns,
		MaxIdleConnsPerHost:   conf.MaxIdleConnsPerHost,
		IdleConnTimeout:       conf.IdleConnTimeout,
		ResponseHeaderTimeout: conf.ResponseHeaderTimeout,
		ExpectContinueTimeout: conf.ExpectContinueTimeout,
	}
	host, _, err := net.SplitHostPort(target)
	if err != nil {
		zap.L().Panic("HTTP transport configure fail", zap.Error(err))
	}
	tr.TLSClientConfig = &tls.Config{
		InsecureSkipVerify: true,                 // We should not spend time for this stuff.
		NextProtos:         []string{"http/1.1"}, // Disable HTTP/2. Use HTTP/2 transport explicitly, if needed.
		ServerName:         host,
	}
	tr.DialContext = dial
	return tr
}

func NewHTTP2Transport(conf TransportConfig, dial netutil.DialerFunc, target string) http.RoundTripper {
	tr := NewTransport(conf, dial, target).(*http.Transport)
	err := http2.ConfigureTransport(tr)
	if err != nil {
		zap.L().Panic("HTTP/2 transport configure fail", zap.Error(err))
	}

	tr.TLSClientConfig.NextProtos = []string{"h2"}
	return withResponseHeaderTimeout(tr, conf.ResponseHeaderTimeout)
}

func NewH2CTransport(conf TransportConfig, dial netutil.DialerFunc, target string) http.RoundTripper {
	rt := &http2.Transport{
		AllowHTTP: true,
		DialTLSContext: func(ctx context.Context, network, addr string, _ *tls.Config) (net.Conn, error) {
			return dial(ctx, network, addr)
		},
		DisableCompression: conf.DisableCompression,
		IdleConnTimeout:    conf.IdleConnTimeout,
	}
	return withResponseHeaderTimeout(rt, conf.ResponseHeaderTimeout)
}

func withResponseHeaderTimeout(rt http.RoundTripper, timeout time.Duration) http.RoundTripper {
	if timeout <= 0 {
		return rt
	}
	return &responseHeaderTimeoutRT{rt: rt, timeout: timeout}
}

type responseHeaderTimeoutRT struct {
	rt      http.RoundTripper
	timeout time.Duration
}

func (t *responseHeaderTimeoutRT) RoundTrip(req *http.Request) (*http.Response, error) {
	parent := req.Context()
	ctx, cancel := context.WithCancel(parent)

	var (
		mu           sync.Mutex
		gotFirstByte bool
	)
	timer := time.AfterFunc(t.timeout, cancel)
	timer.Stop()

	trace := &httptrace.ClientTrace{
		WroteRequest: func(info httptrace.WroteRequestInfo) {
			mu.Lock()
			defer mu.Unlock()
			if info.Err == nil && !gotFirstByte {
				timer.Reset(t.timeout)
			}
		},
		GotFirstResponseByte: func() {
			mu.Lock()
			defer mu.Unlock()
			gotFirstByte = true
			timer.Stop()
		},
	}

	req = req.WithContext(httptrace.WithClientTrace(ctx, trace))
	resp, err := t.rt.RoundTrip(req)
	timer.Stop()

	if err != nil {
		timedOut := parent.Err() == nil && ctx.Err() != nil
		cancel()
		if timedOut {
			return nil, &responseHeaderTimeoutError{timeout: t.timeout, cause: err}
		}
		return nil, err
	}
	resp.Body = &cancelOnCloseBody{ReadCloser: resp.Body, cancel: cancel}
	return resp, nil
}

var _ net.Error = (*responseHeaderTimeoutError)(nil)

type responseHeaderTimeoutError struct {
	timeout time.Duration
	cause   error
}

func (e *responseHeaderTimeoutError) Error() string {
	return fmt.Sprintf("response header timeout exceeded (%s): %v", e.timeout, e.cause)
}

func (e *responseHeaderTimeoutError) Timeout() bool   { return true }
func (e *responseHeaderTimeoutError) Temporary() bool { return false }
func (e *responseHeaderTimeoutError) Unwrap() error   { return e.cause }

type cancelOnCloseBody struct {
	io.ReadCloser
	cancel context.CancelFunc
	once   sync.Once
}

func (b *cancelOnCloseBody) Close() error {
	err := b.ReadCloser.Close()
	b.once.Do(b.cancel)
	return err
}

func NewRedirectingClient(tr http.RoundTripper, redirect bool) Client {
	if redirect {
		return redirectClient{&http.Client{Transport: tr}}
	}
	return noRedirectClient{tr}
}

func CloseIdleConnectionsRoundTripper(tr http.RoundTripper) {
	switch v := tr.(type) {
	case *http.Transport:
		v.CloseIdleConnections()
	case *http2.Transport:
		v.CloseIdleConnections()
	case *responseHeaderTimeoutRT:
		CloseIdleConnectionsRoundTripper(v.rt)
	default:
	}
}

type redirectClient struct{ *http.Client }

func (c redirectClient) CloseIdleConnections() {
	CloseIdleConnectionsRoundTripper(c.Transport)
}

type noRedirectClient struct{ http.RoundTripper }

func (c noRedirectClient) Do(req *http.Request) (*http.Response, error) {
	return c.RoundTripper.RoundTrip(req)
}

func (c noRedirectClient) CloseIdleConnections() {
	CloseIdleConnectionsRoundTripper(c.RoundTripper)
}

// Used to cancel shooting in HTTP/2 gun, when target doesn't support HTTP/2
type panicOnHTTP1Client struct {
	Client
}

const notHTTP2PanicMsg = "Non HTTP/2 connection established. Seems that target doesn't support HTTP/2."

func (c *panicOnHTTP1Client) Do(req *http.Request) (*http.Response, error) {
	res, err := c.Client.Do(req)
	if err != nil {
		var opError *net.OpError
		// Unfortunately, Go doesn't expose tls.alert (https://github.com/golang/go/issues/35234), so we make decisions based on the error message
		if errors.As(err, &opError) && opError.Op == "remote error" && strings.Contains(err.Error(), "no application protocol") {
			zap.L().Panic(notHTTP2PanicMsg, zap.Error(err))
		}
		return nil, err
	}
	err = checkHTTP2(res.TLS)
	if err != nil {
		zap.L().Panic(notHTTP2PanicMsg, zap.Error(err))
	}
	return res, nil
}

func checkHTTP2(state *tls.ConnectionState) error {
	if state == nil {
		return errors.New("http2: non TLS connection")
	}
	if p := state.NegotiatedProtocol; p != http2.NextProtoTLS {
		return errors.Errorf("http2: unexpected ALPN protocol %q; want %q", p, http2.NextProtoTLS)
	}
	if !state.NegotiatedProtocolIsMutual {
		return errors.New("http2: could not negotiate protocol mutually")
	}
	return nil
}

func getHostWithoutPort(target string) string {
	host, _, err := net.SplitHostPort(target)
	if err != nil {
		host = target
	}
	return host
}
