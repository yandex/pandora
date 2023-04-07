// Copyright (c) 2017 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package phttp

import (
	"net"

	"github.com/spf13/afero"
	phttp "github.com/yandex/pandora/components/guns/http"
	httpProvider "github.com/yandex/pandora/components/providers/http"
	"github.com/yandex/pandora/components/providers/http/config"
	"github.com/yandex/pandora/core"
	"github.com/yandex/pandora/core/register"
	"github.com/yandex/pandora/lib/answlog"
	"github.com/yandex/pandora/lib/netutil"
	"go.uber.org/zap"
)

func Import(fs afero.Fs) {
	register.Provider("http", func(conf config.Config) (core.Provider, error) {
		return httpProvider.NewProvider(fs, conf)
	})

	register.Provider("http/json", func(conf config.Config) (core.Provider, error) {
		conf.Decoder = config.DecoderJSONLine
		return httpProvider.NewProvider(fs, conf)
	})

	register.Provider("uri", func(conf config.Config) (core.Provider, error) {
		conf.Decoder = config.DecoderURI
		return httpProvider.NewProvider(fs, conf)
	})

	register.Provider("uripost", func(conf config.Config) (core.Provider, error) {
		conf.Decoder = config.DecoderURIPost
		return httpProvider.NewProvider(fs, conf)
	})

	register.Provider("raw", func(conf config.Config) (core.Provider, error) {
		conf.Decoder = config.DecoderRaw
		return httpProvider.NewProvider(fs, conf)
	})

	register.Gun("http", func(conf phttp.HTTPGunConfig) func() core.Gun {
		targetResolved, _ := PreResolveTargetAddr(&conf.Client, conf.Gun.Target)
		answLog := answlog.Init(conf.Gun.Base.AnswLog.Path)
		return func() core.Gun { return phttp.WrapGun(phttp.NewHTTPGun(conf, answLog, targetResolved)) }
	}, phttp.DefaultHTTPGunConfig)

	register.Gun("http2", func(conf phttp.HTTP2GunConfig) func() (core.Gun, error) {
		targetResolved, _ := PreResolveTargetAddr(&conf.Client, conf.Gun.Target)
		answLog := answlog.Init(conf.Gun.Base.AnswLog.Path)
		return func() (core.Gun, error) {
			gun, err := phttp.NewHTTP2Gun(conf, answLog, targetResolved)
			return phttp.WrapGun(gun), err
		}
	}, phttp.DefaultHTTP2GunConfig)

	register.Gun("connect", func(conf phttp.ConnectGunConfig) func() core.Gun {
		conf.Target, _ = PreResolveTargetAddr(&conf.Client, conf.Target)
		answLog := answlog.Init(conf.BaseGunConfig.AnswLog.Path)
		return func() core.Gun {
			return phttp.WrapGun(phttp.NewConnectGun(conf, answLog))
		}
	}, phttp.DefaultConnectGunConfig)
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
