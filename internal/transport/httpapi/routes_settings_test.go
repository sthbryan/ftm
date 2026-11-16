package httpapi

import "testing"

func TestIsSelectableLanguage(t *testing.T) {
	allowed := []string{"en", "es", "auto"}
	for _, lang := range allowed {
		if !isSelectableLanguage(lang) {
			t.Errorf("isSelectableLanguage(%q) = false, want true", lang)
		}
	}

	denied := []string{"", "klingon", "EN", "es-ES", "auto ", "null"}
	for _, lang := range denied {
		if isSelectableLanguage(lang) {
			t.Errorf("isSelectableLanguage(%q) = true, want false", lang)
		}
	}
}
