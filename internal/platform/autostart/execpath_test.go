package autostart

import "testing"

func TestResolveExecPathPrefersTheAppImageOverTheMountPoint(t *testing.T) {
	tests := []struct {
		name       string
		goos       string
		appImage   string
		executable string
		want       string
	}{
		{
			name:       "an AppImage registers the file, not its temporary mount",
			goos:       "linux",
			appImage:   "/home/gm/Apps/ftm-desktop-x86_64.AppImage",
			executable: "/tmp/.mount_ftm123/usr/bin/ftm-desktop",
			want:       "/home/gm/Apps/ftm-desktop-x86_64.AppImage",
		},
		{
			name:       "a plain linux binary registers itself",
			goos:       "linux",
			appImage:   "",
			executable: "/usr/local/bin/ftm-desktop",
			want:       "/usr/local/bin/ftm-desktop",
		},
		{
			name:       "macOS ignores a stray APPIMAGE",
			goos:       "darwin",
			appImage:   "/home/gm/Apps/ftm-desktop.AppImage",
			executable: "/Applications/ftm-desktop.app/Contents/MacOS/ftm-desktop",
			want:       "/Applications/ftm-desktop.app/Contents/MacOS/ftm-desktop",
		},
		{
			name:       "windows ignores a stray APPIMAGE",
			goos:       "windows",
			appImage:   "/home/gm/Apps/ftm-desktop.AppImage",
			executable: `C:\Program Files\ftm\ftm-desktop.exe`,
			want:       `C:\Program Files\ftm\ftm-desktop.exe`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveExecPath(tt.goos, tt.appImage, tt.executable)
			if got != tt.want {
				t.Errorf("resolveExecPath(%q, %q, %q) = %q, want %q", tt.goos, tt.appImage, tt.executable, got, tt.want)
			}
		})
	}
}
