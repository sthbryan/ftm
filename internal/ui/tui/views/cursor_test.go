package views

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func editorFixture(focus int, name, port string) *TunnelEditor {
	e := NewTunnelEditor()
	e.Focus = focus
	e.Name = name
	e.Port = port
	e.Provider = "cloudflared"
	return e
}

func TestCursorLandsRightAfterTheTypedText(t *testing.T) {
	for _, tc := range []struct {
		field int
		name  string
		port  string
		typed string
	}{
		{0, "Foundry VTT", "30000", "Foundry VTT"},
		{2, "Foundry VTT", "30000", "30000"},
		{2, "Foundry VTT", "3", "3"},
	} {
		e := editorFixture(tc.field, tc.name, tc.port)
		rendered := e.Render()

		if !e.HasCursor {
			t.Fatalf("field %d: no cursor while a text field has focus", tc.field)
		}

		lines := strings.Split(ansi.Strip(rendered), "\n")
		if e.CursorRow >= len(lines) {
			t.Fatalf("field %d: cursor row %d is past the panel", tc.field, e.CursorRow)
		}

		row := []rune(lines[e.CursorRow])
		if e.CursorColumn > len(row) {
			t.Fatalf("field %d: cursor column %d is past the row", tc.field, e.CursorColumn)
		}
		before := strings.TrimRight(string(row[:e.CursorColumn]), " ")

		if !strings.HasSuffix(before, tc.typed) {
			t.Errorf("field %d: cursor sits after %q, want it right after %q", tc.field, before, tc.typed)
		}
	}
}

func TestCursorSitsAtTheStartOfAnEmptyField(t *testing.T) {
	e := editorFixture(0, "", "30000")
	e.Render()

	if !e.HasCursor {
		t.Fatal("no cursor on an empty focused field")
	}
	if e.CursorColumn != valueColumn {
		t.Errorf("cursor column = %d, want %d so it sits before the placeholder", e.CursorColumn, valueColumn)
	}
}

func TestNoCursorOnTheProviderOrTheButton(t *testing.T) {
	for _, focus := range []int{1, submitFocus} {
		e := editorFixture(focus, "Foundry VTT", "30000")
		e.Render()

		if e.HasCursor {
			t.Errorf("focus %d is not a text field but asked for a cursor", focus)
		}
	}
}
