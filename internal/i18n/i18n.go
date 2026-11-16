package i18n

import (
	"embed"
	"fmt"
	"path"
	"strings"
	"sync"

	"golang.org/x/text/language"
	"gopkg.in/yaml.v3"
)

const (
	LangEN = "en"
	LangES = "es"
	LangPT = "pt"
)

const DefaultLang = LangEN

var (
	currentLang     = DefaultLang
	currentLangOnce sync.Once
	langMu          sync.RWMutex
)

type TranslationStore struct {
	mu           sync.RWMutex
	translations map[string]map[string]string
}

var store = &TranslationStore{
	translations: make(map[string]map[string]string),
}

func T(key string) string {
	langMu.RLock()
	lang := currentLang
	langMu.RUnlock()
	return store.T(key, lang)
}

func TF(key string, args ...interface{}) string {
	template := T(key)
	for i, arg := range args {
		placeholder := fmt.Sprintf("{%d}", i)
		template = strings.Replace(template, placeholder, fmt.Sprintf("%v", arg), 1)
	}
	return template
}

func TLang(lang, key string) string {
	return store.T(key, lang)
}

func GetCurrentLang() string {
	langMu.RLock()
	defer langMu.RUnlock()
	return currentLang
}

func SetLanguage(lang string) {
	currentLangOnce.Do(func() {})
	store.mu.Lock()
	defer store.mu.Unlock()
	langMu.Lock()
	defer langMu.Unlock()
	if _, ok := store.translations[lang]; ok {
		currentLang = lang
	}
}

func SetLanguageWithFallback(lang string) {
	store.mu.RLock()
	_, ok := store.translations[lang]
	store.mu.RUnlock()

	langMu.Lock()
	defer langMu.Unlock()
	if ok {
		currentLang = lang
	} else {
		currentLang = DefaultLang
	}
}

func (s *TranslationStore) T(key, lang string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if translations, ok := s.translations[lang]; ok {
		if val, ok := translations[key]; ok {
			return val
		}
	}

	if lang != DefaultLang {
		if translations, ok := s.translations[DefaultLang]; ok {
			if val, ok := translations[key]; ok {
				return val
			}
		}
	}

	return key
}

func LoadTranslations(lang string, data map[string]string) {
	store.mu.Lock()
	defer store.mu.Unlock()
	store.translations[lang] = data
}

func LoadFromYAML(lang string, content []byte) error {
	store.mu.Lock()
	defer store.mu.Unlock()

	var data map[string]string
	if err := yaml.Unmarshal(content, &data); err != nil {
		return fmt.Errorf("failed to parse YAML for lang %s: %w", lang, err)
	}

	store.translations[lang] = data
	return nil
}

// LangAuto is the saved language that means "follow the system".
const LangAuto = "auto"

// UseLanguage switches to the saved language setting, resolving LangAuto (or
// an empty setting) to the system language.
func UseLanguage(configured string) {
	SetLanguageWithFallback(ResolveLanguage(configured))
}

func ResolveLanguage(configured string) string {
	if configured == "" || configured == LangAuto {
		return detectSystemLang()
	}
	return configured
}

func SupportedLanguages() []string {
	return []string{LangEN, LangES, LangPT}
}

func LanguageName(code string) string {
	switch code {
	case LangEN:
		return "English"
	case LangES:
		return "Español"
	case LangPT:
		return "Português"
	default:
		return code
	}
}

func LanguageTag(code string) string {
	switch code {
	case LangEN:
		return "en-US"
	case LangES:
		return "es-ES"
	case LangPT:
		return "pt-BR"
	default:
		return "en-US"
	}
}

func ParseAcceptLanguage(header string) string {
	if header == "" {
		return DefaultLang
	}

	parts := strings.Split(header, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if idx := strings.Index(part, ";"); idx != -1 {
			part = part[:idx]
		}

		tag, err := language.Parse(part)
		if err != nil {
			continue
		}

		base, _ := tag.Base()
		switch base.String() {
		case "en":
			return LangEN
		case "es":
			return LangES
		case "pt":
			return LangPT
		}
	}

	return DefaultLang
}

func LoadFromFS(fs embed.FS, prefix string) error {
	languages := SupportedLanguages()

	for _, lang := range languages {
		path := path.Join(prefix, lang+".yaml")
		content, err := fs.ReadFile(path)
		if err != nil {
			continue
		}

		if err := LoadFromYAML(lang, content); err != nil {
			return err
		}
	}

	return nil
}

func AddFallback(lang string) {
	store.mu.Lock()
	defer store.mu.Unlock()

	en, ok := store.translations[DefaultLang]
	if !ok {
		return
	}

	current, ok := store.translations[lang]
	if !ok {
		store.translations[lang] = en
		return
	}

	for k, v := range en {
		if _, exists := current[k]; !exists {
			current[k] = v
		}
	}
}

func TranslationsMap() map[string]map[string]string {
	store.mu.RLock()
	defer store.mu.RUnlock()

	result := make(map[string]map[string]string)
	for lang, trans := range store.translations {
		result[lang] = trans
	}
	return result
}

func GetTranslations(lang string) map[string]string {
	store.mu.RLock()
	defer store.mu.RUnlock()

	if trans, ok := store.translations[lang]; ok {
		return trans
	}
	if trans, ok := store.translations[DefaultLang]; ok {
		return trans
	}
	return nil
}

func GetCurrentTranslations() map[string]string {
	return GetTranslations(currentLang)
}

func CurrentLanguage() string {
	langMu.RLock()
	defer langMu.RUnlock()
	return currentLang
}

func ChangeLanguage(lang string) {
	store.mu.RLock()
	_, ok := store.translations[lang]
	store.mu.RUnlock()

	if ok {
		langMu.Lock()
		currentLang = lang
		langMu.Unlock()
	}
}

func AvailableLanguages() []string {
	return SupportedLanguages()
}
