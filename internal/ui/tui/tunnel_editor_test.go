package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/sthbryan/ftm/internal/core"
	"github.com/sthbryan/ftm/internal/supervisor"
)

func editorWith(t *testing.T, focus int) *Model {
	t.Helper()

	m := modelWithTunnel(t, "t1", "Foundry VTT")
	m.State = viewNewTunnel
	m.EditorFocus = focus

	return m
}

func TestArrowsMoveBetweenFields(t *testing.T) {
	m := editorWith(t, 0)

	m.handleKey(tea.KeyPressMsg{Code: tea.KeyDown})
	if m.EditorFocus != 1 {
		t.Errorf("down left the focus on %d, want 1", m.EditorFocus)
	}

	m.handleKey(tea.KeyPressMsg{Code: tea.KeyUp})
	if m.EditorFocus != 0 {
		t.Errorf("up left the focus on %d, want 0", m.EditorFocus)
	}
}

func TestFocusWrapsAroundAndNeverStopsOnNothing(t *testing.T) {
	m := editorWith(t, 0)

	seen := make(map[int]bool)
	for i := 0; i < editorStops; i++ {
		seen[m.EditorFocus] = true
		m.handleKey(tea.KeyPressMsg{Code: tea.KeyDown})
	}

	if m.EditorFocus != 0 {
		t.Errorf("a full cycle ended on %d, want back at 0", m.EditorFocus)
	}
	for stop := 0; stop < editorStops; stop++ {
		if !seen[stop] {
			t.Errorf("focus %d is never reachable", stop)
		}
	}
}

func TestUpFromTheFirstFieldWrapsToSubmit(t *testing.T) {
	m := editorWith(t, 0)

	m.handleKey(tea.KeyPressMsg{Code: tea.KeyUp})

	if m.EditorFocus != editorSubmit {
		t.Errorf("focus = %d, want the submit button at %d", m.EditorFocus, editorSubmit)
	}
}

func TestArrowsStillChangeTheProvider(t *testing.T) {
	m := editorWith(t, 1)
	m.Draft.Provider = "cloudflared"

	m.handleKey(tea.KeyPressMsg{Code: tea.KeyRight})

	if m.Draft.Provider == "cloudflared" {
		t.Error("right did not move to the next provider")
	}
	if m.EditorFocus != 1 {
		t.Errorf("changing the provider moved the focus to %d", m.EditorFocus)
	}
}

func TestFirstArrowMovesEvenFromAnUnknownProvider(t *testing.T) {
	m := editorWith(t, 1)
	m.Draft.Provider = "localtunnel"

	m.handleKey(tea.KeyPressMsg{Code: tea.KeyRight})

	if m.Draft.Provider == "localtunnel" {
		t.Fatal("the first press was swallowed rewriting the provider")
	}
	if m.Draft.Provider != string(core.AllProviders()[1]) {
		t.Errorf("provider = %q, want the second in the list", m.Draft.Provider)
	}
}

func TestProviderSelectorReachesEveryProvider(t *testing.T) {
	m := editorWith(t, 1)
	m.Draft.Provider = string(core.AllProviders()[0])

	seen := map[string]bool{}
	for range core.AllProviders() {
		seen[m.Draft.Provider] = true
		m.handleKey(tea.KeyPressMsg{Code: tea.KeyRight})
	}

	for _, provider := range core.AllProviders() {
		if !seen[string(provider)] {
			t.Errorf("%q is never reachable in the editor", provider)
		}
	}
}

func TestEveryOfferedProviderCanActuallyStart(t *testing.T) {
	manager := supervisor.NewManager()

	for _, provider := range core.AllProviders() {
		if err := manager.Start(core.TunnelConfig{ID: "x", Provider: provider}); err != nil {
			if strings.Contains(err.Error(), "unknown provider") {
				t.Errorf("the editor offers %q but the manager cannot start it", provider)
			}
		}
		_ = manager.Stop("x")
	}
}
