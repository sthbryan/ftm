package ui

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
)

func TestPanelFillsExactlyTheRequestedBox(t *testing.T) {
	for _, size := range []struct{ width, height int }{
		{40, 10}, {20, 5}, {120, 30}, {13, 4},
	} {
		out := Panel("Connections", "one\ntwo", size.width, size.height, ThemeDefault.Bronze)

		lines := strings.Split(out, "\n")
		if len(lines) != size.height {
			t.Errorf("%dx%d: rendered %d lines", size.width, size.height, len(lines))
		}
		for i, line := range lines {
			if w := lipgloss.Width(line); w != size.width {
				t.Errorf("%dx%d: line %d is %d cells wide", size.width, size.height, i, w)
			}
		}
	}
}

func TestPanelShowsItsTitleInTheTopBorder(t *testing.T) {
	out := Panel("Connections", "body", 40, 6, ThemeDefault.Bronze)

	top := strings.Split(out, "\n")[0]
	if !strings.Contains(top, "Connections") {
		t.Errorf("the title is missing from the top border: %q", top)
	}
	if strings.Contains(strings.Join(strings.Split(out, "\n")[1:], "\n"), "Connections") {
		t.Error("the title leaked into the panel body")
	}
}

func TestPanelWithoutRoomFallsBackToItsContent(t *testing.T) {
	if out := Panel("Connections", "body", 3, 10, ThemeDefault.Bronze); out != "body" {
		t.Errorf("got %q, want the bare content", out)
	}
}

func TestPanelClipsContentTallerThanTheBox(t *testing.T) {
	content := strings.TrimSuffix(strings.Repeat("row\n", 40), "\n")

	out := Panel("Logs", content, 30, 8, ThemeDefault.Bronze)

	if lines := strings.Count(out, "\n") + 1; lines != 8 {
		t.Errorf("rendered %d lines, want 8", lines)
	}
}
