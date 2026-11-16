package main

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLogsGoToAFileNotTheTerminal(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("AppData", filepath.Join(home, "appdata"))
	t.Setenv("XDG_CONFIG_HOME", "")

	previous := log.Writer()
	t.Cleanup(func() { log.SetOutput(previous) })

	file := redirectLog()
	if file == nil {
		t.Fatal("redirectLog gave up and would keep writing to stderr")
	}
	t.Cleanup(func() { file.Close() })

	if log.Writer() == os.Stderr {
		t.Fatal("logs still go to the terminal the TUI is drawing on")
	}

	log.Print("web: port 40500 unavailable")

	written, err := os.ReadFile(LogPath())
	if err != nil {
		t.Fatalf("reading the log file: %v", err)
	}
	if !strings.Contains(string(written), "port 40500 unavailable") {
		t.Errorf("the log file does not hold the message: %q", written)
	}

	if dir := filepath.Dir(LogPath()); !strings.HasPrefix(dir, home) {
		t.Errorf("the log landed outside the config dir: %s", dir)
	}
}
