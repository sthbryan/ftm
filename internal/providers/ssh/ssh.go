package ssh

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"

	"foundry-tunnel/internal/config"
	"foundry-tunnel/internal/providers"
)

type SSHProvider struct {
	host string
	name string
}

func NewLocalhostRun() providers.Provider {
	return &SSHProvider{
		host: "localhost.run",
		name: "localhost.run",
	}
}

func NewServeo() providers.Provider {
	return &SSHProvider{
		host: "serveo.net",
		name: "serveo",
	}
}

func (p *SSHProvider) Name() string {
	return p.name
}

func (p *SSHProvider) BinaryName() string {
	return "ssh"
}

func (p *SSHProvider) InstallURL() string {
	return ""
}

func (p *SSHProvider) RequiresAuth() bool {
	return false
}

func (p *SSHProvider) FindBinary() string {
	if path, err := exec.LookPath("ssh"); err == nil {
		return path
	}
	return ""
}

func (p *SSHProvider) Start(ctx context.Context, tunnel config.TunnelConfig, logWriter io.Writer) (*providers.Process, error) {
	binary := p.FindBinary()
	if binary == "" {
		return nil, fmt.Errorf("ssh not found. Install OpenSSH")
	}

	ctx, cancel := context.WithCancel(ctx)

	var args []string
	if p.host == "localhost.run" {
		args = []string{
			"-o", "StrictHostKeyChecking=no",
			"-o", "ServerAliveInterval=60",
			"-R", fmt.Sprintf("80:localhost:%d", tunnel.LocalPort),
			"nokey@localhost.run",
		}
	} else if p.host == "serveo.net" {
		args = []string{
			"-o", "StrictHostKeyChecking=no",
			"-o", "ServerAliveInterval=60",
			"-R", fmt.Sprintf("80:localhost:%d", tunnel.LocalPort),
			"serveo.net",
		}
	}

	if len(tunnel.CustomArgs) > 0 {
		args = append(args, tunnel.CustomArgs...)
	}

	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Stdout = logWriter
	cmd.Stderr = logWriter

	if err := cmd.Start(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to start ssh tunnel: %w", err)
	}

	return &providers.Process{
		Cancel: cancel,
	}, nil
}

func (p *SSHProvider) ParseURL(line string) string {
	line = strings.ToLower(line)
	if idx := strings.Index(line, "https://"); idx != -1 {
		rest := line[idx:]
		if endIdx := strings.IndexAny(rest, " \t\n\r"); endIdx != -1 {
			rest = rest[:endIdx]
		}
		if strings.Contains(rest, "localhost.run") || strings.Contains(rest, "lhr.life") || strings.Contains(rest, "serveo.net") {
			return rest
		}
	}
	return ""
}

func (p *SSHProvider) IsReady(line string) bool {
	line = strings.ToLower(line)
	if p.host == "localhost.run" {
		return strings.Contains(line, "localhost.run") || strings.Contains(line, "lhr.life")
	}
	return strings.Contains(line, "serveo.net") || strings.Contains(line, "forwarding")
}
