// Copyright (c) 2017 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package core

import (
	"github.com/spf13/afero"
	"github.com/yandex/pandora/core"
	"github.com/yandex/pandora/core/aggregate"
	"github.com/yandex/pandora/core/aggregate/netsample"
	"github.com/yandex/pandora/core/register"
	"github.com/yandex/pandora/core/schedule"
)

func Import(fs afero.Fs) {
	register.Aggregator("phout", func(conf netsample.PhoutConfig) (core.Aggregator, error) {
		a, err := netsample.GetPhout(fs, conf)
		return netsample.WrapAggregator(a), err
	})

	register.Aggregator("discard", aggregate.NewDiscard)
	register.Aggregator("log", aggregate.NewLog)

	register.Limiter("periodic", schedule.NewPeriodic)
	register.Limiter("unlimited", schedule.NewUnlimited)
	register.Limiter("linear", schedule.NewLinear)

}
