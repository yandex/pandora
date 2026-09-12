package grpcjson

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yandex/pandora/components/providers/grpc/ammo"
)

// Патроны берутся из пула и переиспользуются. Если строка не разобралась, объект нельзя отдавать
// с полями предыдущего патрона: вызывающий отфильтрует его по чужому тегу, а gRPC-пушка выстрелит
// чужим payload и запишет это в отчёт успехом (LOAD-3696).
func TestDecodeAmmoClearsPreviousAmmoOnError(t *testing.T) {
	reused := &ammo.Ammo{}
	reused.Reset("prev-tag", "prev.Call", map[string]string{"k": "v"}, map[string]interface{}{"id": 1})

	got, err := decodeAmmo([]byte("{не json"), reused)

	require.Error(t, err)
	assert.Empty(t, got.Tag, "чужой тег не должен пережить ошибку разбора")
	assert.Empty(t, got.Call, "чужой вызов не должен пережить ошибку разбора")
	assert.Empty(t, got.Payload, "чужой payload не должен пережить ошибку разбора")
}

func TestDecodeAmmoFillsFieldsOnSuccess(t *testing.T) {
	reused := &ammo.Ammo{}
	reused.Reset("prev-tag", "prev.Call", nil, nil)

	got, err := decodeAmmo([]byte(`{"tag":"t","call":"pkg.Service/Method","payload":{"id":7}}`), reused)

	require.NoError(t, err)
	assert.Equal(t, "t", got.Tag)
	assert.Equal(t, "pkg.Service/Method", got.Call)
	assert.Equal(t, float64(7), got.Payload["id"])
	assert.True(t, got.IsValid(), "успешно разобранный патрон валиден")
}
