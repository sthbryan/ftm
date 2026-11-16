package i18n

import "embed"

//go:embed locales/*.yaml
var localeFS embed.FS

func Load() error {
	return LoadFromFS(localeFS, "locales")
}
