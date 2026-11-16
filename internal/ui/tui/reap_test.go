package tui

import (
	"os/exec"
	"testing"
	"time"
)

func TestExternalLauncherIsReaped(t *testing.T) {
	reaped := make(chan struct{})

	previous := onReaped
	onReaped = func(*exec.Cmd) { close(reaped) }
	t.Cleanup(func() { onReaped = previous })

	if err := startAndReap(exec.Command("sh", "-c", "exit 0")); err != nil {
		t.Fatalf("startAndReap: %v", err)
	}

	select {
	case <-reaped:
	case <-time.After(3 * time.Second):
		t.Error("the launcher process was never waited on and stays a zombie")
	}
}

func TestStartAndReapReportsAMissingLauncher(t *testing.T) {
	if err := startAndReap(exec.Command("ftm-no-such-launcher")); err == nil {
		t.Error("a launcher that does not exist reported success")
	}
}
