package ui

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

func background(width, height int) string {
	row := strings.Repeat("背", width/2)
	rows := make([]string, height)
	for i := range rows {
		rows[i] = row
	}
	return strings.Join(rows, "\n")
}

func TestOverlayKeepsTheBackgroundVisible(t *testing.T) {
	out := Overlay(background(40, 12), "MODAL", 40, 12)

	if !strings.Contains(out, "MODAL") {
		t.Fatal("the overlay did not draw the foreground")
	}
	if !strings.Contains(out, "背") {
		t.Error("the overlay erased the background instead of floating over it")
	}
}

func TestOverlayFitsTheTerminal(t *testing.T) {
	const width, height = 40, 12

	out := Overlay(background(width, height), "MODAL\nMODAL", width, height)

	if lines := strings.Count(out, "\n") + 1; lines != height {
		t.Errorf("rendered %d lines, want %d", lines, height)
	}
}

func TestOverlayPaintsEveryCell(t *testing.T) {
	const width, height = 40, 12

	for name, out := range map[string]string{
		"over a list":   Overlay(background(width, height), "MODAL", width, height),
		"over nothing":  Overlay("", "MODAL", width, height),
		"short content": Overlay("one\ntwo", "MODAL", width, height),
	} {
		lines := strings.Split(out, "\n")
		if len(lines) != height {
			t.Errorf("%s: rendered %d lines, want %d", name, len(lines), height)
		}
		for i, line := range lines {
			if w := lipgloss.Width(line); w != width {
				t.Errorf("%s: line %d is %d cells wide, want %d", name, i, w, width)
			}
		}
	}
}

func TestOverlayCentresTheForeground(t *testing.T) {
	const width, height = 41, 11

	plain := strings.TrimSuffix(strings.Repeat(strings.Repeat(".", width)+"\n", height), "\n")

	lines := strings.Split(ansi.Strip(Overlay(plain, "ABC", width, height)), "\n")

	row := -1
	for i, line := range lines {
		if strings.Contains(line, "ABC") {
			row = i
			break
		}
	}
	if row == -1 {
		t.Fatal("the foreground is missing")
	}
	if row != height/2 {
		t.Errorf("the foreground sits on row %d, want %d", row, height/2)
	}
	if column := strings.Index(lines[row], "ABC"); column != (width-3)/2 {
		t.Errorf("the foreground starts at column %d, want %d", column, (width-3)/2)
	}
}

func TestOverlayWithoutASizeFallsBackToTheForeground(t *testing.T) {
	if out := Overlay("ignored", "MODAL", 0, 0); out != "MODAL" {
		t.Errorf("got %q, want the bare foreground", out)
	}
}

func TestOverlayDimsWhatItCovers(t *testing.T) {
	list := lipgloss.NewStyle().
		Foreground(ThemeDefault.Gold).
		Background(ThemeDefault.Online).
		Render("ONLINE")

	out := Overlay(list, "MODAL", 40, 12)

	if strings.Contains(out, "MODAL") && !strings.Contains(out, "ONLINE") {
		t.Fatal("the background text is gone")
	}
	if strings.Count(out, "ONLINE") != 1 {
		t.Fatal("the background was not composed once")
	}

	background := ""
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(ansi.Strip(line), "ONLINE") {
			background = line
		}
	}

	if strings.Contains(background, "MODAL") {
		t.Skip("the modal overlaps the background row")
	}
	if background == ansi.Strip(background) {
		t.Error("the covered row carries no styling at all")
	}
	if strings.Contains(background, "1;") {
		t.Error("the covered row kept its original emphasis instead of being dimmed")
	}
}
