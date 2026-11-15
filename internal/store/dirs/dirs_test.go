package dirs

import (
	"path/filepath"
	"testing"
)

func TestResolve(t *testing.T) {
	tests := []struct {
		name                         string
		goos, home, userCfg, xdgRun  string
		wantConfig, wantRun, wantLeg string
	}{
		{
			name: "windows uses AppData",
			goos: "windows", home: `C:\Users\gm`, userCfg: `C:\Users\gm\AppData\Roaming`,
			wantConfig: filepath.Join(`C:\Users\gm\AppData\Roaming`, AppName),
			wantRun:    filepath.Join(`C:\Users\gm\AppData\Roaming`, AppName, "run"),
			wantLeg:    filepath.Join(`C:\Users\gm`, ".config", AppName),
		},
		{
			name: "linux follows XDG",
			goos: "linux", home: "/home/gm", userCfg: "/home/gm/.config", xdgRun: "/run/user/1000",
			wantConfig: filepath.Join("/home/gm/.config", AppName),
			wantRun:    filepath.Join("/run/user/1000", AppName),
			wantLeg:    filepath.Join("/home/gm", ".config", AppName),
		},
		{
			name: "linux without a runtime dir keeps it under config",
			goos: "linux", home: "/home/gm", userCfg: "/srv/cfg",
			wantConfig: filepath.Join("/srv/cfg", AppName),
			wantRun:    filepath.Join("/srv/cfg", AppName, "run"),
			wantLeg:    filepath.Join("/home/gm", ".config", AppName),
		},
		{
			name: "linux ignores a relative runtime dir",
			goos: "linux", home: "/home/gm", userCfg: "/home/gm/.config", xdgRun: "run",
			wantConfig: filepath.Join("/home/gm/.config", AppName),
			wantRun:    filepath.Join("/home/gm/.config", AppName, "run"),
			wantLeg:    filepath.Join("/home/gm", ".config", AppName),
		},
		{
			name: "macos uses Application Support",
			goos: "darwin", home: "/Users/gm", userCfg: "/Users/gm/Library/Application Support", xdgRun: "/tmp/x",
			wantConfig: filepath.Join("/Users/gm/Library/Application Support", AppName),
			wantRun:    filepath.Join("/Users/gm/Library/Application Support", AppName, "run"),
			wantLeg:    filepath.Join("/Users/gm", ".config", AppName),
		},
		{
			name: "no user config dir falls back to the legacy layout",
			goos: "windows", home: "/home/gm",
			wantConfig: filepath.Join("/home/gm", ".config", AppName),
			wantRun:    filepath.Join("/home/gm", ".config", AppName, "run"),
			wantLeg:    filepath.Join("/home/gm", ".config", AppName),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := Resolve(tt.goos, tt.home, tt.userCfg, tt.xdgRun)
			if d.Config != tt.wantConfig {
				t.Errorf("Config = %q, want %q", d.Config, tt.wantConfig)
			}
			if want := filepath.Join(tt.wantConfig, "bin"); d.Bin != want {
				t.Errorf("Bin = %q, want %q", d.Bin, want)
			}
			if d.Runtime != tt.wantRun {
				t.Errorf("Runtime = %q, want %q", d.Runtime, tt.wantRun)
			}
			if d.Legacy != tt.wantLeg {
				t.Errorf("Legacy = %q, want %q", d.Legacy, tt.wantLeg)
			}
			if want := filepath.Join(tt.wantConfig, "config.yaml"); d.ConfigFile() != want {
				t.Errorf("ConfigFile() = %q, want %q", d.ConfigFile(), want)
			}
		})
	}
}

func TestDefaultFollowsTheEnvironment(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("AppData", filepath.Join(home, "appdata"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "xdg"))

	d := Default()
	if d.Legacy != filepath.Join(home, ".config", AppName) {
		t.Errorf("Legacy = %q, want it under the isolated home", d.Legacy)
	}
	if rel, err := filepath.Rel(home, d.Config); err != nil || rel == ".." || filepath.IsAbs(rel) || rel[:2] == ".." {
		t.Errorf("Config = %q, want it under the isolated home %q", d.Config, home)
	}
}
