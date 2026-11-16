package main

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// The desktop shell talks to the core through internal/service; it must not
// pull in the TUI or Bubble Tea.
func TestDoesNotImportTheTUI(t *testing.T) {
	out, err := exec.Command("go", "list", "-deps", ".").Output()
	if err != nil {
		t.Fatalf("go list -deps: %v", err)
	}
	for _, dep := range strings.Fields(string(out)) {
		if dep == "github.com/sthbryan/ftm/internal/app" ||
			strings.HasPrefix(dep, "github.com/sthbryan/ftm/internal/app/") ||
			strings.HasPrefix(dep, "charm.land/bubbletea") {
			t.Errorf("ftm-desktop depends on %s", dep)
		}
	}
}

type fakeWindow struct{ calls []string }

func (f *fakeWindow) Restore()                 { f.calls = append(f.calls, "restore") }
func (f *fakeWindow) Show() application.Window { f.calls = append(f.calls, "show"); return nil }
func (f *fakeWindow) Focus()                   { f.calls = append(f.calls, "focus") }

// A second launch brings a minimised or hidden window to the front.
func TestRaiseRestoresShowsAndFocuses(t *testing.T) {
	w := &fakeWindow{}
	raise(w)
	if got := strings.Join(w.calls, ","); got != "restore,show,focus" {
		t.Fatalf("raise called %s", got)
	}
}

func TestMiniURLEscapesTheTunnelID(t *testing.T) {
	got := miniURL("http://127.0.0.1:4000", "a b&c")
	if got != "http://127.0.0.1:4000/mini?id=a+b%26c" {
		t.Fatalf("miniURL = %s", got)
	}
}
