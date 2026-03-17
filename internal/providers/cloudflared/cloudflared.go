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

	"foundry-tunnel/internal/config"
	"foundry-tunnel/internal/providers"
)

type CloudflaredProvider struct {
	installer *providers.Installer
}

func New() providers.Provider {
	return &CloudflaredProvider{
		installer: providers.NewInstaller(),
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

func (p *CloudflaredProvider) InstallURL() string {
	return "https://developers.cloudflare.com/cloudflare-one/connections/connect-networks/downloads/"
}

func (p *CloudflaredProvider) RequiresAuth() bool {
	return false
}

func (p *CloudflaredProvider) FindBinary() string {
	if path, err := exec.LookPath(p.BinaryName()); err == nil {
		return path
	}
	
	home, _ := os.UserHomeDir()
	candidates := []string{
		filepath.Join(p.installer.BinDir(), p.BinaryName()),
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
		var err error
		binary, err = p.installer.EnsureInstalled(p)
		if err != nil {
			return nil, fmt.Errorf("failed to install cloudflared: %w", err)
		}
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

func (p *CloudflaredProvider) ParseURL(line string) string {
	line = strings.ToLower(line)
	
	if idx := strings.Index(line, "https://"); idx != -1 {
		rest := line[idx:]
		if endIdx := strings.IndexFunc(rest, func(r rune) bool {
			return r == ' ' || r == '\t' || r == '\n' || r == '\r'
		}); endIdx != -1 {
			rest = rest[:endIdx]
		}
		if strings.Contains(rest, "trycloudflare.com") {
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
