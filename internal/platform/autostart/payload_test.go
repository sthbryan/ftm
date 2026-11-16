package autostart

import "testing"

func TestDesktopEntryRoundTripsPathsWithSpaces(t *testing.T) {
	paths := []string{
		"/usr/local/bin/ftm-desktop",
		"/home/gm/Apps/Foundry Tunnel Manager.AppImage",
		`/home/gm/we"ird/ftm-desktop`,
	}

	for _, want := range paths {
		got := desktopEntryExec(desktopEntry(want))
		if got != want {
			t.Errorf("desktopEntryExec(desktopEntry(%q)) = %q, want %q", want, got, want)
		}
	}
}

func TestDesktopEntryIsEnabledByDefault(t *testing.T) {
	if !desktopEntryEnabled(desktopEntry("/usr/local/bin/ftm-desktop")) {
		t.Error("a freshly written entry reads as disabled, want enabled")
	}
}

func TestDesktopEntryHonoursDesktopEnvironmentOptOut(t *testing.T) {
	tests := []struct {
		name     string
		contents string
		want     bool
	}{
		{
			name:     "hidden marks it disabled",
			contents: "[Desktop Entry]\nType=Application\nExec=/bin/ftm\nHidden=true\n",
			want:     false,
		},
		{
			name:     "gnome opt-out marks it disabled",
			contents: "[Desktop Entry]\nType=Application\nExec=/bin/ftm\nX-GNOME-Autostart-enabled=false\n",
			want:     false,
		},
		{
			name:     "gnome opt-in stays enabled",
			contents: "[Desktop Entry]\nType=Application\nExec=/bin/ftm\nX-GNOME-Autostart-enabled=true\n",
			want:     true,
		},
		{
			name:     "a bare entry stays enabled",
			contents: "[Desktop Entry]\nType=Application\nExec=/bin/ftm\n",
			want:     true,
		},
		{
			name:     "comments are ignored",
			contents: "[Desktop Entry]\n# Hidden=true\nExec=/bin/ftm\n",
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := desktopEntryEnabled(tt.contents); got != tt.want {
				t.Errorf("desktopEntryEnabled() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLaunchAgentRoundTripsEscapedPaths(t *testing.T) {
	paths := []string{
		"/Applications/ftm-desktop.app/Contents/MacOS/ftm-desktop",
		"/Users/gm/Apps/Foundry & Tunnels.app/Contents/MacOS/ftm-desktop",
	}

	for _, want := range paths {
		got := launchAgentProgram(launchAgent(want))
		if got != want {
			t.Errorf("launchAgentProgram(launchAgent(%q)) = %q, want %q", want, got, want)
		}
	}
}

func TestLaunchAgentProgramToleratesGarbage(t *testing.T) {
	if got := launchAgentProgram("not a plist"); got != "" {
		t.Errorf("launchAgentProgram(garbage) = %q, want empty", got)
	}
}

func TestStartupApprovalDisabled(t *testing.T) {
	tests := []struct {
		name  string
		value []byte
		want  bool
	}{
		{name: "missing value reads as enabled", value: nil, want: false},
		{name: "windows enabled marker", value: []byte{2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, want: false},
		{name: "windows disabled marker", value: []byte{3, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, want: true},
		{name: "task manager re-enabled marker", value: []byte{6, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := startupApprovalDisabled(tt.value); got != tt.want {
				t.Errorf("startupApprovalDisabled(%v) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

func TestWindowsCommandRoundTrip(t *testing.T) {
	want := `C:\Program Files\ftm\ftm-desktop.exe`

	if got := unquoteWindowsCommand(quoteWindowsCommand(want)); got != want {
		t.Errorf("unquoteWindowsCommand(quoteWindowsCommand(%q)) = %q, want %q", want, got, want)
	}

	if got := unquoteWindowsCommand(want); got != want {
		t.Errorf("unquoteWindowsCommand(%q) = %q, want the unquoted path unchanged", want, got)
	}
}
