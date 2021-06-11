package provider

import (
	"testing"

	"a.yandex-team.ru/load/projects/pandora/core"
	"a.yandex-team.ru/load/projects/pandora/lib/ginkgoutil"
)

func TestProvider(t *testing.T) {
	ginkgoutil.RunSuite(t, "AmmoQueue Suite")
}

func testDeps() core.ProviderDeps {
	return core.ProviderDeps{Log: ginkgoutil.NewLogger()}
}
