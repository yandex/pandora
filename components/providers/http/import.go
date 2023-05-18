package http

import (
	"github.com/spf13/afero"
	"github.com/yandex/pandora/components/providers/http/config"
	"github.com/yandex/pandora/components/providers/http/middleware"
	headerdate "github.com/yandex/pandora/components/providers/http/middleware/headerdate"
	httpRegister "github.com/yandex/pandora/components/providers/http/register"
	"github.com/yandex/pandora/core"
	"github.com/yandex/pandora/core/register"
)

func Import(fs afero.Fs) {
	register.Provider("http", func(cfg config.Config) (core.Provider, error) {
		return NewProvider(fs, cfg)
	})

	register.Provider("http/json", func(cfg config.Config) (core.Provider, error) {
		cfg.Decoder = config.DecoderJSONLine
		return NewProvider(fs, cfg)
	})

	register.Provider("uri", func(cfg config.Config) (core.Provider, error) {
		cfg.Decoder = config.DecoderURI
		return NewProvider(fs, cfg)
	})

	register.Provider("uripost", func(cfg config.Config) (core.Provider, error) {
		cfg.Decoder = config.DecoderURIPost
		return NewProvider(fs, cfg)
	})

	register.Provider("raw", func(cfg config.Config) (core.Provider, error) {
		cfg.Decoder = config.DecoderRaw
		return NewProvider(fs, cfg)
	})

	httpRegister.HTTPMW("header/date", func(cfg headerdate.Config) (middleware.Middleware, error) {
		return headerdate.NewMiddleware(cfg)
	})

}
