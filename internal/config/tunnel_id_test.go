package config

import (
	"testing"
)

func TestSlugify(t *testing.T) {
	tests := map[string]string{
		"Curse of Strahd":  "curse-of-strahd",
		"Curse of Strahd!": "curse-of-strahd",
		"  spaced  out  ":  "spaced-out",
		"Sesión uno":       "sesión-uno",
		"a/b?c#d":          "a-b-c-d",
		"Port 30000":       "port-30000",
		"---":              "",
		"":                 "",
	}

	for input, want := range tests {
		if got := slugify(input); got != want {
			t.Errorf("slugify(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestNewTunnelIDIsUniquePerName(t *testing.T) {
	cfg := DefaultConfig()

	first := cfg.NewTunnelID("Curse of Strahd")
	cfg.AddTunnel(TunnelConfig{ID: first, Name: "Curse of Strahd"})

	second := cfg.NewTunnelID("Curse of Strahd")
	cfg.AddTunnel(TunnelConfig{ID: second, Name: "Curse of Strahd"})

	third := cfg.NewTunnelID("Curse of Strahd")

	if first != "curse-of-strahd" {
		t.Errorf("first id = %q, want the slug of the name", first)
	}
	if second == first || third == first || third == second {
		t.Fatalf("ids collide: %q, %q, %q", first, second, third)
	}
}

func TestNewTunnelIDFallsBackWhenTheNameHasNoSlug(t *testing.T) {
	cfg := DefaultConfig()

	first := cfg.NewTunnelID("...")
	cfg.AddTunnel(TunnelConfig{ID: first, Name: "..."})
	second := cfg.NewTunnelID("!!!")

	if first != fallbackTunnelID {
		t.Errorf("id = %q, want %q for a name with nothing to slug", first, fallbackTunnelID)
	}
	if second == first {
		t.Errorf("second unslugabble name reused %q", first)
	}
}

func TestBackToBackCreationsGetDistinctIDs(t *testing.T) {
	cfg := DefaultConfig()

	seen := make(map[string]bool)
	for i := 0; i < 50; i++ {
		id := cfg.NewTunnelID("Mesa")
		if seen[id] {
			t.Fatalf("id %q handed out twice after %d creations", id, i)
		}
		seen[id] = true
		cfg.AddTunnel(TunnelConfig{ID: id, Name: "Mesa"})
	}
}
