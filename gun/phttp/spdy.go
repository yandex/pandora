package phttp

import (
	"context"
	"crypto/tls"
	"log"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/amahi/spdy"
	"github.com/yandex/pandora/aggregate"
	"github.com/yandex/pandora/config"
	"github.com/yandex/pandora/gun"
)

type SPDYDialConfig struct {
	Timeout       time.Duration `config:"timeout"`
	DualStack     bool          `config:"dual-stack"`
	FallbackDelay time.Duration `config:"fallback-delay"`
}

type SPDYGunConfig struct {
	Dial       SPDYDialConfig `config:"dial"`
	PingPeriod time.Duration
	Target     string
}

func NewSPDYGun(conf SPDYGunConfig) *SPDYGun {
	// TODO: test gun and remove panic
	log.Panic("SPDY gun is not supported at that moment")
	closeCtx, onClose := context.WithCancel(context.Background())
	dialer := &net.Dialer{}
	err := config.Map(&dialer, conf.Dial)
	if err != nil {
		log.Panic("Dial config map error: ", err)
	}

	var g SPDYGun
	g = SPDYGun{
		Base: Base{
			Do:      g.Do,
			Connect: g.Connect,
		},
		config:   conf,
		dialer:   dialer,
		onClose:  onClose,
		closeCtx: closeCtx,
	}
	return &g
}

type SPDYGun struct {
	Base
	config   SPDYGunConfig
	onClose  context.CancelFunc
	closeCtx context.Context
	dialer   *net.Dialer

	startPingOnce sync.Once
	// CAS connecting 0 -> 1, when you need connect.
	connecting int32
	clientLock sync.Mutex
	client     *spdy.Client // Lazy set in connect.
}

var _ gun.Gun = (*SPDYGun)(nil)

func (g *SPDYGun) Do(req *http.Request) (*http.Response, error) {
	req.Method = "https"
	// TODO: need reconnect on error?
	return g.getClient().Do(req)
}

func (g *SPDYGun) Close() {
	g.clientLock.Lock()
	defer g.clientLock.Unlock()
	g.onClose()
	if g.client != nil {
		g.client.Close()
	}
	g.Base.Do = nil
	g.Base.Connect = nil
}

func (g *SPDYGun) getClient() *spdy.Client {
	g.clientLock.Lock()
	c := g.client
	g.clientLock.Unlock()
	return c
}

func (g *SPDYGun) Connect(ctx context.Context) error {
	if g.getClient() == nil {
		return g.connect(ctx)
	}
	return nil
}

func (g *SPDYGun) connect(ctx context.Context) (err error) {
	ok := atomic.CompareAndSwapInt32(&g.connecting, 0, 1)
	if !ok {
		return
	}
	g.clientLock.Lock()
	defer func() {
		atomic.StoreInt32(&g.connecting, 0)
		g.clientLock.Unlock()
	}()
	g.startPingOnce.Do(g.startAutoPing)
	ss := aggregate.AcquireSample("CONNECT")
	// TODO: metrics
	defer func() {
		if err != nil {
			ss.SetErr(err)
		}
		g.Results <- ss
	}()
	tlsConfig := tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{"spdy/3.1"},
	}
	g.dialer.Cancel = ctx.Done()
	conn, err := tls.DialWithDialer(g.dialer, "tcp", g.config.Target, &tlsConfig)
	if err != nil {
		return
	}
	client, err := spdy.NewClientConn(conn)
	if err != nil {
		return err
	} else {
		ss.SetProtoCode(http.StatusOK)
	}
	if g.client != nil {
		g.client.Close()
	}
	g.client = client
	return nil
}

func (g *SPDYGun) Ping() {
	deadline := time.Now().Add(g.config.Dial.Timeout)
	// Can block if connecting. Timeout includes connect time.
	client := g.getClient()
	if client == nil {
		// Not connected yet. Ignore.
		return
	}
	ss := aggregate.AcquireSample("PING")
	pinged, err := client.Ping(deadline.Sub(time.Now()))
	if err != nil {
		log.Printf("Client: ping: %s\n", err)
	} else if !pinged {
		log.Println("Client: ping: timed out")
	}
	if err == nil && pinged {
		ss.SetProtoCode(http.StatusOK)
	} else {
		ss.SetErr(context.DeadlineExceeded)
	}
	g.Results <- ss
	if err != nil {
		g.connect(context.Background())
	}
}

func (g *SPDYGun) startAutoPing() {
	if g.config.PingPeriod <= 0 {
		return
	}
	go func() {
		ticker := time.NewTicker(g.config.PingPeriod)
		defer ticker.Stop()
		for {
			select {
			case _ = <-ticker.C:
				g.Ping()
			case <-g.closeCtx.Done():
				return
			}
		}
	}()
}
