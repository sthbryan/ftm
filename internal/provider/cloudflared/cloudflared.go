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

	config, err := emptyConfig()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to prepare cloudflared: %w", err)
	}
	args := quickTunnelArgs(config, tunnel.LocalPort)

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

// quickTunnelArgs runs a quick tunnel to the local port. --config points at an
// empty file so a config.yml the user keeps for their own named tunnels
// (~/.cloudflared, /etc/cloudflared) is never loaded: its ingress rules would
// not know the trycloudflare host and every request would get a 404. The
// target is 127.0.0.1 because ftm's proxy listens on IPv4 only, and
// "localhost" can resolve to ::1 first.
func quickTunnelArgs(config string, port int) []string {
	return []string{
		"tunnel",
		"--config", config,
		"--url", fmt.Sprintf("http://127.0.0.1:%d", port),
	}
}

// emptyConfig returns the path of an empty cloudflared config file, creating
// it in the temp directory when it is missing.
func emptyConfig() (string, error) {
	path := filepath.Join(os.TempDir(), "ftm-cloudflared.yml")
	if _, err := os.Stat(path); err == nil {
		return path, nil
	}
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		return "", err
	}
	return path, nil
}
