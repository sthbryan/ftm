package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

func pressKey(m *Model, k string) tea.Cmd {
	_, cmd := m.handleKey(tea.KeyPressMsg{Code: rune(k[0]), Text: k})
	return cmd
}

func pressEscape(m *Model) tea.Cmd {
	_, cmd := m.handleKey(tea.KeyPressMsg{Code: tea.KeyEscape})
	return cmd
}

func TestSubViewsReturnToTheListInsteadOfQuitting(t *testing.T) {
	for _, state := range []viewState{viewLogs, viewSettings, viewDownloading} {
		for name, press := range map[string]func(*Model) tea.Cmd{"q": func(m *Model) tea.Cmd { return pressKey(m, "q") }, "esc": pressEscape} {
			m := modelWithTunnel(t, "t1", "Foundry VTT")
			m.State = state

			if cmd := press(m); cmd != nil {
				t.Errorf("%v + %s issued a command, want no quit", state, name)
			}
			if m.State != viewList {
				t.Errorf("%v + %s left State = %v, want viewList", state, name, m.State)
			}
		}
	}
}

func TestQuitClosesTheAppOnlyFromTheList(t *testing.T) {
	m := modelWithTunnel(t, "t1", "Foundry VTT")

	cmd := pressKey(m, "q")
	if cmd == nil {
		t.Fatal("q did not quit from the list view")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Errorf("q produced %T, want tea.QuitMsg", cmd())
	}
}

func TestEscapeDoesNothingOnTheList(t *testing.T) {
	m := modelWithTunnel(t, "t1", "Foundry VTT")

	if cmd := pressEscape(m); cmd != nil {
		t.Error("escape quit from the list view")
	}
	if m.State != viewList {
		t.Errorf("State = %v, want viewList", m.State)
	}
}

func TestForceQuitAlwaysCloses(t *testing.T) {
	for _, state := range []viewState{viewList, viewLogs, viewSettings, viewNewTunnel} {
		m := modelWithTunnel(t, "t1", "Foundry VTT")
		m.State = state

		_, cmd := m.handleKey(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
		if cmd == nil {
			t.Fatalf("ctrl+c did not quit from %v", state)
		}
		if _, ok := cmd().(tea.QuitMsg); !ok {
			t.Errorf("ctrl+c from %v produced %T, want tea.QuitMsg", state, cmd())
		}
	}
}
