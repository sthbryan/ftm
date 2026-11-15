package i18n

import (
	"regexp"
	"sort"
	"strings"
	"testing"
)

var placeholder = regexp.MustCompile(`\{[a-z0-9_]+\}`)

func placeholders(s string) string {
	found := placeholder.FindAllString(s, -1)
	sort.Strings(found)
	return strings.Join(found, " ")
}

// A translation that drops or renames a placeholder shows a raw {name} or
// loses the value, so every locale must carry the same set as English.
func TestPlaceholdersMatchAcrossLocales(t *testing.T) {
	loadEmbedded(t)
	en := GetTranslations(DefaultLang)

	for _, lang := range SupportedLanguages() {
		if lang == DefaultLang {
			continue
		}
		for key, text := range GetTranslations(lang) {
			if want, got := placeholders(en[key]), placeholders(text); want != got {
				t.Errorf("%s.yaml %s has placeholders [%s], en.yaml has [%s]", lang, key, got, want)
			}
		}
	}
}

// The mechanical rules from docs/voice.md.
func TestCopyFollowsVoiceGuide(t *testing.T) {
	loadEmbedded(t)

	banned := []struct {
		pattern *regexp.Regexp
		why     string
	}{
		{regexp.MustCompile(`[!¡]`), "no exclamation marks"},
		{regexp.MustCompile(`[^.]\.\.\.$|[^.]\.\.\.[^.\]]`), "use … for work in progress, not ..."},
		{regexp.MustCompile(`%[sdv]`), "use {0}-style placeholders; TF does not read printf verbs"},
		{regexp.MustCompile(`(?i)\b(oops|whoops|uh-oh|successfully|seamless(ly)?|effortless(ly)?|simply|just|awesome|amazing|powerful|magic(al)?|unleash|supercharge|exitosamente|com sucesso|incre[ií]ble|incr[ií]vel)\b`), "no filler or hype words"},
		{regexp.MustCompile(`(?i)\b(binar(y|io|ies|ios)|bin[aá]rio|TUI|endpoint|localhost(:| |$)|stdout|stderr|exit code|null|undefined)\b`), "no internal plumbing"},
		{regexp.MustCompile(`[\x{1F300}-\x{1FAFF}\x{2600}-\x{26FF}\x{2728}]`), "no emoji"},
	}

	for _, lang := range SupportedLanguages() {
		for key, text := range GetTranslations(lang) {
			for _, rule := range banned {
				if rule.pattern.MatchString(text) {
					t.Errorf("%s.yaml %s = %q: %s", lang, key, text, rule.why)
				}
			}
		}
	}
}
