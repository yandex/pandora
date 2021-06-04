// Copyright (c) 2017 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package phttp

import (
	"net"

	"github.com/spf13/afero"
	"go.uber.org/zap"

	"github.com/yandex/pandora/components/phttp"
	"github.com/yandex/pandora/components/phttp/ammo/simple/jsonline"
	"github.com/yandex/pandora/components/phttp/ammo/simple/raw"
	"github.com/yandex/pandora/components/phttp/ammo/simple/uri"
	"github.com/yandex/pandora/core"
	"github.com/yandex/pandora/core/register"
	"github.com/yandex/pandora/lib/netutil"
)

func Import(fs afero.Fs) {
	register.Provider("http/json", func(conf jsonline.Config) core.Provider {
		return jsonline.NewProvider(fs, conf)
	})

	register.Provider("uri", func(conf uri.Config) core.Provider {
		return uri.NewProvider(fs, conf)
	})

	register.Provider("raw", func(conf raw.Config) core.Provider {
		return raw.NewProvider(fs, conf)
	})

	register.Gun("http", func(conf phttp.HTTPGunConfig) func() core.Gun {
		_ = preResolveTargetAddr(&conf.Client, &conf.Gun.Target)
		return func() core.Gun { return phttp.WrapGun(phttp.NewHTTPGun(conf)) }
	}, phttp.DefaultHTTPGunConfig)

	register.Gun("http2", func(conf phttp.HTTP2GunConfig) func() (core.Gun, error) {
		_ = preResolveTargetAddr(&conf.Client, &conf.Gun.Target)
		return func() (core.Gun, error) {
			gun, err := phttp.NewHTTP2Gun(conf)
			return phttp.WrapGun(gun), err
		}
	}, phttp.DefaultHTTP2GunConfig)

	register.Gun("connect", func(conf phttp.ConnectGunConfig) func() core.Gun {
		_ = preResolveTargetAddr(&conf.Client, &conf.Target)
		return func() core.Gun {
			return phttp.WrapGun(phttp.NewConnectGun(conf))
		}
	}, phttp.DefaultConnectGunConfig)
}

// DNS resolve optimisation.
// When DNSCache turned off - do nothing extra, host will be resolved on every shoot.
// When using resolved target, don't use DNS caching logic - it is useless.
// If we can resolve accessible target addr - use it as target, not use caching.
// Otherwise just use DNS cache - we should not fail shooting, we should try to
// connect on every shoot. DNS cache will save resolved addr after first successful connect.
func preResolveTargetAddr(clientConf *phttp.ClientConfig, target *string) (err error) {
	if !clientConf.Dialer.DNSCache {
		return
	}
	if endpointIsResolved(*target) {
		clientConf.Dialer.DNSCache = false
		return
	}
	resolved, err := netutil.LookupReachable(*target)
	if err != nil {
		zap.L().Warn("DNS target pre resolve failed",
			zap.String("target", *target), zap.Error(err))
		return
	}
	clientConf.Dialer.DNSCache = false
	*target = resolved
	return
}

func endpointIsResolved(endpoint string) bool {
	host, _, err := net.SplitHostPort(endpoint)
	if err != nil {
		return false
	}
	return net.ParseIP(host) != nil
}
