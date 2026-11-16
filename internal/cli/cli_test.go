package cli

import (
	"errors"
	"testing"

	"github.com/sthbryan/ftm/internal/core"
	"github.com/sthbryan/ftm/internal/i18n"
)

func peekLanguage(lang string) func() (core.Config, error) {
	return func() (core.Config, error) {
		cfg := core.DefaultConfig()
		cfg.Language = lang
		return *cfg, nil
	}
}

func TestInitAppliesConfiguredLanguage(t *testing.T) {
	if err := Init(peekLanguage(i18n.LangES)); err != nil {
		t.Fatalf("Init() failed: %v", err)
	}

	if got := i18n.CurrentLanguage(); got != i18n.LangES {
		t.Fatalf("CurrentLanguage() = %q, want %q", got, i18n.LangES)
	}
}

func TestInitFallsBackWhenConfigIsUnreadable(t *testing.T) {
	broken := func() (core.Config, error) { return core.Config{}, errors.New("bad yaml") }

	if err := Init(broken); err != nil {
		t.Fatalf("Init() returned an error for a broken config: %v, want a fallback", err)
	}

	supported := false
	for _, lang := range i18n.SupportedLanguages() {
		if i18n.CurrentLanguage() == lang {
			supported = true
		}
	}
	if !supported {
		t.Fatalf("CurrentLanguage() = %q, want a supported language", i18n.CurrentLanguage())
	}
}

func TestInitLoadsTranslations(t *testing.T) {
	if err := Init(peekLanguage(core.LanguageAuto)); err != nil {
		t.Fatalf("Init() failed: %v", err)
	}

	if got := i18n.T("connections"); got == "connections" {
		t.Fatal("translations were not loaded: T(connections) echoed the key")
	}
}
