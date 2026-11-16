package ui

import (
	"testing"
	"unicode/utf8"

	"github.com/mattn/go-runewidth"
)

func TestTruncateKeepsShortStringsIntact(t *testing.T) {
	for _, s := range []string{"", "Foundry", "Configuración", "12345678901234567890"[:18]} {
		if got := Truncate(s, 18); got != s {
			t.Errorf("Truncate(%q, 18) = %q, want it unchanged", s, got)
		}
	}
}

func TestTruncateProducesValidUTF8(t *testing.T) {
	names := []string{
		"Configuración de la partida",
		"Mundo de Ñoño con acentos áéíóú",
		"日本語のトンネル名前です",
		"Foundry VTT (Default) — main",
	}

	for _, name := range names {
		for width := 1; width <= 24; width++ {
			got := Truncate(name, width)

			if !utf8.ValidString(got) {
				t.Fatalf("Truncate(%q, %d) = %q, which is not valid UTF-8", name, width, got)
			}
			if runewidth.StringWidth(got) > width {
				t.Fatalf("Truncate(%q, %d) = %q, width %d exceeds %d",
					name, width, got, runewidth.StringWidth(got), width)
			}
		}
	}
}

func TestTruncateAddsEllipsisWhenItCuts(t *testing.T) {
	got := Truncate("Configuración de la partida", 18)

	if got == "Configuración de la partida" {
		t.Fatal("Truncate did not shorten a long name")
	}
	if []rune(got)[len([]rune(got))-1] != '…' {
		t.Errorf("Truncate(...) = %q, want it to end with an ellipsis", got)
	}
}

func TestTruncateWithNonPositiveWidth(t *testing.T) {
	for _, width := range []int{0, -1, -50} {
		if got := Truncate("Foundry", width); got != "" {
			t.Errorf("Truncate(%q, %d) = %q, want empty", "Foundry", width, got)
		}
	}
}
