package bore

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"

	"github.com/sthbryan/ftm/internal/config"
	"github.com/sthbryan/ftm/internal/providers"
)

type BoreProvider struct {
	installer *Installer
}

func New() providers.Provider {
	return &BoreProvider{
		installer: NewInstaller(),
	}
}

func (p *BoreProvider) Name() string {
	return "bore"
}

func (p *BoreProvider) BinaryName() string {
	if runtime.GOOS == "windows" {
		return "bore.exe"
	}
	return "bore"
}

func (p *BoreProvider) IsInstalled() bool {
	if _, err := exec.LookPath(p.BinaryName()); err == nil {
		return true
	}
	return p.installer.IsInstalled()
}

func (p *BoreProvider) Install(progress chan<- providers.DownloadProgress) error {
	return p.installer.Install(progress)
}

func (p *BoreProvider) FindBinary() string {
	if path, err := exec.LookPath(p.BinaryName()); err == nil {
		return path
	}

	if p.installer.IsInstalled() {
		return p.installer.BoreBin()
	}

	candidates := []string{
		"/usr/local/bin/" + p.BinaryName(),
		"/usr/bin/" + p.BinaryName(),
	}

	if runtime.GOOS == "windows" {
		candidates = []string{
			p.BinaryName(),
			filepath.Join(os.Getenv("ProgramFiles"), "bore", p.BinaryName()),
		}
	}

	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return ""
}

func (p *BoreProvider) Start(ctx context.Context, tunnel config.TunnelConfig, logWriter io.Writer) (*providers.Process, error) {
	binary := p.FindBinary()
	if binary == "" {
		if err := p.installer.Install(nil); err != nil {
			return nil, fmt.Errorf("failed to install bore: %w", err)
		}
		binary = p.installer.BoreBin()
	}

	ctx, cancel := context.WithCancel(ctx)

	args := []string{
		"local",
		fmt.Sprintf("%d", tunnel.LocalPort),
		"--to", "bore.pub",
	}

	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Stdout = logWriter
	cmd.Stderr = logWriter

	proc, err := providers.StartProcess(cmd, cancel)
	if err != nil {
		return nil, fmt.Errorf("failed to start bore: %w", err)
	}

	return proc, nil
}

var boreEndpointRegex = regexp.MustCompile(`bore\.pub:\d+`)

func (p *BoreProvider) ParseURL(line string) string {
	if endpoint := boreEndpointRegex.FindString(providers.StripANSI(line)); endpoint != "" {
		return "http://" + endpoint
	}

	return ""
}

func (p *BoreProvider) BundledBinary() string {
	return p.installer.BoreBin()
}

func (p *BoreProvider) LatestRelease(ctx context.Context) (*providers.Release, error) {
	return p.installer.LatestRelease(ctx)
}

func (p *BoreProvider) InstallRelease(ctx context.Context, rel *providers.Release, progress chan<- providers.DownloadProgress) error {
	return p.installer.InstallRelease(ctx, rel, progress)
}
