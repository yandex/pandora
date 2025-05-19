package answ

import (
	"github.com/yandex/pandora/core/register"
	"github.com/yandex/pandora/lib/answlog"
)

func Import() {
	register.Plugin[answlog.LogMasker]("pass", func() *answlog.PassAllMasker {
		return &answlog.PassAllMasker{}
	})
}
