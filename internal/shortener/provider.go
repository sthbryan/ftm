package shortener

type Provider interface {
	Name() string
	Shorten(longURL, custom string) (string, error)
}

type UpdateableProvider interface {
	Provider
	Update(shortURL, newLongURL string) (string, error)
}

type NoOpProvider struct{}

func (n *NoOpProvider) Name() string { return "none" }
func (n *NoOpProvider) Shorten(_, _ string) (string, error) {
	return "", nil
}

type ShortenError struct {
	Reason  string
	Message string
}

func (e ShortenError) Error() string {
	return "shortener: " + e.Reason + " - " + e.Message
}

func IsDomainBlocked(err error) bool {
	if se, ok := err.(ShortenError); ok {
		return se.Reason == "DOMAIN_BLOCKED"
	}
	return false
}

func IsAlreadyExists(err error) bool {
	if se, ok := err.(ShortenError); ok {
		return se.Reason == "ALREADY_EXISTS"
	}
	return false
}
