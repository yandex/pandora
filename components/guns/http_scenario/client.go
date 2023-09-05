package httpscenario

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"

	"go.uber.org/zap"
	"golang.org/x/net/http2"
)

//go:generate go run github.com/vektra/mockery/v2@v2.22.1 --inpackage --name=Client --filename=mock_client.go

type Client interface {
	Do(req *http.Request) (*http.Response, error)
	CloseIdleConnections() // We should close idle conns after gun close.
}

func newClient(tr *http.Transport, redirect bool) Client {
	if redirect {
		return redirectClient{Client: &http.Client{Transport: tr}}
	}
	return noRedirectClient{Transport: tr}
}

type redirectClient struct{ *http.Client }

func (c redirectClient) CloseIdleConnections() {
	c.Transport.(*http.Transport).CloseIdleConnections()
}

type noRedirectClient struct{ *http.Transport }

func (c noRedirectClient) Do(req *http.Request) (*http.Response, error) {
	return c.Transport.RoundTrip(req)
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
		return fmt.Errorf("http2: unexpected ALPN protocol %q; want %q", p, http2.NextProtoTLS)
	}
	if !state.NegotiatedProtocolIsMutual {
		return errors.New("http2: could not negotiate protocol mutually")
	}
	return nil
}
