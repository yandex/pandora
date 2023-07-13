package scenario

import (
	"net"

	"github.com/spf13/afero"
	"go.uber.org/zap"

	phttp "github.com/yandex/pandora/components/guns/http"
	"github.com/yandex/pandora/core"
	"github.com/yandex/pandora/core/aggregator/netsample"
	"github.com/yandex/pandora/core/register"
	"github.com/yandex/pandora/lib/answlog"
	"github.com/yandex/pandora/lib/netutil"
)

func WrapGun(g Gun) core.Gun {
	if g == nil {
		return nil
	}
	return &gunWrapper{g}
}

type gunWrapper struct{ Gun }

func (g *gunWrapper) Shoot(ammo core.Ammo) {
	g.Gun.Shoot(ammo.(Ammo))
}

func (g *gunWrapper) Bind(a core.Aggregator, deps core.GunDeps) error {
	return g.Gun.Bind(netsample.UnwrapAggregator(a), deps)
}

func Import(fs afero.Fs) {
	register.Gun("http/scenario", func(conf phttp.HTTPGunConfig) func() core.Gun {
		targetResolved, _ := PreResolveTargetAddr(&conf.Client, conf.Gun.Target)
		answLog := answlog.Init(conf.Gun.Base.AnswLog.Path)
		return func() core.Gun {
			return WrapGun(NewHTTPGun(conf, answLog, targetResolved))
		}
	}, phttp.DefaultHTTPGunConfig)

	register.Gun("http2/scenario", func(conf phttp.HTTP2GunConfig) func() (core.Gun, error) {
		targetResolved, _ := PreResolveTargetAddr(&conf.Client, conf.Gun.Target)
		answLog := answlog.Init(conf.Gun.Base.AnswLog.Path)
		return func() (core.Gun, error) {
			gun, err := NewHTTP2Gun(conf, answLog, targetResolved)
			return WrapGun(gun), err
		}
	}, phttp.DefaultHTTP2GunConfig)
}

// DNS resolve optimisation.
// When DNSCache turned off - do nothing extra, host will be resolved on every shoot.
// When using resolved target, don't use DNS caching logic - it is useless.
// If we can resolve accessible target addr - use it as target, not use caching.
// Otherwise just use DNS cache - we should not fail shooting, we should try to
// connect on every shoot. DNS cache will save resolved addr after first successful connect.
func PreResolveTargetAddr(clientConf *phttp.ClientConfig, target string) (string, error) {
	if !clientConf.Dialer.DNSCache {
		return target, nil
	}
	if endpointIsResolved(target) {
		clientConf.Dialer.DNSCache = false
		return target, nil
	}
	resolved, err := netutil.LookupReachable(target, clientConf.Dialer.Timeout)
	if err != nil {
		zap.L().Warn("DNS target pre resolve failed", zap.String("target", target), zap.Error(err))
		return target, err
	}
	clientConf.Dialer.DNSCache = false
	return resolved, nil
}

func endpointIsResolved(endpoint string) bool {
	host, _, err := net.SplitHostPort(endpoint)
	if err != nil {
		return false
	}
	return net.ParseIP(host) != nil
}
