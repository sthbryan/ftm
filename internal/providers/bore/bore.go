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
	"strings"

	"github.com/sthbryan/ftm/internal/config"
	"github.com/sthbryan/ftm/internal/providers"
)

type BoreProvider struct {
	installer *Installer
}

func New() providers.Provider {
	configDir, _ := os.UserHomeDir()
	if configDir == "" {
		configDir = "."
	}
	baseDir := filepath.Join(configDir, ".config", "foundry-tunnel", "bin")

	return &BoreProvider{
		installer: NewInstaller(baseDir),
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

	home, _ := os.UserHomeDir()
	candidates := []string{
		filepath.Join(home, ".config", "foundry-tunnel", "bin", p.BinaryName()),
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

	if err := cmd.Start(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to start bore: %w", err)
	}

	return &providers.Process{
		Cancel: cancel,
	}, nil
}

var boreURLRegex = regexp.MustCompile(`bore\.pub:\d+`)

func (p *BoreProvider) ParseURL(line string) string {
	matches := boreURLRegex.FindString(line)
	if matches != "" {
		return matches
	}
	return ""
}

func (p *BoreProvider) IsReady(line string) bool {
	lineLower := strings.ToLower(line)
	return strings.Contains(lineLower, "bore.pub") &&
		(strings.Contains(lineLower, "listening") || strings.Contains(lineLower, "port"))
}
