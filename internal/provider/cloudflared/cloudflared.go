package cloudflared

import (
	"context"
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

type CloudflaredProvider struct {
	installer *Installer
}

func New() provider.Provider {
	return &CloudflaredProvider{
		installer: NewInstaller(),
	}
}

func (p *CloudflaredProvider) Name() string {
	return "Cloudflare Tunnel"
}

func (p *CloudflaredProvider) BinaryName() string {
	if runtime.GOOS == "windows" {
		return "cloudflared.exe"
	}
	return "cloudflared"
}

func (p *CloudflaredProvider) FindBinary() string {
	if path, err := exec.LookPath(p.BinaryName()); err == nil {
		return path
	}

	if p.installer.IsInstalled() {
		return p.installer.CloudflaredBin()
	}

	home, _ := os.UserHomeDir()
	candidates := []string{
		filepath.Join(home, ".cloudflared", p.BinaryName()),
		"/usr/local/bin/" + p.BinaryName(),
		"/usr/bin/" + p.BinaryName(),
		"./" + p.BinaryName(),
	}

	if runtime.GOOS == "windows" {
		candidates = append(candidates,
			filepath.Join(os.Getenv("ProgramFiles"), "cloudflared", p.BinaryName()),
		)
	}

	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return ""
}

func (p *CloudflaredProvider) Start(ctx context.Context, tunnel core.TunnelConfig, logWriter io.Writer) (*provider.Process, error) {
	binary := p.FindBinary()
	if binary == "" {
		return nil, fmt.Errorf("installing")
	}

	ctx, cancel := context.WithCancel(ctx)

	args := []string{
		"tunnel",
		"--url", fmt.Sprintf("http://localhost:%d", tunnel.LocalPort),
	}

	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Stdout = logWriter
	cmd.Stderr = logWriter

	proc, err := provider.StartProcess(cmd, cancel)
	if err != nil {
		return nil, fmt.Errorf("failed to start cloudflared: %w", err)
	}

	return proc, nil
}

func (p *CloudflaredProvider) ParseURL(line string) string {
	if strings.Contains(line, "dest=") {
		return ""
	}

	return provider.ExtractURL(line, func(host string) bool {
		return provider.IsSubdomainOf(host, "trycloudflare.com")
	})
}

func (p *CloudflaredProvider) IsInstalled() bool {
	return p.installer.IsInstalled()
}

func (p *CloudflaredProvider) Install(progress chan<- provider.DownloadProgress) error {
	return p.installer.Install(progress)
}

func (p *CloudflaredProvider) BundledBinary() string {
	return p.installer.CloudflaredBin()
}

func (p *CloudflaredProvider) LatestRelease(ctx context.Context) (*provider.Release, error) {
	return p.installer.LatestRelease(ctx)
}

func (p *CloudflaredProvider) InstallRelease(ctx context.Context, rel *provider.Release, progress chan<- provider.DownloadProgress) error {
	return p.installer.InstallRelease(ctx, rel, progress)
}
