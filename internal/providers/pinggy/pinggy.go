package pinggy

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"strings"

	"foundry-tunnel/internal/config"
	"foundry-tunnel/internal/providers"
)

type PinggyProvider struct{}

func New() providers.Provider {
	return &PinggyProvider{}
}

func (p *PinggyProvider) Name() string {
	return "pinggy"
}

func (p *PinggyProvider) BinaryName() string {
	return "ssh"
}

func (p *PinggyProvider) InstallURL() string {
	return ""
}

func (p *PinggyProvider) RequiresAuth() bool {
	return false
}

func (p *PinggyProvider) FindBinary() string {
	if path, err := exec.LookPath("ssh"); err == nil {
		return path
	}
	return ""
}

func (p *PinggyProvider) Start(ctx context.Context, tunnel config.TunnelConfig, logWriter io.Writer) (*providers.Process, error) {
	binary := p.FindBinary()
	if binary == "" {
		return nil, fmt.Errorf("ssh not found. Install OpenSSH")
	}

	ctx, cancel := context.WithCancel(ctx)

	args := []string{
		"-p", "443",
		"-o", "StrictHostKeyChecking=no",
		"-o", "ServerAliveInterval=30",
		"-R", fmt.Sprintf("0:localhost:%d", tunnel.LocalPort),
		"a.pinggy.io",
	}

	if len(tunnel.CustomArgs) > 0 {
		args = append(args, tunnel.CustomArgs...)
	}

	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Stdout = logWriter
	cmd.Stderr = logWriter

	if err := cmd.Start(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to start pinggy tunnel: %w", err)
	}

	return &providers.Process{
		Cancel: cancel,
	}, nil
}

var ansiEscape = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

func stripANSI(s string) string {
	return ansiEscape.ReplaceAllString(s, "")
}

var pinggyRegex = regexp.MustCompile(`https?://[a-z0-9]+\.pinggy\.io`)

func (p *PinggyProvider) ParseURL(line string) string {
	clean := stripANSI(line)
	cleanLower := strings.ToLower(clean)

	matches := pinggyRegex.FindStringSubmatch(cleanLower)
	if len(matches) > 0 {
		return matches[0]
	}

	if strings.Contains(cleanLower, "pinggy.io") {
		if idx := strings.Index(cleanLower, "https://"); idx != -1 {
			rest := clean[idx:]
			if endIdx := strings.IndexAny(rest, " \t\n\r,"); endIdx != -1 {
				return rest[:endIdx]
			}
			return rest
		}
	}

	return ""
}

func (p *PinggyProvider) IsReady(line string) bool {
	clean := stripANSI(line)
	cleanLower := strings.ToLower(clean)

	return strings.Contains(cleanLower, "pinggy.io") ||
		strings.Contains(cleanLower, "connected")
}
