// Package clipboard writes text to the system clipboard.
package clipboard

// WriteAll copies lines joined by newlines.
func WriteAll(lines []string) error {
	var text string
	for i, line := range lines {
		if i > 0 {
			text += "\n"
		}
		text += line
	}
	return Write(text)
}
