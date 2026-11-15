package cli

import (
	"github.com/sthbryan/ftm/internal/i18n"
	"github.com/sthbryan/ftm/internal/store"
)

//

func Init() error {
	if err := i18n.Load(); err != nil {
		return err
	}

	// Only the language is needed here, and another ftm may own the config,
	// so read it without writing.
	cfg, err := store.Peek(store.DefaultDirs())
	if err != nil {

		i18n.SetLanguageWithFallback(i18n.ResolveLanguage(""))
		return nil
	}

	i18n.InitFromConfig(&cfg)

	return nil
}
