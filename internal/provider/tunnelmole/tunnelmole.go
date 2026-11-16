package tunnelmole

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/sthbryan/ftm/internal/core"
	"github.com/sthbryan/ftm/internal/provider"
)

var ErrRosettaRequired = errors.New("tunnelmole requires Rosetta 2 to run on Apple Silicon. Install it with: softwareupdate --install-rosetta")

func RosettaInstalled() bool {
	_, err := os.Stat("/Library/Apple/usr/libexec/oah")
	return err == nil
}

func needsRosetta(err string) bool {
	return strings.Contains(err, "bad CPU type") || strings.Contains(err, "executable")
}

type TunnelmoleProvider struct {
	installer *Installer
}

func New() provider.Provider {
	return &TunnelmoleProvider{
		installer: NewInstaller(),
	}
}

func (p *TunnelmoleProvider) Name() string {
	return "tunnelmole"
}

func (p *TunnelmoleProvider) BinaryName() string {
	return "tunnelmole"
}

func (p *TunnelmoleProvider) IsInstalled() bool {

	if _, err := exec.LookPath("tmole"); err == nil {
		return true
	}
	if _, err := exec.LookPath("tunnelmole"); err == nil {
		return true
	}

	return p.installer.IsInstalled()
}

func (p *TunnelmoleProvider) Install(progress chan<- provider.DownloadProgress) error {
	return p.installer.Install(progress)
}

func (p *TunnelmoleProvider) FindBinary() string {

	if path, err := exec.LookPath("tmole"); err == nil {
		return path
	}
	if path, err := exec.LookPath("tunnelmole"); err == nil {
		return path
	}

	if p.installer.IsInstalled() {
		return p.installer.TunnelmoleBin()
	}

	home, _ := os.UserHomeDir()
	candidates := []string{
		filepath.Join(home, ".npm-global", "bin", "tmole"),
		filepath.Join(home, ".npm-global", "bin", "tunnelmole"),
		"/usr/local/bin/tmole",
		"/usr/local/bin/tunnelmole",
	}

	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return ""
}

func (p *TunnelmoleProvider) Start(ctx context.Context, tunnel core.TunnelConfig, logWriter io.Writer) (*provider.Process, error) {
	binary := p.FindBinary()
	if binary == "" {
		return nil, fmt.Errorf("installing")
	}

	baseCtx := ctx
	ctx, cancel := context.WithCancel(baseCtx)

	args := []string{fmt.Sprintf("%d", tunnel.LocalPort)}

	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Stdout = logWriter
	cmd.Stderr = logWriter

	proc, err := provider.StartProcess(cmd, cancel)
	if err == nil {
		return proc, nil
	}

	if !needsRosetta(err.Error()) || runtime.GOOS != "darwin" || runtime.GOARCH != "arm64" {
		return nil, fmt.Errorf("failed to start tunnelmole: %w", err)
	}

	if !RosettaInstalled() {
		return nil, ErrRosettaRequired
	}

	ctx, cancel = context.WithCancel(baseCtx)

	cmd = exec.CommandContext(ctx, "/Library/Apple/usr/libexec/oah", append([]string{"-r", binary}, args...)...)
	cmd.Stdout = logWriter
	cmd.Stderr = logWriter

	proc, err = provider.StartProcess(cmd, cancel)
	if err != nil {
		return nil, fmt.Errorf("failed to start tunnelmole: %w", err)
	}

	return proc, nil
}

func (p *TunnelmoleProvider) ParseURL(line string) string {
	return provider.ExtractURL(line, func(host string) bool {
		if strings.HasPrefix(host, "dashboard.") || strings.HasPrefix(host, "www.") {
			return false
		}
		return provider.IsSubdomainOf(host, "tunnelmole.net", "tunnelmole.com")
	})
}

func (p *TunnelmoleProvider) BundledBinary() string {
	return p.installer.TunnelmoleBin()
}

func (p *TunnelmoleProvider) LatestRelease(ctx context.Context) (*provider.Release, error) {
	return p.installer.LatestRelease(ctx)
}

func (p *TunnelmoleProvider) InstallRelease(ctx context.Context, rel *provider.Release, progress chan<- provider.DownloadProgress) error {
	return p.installer.InstallRelease(ctx, rel, progress)
}
