package engine

import (
	"context"
	"log"

	"github.com/yandex/pandora/config"
	"github.com/yandex/pandora/utils"
)

type Engine struct {
	cfg config.Global
}

func New(cfg config.Global) *Engine {
	return &Engine{cfg}
}

func (e *Engine) Serve(ctx context.Context) error {
	pools := make([]*UserPool, 0, len(e.cfg.Pools))
	for _, upc := range e.cfg.Pools {
		up, err := NewUserPoolFromConfig(&upc)
		if err != nil {
			log.Printf("Could not create user pool: %s", err)
			continue
		}
		pools = append(pools, up)
	}
	promises := utils.Promises{}
	for _, up := range pools {
		promises = append(promises, utils.PromiseCtx(ctx, up.Start))
	}
	select {
	case err := <-promises.All():
		if err != nil {
			return err
		}
	case <-ctx.Done():
	}
	log.Println("Done")
	return nil
}
