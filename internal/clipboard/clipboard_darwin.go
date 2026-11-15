//go:build darwin

package clipboard

func Write(text string) error {
	return writeVia("pbcopy", nil, text)
}
