package grpc

import (
	"context"
	"errors"
	"testing"

	"github.com/jhump/protoreflect/desc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yandex/pandora/components/answ/sampler"
	"github.com/yandex/pandora/components/providers/grpc/ammo"
	"github.com/yandex/pandora/core"
	"github.com/yandex/pandora/core/aggregator/netsample"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func Test_replacePort(t *testing.T) {
	tests := []struct {
		name string
		host string
		port int64
		want string
	}{
		{
			name: "zero port",
			host: "[2a02:6b8:c02:901:0:fc5f:9a6c:4]:8888",
			port: 0,
			want: "[2a02:6b8:c02:901:0:fc5f:9a6c:4]:8888",
		},
		{
			name: "replace ipv6",
			host: "[2a02:6b8:c02:901:0:fc5f:9a6c:4]:8888",
			port: 9999,
			want: "[2a02:6b8:c02:901:0:fc5f:9a6c:4]:9999",
		},
		{
			name: "add port to ipv6",
			host: "[2a02:6b8:c02:901:0:fc5f:9a6c:4]",
			port: 9999,
			want: "[2a02:6b8:c02:901:0:fc5f:9a6c:4]:9999",
		},
		{
			name: "replace ipv4",
			host: "127.0.0.1:8888",
			port: 9999,
			want: "127.0.0.1:9999",
		},
		{
			name: "replace host",
			host: "localhost:8888",
			port: 9999,
			want: "localhost:9999",
		},
		{
			name: "add port",
			host: "localhost",
			port: 9999,
			want: "localhost:9999",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := replacePort(tt.host, tt.port)
			require.Equal(t, tt.want, got)
		})
	}
}

// Числа из этой функции уезжают в отчёт стрельбы как HTTP-подобные статусы: по ним считается доля
// успешных запросов и раскладка ошибок. Сдвиг любого кода тихо испортит вердикт нагрузочного
// теста, а не уронит прогон, поэтому все ветки перечислены поимённо.
func TestConvertGrpcStatus(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{"ok", status.Error(codes.OK, ""), 200},
		{"canceled", status.Error(codes.Canceled, ""), 499},
		{"invalid argument", status.Error(codes.InvalidArgument, ""), 400},
		{"deadline exceeded", status.Error(codes.DeadlineExceeded, ""), 504},
		{"not found", status.Error(codes.NotFound, ""), 404},
		{"already exists", status.Error(codes.AlreadyExists, ""), 409},
		{"permission denied", status.Error(codes.PermissionDenied, ""), 403},
		{"resource exhausted", status.Error(codes.ResourceExhausted, ""), 429},
		{"failed precondition", status.Error(codes.FailedPrecondition, ""), 400},
		{"aborted", status.Error(codes.Aborted, ""), 409},
		{"out of range", status.Error(codes.OutOfRange, ""), 400},
		{"unimplemented", status.Error(codes.Unimplemented, ""), 501},
		{"unavailable", status.Error(codes.Unavailable, ""), 503},
		{"unauthenticated", status.Error(codes.Unauthenticated, ""), 401},
		{"internal", status.Error(codes.Internal, ""), 500},
		{"data loss", status.Error(codes.DataLoss, ""), 500},
		{"unknown", status.Error(codes.Unknown, ""), 500},
		// Не-gRPC ошибка приходит с сетевого слоя (оборванное соединение, отказ резолва) —
		// она обязана попасть в 500, а не притвориться успехом.
		{"негрпц ошибка", errors.New("connection refused"), 500},
		{"ошибки нет", nil, 200},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, ConvertGrpcStatus(tt.err))
		})
	}
}

// Сэмплирование ответов включено по умолчанию с фактором 10: выключенный фактор залил бы диск
// полным логом ответов на боевой стрельбе, а нулевой — отключил бы запись вообще.
func TestDefaultGunConfig(t *testing.T) {
	conf := DefaultGunConfig()

	require.False(t, conf.AnswLog.Enabled, "лог ответов по умолчанию выключен")
	require.True(t, conf.AnswLog.Sampling.Enabled, "сэмплирование по умолчанию включено")
	require.NotNil(t, conf.AnswLog.Masking, "маскировщик обязан быть задан")

	pattern, ok := conf.AnswLog.Sampling.Pattern.(sampler.FactorPattern)
	require.True(t, ok, "паттерн сэмплирования должен быть факторным")
	require.Equal(t, 10, pattern.Factor)
}

// collectAggregator собирает отчёты вместо настоящего агрегатора: тесты ниже смотрят, ЧТО
// уехало в отчёт по несостоявшемуся выстрелу.
type collectAggregator struct {
	samples []core.Sample
}

func (a *collectAggregator) Run(context.Context, core.AggregatorDeps) error { return nil }

func (a *collectAggregator) Report(s core.Sample) { a.samples = append(a.samples, s) }

func invalidAmmoGun(aggr core.Aggregator) *Gun {
	return &Gun{
		Aggr:    aggr,
		GunDeps: core.GunDeps{Log: zap.NewNop()},
		// Services пуст намеренно: если бы выстрел всё же начался, он свалился бы на
		// «invalid ammo.Call» и тоже не дошёл до сети. Различаем случаи по тегу в отчёте.
		Services: map[string]desc.MethodDescriptor{},
	}
}

// Патрон, который не разобрался, приходит в пушку с полями ПРЕДЫДУЩЕГО патрона: провайдер берёт
// объект из пула и при ошибке только помечает его Invalidate(), не очищая. HTTP-пушка этот флаг
// проверяет давно (guns/http/base.go), gRPC — не проверяла, поэтому в цель уходил повтор прошлого
// запроса и писался в отчёт как обычный выстрел (LOAD-3696).
func TestShootSkipsInvalidAmmo(t *testing.T) {
	aggr := &collectAggregator{}
	gun := invalidAmmoGun(aggr)

	am := &ammo.Ammo{}
	am.Reset("prev-tag", "pkg.Service/Method", nil, map[string]interface{}{"id": 1})
	am.Invalidate()

	gun.shoot(am)

	require.Len(t, aggr.samples, 1, "несостоявшийся выстрел всё равно попадает в отчёт")
	reported := aggr.samples[0].(*netsample.Sample)
	assert.Contains(t, reported.Tags(), EmptyTag, "в отчёт идёт пустой тег, а не тег прошлого патрона")
	assert.Equal(t, 0, reported.ProtoCode(), "несостоявшийся выстрел не получает код ответа")
}

// Обратный случай: валидный патрон отчитывается своим тегом — фикс выше не глушит обычные выстрелы.
func TestShootKeepsTagForValidAmmo(t *testing.T) {
	aggr := &collectAggregator{}
	gun := invalidAmmoGun(aggr)

	am := &ammo.Ammo{}
	am.Reset("my-tag", "pkg.Service/Unknown", nil, nil)

	gun.shoot(am)

	require.Len(t, aggr.samples, 1)
	reported := aggr.samples[0].(*netsample.Sample)
	assert.Contains(t, reported.Tags(), "my-tag")
	assert.NotContains(t, reported.Tags(), EmptyTag)
}
