package main

import (
	"path/filepath"
	"strings"
	"testing"
)

// Tests run with the package directory as the working directory.
func testTokensPath() string {
	return filepath.Join("..", "..", tokensPath)
}

func TestLoadIsValid(t *testing.T) {
	src, err := load(testTokensPath())
	if err != nil {
		t.Fatalf("load %s: %v", tokensPath, err)
	}

	if len(src.Themes) != 24 {
		t.Fatalf("got %d themes, want 24", len(src.Themes))
	}
}

func TestRenderCoversEveryTokenOfEveryTheme(t *testing.T) {
	src, err := load(testTokensPath())
	if err != nil {
		t.Fatalf("load %s: %v", tokensPath, err)
	}

	reference := src.Themes[themeNames(src)[0]].Colors
	outputs := render(src)

	if len(outputs) != len(src.Themes) {
		t.Fatalf("rendered %d files for %d themes", len(outputs), len(src.Themes))
	}

	for name, theme := range src.Themes {
		path := themesDir + "/_" + name + ".css"
		body, ok := outputs[path]
		if !ok {
			t.Fatalf("theme %q has no rendered file", name)
		}

		for token := range reference {
			want := token + ": " + theme.Colors[token] + ";"
			if _, ok := theme.Colors[token]; !ok {
				t.Fatalf("theme %q is missing %s", name, token)
			}
			if !strings.Contains(body, want) {
				t.Fatalf("%s does not contain %q", path, want)
			}
		}
	}
}

func TestValidateRejectsIncompleteTheme(t *testing.T) {
	src, err := load(testTokensPath())
	if err != nil {
		t.Fatalf("load %s: %v", tokensPath, err)
	}

	src.Themes["broken"] = themeTokens{Mode: "dark", Colors: map[string]string{"--color-bg": "#000000"}}

	if err := validate(src); err == nil {
		t.Fatal("validate accepted a theme with one token")
	}
}
