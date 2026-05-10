package i18n

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/sthbryan/ftm/internal/config"
	"golang.org/x/text/language"
	"gopkg.in/yaml.v3"
)

const (
	LangEN = "en"
	LangES = "es"
)

const DefaultLang = LangEN

var (
	currentLang     = DefaultLang
	currentLangOnce sync.Once
)

type TranslationStore struct {
	mu           sync.RWMutex
	translations map[string]map[string]string
}

var store = &TranslationStore{
	translations: make(map[string]map[string]string),
}

func T(key string) string {
	return store.T(key, currentLang)
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
	return currentLang
}

func SetLanguage(lang string) {
	currentLangOnce.Do(func() {})
	store.mu.Lock()
	defer store.mu.Unlock()
	if _, ok := store.translations[lang]; ok {
		currentLang = lang
	}
}

func SetLanguageWithFallback(lang string) {
	store.mu.RLock()
	_, ok := store.translations[lang]
	store.mu.RUnlock()

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

func InitFromConfig(cfg *config.Config) {

	systemLang := detectSystemLang()

	if cfg.Language != "" {
		SetLanguageWithFallback(cfg.Language)
		return
	}

	SetLanguageWithFallback(systemLang)
}

func detectSystemLang() string {
	lang := os.Getenv("LANG")
	if lang == "" {
		return DefaultLang
	}

	tag, err := language.Parse(lang)
	if err != nil {
		return DefaultLang
	}

	base, _ := tag.Base()
	switch base.String() {
	case "en":
		return LangEN
	case "es":
		return LangES
	default:
		return DefaultLang
	}
}

func SupportedLanguages() []string {
	return []string{LangEN, LangES}
}

func LanguageName(code string) string {
	switch code {
	case LangEN:
		return "English"
	case LangES:
		return "Español"
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
		}
	}

	return DefaultLang
}

func LoadFromFS(fs embed.FS, prefix string) error {
	languages := SupportedLanguages()

	for _, lang := range languages {
		path := filepath.Join(prefix, lang+".yaml")
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
	return currentLang
}

func ChangeLanguage(lang string) {
	store.mu.RLock()
	_, ok := store.translations[lang]
	store.mu.RUnlock()

	if ok {
		currentLang = lang
	}
}

func AvailableLanguages() []string {
	return SupportedLanguages()
}
