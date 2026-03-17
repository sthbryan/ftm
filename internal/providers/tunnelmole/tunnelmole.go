package tunnelmole

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

type TunnelmoleProvider struct{}

func New() providers.Provider {
	return &TunnelmoleProvider{}
}

func (p *TunnelmoleProvider) Name() string {
	return "Tunnelmole"
}

func (p *TunnelmoleProvider) BinaryName() string {
	return "tunnelmole"
}

func (p *TunnelmoleProvider) InstallURL() string {
	return "https://tunnelmole.com/downloads"
}

func (p *TunnelmoleProvider) RequiresAuth() bool {
	return false
}

func (p *TunnelmoleProvider) FindBinary() string {
	if path, err := exec.LookPath("tmole"); err == nil {
		return path
	}
	if path, err := exec.LookPath("tunnelmole"); err == nil {
		return path
	}
	
	home, _ := os.UserHomeDir()
	candidates := []string{
		filepath.Join(home, ".npm-global", "bin", "tmole"),
		filepath.Join(home, ".npm-global", "bin", "tunnelmole"),
		"/usr/local/bin/tmole",
		"/usr/local/bin/tunnelmole",
	}
	
	if runtime.GOOS == "windows" {
		npmPath := os.Getenv("APPDATA")
		candidates = append(candidates,
			filepath.Join(npmPath, "npm", "tmole.cmd"),
			filepath.Join(npmPath, "npm", "tunnelmole.cmd"),
		)
	}
	
	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	
	return ""
}

func (p *TunnelmoleProvider) Start(ctx context.Context, tunnel config.TunnelConfig, logWriter io.Writer) (*providers.Process, error) {
	binary := p.FindBinary()
	if binary == "" {
		return nil, fmt.Errorf("tunnelmole not found. Install with: npm install -g tunnelmole")
	}

	ctx, cancel := context.WithCancel(ctx)
	
	args := []string{fmt.Sprintf("%d", tunnel.LocalPort)}
	if len(tunnel.CustomArgs) > 0 {
		args = append(args, tunnel.CustomArgs...)
	}
	
	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Stdout = logWriter
	cmd.Stderr = logWriter
	
	if err := cmd.Start(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to start tunnelmole: %w", err)
	}

	return &providers.Process{
		Cancel: cancel,
	}, nil
}

func (p *TunnelmoleProvider) ParseURL(line string) string {
	line = strings.ToLower(line)
	
	if idx := strings.Index(line, "https://"); idx != -1 {
		rest := line[idx:]
		if endIdx := strings.IndexFunc(rest, func(r rune) bool {
			return r == ' ' || r == '\t' || r == '\n' || r == '\r' || r == ')'
		}); endIdx != -1 {
			rest = rest[:endIdx]
		}
		if strings.Contains(rest, "tunnelmole") || strings.Contains(rest, "net") {
			return rest
		}
	}
	
	return ""
}

func (p *TunnelmoleProvider) IsReady(line string) bool {
	line = strings.ToLower(line)
	return strings.Contains(line, "tunnelmole") ||
		   strings.Contains(line, "your site is available at")
}
