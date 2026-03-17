package shortener

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	isgdAPILong   = "https://is.gd/create.php?format=simple&url=%s"
	isgdAPICustom = "https://is.gd/create.php?format=simple&url=%s&shorturl=%s"
)

type ISGDClient struct {
	client *http.Client
}

type ShortenError struct {
	Reason  string
	Message string
}

func (e ShortenError) Error() string {
	return fmt.Sprintf("is.gd: %s - %s", e.Reason, e.Message)
}

func NewISGD() *ISGDClient {
	return &ISGDClient{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *ISGDClient) Shorten(longURL, custom string) (string, error) {
	encodedURL := url.QueryEscape(longURL)

	var apiURL string
	if custom != "" {
		custom = cleanCustomURL(custom)
		apiURL = fmt.Sprintf(isgdAPICustom, encodedURL, custom)
	} else {
		apiURL = fmt.Sprintf(isgdAPILong, encodedURL)
	}

	resp, err := c.client.Get(apiURL)
	if err != nil {
		return "", fmt.Errorf("failed to contact is.gd: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	shortURL := strings.TrimSpace(string(body))

	if strings.Contains(shortURL, "Error: ") {
		errMsg := strings.TrimPrefix(shortURL, "Error: ")

		if strings.Contains(errMsg, "blacklist") {
			return "", ShortenError{
				Reason:  "DOMAIN_BLOCKED",
				Message: "This tunnel domain is blocked by is.gd (try playit.gg instead)",
			}
		}

		if strings.Contains(errMsg, "already taken") || strings.Contains(errMsg, "short URL") {
			if custom != "" {
				return c.Shorten(longURL, "")
			}
			return "", ShortenError{
				Reason:  "CUSTOM_TAKEN",
				Message: "Custom URL already taken",
			}
		}

		return "", ShortenError{
			Reason:  "API_ERROR",
			Message: errMsg,
		}
	}

	if !strings.HasPrefix(shortURL, "http") {
		return "", fmt.Errorf("unexpected response: %s", shortURL)
	}

	return shortURL, nil
}

func (c *ISGDClient) Update(custom, newLongURL string) (string, error) {
	return c.Shorten(newLongURL, custom)
}

func cleanCustomURL(s string) string {
	s = strings.TrimPrefix(s, "is.gd/")
	s = strings.TrimPrefix(s, "https://is.gd/")
	s = strings.TrimPrefix(s, "http://is.gd/")

	var result strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			result.WriteRune(r)
		}
	}

	return strings.ToLower(result.String())
}

func IsDomainBlocked(err error) bool {
	if shortenErr, ok := err.(ShortenError); ok {
		return shortenErr.Reason == "DOMAIN_BLOCKED"
	}
	return false
}
