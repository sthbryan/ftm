package autostart

import (
	"fmt"
	"html"
	"strings"
)

const desktopEntryIcon = "ftm-desktop"

func desktopEntry(path string) string {
	return fmt.Sprintf(`[Desktop Entry]
Type=Application
Name=%s
Exec=%s
Icon=%s
Terminal=false
X-GNOME-Autostart-enabled=true
`, appName, quoteDesktopExec(path), desktopEntryIcon)
}

func quoteDesktopExec(path string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `"`, `\"`, "`", "\\`", `$`, `\$`)
	return `"` + replacer.Replace(path) + `"`
}

func unquoteDesktopExec(value string) string {
	value = strings.TrimSpace(value)
	if !strings.HasPrefix(value, `"`) {
		return value
	}

	value = strings.TrimSuffix(strings.TrimPrefix(value, `"`), `"`)
	replacer := strings.NewReplacer(`\"`, `"`, "\\`", "`", `\$`, `$`, `\\`, `\`)

	return replacer.Replace(value)
}

func desktopEntryValue(contents, key string) string {
	for _, line := range strings.Split(contents, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		name, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}

		if strings.EqualFold(strings.TrimSpace(name), key) {
			return strings.TrimSpace(value)
		}
	}

	return ""
}

func desktopEntryEnabled(contents string) bool {
	if strings.EqualFold(desktopEntryValue(contents, "Hidden"), "true") {
		return false
	}

	if autostart := desktopEntryValue(contents, "X-GNOME-Autostart-enabled"); autostart != "" {
		return !strings.EqualFold(autostart, "false")
	}

	return true
}

func desktopEntryExec(contents string) string {
	return unquoteDesktopExec(desktopEntryValue(contents, "Exec"))
}

func launchAgent(path string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>%s</string>
	<key>ProgramArguments</key>
	<array>
		<string>%s</string>
	</array>
	<key>RunAtLoad</key>
	<true/>
	<key>ProcessType</key>
	<string>Interactive</string>
</dict>
</plist>
`, appLabel, html.EscapeString(path))
}

func launchAgentProgram(contents string) string {
	_, rest, found := strings.Cut(contents, "<array>")
	if !found {
		return ""
	}

	block, _, found := strings.Cut(rest, "</array>")
	if !found {
		return ""
	}

	_, rest, found = strings.Cut(block, "<string>")
	if !found {
		return ""
	}

	value, _, found := strings.Cut(rest, "</string>")
	if !found {
		return ""
	}

	return html.UnescapeString(strings.TrimSpace(value))
}

func startupApprovalDisabled(value []byte) bool {
	return len(value) > 0 && value[0]&1 == 1
}

func quoteWindowsCommand(path string) string {
	return `"` + path + `"`
}

func unquoteWindowsCommand(value string) string {
	value = strings.TrimSpace(value)
	if !strings.HasPrefix(value, `"`) {
		return value
	}

	value = value[1:]
	if end := strings.Index(value, `"`); end >= 0 {
		return value[:end]
	}

	return value
}
