package grpc

import (
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/dynamic/grpcdynamic"
	"github.com/yandex/pandora/core/clientpool"
)

type SharedDeps struct {
	services   map[string]desc.MethodDescriptor
	clientPool *clientpool.Pool[grpcdynamic.Stub]
}
