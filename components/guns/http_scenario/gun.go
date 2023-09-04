package httpscenario

import (
	"context"
	"io"
	"net/http"

	phttp "github.com/yandex/pandora/components/guns/http"
	"github.com/yandex/pandora/core"
	"github.com/yandex/pandora/core/aggregator/netsample"
	"go.uber.org/zap"
)

type Gun interface {
	Shoot(ammo Ammo)
	Bind(sample netsample.Aggregator, deps core.GunDeps) error
}

const (
	EmptyTag = "__EMPTY__"
)

type BaseGun struct {
	DebugLog   bool // Automaticaly set in Bind if Log accepts debug messages.
	Config     phttp.BaseGunConfig
	Connect    func(ctx context.Context) error // Optional hook.
	OnClose    func() error                    // Optional. Called on Close().
	Aggregator netsample.Aggregator            // Lazy set via BindResultTo.
	AnswLog    *zap.Logger
	core.GunDeps
	scheme         string
	hostname       string
	targetResolved string
	client         Client
	templater      Templater
}

var _ Gun = (*BaseGun)(nil)
var _ io.Closer = (*BaseGun)(nil)

func (g *BaseGun) Bind(aggregator netsample.Aggregator, deps core.GunDeps) error {
	log := deps.Log
	if ent := log.Check(zap.DebugLevel, "Gun bind"); ent != nil {
		// Enable debug level logging during shooting. Creating log entries isn't free.
		g.DebugLog = true
	}

	if g.Aggregator != nil {
		log.Panic("already binded")
	}
	if aggregator == nil {
		log.Panic("nil aggregator")
	}
	g.Aggregator = aggregator
	g.GunDeps = deps

	return nil
}

// Shoot is thread safe iff Do and Connect hooks are thread safe.
func (g *BaseGun) Shoot(ammo Ammo) {
	if g.Aggregator == nil {
		zap.L().Panic("must bind before shoot")
	}
	if g.Connect != nil {
		err := g.Connect(g.Ctx)
		if err != nil {
			g.Log.Warn("Connect fail", zap.Error(err))
			return
		}
	}

	err := g.shoot(ammo)
	if err != nil {
		g.Log.Warn("Invalid ammo", zap.Uint64("request", ammo.ID()), zap.Error(err))
		return
	}
}

func (g *BaseGun) Do(req *http.Request) (*http.Response, error) {
	return g.client.Do(req)
}

func (g *BaseGun) Close() error {
	if g.OnClose != nil {
		return g.OnClose()
	}
	return nil
}

func (g *BaseGun) shoot(ammo Ammo) error {
	// implement scenario generator
	return nil
}
