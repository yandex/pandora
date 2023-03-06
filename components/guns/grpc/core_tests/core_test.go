package core

import (
	"testing"

	"github.com/yandex/pandora/components/guns/grpc"
	"github.com/yandex/pandora/core/warmup"
)

func TestGrpcGunImplementsWarmedUp(t *testing.T) {
	_ = warmup.WarmedUp(&grpc.Gun{})
}
