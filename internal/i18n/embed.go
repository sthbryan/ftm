package i18n

import (
	"embed"
	"os"
)

//go:embed locales/*.yaml
var localeFS embed.FS

func Load() error {
	return LoadFromFS(localeFS, "locales")
}

func LoadExtra(lang, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return LoadFromYAML(lang, data)
}
