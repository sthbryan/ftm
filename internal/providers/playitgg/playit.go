package playitgg

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

type PlayitggProvider struct {
	installer *providers.Installer
}

func New() providers.Provider {
	return &PlayitggProvider{
		installer: providers.NewInstaller(),
	}
}

func (p *PlayitggProvider) Name() string {
	return "Playit.gg"
}

func (p *PlayitggProvider) BinaryName() string {
	if runtime.GOOS == "windows" {
		return "playit.exe"
	}
	return "playit"
}

func (p *PlayitggProvider) InstallURL() string {
	return "https://playit.gg/download"
}

func (p *PlayitggProvider) RequiresAuth() bool {
	return false
}

func (p *PlayitggProvider) FindBinary() string {
	if path, err := exec.LookPath(p.BinaryName()); err == nil {
		return path
	}
	
	home, _ := os.UserHomeDir()
	candidates := []string{
		filepath.Join(p.installer.BinDir(), p.BinaryName()),
		filepath.Join(home, ".local", "bin", p.BinaryName()),
		filepath.Join(home, "bin", p.BinaryName()),
		"/usr/local/bin/" + p.BinaryName(),
		"./" + p.BinaryName(),
	}
	
	if runtime.GOOS == "windows" {
		candidates = append(candidates, 
			filepath.Join(os.Getenv("LOCALAPPDATA"), "playit", p.BinaryName()),
			filepath.Join(os.Getenv("PROGRAMFILES"), "playit", p.BinaryName()),
		)
	}
	
	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	
	return ""
}

func (p *PlayitggProvider) Start(ctx context.Context, tunnel config.TunnelConfig, logWriter io.Writer) (*providers.Process, error) {
	binary := p.FindBinary()
	if binary == "" {
		var err error
		binary, err = p.installer.EnsureInstalled(p)
		if err != nil {
			return nil, fmt.Errorf("failed to install playit: %w", err)
		}
	}

	ctx, cancel := context.WithCancel(ctx)
	
	args := []string{"--local", fmt.Sprintf("localhost:%d", tunnel.LocalPort)}
	if len(tunnel.CustomArgs) > 0 {
		args = append(args, tunnel.CustomArgs...)
	}
	
	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Stdout = logWriter
	cmd.Stderr = logWriter
	
	if err := cmd.Start(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to start playit: %w", err)
	}

	return &providers.Process{
		Cancel: cancel,
	}, nil
}

func (p *PlayitggProvider) ParseURL(line string) string {
	line = strings.ToLower(line)
	
	if idx := strings.Index(line, "https://"); idx != -1 {
		rest := line[idx:]
		if endIdx := strings.IndexFunc(rest, func(r rune) bool {
			return r == ' ' || r == '\t' || r == '\n' || r == '\r'
		}); endIdx != -1 {
			rest = rest[:endIdx]
		}
		if strings.Contains(rest, "playit.gg") || strings.Contains(rest, "playit") {
			return rest
		}
	}
	
	return ""
}

func (p *PlayitggProvider) IsReady(line string) bool {
	line = strings.ToLower(line)
	return strings.Contains(line, "connected") || 
		   strings.Contains(line, "tunnel ready") ||
		   strings.Contains(line, "playit.gg")
}
