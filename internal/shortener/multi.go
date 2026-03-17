package shortener

import "fmt"

type MultiProvider struct {
	bitly   *BitlyClient
	hasKey  bool
}

func NewMulti(apiKeys map[string]string) *MultiProvider {
	m := &MultiProvider{}
	
	if apiKeys != nil && apiKeys["bitly"] != "" {
		m.bitly = NewBitly(apiKeys["bitly"])
		m.hasKey = true
	}
	
	return m
}

func (m *MultiProvider) Name() string {
	if m.hasKey {
		return "bitly"
	}
	return "none"
}

func (m *MultiProvider) Shorten(longURL, custom string) (string, error) {
	if !m.hasKey {
		return "", fmt.Errorf("no API key configured")
	}
	return m.bitly.Shorten(longURL, custom)
}

func (m *MultiProvider) Update(shortURL, newLongURL string) (string, error) {
	if !m.hasKey {
		return "", fmt.Errorf("no API key configured")
	}
	return m.bitly.Update(shortURL, newLongURL)
}
