package ui

import (
	"image/color"
	"testing"
)

func luminance(c color.Color) float64 {
	r, g, b, _ := c.RGBA()
	return (0.2126*float64(r) + 0.7152*float64(g) + 0.0722*float64(b)) / 65535
}

func TestThemeInvertsWithTheBackground(t *testing.T) {
	dark := BuildTheme(true)
	light := BuildTheme(false)

	if luminance(dark.Text) <= 0.5 {
		t.Error("text on a dark background is not light")
	}
	if luminance(light.Text) >= 0.5 {
		t.Error("text on a light background is not dark")
	}

	if luminance(dark.Offline) >= 0.5 {
		t.Error("the offline row background is not dark on a dark terminal")
	}
	if luminance(light.Offline) <= 0.5 {
		t.Error("the offline row background is not light on a light terminal")
	}
}

func TestThemeKeepsTextReadableOnRowBackgrounds(t *testing.T) {
	for _, tc := range []struct {
		name   string
		isDark bool
	}{
		{"dark terminal", true},
		{"light terminal", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			theme := BuildTheme(tc.isDark)
			text := luminance(theme.Text)

			backgrounds := map[string]color.Color{
				"online":     theme.Online,
				"offline":    theme.Offline,
				"connecting": theme.Connecting,
				"error":      theme.Error,
				"stopped":    theme.Stopped,
			}

			for label, bg := range backgrounds {
				if diff := text - luminance(bg); diff < 0.3 && diff > -0.3 {
					t.Errorf("%s row: text and background are too close in luminance (%.2f vs %.2f)",
						label, text, luminance(bg))
				}
			}
		})
	}
}

func TestButtonsStayLegibleOnBothBackgrounds(t *testing.T) {
	for _, tc := range []struct {
		name   string
		isDark bool
	}{
		{"dark terminal", true},
		{"light terminal", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			theme := BuildTheme(tc.isDark)

			for _, pair := range []struct {
				label      string
				background color.Color
				foreground color.Color
			}{
				{"button", theme.Button, theme.ButtonText},
				{"active button", theme.ButtonActive, theme.ButtonActiveText},
			} {
				diff := luminance(pair.foreground) - luminance(pair.background)
				if diff < 0.35 && diff > -0.35 {
					t.Errorf("%s: label and fill are too close in luminance (%.2f vs %.2f)",
						pair.label, luminance(pair.foreground), luminance(pair.background))
				}
			}
		})
	}
}

func TestSetDarkBackgroundSwapsTheActiveTheme(t *testing.T) {
	original := ThemeDefault
	t.Cleanup(func() { ThemeDefault = original })

	SetDarkBackground(false)
	lightText := luminance(ThemeDefault.Text)

	SetDarkBackground(true)
	darkText := luminance(ThemeDefault.Text)

	if lightText >= darkText {
		t.Errorf("switching to a dark background did not lighten the text (%.2f then %.2f)", lightText, darkText)
	}
}
