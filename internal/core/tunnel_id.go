package core

import (
	"fmt"
	"strings"
	"unicode"
)

const fallbackTunnelID = "tunnel"

func slugify(name string) string {
	var b strings.Builder

	lastDash := true
	for _, r := range strings.ToLower(strings.TrimSpace(name)) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
			lastDash = false
		case !lastDash:
			b.WriteRune('-')
			lastDash = true
		}
	}

	return strings.Trim(b.String(), "-")
}

func (c *Config) NewTunnelID(name string) string {
	base := slugify(name)
	if base == "" {
		base = fallbackTunnelID
	}

	if c.GetTunnel(base) == nil {
		return base
	}

	for suffix := 2; ; suffix++ {
		candidate := fmt.Sprintf("%s-%d", base, suffix)
		if c.GetTunnel(candidate) == nil {
			return candidate
		}
	}
}

func (c *Config) dedupeTunnelIDs() bool {
	seen := make(map[string]bool, len(c.Tunnels))
	changed := false

	for i := range c.Tunnels {
		id := c.Tunnels[i].ID

		if id != "" && !seen[id] {
			seen[id] = true
			continue
		}

		base := slugify(c.Tunnels[i].Name)
		if base == "" {
			base = fallbackTunnelID
		}

		replacement := base
		for suffix := 2; seen[replacement] || replacement == ""; suffix++ {
			replacement = fmt.Sprintf("%s-%d", base, suffix)
		}

		c.Tunnels[i].ID = replacement
		seen[replacement] = true
		changed = true
	}

	return changed
}
