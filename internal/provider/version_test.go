package provider

import "testing"

func TestParseVersion(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   string
	}{
		{"cloudflared", "cloudflared version 2025.8.1 (built 2025-08-13-1219 UTC)\n", "2025.8.1"},
		{"cloudflared old", "cloudflared version 2024.6.1 (built 2024-06-12-0905 UTC)", "2024.6.1"},
		{"bore", "bore-cli 0.6.0\n", "0.6.0"},
		{"pinggy", "Pinggy CLI version: 0.6.7\n", "0.6.7"},
		{"tmole", "2.4.0\n", "2.4.0"},
		{"v prefix", "tool v1.2.3", "1.2.3"},
		{"two segments", "tool 1.2", "1.2"},
		{"ansi", "\x1b[1mbore-cli 0.5.3\x1b[0m", "0.5.3"},
		{"banner on a later line", "Welcome\nPinggy CLI version: 0.6.7", "0.6.7"},
		{"no version", "usage: tool [flags]", ""},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ParseVersion(tt.output); got != tt.want {
				t.Errorf("ParseVersion(%q) = %q, want %q", tt.output, got, tt.want)
			}
		})
	}
}

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"2025.8.1", "2025.8.1", 0},
		{"2026.11.0", "2025.8.1", 1},
		{"2025.8.1", "2025.10.0", -1}, // numeric, not lexical
		{"2025.10.0", "2025.9.2", 1},
		{"0.6.0", "0.5.3", 1},
		{"v0.6.0", "0.6.0", 0},
		{"1.2", "1.2.0", 0},
		{"1.2.1", "1.2", 1},
		{"1.0.0-rc1", "1.0.0", 0},
		{"2.4.0", "2.3.9", 1},
		{"", "", 0},
	}

	for _, tt := range tests {
		if got := CompareVersions(tt.a, tt.b); got != tt.want {
			t.Errorf("CompareVersions(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
		if got := CompareVersions(tt.b, tt.a); got != -tt.want {
			t.Errorf("CompareVersions(%q, %q) = %d, want %d", tt.b, tt.a, got, -tt.want)
		}
	}
}

func TestIsNewer(t *testing.T) {
	tests := []struct {
		latest, installed string
		want              bool
	}{
		{"2026.11.0", "2024.6.1", true},
		{"2024.6.1", "2024.6.1", false},
		{"0.6.0", "0.6.1", false},
		{"", "1.0.0", false},
		{"1.0.0", "", false},
	}

	for _, tt := range tests {
		if got := IsNewer(tt.latest, tt.installed); got != tt.want {
			t.Errorf("IsNewer(%q, %q) = %v, want %v", tt.latest, tt.installed, got, tt.want)
		}
	}
}
