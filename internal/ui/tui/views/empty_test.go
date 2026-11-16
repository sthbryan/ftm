package views

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/sthbryan/ftm/internal/i18n"
)

func plain(s string) string {
	var b strings.Builder
	skipping := false

	for _, r := range s {
		switch {
		case r == '\x1b':
			skipping = true
		case skipping && r == 'm':
			skipping = false
		case !skipping:
			b.WriteRune(r)
		}
	}

	return b.String()
}

func TestWelcomeScreenWalksThroughTheThreeSteps(t *testing.T) {
	v := NewEmptyState()
	v.Width, v.Height = 100, 30
	v.Dashboard = "http://localhost:40500"

	out := plain(v.Render())

	if lead := strings.Fields(i18n.T("welcome_lead")); !strings.Contains(out, strings.Join(lead[:4], " ")) {
		t.Error("welcome screen is missing the lead sentence")
	}

	for _, key := range []string{
		"welcome_title",
		"welcome_step_provider_title",
		"welcome_step_port_title",
		"welcome_step_share_title",
		"create_first",
	} {
		if !strings.Contains(out, i18n.T(key)) {
			t.Errorf("welcome screen is missing %q", key)
		}
	}
}

func TestWelcomeScreenDropsTheStepsRatherThanClipping(t *testing.T) {
	v := NewEmptyState()
	v.Width, v.Height = 50, 20
	v.Dashboard = "http://localhost:40500"

	out := plain(v.Render())

	if strings.Contains(out, i18n.T("welcome_step_provider_title")) {
		t.Error("the steps survived at a height that cannot hold them")
	}
	if !strings.Contains(out, i18n.T("create_first")) {
		t.Error("the call to action was dropped, which is the one line that must survive")
	}
	if got := len(strings.Split(out, "\n")); got != 20 {
		t.Errorf("rendered %d lines, want 20 to match the terminal", got)
	}
}

func TestWelcomeScreenNeverOutgrowsANarrowTerminal(t *testing.T) {
	for _, width := range []int{40, 50, 62, 80, 120} {
		v := NewEmptyState()
		v.Width, v.Height = width, 30
		v.Dashboard = "http://localhost:40500"

		for i, line := range strings.Split(v.Render(), "\n") {
			if w := lipgloss.Width(line); w != width {
				t.Errorf("width %d: line %d is %d cells wide, want %d", width, i, w, width)
			}
		}
	}
}

func TestEmptyHomeKeepsTheListChrome(t *testing.T) {
	v := listWith(nil, 100, 30, 0)

	out := plain(v.Render())

	for _, key := range []string{"app_name_tui", "connections", "no_tunnels", "add_tunnel_prompt"} {
		if !strings.Contains(out, i18n.T(key)) {
			t.Errorf("empty home is missing %q", key)
		}
	}

	if !strings.Contains(out, "http://localhost:40500") {
		t.Error("empty home dropped the dashboard address from the header")
	}
	if strings.Contains(out, i18n.T("welcome_title")) {
		t.Error("empty home is showing the welcome panel")
	}
}

func TestEmptyHomeStaysSingleColumn(t *testing.T) {
	v := listWith(nil, 140, 30, 0)

	if out := plain(v.Render()); strings.Contains(out, i18n.T("selected_tunnel")) {
		t.Error("the detail pane rendered with nothing to select")
	}
}

func TestEmptyHomeFillsTheTerminal(t *testing.T) {
	for _, size := range [][2]int{{80, 24}, {100, 30}, {140, 40}} {
		v := listWith(nil, size[0], size[1], 0)

		lines := strings.Split(v.Render(), "\n")
		if len(lines) > size[1] {
			t.Errorf("%dx%d: rendered %d lines, taller than the terminal", size[0], size[1], len(lines))
		}
	}
}
