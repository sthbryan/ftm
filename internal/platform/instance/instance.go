// Package instance keeps one ftm process per user in charge of the tunnels.
//
// The owner holds an exclusive OS lock on <runtime>/ftm.lock for as long as it
// runs, and publishes <runtime>/ftm.json with its port and a random token once
// its server listens. Another process finds it through Find or a failed
// Acquire and attaches to it instead of starting its own manager. The OS drops
// the lock when the owner exits, however it exits, so a runtime file whose lock
// is free belongs to a process that is gone and is treated as stale.
package instance

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	lockName    = "ftm.lock"
	runtimeName = "ftm.json"
)

// errLocked is what tryLock returns when another handle holds the lock.
var errLocked = errors.New("instance lock is held")

// Info is what the owner publishes in the runtime file.
type Info struct {
	PID     int       `json:"pid"`
	Port    int       `json:"port"`
	Token   string    `json:"token"`
	Started time.Time `json:"started"`
}

// URL is the owner's local HTTP address.
func (i Info) URL() string { return fmt.Sprintf("http://127.0.0.1:%d", i.Port) }

// RunningError is returned by Acquire when another process owns the lock.
// Info is nil when that process has not published its runtime file yet.
type RunningError struct {
	Info *Info
}

func (e *RunningError) Error() string {
	if e.Info == nil {
		return "Foundry Tunnel Manager is starting in another process"
	}
	return fmt.Sprintf("Foundry Tunnel Manager is already running at %s (pid %d)", e.Info.URL(), e.Info.PID)
}

// Lock is the held single-instance lock.
type Lock struct {
	dir   string
	file  *os.File
	token string
}

// publishWait is how long a second process waits for the owner's runtime file
// when it catches the owner between taking the lock and listening.
var publishWait = 3 * time.Second

// Acquire takes the single-instance lock in dir. When another process holds
// it, it returns a *RunningError describing that process.
func Acquire(dir string) (*Lock, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("failed to create runtime dir: %w", err)
	}

	f, err := os.OpenFile(filepath.Join(dir, lockName), os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("failed to open instance lock: %w", err)
	}

	if err := tryLock(f); err != nil {
		_ = f.Close()
		if !errors.Is(err, errLocked) {
			return nil, fmt.Errorf("failed to take instance lock: %w", err)
		}
		return nil, &RunningError{Info: waitForInfo(dir)}
	}

	// We own the lock, so any runtime file left here is from a process that
	// died without cleaning up.
	_ = os.Remove(filepath.Join(dir, runtimeName))

	token, err := newToken()
	if err != nil {
		_ = unlock(f)
		_ = f.Close()
		return nil, err
	}

	return &Lock{dir: dir, file: f, token: token}, nil
}

// Token is the secret this instance publishes for its clients.
func (l *Lock) Token() string { return l.token }

// Publish writes the runtime file so other processes can reach this one.
func (l *Lock) Publish(port int) error {
	info := Info{PID: os.Getpid(), Port: port, Token: l.token, Started: time.Now().UTC()}
	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return err
	}

	tmp, err := os.CreateTemp(l.dir, runtimeName+".tmp*")
	if err != nil {
		return fmt.Errorf("failed to write runtime file: %w", err)
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to write runtime file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to write runtime file: %w", err)
	}
	if err := os.Chmod(tmpPath, 0o600); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to write runtime file: %w", err)
	}
	if err := os.Rename(tmpPath, filepath.Join(l.dir, runtimeName)); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to write runtime file: %w", err)
	}
	return nil
}

// Release removes the runtime file and drops the lock. It is safe to call
// more than once.
func (l *Lock) Release() error {
	if l == nil || l.file == nil {
		return nil
	}

	err := os.Remove(filepath.Join(l.dir, runtimeName))
	if errors.Is(err, os.ErrNotExist) {
		err = nil
	}

	// The lock file itself stays: removing it would let a waiting process lock
	// a file that a newer one then recreates under a different inode.
	err = errors.Join(err, unlock(l.file), l.file.Close())
	l.file = nil
	return err
}

// Find reports the running owner in dir, or nil when there is none. A runtime
// file left by a process that is gone is removed.
func Find(dir string) (*Info, error) {
	f, err := os.OpenFile(filepath.Join(dir, lockName), os.O_RDWR, 0)
	if errors.Is(err, os.ErrNotExist) {
		_ = os.Remove(filepath.Join(dir, runtimeName))
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to open instance lock: %w", err)
	}
	defer f.Close()

	if err := tryLock(f); err != nil {
		if !errors.Is(err, errLocked) {
			return nil, fmt.Errorf("failed to check instance lock: %w", err)
		}
		return readInfo(dir), nil
	}

	_ = os.Remove(filepath.Join(dir, runtimeName))
	_ = unlock(f)
	return nil, nil
}

func waitForInfo(dir string) *Info {
	deadline := time.Now().Add(publishWait)
	for {
		if info := readInfo(dir); info != nil {
			return info
		}
		if time.Now().After(deadline) {
			return nil
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func readInfo(dir string) *Info {
	data, err := os.ReadFile(filepath.Join(dir, runtimeName))
	if err != nil {
		return nil
	}
	var info Info
	if err := json.Unmarshal(data, &info); err != nil || info.Port <= 0 {
		return nil
	}
	return &info
}

func newToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate instance token: %w", err)
	}
	return hex.EncodeToString(b), nil
}
