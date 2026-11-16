package provider

import (
	"net/url"
	"regexp"
	"strings"
)

var (
	ansiEscape = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)
	urlToken   = regexp.MustCompile("(?i)https?://[^\\s\"'`<>,\\\\]+")
)

func StripANSI(s string) string {
	return ansiEscape.ReplaceAllString(s, "")
}

func ExtractURL(line string, allow func(host string) bool) string {
	for _, token := range urlToken.FindAllString(StripANSI(line), -1) {
		token = strings.TrimRight(token, ".;:!?)]}>")

		parsed, err := url.Parse(token)
		if err != nil {
			continue
		}

		host := strings.ToLower(parsed.Hostname())
		if host != "" && allow(host) {
			return token
		}
	}

	return ""
}

func IsSubdomainOf(host string, domains ...string) bool {
	for _, domain := range domains {
		if strings.HasSuffix(host, "."+domain) {
			return true
		}
	}
	return false
}
