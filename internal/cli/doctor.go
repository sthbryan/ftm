package cli

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"time"

	"github.com/spf13/cobra"

	"github.com/sthbryan/ftm/internal/core"
	"github.com/sthbryan/ftm/internal/i18n"
)

// DoctorEnv is what doctor looks at on this computer. Tests swap it.
type DoctorEnv struct {
	GOOS     string
	LookPath func(string) (string, error)
	HomeDir  func() (string, error)
	Stat     func(string) (os.FileInfo, error)
	// Writable reports whether files can be created in dir.
	Writable func(dir string) error
	// Answers reports whether something accepts connections on the local
	// port.
	Answers func(port int) bool
}

func realDoctorEnv() *DoctorEnv {
	return &DoctorEnv{
		GOOS:     runtime.GOOS,
		LookPath: exec.LookPath,
		HomeDir:  os.UserHomeDir,
		Stat:     os.Stat,
		Writable: func(dir string) error {
			f, err := os.CreateTemp(dir, ".ftm-doctor-*")
			if err != nil {
				return err
			}
			name := f.Name()
			_ = f.Close()
			return os.Remove(name)
		},
		Answers: func(port int) bool {
			conn, err := net.DialTimeout("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)), 500*time.Millisecond)
			if err != nil {
				return false
			}
			_ = conn.Close()
			return true
		},
	}
}

type checkStatus int

const (
	checkOK checkStatus = iota
	checkWarn
	checkFail
)

// check is one line of the doctor's report.
type check struct {
	status checkStatus
	label  string
	detail string
	hint   string
}

func (c check) String() string {
	mark := "✓"
	switch c.status {
	case checkWarn:
		mark = "!"
	case checkFail:
		mark = "✗"
	}
	s := fmt.Sprintf("%s %s: %s", mark, c.label, c.detail)
	if c.hint != "" && c.status != checkOK {
		s += "\n    → " + c.hint
	}
	return s
}

func newDoctorCmd(d *Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check that ftm can open connections on this computer",
		Long: "Check the tunnel programs, the ssh key, the Foundry port, the config folder and " +
			"start at login, with a fix for each problem.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			env := d.Doctor
			if env == nil {
				env = realDoctorEnv()
			}
			checks := runDoctor(cmd.Context(), d, env)
			failed := false
			for _, c := range checks {
				if err := println(cmd, c.String()); err != nil {
					return err
				}
				failed = failed || c.status == checkFail
			}
			if failed {
				return exitCode(1)
			}
			return nil
		},
	}
}

func runDoctor(ctx context.Context, d *Deps, env *DoctorEnv) []check {
	var checks []check

	checks = append(checks, configChecks(d, env)...)

	s, err := d.connect(false)
	if err != nil {
		checks = append(checks, check{checkFail, i18n.T("cli_doctor_ftm"), err.Error(), i18n.T("cli_doctor_ftm_hint")})
		return append(checks, sshChecks(env)...)
	}
	defer s.Close()

	if s.client != nil {
		checks = append(checks, check{checkOK, i18n.T("cli_doctor_ftm"), i18n.TF("cli_doctor_ftm_running", s.client.URL()), ""})
	} else {
		checks = append(checks, check{checkOK, i18n.T("cli_doctor_ftm"), i18n.T("cli_doctor_ftm_idle"), ""})
	}

	checks = append(checks, binaryChecks(ctx, s)...)
	checks = append(checks, sshChecks(env)...)
	checks = append(checks, foundryCheck(ctx, s, env))
	checks = append(checks, autostartCheck(ctx, s))
	return checks
}

func configChecks(d *Deps, env *DoctorEnv) []check {
	label := i18n.T("cli_doctor_config_dir")
	dir := d.Dirs.Config
	info, err := env.Stat(dir)
	switch {
	case errors.Is(err, os.ErrNotExist):
		return []check{{checkWarn, label, i18n.TF("cli_doctor_config_missing", dir), i18n.T("cli_doctor_config_missing_hint")}}
	case err != nil:
		return []check{{checkFail, label, err.Error(), i18n.TF("cli_doctor_config_perm_hint", dir)}}
	case !info.IsDir():
		return []check{{checkFail, label, i18n.TF("cli_doctor_config_not_dir", dir), i18n.TF("cli_doctor_config_not_dir_hint", dir)}}
	}
	if err := env.Writable(dir); err != nil {
		return []check{{checkFail, label, i18n.TF("cli_doctor_config_readonly", dir), i18n.TF("cli_doctor_config_perm_hint", dir)}}
	}
	checks := []check{{checkOK, label, dir, ""}}

	fileLabel := i18n.T("cli_doctor_config_file")
	if _, err := env.Stat(d.Dirs.ConfigFile); errors.Is(err, os.ErrNotExist) {
		return checks
	}
	if d.Peek != nil {
		if _, err := d.Peek(); err != nil {
			return append(checks, check{checkFail, fileLabel, err.Error(), i18n.TF("cli_doctor_config_file_hint", d.Dirs.ConfigFile)})
		}
	}
	return append(checks, check{checkOK, fileLabel, d.Dirs.ConfigFile, ""})
}

func binaryChecks(ctx context.Context, s *session) []check {
	report, err := s.Providers(ctx, false)
	label := i18n.T("cli_doctor_binaries")
	if err != nil {
		return []check{{checkFail, label, err.Error(), i18n.T("cli_doctor_ftm_hint")}}
	}
	var checks []check
	for _, p := range report {
		if isSSHProvider(p.ID) || p.Source == string(core.SourceSystem) {
			// These run on ssh, which sshChecks covers.
			continue
		}
		name := i18n.ProviderText(p.ID)
		switch {
		case p.Installed:
			checks = append(checks, check{checkOK, name, fmt.Sprintf("%s (%s)", dash(p.Version), sourceLabel(p.Source)), ""})
		default:
			checks = append(checks, check{checkWarn, name, i18n.T("provider_not_installed"), i18n.TF("cli_doctor_binary_hint", p.ID)})
		}
	}
	return checks
}

func isSSHProvider(id string) bool {
	return id == string(core.ProviderLocalhostRun) || id == string(core.ProviderServeo)
}

func sshChecks(env *DoctorEnv) []check {
	label := i18n.T("cli_doctor_ssh")
	if _, err := env.LookPath("ssh"); err != nil {
		hint := i18n.T("cli_doctor_ssh_hint_linux")
		switch env.GOOS {
		case "windows":
			hint = i18n.T("cli_doctor_ssh_hint_windows")
		case "darwin":
			hint = i18n.T("cli_doctor_ssh_hint_mac")
		}
		return []check{{checkWarn, label, i18n.T("cli_doctor_ssh_missing"), hint}}
	}
	checks := []check{{checkOK, label, i18n.T("cli_doctor_ssh_found"), ""}}

	keyLabel := i18n.T("cli_doctor_ssh_key")
	home, err := env.HomeDir()
	if err != nil {
		return append(checks, check{checkWarn, keyLabel, err.Error(), ""})
	}
	key := filepath.Join(home, ".ssh", "id_rsa")
	if _, err := env.Stat(key); err != nil {
		return append(checks, check{checkWarn, keyLabel, i18n.TF("cli_doctor_ssh_key_missing", key),
			i18n.TF("cli_doctor_ssh_key_hint", fmt.Sprintf(`ssh-keygen -t rsa -b 2048 -f "%s" -N ""`, key))})
	}
	return append(checks, check{checkOK, keyLabel, key, ""})
}

func foundryCheck(ctx context.Context, s *session, env *DoctorEnv) check {
	label := i18n.T("cli_doctor_foundry")
	ports := []int{defaultFoundryPort}
	if tunnels, err := s.Tunnels(ctx); err == nil {
		for _, t := range tunnels {
			if !slices.Contains(ports, t.Port) {
				ports = append(ports, t.Port)
			}
		}
	}
	for _, p := range ports {
		if env.Answers(p) {
			return check{checkOK, label, i18n.TF("cli_doctor_foundry_ok", p), ""}
		}
	}
	list := ""
	for i, p := range ports {
		if i > 0 {
			list += ", "
		}
		list += strconv.Itoa(p)
	}
	return check{checkWarn, label, i18n.TF("cli_doctor_foundry_silent", list), i18n.T("cli_doctor_foundry_hint")}
}

func autostartCheck(ctx context.Context, s *session) check {
	label := i18n.T("cli_doctor_autostart")
	settings, err := s.Settings(ctx)
	switch {
	case err != nil:
		return check{checkFail, label, err.Error(), i18n.T("cli_doctor_autostart_hint")}
	case !settings.AutostartSupported:
		return check{checkOK, label, i18n.T("cli_autostart_desktop_only"), ""}
	case settings.AutostartEnabled:
		return check{checkOK, label, i18n.T("cli_doctor_autostart_on"), ""}
	}
	return check{checkOK, label, i18n.T("cli_doctor_autostart_off"), ""}
}
