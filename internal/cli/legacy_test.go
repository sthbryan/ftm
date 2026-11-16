package cli

import (
	"slices"
	"strings"
	"testing"
)

func TestOldFlagsStillWork(t *testing.T) {
	cases := []struct{ in, want string }{
		{"", ""},
		{"-web", "serve --open"},
		{"--web", "serve --open"},
		{"-server", "serve"},
		{"-server -port 8080", "serve --port 8080"},
		{"-web -port=8080", "serve --open --port 8080"},
		{"-port 8080", "--port 8080"},
		{"-update", "self-update"},
		{"-check", "self-update --check"},
		{"-update -check", "self-update --check"},
		{"-uninstall", "uninstall --yes"},
		{"-version", "version"},
		{"--version", "version"},
		{"-web=false", ""},
		{"--help", "--help"},
		{"serve -port 9000", "serve --port 9000"},
		{"ls --json", "ls --json"},
		{"add x --port 30000", "add x --port 30000"},
	}
	for _, c := range cases {
		got := strings.Join(translateLegacy(strings.Fields(c.in)), " ")
		if got != c.want {
			t.Errorf("translateLegacy(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestOldFlagsAreHidden(t *testing.T) {
	h := newHarness(t, modes[0])
	out := h.mustRun("--help")
	_, flags, _ := strings.Cut(out, "FLAGS")
	if strings.Contains(flags, "--port") {
		t.Fatalf("help shows the old --port flag:\n%s", out)
	}
	if out := h.mustRun("-version"); !strings.Contains(out, "Foundry Tunnel Manager v") {
		t.Fatalf("-version = %q", out)
	}
	if out := h.mustRun("-server"); !strings.Contains(out, "Web view") {
		t.Fatalf("-server = %q", out)
	}
	root := newRoot(&h.deps)
	if f := root.Flags().Lookup("port"); f == nil || !f.Hidden {
		t.Fatal("--port must exist on the root and be hidden")
	}
	var names []string
	for _, c := range root.Commands() {
		names = append(names, c.Name())
	}
	for _, want := range []string{"up", "down", "share", "ls", "add", "rm", "edit", "logs", "providers", "doctor", "serve", "config", "self-update", "uninstall", "version"} {
		if !slices.Contains(names, want) {
			t.Errorf("no %s command", want)
		}
	}
}
