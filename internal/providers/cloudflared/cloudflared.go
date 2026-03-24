package cloudflared

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

	"github.com/deadbryam/ftm/internal/config"
	"github.com/deadbryam/ftm/internal/providers"
)

type CloudflaredProvider struct {
	installer *Installer
}

func New() providers.Provider {
	configDir, _ := os.UserHomeDir()
	if configDir == "" {
		configDir = "."
	}
	baseDir := filepath.Join(configDir, ".config", "foundry-tunnel", "bin")

	return &CloudflaredProvider{
		installer: NewInstaller(baseDir),
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

func (p *CloudflaredProvider) Start(ctx context.Context, tunnel config.TunnelConfig, logWriter io.Writer) (*providers.Process, error) {
	binary := p.FindBinary()
	if binary == "" {
		if err := p.installer.Install(nil); err != nil {
			return nil, fmt.Errorf("failed to install cloudflared: %w", err)
		}
		binary = p.installer.CloudflaredBin()
	}

	ctx, cancel := context.WithCancel(ctx)

	args := []string{
		"tunnel",
		"--url", fmt.Sprintf("http://localhost:%d", tunnel.LocalPort),
	}
	if len(tunnel.CustomArgs) > 0 {
		args = append(args, tunnel.CustomArgs...)
	}

	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Stdout = logWriter
	cmd.Stderr = logWriter

	if err := cmd.Start(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to start cloudflared: %w", err)
	}

	return &providers.Process{
		Cancel: cancel,
	}, nil
}

var cloudflareURLRegex = regexp.MustCompile(`https?://[a-zA-Z0-9-]+\.trycloudflare\.com`)

func (p *CloudflaredProvider) ParseURL(line string) string {
	matches := cloudflareURLRegex.FindStringSubmatch(line)
	if len(matches) > 0 {
		return matches[0]
	}

	lineLower := strings.ToLower(line)
	if idx := strings.Index(lineLower, "https://"); idx != -1 {
		rest := line[idx:]
		if endIdx := strings.IndexAny(rest, " \t\n\r"); endIdx != -1 {
			rest = rest[:endIdx]
		}
		if strings.Contains(strings.ToLower(rest), "trycloudflare.com") {
			return rest
		}
	}

	return ""
}

func (p *CloudflaredProvider) IsReady(line string) bool {
	line = strings.ToLower(line)
	return strings.Contains(line, "trycloudflare.com") ||
		strings.Contains(line, "started tunnel")
}
