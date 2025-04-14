package filter

import (
	"slices"

	"github.com/yandex/pandora/components/answ"
	"github.com/yandex/pandora/lib/answlog"
	"go.uber.org/zap/zapcore"
)

func NewHTTPStatusCodeFilter(filter string) answlog.LogFilter {
	if filter == "" {
		return &answlog.PassAllFilter{}
	}
	return &httpStatusCodeFilter{filter: filter}
}

func NewGRPCStatusCodeFilter(filter string) answlog.LogFilter {
	if filter == "" {
		return &answlog.PassAllFilter{}
	}
	return &grpcStatusCodeFilter{filter: filter}
}

type httpStatusCodeFilter struct {
	filter string
}

func (f *httpStatusCodeFilter) FilterAnsw(fields []zapcore.Field) bool {
	field, ok := answ.GetField(fields, answlog.FilterAndSampleGroup)
	if !ok {
		return true
	}

	answCode := field.Integer

	switch f.filter {
	case FilterAll:
		return true
	case FilterWarning:
		return answCode >= 400
	case FilterError:
		return answCode >= 500
	default:
		return true
	}
}

type grpcStatusCodeFilter struct {
	filter string
}

func (f *grpcStatusCodeFilter) FilterAnsw(fields []zapcore.Field) bool {
	field, ok := answ.GetField(fields, answlog.FilterAndSampleGroup)
	if !ok {
		return true
	}

	answCode := field.Integer

	switch f.filter {
	case FilterAll:
		return true
	case FilterWarning:
		return answCode != 0
	case FilterError:
		return slices.Contains(errorGrpcCodes, answCode)
	default:
		return true
	}
}
