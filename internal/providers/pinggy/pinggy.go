package pinggy

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"foundry-tunnel/internal/config"
	"foundry-tunnel/internal/providers"
)

type PinggyCliProvider struct {
	installer *Installer
}

func New() providers.Provider {
	configDir, _ := os.UserHomeDir()
	if configDir == "" {
		configDir = "."
	}
	baseDir := filepath.Join(configDir, ".config", "foundry-tunnel", "bin")

	return &PinggyCliProvider{
		installer: NewInstaller(baseDir),
	}
}

func (p *PinggyCliProvider) Name() string {
	return "pinggy"
}

func (p *PinggyCliProvider) BinaryName() string {
	return "pinggy"
}

func (p *PinggyCliProvider) InstallURL() string {
	return "https://pinggy.io/cli/"
}

func (p *PinggyCliProvider) RequiresAuth() bool {
	return false
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
	if len(tunnel.CustomArgs) > 0 {
		args = append(args, tunnel.CustomArgs...)
	}

	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Stdout = logWriter
	cmd.Stderr = logWriter

	if err := cmd.Start(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to start pinggy: %w", err)
	}

	return &providers.Process{
		Cancel: cancel,
	}, nil
}

var pinggyRegex = regexp.MustCompile(`https?://[a-z0-9-]+\.[a-z]+\.pinggy\.(io|link)`)

func (p *PinggyCliProvider) ParseURL(line string) string {
	lineLower := strings.ToLower(line)

	if strings.Contains(lineLower, "dashboard.pinggy.io") {
		return ""
	}

	matches := pinggyRegex.FindStringSubmatch(lineLower)
	if len(matches) > 0 {
		return matches[0]
	}

	if strings.Contains(lineLower, "pinggy.link") ||
		(strings.Contains(lineLower, ".pinggy.io") && !strings.Contains(lineLower, "dashboard")) {
		if idx := strings.Index(lineLower, "https://"); idx != -1 {
			rest := line[idx:]
			if endIdx := strings.IndexAny(rest, " \t\n\r,"); endIdx != -1 {
				return rest[:endIdx]
			}
			return rest
		}
		if idx := strings.Index(lineLower, "http://"); idx != -1 {
			rest := line[idx:]
			if endIdx := strings.IndexAny(rest, " \t\n\r,"); endIdx != -1 {
				return rest[:endIdx]
			}
			return rest
		}
	}

	return ""
}

func (p *PinggyCliProvider) IsReady(line string) bool {
	lineLower := strings.ToLower(line)

	return strings.Contains(lineLower, "pinggy.link") ||
		(strings.Contains(lineLower, ".pinggy.io") && !strings.Contains(lineLower, "dashboard")) ||
		strings.Contains(lineLower, "connected")
}
