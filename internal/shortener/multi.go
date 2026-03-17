package shortener

type MultiProvider struct {
	providers []Provider
	apiKeys   map[string]string
}

func NewMulti(apiKeys map[string]string) *MultiProvider {
	m := &MultiProvider{
		providers: []Provider{
			NewCleanURI(),
			NewTinyURL(),
		},
		apiKeys: apiKeys,
	}
	
	// Add Bitly if API key exists
	if apiKeys != nil && apiKeys["bitly"] != "" {
		m.providers = append([]Provider{NewBitly(apiKeys["bitly"])}, m.providers...)
	}
	
	return m
}

func (m *MultiProvider) Name() string {
	return "multi"
}

func (m *MultiProvider) Shorten(longURL, custom string) (string, error) {
	var lastErr error
	
	for _, p := range m.providers {
		shortURL, err := p.Shorten(longURL, custom)
		if err == nil {
			return shortURL, nil
		}
		
		lastErr = err
		
		if !IsDomainBlocked(err) {
			return "", err
		}
	}
	
	return "", lastErr
}
