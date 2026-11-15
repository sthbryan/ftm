package instance

import (
	"bufio"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

func TestSecondAcquireFindsTheOwner(t *testing.T) {
	dir := t.TempDir()

	owner, err := Acquire(dir)
	if err != nil {
		t.Fatalf("first Acquire: %v", err)
	}
	defer func() { _ = owner.Release() }()

	if err := owner.Publish(40507); err != nil {
		t.Fatal(err)
	}

	_, err = Acquire(dir)
	var running *RunningError
	if !errors.As(err, &running) {
		t.Fatalf("second Acquire = %v, want a *RunningError", err)
	}
	if running.Info == nil || running.Info.Port != 40507 || running.Info.Token != owner.Token() || running.Info.PID != os.Getpid() {
		t.Fatalf("RunningError.Info = %+v, want the owner's port, token and pid", running.Info)
	}
	if running.Info.URL() != "http://127.0.0.1:40507" {
		t.Errorf("URL() = %q", running.Info.URL())
	}

	info, err := Find(dir)
	if err != nil || info == nil || info.Port != 40507 {
		t.Fatalf("Find = %+v, %v, want the owner", info, err)
	}
}

func TestAcquireBeforePublishReportsAStartingOwner(t *testing.T) {
	publishWait = 100 * time.Millisecond
	t.Cleanup(func() { publishWait = 3 * time.Second })

	dir := t.TempDir()
	owner, err := Acquire(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = owner.Release() }()

	_, err = Acquire(dir)
	var running *RunningError
	if !errors.As(err, &running) || running.Info != nil {
		t.Fatalf("Acquire = %v, want a *RunningError without info", err)
	}
}

func TestReleaseRemovesTheRuntimeFile(t *testing.T) {
	dir := t.TempDir()

	owner, err := Acquire(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := owner.Publish(40501); err != nil {
		t.Fatal(err)
	}
	if err := owner.Release(); err != nil {
		t.Fatalf("Release: %v", err)
	}
	if err := owner.Release(); err != nil {
		t.Fatalf("second Release: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, runtimeName)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("runtime file still there after Release (stat err %v)", err)
	}
	if info, err := Find(dir); err != nil || info != nil {
		t.Fatalf("Find = %+v, %v after Release, want nothing", info, err)
	}

	next, err := Acquire(dir)
	if err != nil {
		t.Fatalf("Acquire after Release: %v", err)
	}
	_ = next.Release()
}

func TestARuntimeFileWithoutAnOwnerIsStale(t *testing.T) {
	dir := t.TempDir()
	stale := filepath.Join(dir, runtimeName)
	if err := os.WriteFile(filepath.Join(dir, lockName), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(stale, []byte(`{"pid":999999,"port":40502,"token":"old"}`), 0o600); err != nil {
		t.Fatal(err)
	}

	info, err := Find(dir)
	if err != nil || info != nil {
		t.Fatalf("Find = %+v, %v, want a stale file ignored", info, err)
	}
	if _, err := os.Stat(stale); !errors.Is(err, os.ErrNotExist) {
		t.Error("Find left the stale runtime file behind")
	}

	if err := os.WriteFile(stale, []byte(`{"pid":999999,"port":40502,"token":"old"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	owner, err := Acquire(dir)
	if err != nil {
		t.Fatalf("Acquire over a stale runtime file: %v", err)
	}
	defer func() { _ = owner.Release() }()
	if _, err := os.Stat(stale); !errors.Is(err, os.ErrNotExist) {
		t.Error("Acquire kept the stale runtime file")
	}
}

func TestFindWithNothingThere(t *testing.T) {
	info, err := Find(filepath.Join(t.TempDir(), "missing"))
	if err != nil || info != nil {
		t.Fatalf("Find = %+v, %v, want nothing", info, err)
	}
}

const helperEnv = "FTM_INSTANCE_HELPER_DIR"

// TestHelperOwner is not a test: TestAnOwnerThatDiesLeavesAStaleFile runs it in
// a child process that takes the lock and waits to be killed.
func TestHelperOwner(t *testing.T) {
	dir := os.Getenv(helperEnv)
	if dir == "" {
		t.Skip("helper process only")
	}
	lock, err := Acquire(dir)
	if err != nil {
		os.Stdout.WriteString("error " + err.Error() + "\n")
		os.Exit(1)
	}
	if err := lock.Publish(40503); err != nil {
		os.Exit(1)
	}
	os.Stdout.WriteString("ready\n")
	time.Sleep(time.Minute)
	os.Exit(0)
}

func TestAnOwnerThatDiesLeavesAStaleFile(t *testing.T) {
	if testing.Short() {
		t.Skip("spawns a process")
	}
	dir := t.TempDir()

	cmd := exec.Command(os.Args[0], "-test.run=^TestHelperOwner$")
	cmd.Env = append(os.Environ(), helperEnv+"="+dir)
	out, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cmd.Process.Kill(); _ = cmd.Wait() })

	line, err := bufio.NewReader(out).ReadString('\n')
	if err != nil || line != "ready\n" {
		t.Fatalf("helper said %q, %v", line, err)
	}

	info, err := Find(dir)
	if err != nil || info == nil || info.PID != cmd.Process.Pid {
		t.Fatalf("Find = %+v, %v, want the helper (pid %s)", info, err, strconv.Itoa(cmd.Process.Pid))
	}
	if _, err := Acquire(dir); !errors.As(err, new(*RunningError)) {
		t.Fatalf("Acquire while the helper runs = %v, want a *RunningError", err)
	}

	// Killing it skips Release, as a crash would.
	if err := cmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	_ = cmd.Wait()

	if _, err := os.Stat(filepath.Join(dir, runtimeName)); err != nil {
		t.Fatalf("the killed helper should have left its runtime file: %v", err)
	}
	if info, err := Find(dir); err != nil || info != nil {
		t.Fatalf("Find = %+v, %v after the owner died, want it treated as stale", info, err)
	}

	owner, err := Acquire(dir)
	if err != nil {
		t.Fatalf("Acquire after the owner died: %v", err)
	}
	_ = owner.Release()
}
