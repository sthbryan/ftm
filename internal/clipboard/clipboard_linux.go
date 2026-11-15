//go:build linux

package clipboard

func Write(text string) error {
	if err := writeVia("wl-copy", nil, text); err == nil {
		return nil
	}

	return writeVia("xclip", []string{"-selection", "clipboard"}, text)
}
