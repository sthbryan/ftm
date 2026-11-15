package pinggy

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.com/sthbryan/ftm/internal/config"
	"github.com/sthbryan/ftm/internal/providers"
)

type PinggyCliProvider struct {
	installer *Installer
}

func New() providers.Provider {
	return &PinggyCliProvider{
		installer: NewInstaller(),
	}
}

func (p *PinggyCliProvider) Name() string {
	return "pinggy"
}

func (p *PinggyCliProvider) BinaryName() string {
	return "pinggy"
}

func (p *PinggyCliProvider) IsInstalled() bool {
	if _, err := exec.LookPath("pinggy"); err == nil {
		return true
	}
	return p.installer.IsInstalled()
}

func (p *PinggyCliProvider) Install(progress chan<- providers.DownloadProgress) error {
	return p.installer.Install(progress)
}

func (p *PinggyCliProvider) FindBinary() string {
	if path, err := exec.LookPath("pinggy"); err == nil {
		return path
	}

	if p.installer.IsInstalled() {
		return p.installer.PinggyBin()
	}

	return ""
}

func (p *PinggyCliProvider) Start(ctx context.Context, tunnel config.TunnelConfig, logWriter io.Writer) (*providers.Process, error) {
	binary := p.FindBinary()
	if binary == "" {
		return nil, fmt.Errorf("installing")
	}

	ctx, cancel := context.WithCancel(ctx)

	args := []string{
		"-l", fmt.Sprintf("http://localhost:%d", tunnel.LocalPort),
	}

	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Stdout = logWriter
	cmd.Stderr = logWriter

	proc, err := providers.StartProcess(cmd, cancel)
	if err != nil {
		return nil, fmt.Errorf("failed to start pinggy: %w", err)
	}

	return proc, nil
}

var pinggyReserved = map[string]bool{
	"dashboard": true,
	"docs":      true,
	"www":       true,
}

func (p *PinggyCliProvider) ParseURL(line string) string {
	return providers.ExtractURL(line, isPinggyTunnelHost)
}

func isPinggyTunnelHost(host string) bool {
	labels := strings.Split(host, ".")
	if len(labels) < 3 || pinggyReserved[labels[0]] {
		return false
	}

	for _, label := range labels[1 : len(labels)-1] {
		if strings.HasPrefix(label, "pinggy") {
			return true
		}
	}

	return false
}

func (p *PinggyCliProvider) BundledBinary() string {
	return p.installer.PinggyBin()
}

func (p *PinggyCliProvider) LatestRelease(ctx context.Context) (*providers.Release, error) {
	return p.installer.LatestRelease(ctx)
}

func (p *PinggyCliProvider) InstallRelease(ctx context.Context, rel *providers.Release, progress chan<- providers.DownloadProgress) error {
	return p.installer.InstallRelease(ctx, rel, progress)
}
