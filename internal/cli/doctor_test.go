package cli

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sthbryan/ftm/internal/core"
)

type fakeInfo struct {
	fs.FileInfo
	dir bool
}

func (f fakeInfo) IsDir() bool { return f.dir }

func healthyEnv(home string, existing map[string]bool) *DoctorEnv {
	return &DoctorEnv{
		GOOS:     "windows",
		LookPath: func(string) (string, error) { return `C:\Windows\System32\OpenSSH\ssh.exe`, nil },
		HomeDir:  func() (string, error) { return home, nil },
		Stat: func(p string) (os.FileInfo, error) {
			if dir, ok := existing[p]; ok {
				return fakeInfo{dir: dir}, nil
			}
			return nil, os.ErrNotExist
		},
		Writable: func(string) error { return nil },
		Answers:  func(port int) bool { return port == 30000 },
	}
}

func TestDoctorAllGood(t *testing.T) {
	for _, m := range modes {
		t.Run(m.name, func(t *testing.T) {
			h := newHarness(t, m)
			key := filepath.Join("home", ".ssh", "id_rsa")
			h.deps.Doctor = healthyEnv("home", map[string]bool{h.deps.Dirs.Config: true, key: false})
			out := h.mustRun("doctor")
			for _, want := range []string{"✓ Config folder", "✓ ssh: found", "✓ ssh key", "✓ Foundry: answers on port 30000", "✓ Start at login"} {
				if !strings.Contains(out, want) {
					t.Errorf("doctor output lacks %q:\n%s", want, out)
				}
			}
			if m.attached && !strings.Contains(out, "✓ ftm: open, web view at") {
				t.Errorf("doctor doesn't say ftm is open:\n%s", out)
			}
		})
	}
}

func TestDoctorGivesAFixForEachProblem(t *testing.T) {
	h := newHarness(t, modes[0])
	env := healthyEnv("home", map[string]bool{h.deps.Dirs.Config: true, h.deps.Dirs.ConfigFile: false})
	env.LookPath = func(string) (string, error) { return "", errors.New("not found") }
	env.Answers = func(int) bool { return false }
	h.deps.Doctor = env
	h.deps.Peek = func() (core.Config, error) { return core.Config{}, errors.New("yaml: line 3: bad indentation") }

	code := h.run("doctor")
	out := h.out.String()
	if code != 1 {
		t.Fatalf("an unreadable config must fail doctor; code %d\n%s", code, out)
	}
	for _, want := range []string{
		"✗ Config file: yaml: line 3",
		"→ ftm can't read",
		"! ssh: not found",
		"Optional features",
		"! Foundry: nothing answers on port 30000",
		"Configuration > Port",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("doctor output lacks %q:\n%s", want, out)
		}
	}
}

func TestDoctorSSHHintPerPlatform(t *testing.T) {
	for goos, want := range map[string]string{"windows": "Optional features", "linux": "apt install", "darwin": "xcode-select"} {
		env := healthyEnv("home", nil)
		env.GOOS = goos
		env.LookPath = func(string) (string, error) { return "", errors.New("no") }
		checks := sshChecks(env)
		if len(checks) != 1 || !strings.Contains(checks[0].hint, want) {
			t.Errorf("%s: %+v", goos, checks)
		}
	}
}

func TestDoctorConfigFolder(t *testing.T) {
	h := newHarness(t, modes[0])
	env := healthyEnv("home", map[string]bool{})
	if c := configChecks(&h.deps, env); c[0].status != checkWarn {
		t.Errorf("missing folder: %+v", c)
	}
	env = healthyEnv("home", map[string]bool{h.deps.Dirs.Config: true})
	env.Writable = func(string) error { return errors.New("denied") }
	if c := configChecks(&h.deps, env); c[0].status != checkFail || c[0].hint == "" {
		t.Errorf("read-only folder: %+v", c)
	}
	env = healthyEnv("home", map[string]bool{h.deps.Dirs.Config: false})
	if c := configChecks(&h.deps, env); c[0].status != checkFail {
		t.Errorf("file in place of the folder: %+v", c)
	}
}
