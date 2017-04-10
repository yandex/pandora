// Copyright (c) 2017 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package main

import (
	"github.com/spf13/afero"

	"github.com/yandex/pandora/cli"
	"github.com/yandex/pandora/components/example"
	"github.com/yandex/pandora/components/phttp/import"
	"github.com/yandex/pandora/core"
	"github.com/yandex/pandora/core/aggregate"
	"github.com/yandex/pandora/core/aggregate/netsample"
	"github.com/yandex/pandora/core/register"
	"github.com/yandex/pandora/core/schedule"
)

func init() {
	// TODO(skipor): move all registrations to its packages.
	// TODO(skipor): make and register NewDefaultConfig funcs.

	fs := afero.NewOsFs()
	register.Aggregator("phout", func(conf netsample.PhoutConfig) (core.Aggregator, error) {
		a, err := netsample.GetPhout(fs, conf)
		return netsample.WrapAggregator(a), err
	})

	register.Limiter("periodic", schedule.NewPeriodic)
	register.Limiter("unlimited", schedule.NewUnlimited)
	register.Limiter("linear", schedule.NewLinear)

	register.Aggregator("discard", aggregate.NewDiscard)
	register.Aggregator("log", aggregate.NewLog)

	register.Provider("example", example.NewProvider, example.NewDefaultProviderConfig)
	register.Gun("example", example.NewGun)

	// Components should not write anything to files.
	readOnlyFs := afero.NewReadOnlyFs(fs)

	phttp.Import(readOnlyFs)
}

func main() {
	cli.Run()
}
