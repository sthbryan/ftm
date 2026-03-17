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

	baseArgs := []string{
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "BatchMode=yes",
		"-o", "ServerAliveInterval=30",
		"-o", "ServerAliveCountMax=3",
		"-o", "ConnectTimeout=10",
		"-o", "LogLevel=ERROR",
	}

	var args []string
	if p.host == "localhost.run" {
		args = append(baseArgs,
			"-R", fmt.Sprintf("80:localhost:%d", tunnel.LocalPort),
			"nokey@localhost.run",
		)
	} else if p.host == "serveo.net" {
		args = append(baseArgs,
			"-R", fmt.Sprintf("80:localhost:%d", tunnel.LocalPort),
			"serveo.net",
		)
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

var localhostRunRegex = regexp.MustCompile(`https?://[a-z0-9]+\.lhr\.life`)
var serveoRegex = regexp.MustCompile(`https?://[a-z0-9-]+\.serveousercontent\.com`)

func (p *SSHProvider) ParseURL(line string) string {
	clean := stripANSI(line)
	cleanLower := strings.ToLower(clean)
	
	if p.host == "localhost.run" {
		matches := localhostRunRegex.FindStringSubmatch(cleanLower)
		if len(matches) > 0 {
			return matches[0]
		}
		if strings.Contains(cleanLower, ".lhr.life") {
			if idx := strings.Index(cleanLower, "https://"); idx != -1 {
				rest := clean[idx:]
				if endIdx := strings.IndexAny(rest, " \t\n\r,"); endIdx != -1 {
					return rest[:endIdx]
				}
				return rest
			}
		}
	} else if p.host == "serveo.net" {
		matches := serveoRegex.FindStringSubmatch(cleanLower)
		if len(matches) > 0 {
			return matches[0]
		}
		if strings.Contains(cleanLower, "serveo") || strings.Contains(cleanLower, "serveousercontent") {
			if idx := strings.Index(cleanLower, "https://"); idx != -1 {
				rest := clean[idx:]
				if endIdx := strings.IndexAny(rest, " \t\n\r"); endIdx != -1 {
					return rest[:endIdx]
				}
				return rest
			}
		}
	}
	
	return ""
}

func (p *SSHProvider) IsReady(line string) bool {
	clean := stripANSI(line)
	cleanLower := strings.ToLower(clean)
	
	if p.host == "localhost.run" {
		return strings.Contains(cleanLower, ".lhr.life") ||
		       strings.Contains(cleanLower, "tunneled")
	}
	
	return strings.Contains(cleanLower, "serveo") || 
	       strings.Contains(cleanLower, "forwarding")
}
