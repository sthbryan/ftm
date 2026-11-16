package autostart

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	appLabel = "com.justcallmebryan.ftm"
	appName  = "Foundry Tunnel Manager"
)

func execPath() (string, error) {
	executable, err := os.Executable()
	if err != nil {
		return "", err
	}

	if resolved, err := filepath.EvalSymlinks(executable); err == nil {
		executable = resolved
	}

	return resolveExecPath(runtime.GOOS, os.Getenv("APPIMAGE"), executable), nil
}

func resolveExecPath(goos, appImage, executable string) string {
	if goos == "linux" && appImage != "" {
		return filepath.Clean(appImage)
	}
	return filepath.Clean(executable)
}

func samePath(a, b string) bool {
	a = filepath.Clean(a)
	b = filepath.Clean(b)

	if runtime.GOOS == "windows" {
		return strings.EqualFold(a, b)
	}

	return a == b
}
