// Package dirs is the one place that knows where ftm keeps its files. It
// imports nothing from the app so that every package, config included, can
// ask it for a path instead of building one.
package dirs

import (
	"os"
	"path"
	"path/filepath"
	"runtime"
)

const (
	// AppName names the folder ftm uses under each base directory.
	AppName = "foundry-tunnel"

	configFile = "config.yaml"
	logFile    = "ftm.log"
)

// Dirs holds every directory ftm writes to.
//
//   - Config: config.yaml and the log. %AppData%\foundry-tunnel on Windows,
//     $XDG_CONFIG_HOME/foundry-tunnel (~/.config) on Linux, Application
//     Support on macOS.
//   - Bin: downloaded provider binaries, under Config.
//   - Runtime: the single-instance lock and the runtime file. Under Config,
//     except on Linux with $XDG_RUNTIME_DIR set, which is the place meant for it.
//   - Legacy: where every platform used to keep the config (~/.config), read
//     once to move an old config to Config.
type Dirs struct {
	Config  string
	Bin     string
	Runtime string
	Legacy  string
}

// Default resolves the directories for the current user and platform.
func Default() Dirs {
	return Resolve(runtime.GOOS, userHomeDir(), userConfigDir(), os.Getenv("XDG_RUNTIME_DIR"))
}

// Resolve is the pure part of Default, so every platform's layout can be
// tested from any host. userConfig is what os.UserConfigDir returns there.
func Resolve(goos, home, userConfig, xdgRuntime string) Dirs {
	if home == "" {
		home = "."
	}

	legacy := filepath.Join(home, ".config", AppName)

	config := legacy
	if userConfig != "" {
		config = filepath.Join(userConfig, AppName)
	}

	runtimeDir := filepath.Join(config, "run")
	if goos == "linux" && path.IsAbs(xdgRuntime) {
		runtimeDir = filepath.Join(xdgRuntime, AppName)
	}

	return Dirs{
		Config:  config,
		Bin:     filepath.Join(config, "bin"),
		Runtime: runtimeDir,
		Legacy:  legacy,
	}
}

// ConfigFile is the path of config.yaml.
func (d Dirs) ConfigFile() string { return filepath.Join(d.Config, configFile) }

// LegacyConfigFile is where an older build left config.yaml.
func (d Dirs) LegacyConfigFile() string { return filepath.Join(d.Legacy, configFile) }

// LogFile is the path of the TUI's log.
func (d Dirs) LogFile() string { return filepath.Join(d.Config, logFile) }

func userHomeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return home
}

func userConfigDir() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return dir
}
