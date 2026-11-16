package provider

import (
	"context"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"
	"time"
)

func startSleeper(t *testing.T, script string) *Process {
	t.Helper()

	if runtime.GOOS == "windows" {
		t.Skip("uses a POSIX shell")
	}

	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, "sh", "-c", script)

	proc, err := StartProcess(cmd, cancel)
	if err != nil {
		cancel()
		t.Fatalf("StartProcess failed: %v", err)
	}

	return proc
}

func TestStartProcessReportsExit(t *testing.T) {
	proc := startSleeper(t, "exit 0")

	select {
	case <-proc.Exited():
	case <-time.After(5 * time.Second):
		t.Fatal("Exited() never closed for a process that finished")
	}

	if err := proc.Err(); err != nil {
		t.Fatalf("Err() = %v, want nil for a clean exit", err)
	}
}

func TestStartProcessReportsFailure(t *testing.T) {
	proc := startSleeper(t, "exit 3")

	select {
	case <-proc.Exited():
	case <-time.After(5 * time.Second):
		t.Fatal("Exited() never closed")
	}

	if proc.Err() == nil {
		t.Fatal("Err() = nil for a process that exited 3, want an error")
	}
}

func TestStopTerminatesProcess(t *testing.T) {
	proc := startSleeper(t, "sleep 60")

	done := make(chan struct{})
	go func() {
		proc.Stop()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("Stop() did not return")
	}

	select {
	case <-proc.Exited():
	case <-time.After(time.Second):
		t.Fatal("Stop() returned but the process is still running")
	}
}

func TestStopKillsGrandchildren(t *testing.T) {

	pidFile := t.TempDir() + "/grandchild.pid"
	proc := startSleeper(t, "sh -c 'sleep 60 & echo $! > "+pidFile+"; wait' ")

	deadline := time.Now().Add(5 * time.Second)
	var pid string
	for time.Now().Before(deadline) {
		if data, err := readTrimmed(pidFile); err == nil && data != "" {
			pid = data
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if pid == "" {
		t.Skip("could not capture a grandchild pid on this system")
	}

	proc.Stop()

	deadline = time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if exec.Command("kill", "-0", pid).Run() != nil {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}

	_ = exec.Command("kill", "-9", pid).Run()
	t.Fatalf("grandchild %s survived Stop(); the process tree was not torn down", pid)
}

func TestStopIsIdempotent(t *testing.T) {
	proc := startSleeper(t, "sleep 60")

	proc.Stop()
	proc.Stop()

	select {
	case <-proc.Exited():
	case <-time.After(time.Second):
		t.Fatal("process still running after two Stop() calls")
	}
}

func readTrimmed(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}
