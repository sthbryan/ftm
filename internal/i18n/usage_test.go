package i18n

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// Keys built at runtime from a prefix and a value. Each family lists every
// value the code can produce, so the test can check the full keys. A template
// key with a prefix missing here fails the test: add the family when you add
// one to the code.
var dynamicKeyFamilies = map[string][]string{
	"foundry_": {"answers", "other", "silent", "unknown"}, // web/src/lib/ritual.ts FoundryCheck
	// ConnectionPage provider cards; the ids /api/providers returns.
	"provider_note_": {"cloudflared", "tunnelmole", "localhostrun", "serveo", "pinggy", "bore"},
	// ConnectionPage provider table, PROVIDER_FACTS limit.
	"provider_limit_": {"none", "60"},
	"lang_":           {LangEN, LangES, LangPT}, // SettingsPage language picker
	// SettingsPage provider rows; process.BinarySource ("none" is skipped there).
	"providers_source_": {"bundled", "path", "system", "none"},
}

var (
	goKeyPatterns = []*regexp.Regexp{
		regexp.MustCompile(`\b(?:T|TF|Tr)\(\s*"([a-z0-9_]+)"`),
		// Key pairs in table literals, e.g. the TUI welcome steps.
		regexp.MustCompile(`\{"([a-z0-9_]+)", "([a-z0-9_]+)"\}`),
	}

	tsKeyPatterns = []*regexp.Regexp{
		regexp.MustCompile(`\bt\(\s*['"]([a-z0-9_]+)['"]`),
		regexp.MustCompile(`\bt\(\s*isEdit \? '([a-z0-9_]+)' : '([a-z0-9_]+)'`),
		regexp.MustCompile(`textKey: '([a-z0-9_]+)'`),
		regexp.MustCompile(`'(error_hint_[a-z0-9_]+)'`),
		regexp.MustCompile(`copyWithToast\([^,]+,\s*'([a-z0-9_]+)'\)`),
	}
	tsTemplateKey = regexp.MustCompile("\\bt\\(\\s*`([a-z0-9_]+)\\$\\{")
)

// usedKeys scans the Go sources and web/src for translation keys.
func usedKeys(t *testing.T) map[string][]string {
	t.Helper()
	root := filepath.Join("..", "..")
	used := map[string][]string{}
	add := func(key, where string) { used[key] = append(used[key], where) }

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		rel = filepath.ToSlash(rel)
		if d.IsDir() {
			switch d.Name() {
			case ".git", "node_modules", "dist", "build", "vendor":
				return filepath.SkipDir
			}
			// Developer tools (tools/openapi, tools/fakefoundry) are not
			// translated; their strings are not locale keys.
			if rel == "tools" {
				return filepath.SkipDir
			}
			return nil
		}

		isGo := strings.HasSuffix(rel, ".go") && !strings.HasSuffix(rel, "_test.go")
		isTS := strings.HasPrefix(rel, "web/src/") &&
			(strings.HasSuffix(rel, ".ts") || strings.HasSuffix(rel, ".tsx")) &&
			!strings.Contains(rel, ".test.")
		if !isGo && !isTS {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		src := string(data)

		patterns := tsKeyPatterns
		if isGo {
			patterns = goKeyPatterns
		}
		for _, re := range patterns {
			for _, m := range re.FindAllStringSubmatch(src, -1) {
				for _, key := range m[1:] {
					add(key, rel)
				}
			}
		}
		if isGo {
			return nil
		}
		for _, m := range tsTemplateKey.FindAllStringSubmatch(src, -1) {
			values, ok := dynamicKeyFamilies[m[1]]
			if !ok {
				t.Errorf("%s builds keys from prefix %q; list its values in dynamicKeyFamilies", rel, m[1])
				continue
			}
			for _, v := range values {
				add(m[1]+v, rel)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("scanning sources: %v", err)
	}
	if len(used) == 0 {
		t.Fatal("found no translation keys in the sources")
	}
	return used
}

func loadEmbedded(t *testing.T) {
	t.Helper()
	seed(t, map[string]map[string]string{}, DefaultLang)
	if err := Load(); err != nil {
		t.Fatalf("Load() failed: %v", err)
	}
}

func TestEveryUsedKeyExistsInEveryLocale(t *testing.T) {
	used := usedKeys(t)
	loadEmbedded(t)

	for _, lang := range SupportedLanguages() {
		catalogue := GetTranslations(lang)
		var missing []string
		for key, where := range used {
			if _, ok := catalogue[key]; !ok {
				missing = append(missing, key+" (used in "+where[0]+")")
			}
		}
		sort.Strings(missing)
		for _, m := range missing {
			t.Errorf("%s.yaml is missing %s", lang, m)
		}
	}
}

func TestEveryLocaleKeyIsUsed(t *testing.T) {
	used := usedKeys(t)
	loadEmbedded(t)

	var unused []string
	for key := range GetTranslations(DefaultLang) {
		if _, ok := used[key]; !ok {
			unused = append(unused, key)
		}
	}
	sort.Strings(unused)
	for _, key := range unused {
		t.Errorf("%q is in the locales but nothing uses it; remove it", key)
	}
}
