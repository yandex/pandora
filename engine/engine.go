package engine

import (
	"golang.org/x/net/context"
	"log"
)

type Engine struct {
	cfg GlobalConfig
}

func New(cfg GlobalConfig) *Engine {
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
	for _, up := range pools {
		up.Start()
	}
	for _, up := range pools {
		<-up.done
	}

	log.Println("Done")
	return nil
}