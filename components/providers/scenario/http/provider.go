package http

import (
	"fmt"

	"github.com/spf13/afero"
	gun "github.com/yandex/pandora/components/guns/http_scenario"
	"github.com/yandex/pandora/components/providers/scenario"
	"github.com/yandex/pandora/components/providers/scenario/config"
	"github.com/yandex/pandora/core"
)

var _ core.Provider = (*scenario.Provider[*gun.Scenario])(nil)

const defaultSinkSize = 100

func NewProvider(fs afero.Fs, conf scenario.ProviderConfig) (core.Provider, error) {
	const op = "scenario_http.NewProvider"
	ammoCfg, err := config.ReadAmmoConfig(fs, conf.File)
	if err != nil {
		return nil, fmt.Errorf("%s ReadAmmoConfig %w", op, err)
	}
	vs, err := config.ExtractVariableStorage(ammoCfg)
	if err != nil {
		return nil, fmt.Errorf("%s buildVariableStorage %w", op, err)
	}

	ammos, err := decodeAmmo(ammoCfg, vs)
	if err != nil {
		return nil, fmt.Errorf("%s decodeAmmo %w", op, err)
	}

	p := &scenario.Provider[*gun.Scenario]{}
	p.SetConfig(conf)
	p.SetSink(make(chan *gun.Scenario, defaultSinkSize))
	p.SetAmmos(ammos)

	return p, nil
}
