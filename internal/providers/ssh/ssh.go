package ssh

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

var ansiEscape = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

func stripANSI(s string) string {
	return ansiEscape.ReplaceAllString(s, "")
}

var urlRegex = regexp.MustCompile(`https?://[a-zA-Z0-9][-a-zA-Z0-9.]*[a-zA-Z0-9]`)

func (p *SSHProvider) ParseURL(line string) string {
	clean := stripANSI(line)
	cleanLower := strings.ToLower(clean)
	
	if p.host == "localhost.run" {
		if !strings.Contains(cleanLower, "localhost.run") && !strings.Contains(cleanLower, "lhr.life") {
			return ""
		}
	} else if p.host == "serveo.net" {
		if !strings.Contains(cleanLower, "serveo") && !strings.Contains(cleanLower, "serveousercontent") {
			return ""
		}
	}
	
	matches := urlRegex.FindStringSubmatch(clean)
	if len(matches) > 0 {
		return matches[0]
	}
	
	if idx := strings.Index(cleanLower, "https://"); idx != -1 {
		rest := clean[idx:]
		if endIdx := strings.IndexAny(rest, " \t\n\r"); endIdx != -1 {
			rest = rest[:endIdx]
		}
		return rest
	}
	
	return ""
}

func (p *SSHProvider) IsReady(line string) bool {
	clean := stripANSI(line)
	cleanLower := strings.ToLower(clean)
	
	if p.host == "localhost.run" {
		return strings.Contains(cleanLower, "localhost.run") || 
		       strings.Contains(cleanLower, "lhr.life")
	}
	
	return strings.Contains(cleanLower, "serveo") || 
	       strings.Contains(cleanLower, "forwarding")
}
