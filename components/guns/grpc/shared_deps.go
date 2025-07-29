package grpc

import (
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/dynamic/grpcdynamic"
	"github.com/yandex/pandora/core/clientpool"
	"github.com/yandex/pandora/lib/answlog"
)

type SharedDeps struct {
	services   map[string]desc.MethodDescriptor
	clientPool *clientpool.Pool[grpcdynamic.Stub]
	answlog    *answlog.Logger
}
