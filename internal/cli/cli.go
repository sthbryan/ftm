package cli

import (
	"github.com/sthbryan/ftm/internal/core"
	"github.com/sthbryan/ftm/internal/i18n"
)

// Init loads the translations and switches to the saved language. peek reads
// the config without writing it, since another ftm may own it; when it fails
// the system language is used.
func Init(peek func() (core.Config, error)) error {
	if err := i18n.Load(); err != nil {
		return err
	}

	cfg, err := peek()
	if err != nil {
		i18n.UseLanguage("")
		return nil
	}

	i18n.UseLanguage(cfg.Language)

	return nil
}
