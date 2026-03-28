package ssh

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/sthbryan/ftm/internal/config"
	"github.com/sthbryan/ftm/internal/providers"
)

type SSHProvider struct {
	host       string
	name       string
	sshKeyPath string
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

func (p *SSHProvider) FindBinary() string {
	if path, err := exec.LookPath("ssh"); err == nil {
		return path
	}
	return ""
}

func (p *SSHProvider) ensureSSHKey() (string, error) {
	if p.sshKeyPath != "" {
		return p.sshKeyPath, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	keyPath := filepath.Join(home, ".ssh", "id_rsa")

	if _, err := os.Stat(keyPath); err == nil {
		p.sshKeyPath = keyPath
		return keyPath, nil
	}

	sshDir := filepath.Join(home, ".ssh")
	if err := os.MkdirAll(sshDir, 0700); err != nil {
		return "", fmt.Errorf("failed to create .ssh directory: %w", err)
	}

	cmd := exec.Command("ssh-keygen", "-t", "rsa", "-b", "2048", "-f", keyPath, "-N", "", "-C", "foundry-tunnel")
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to generate SSH key: %w", err)
	}

	p.sshKeyPath = keyPath
	return keyPath, nil
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
		"-o", "ServerAliveInterval=30",
		"-o", "ServerAliveCountMax=3",
		"-o", "ConnectTimeout=10",
		"-o", "LogLevel=ERROR",
	}

	var args []string
	switch p.host {
	case "localhost.run":
		args = append(baseArgs,
			"-o", "BatchMode=yes",
			"-R", fmt.Sprintf("80:localhost:%d", tunnel.LocalPort),
			"nokey@localhost.run",
		)
	case "serveo.net":
		keyPath, err := p.ensureSSHKey()
		if err != nil {
			cancel()
			return nil, err
		}

		args = append(baseArgs,
			"-i", keyPath,
			"-o", "IdentitiesOnly=yes",
			"-R", fmt.Sprintf("80:localhost:%d", tunnel.LocalPort),
			"serveo.net",
		)
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

	switch p.host {
	case "localhost.run":
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
	case "serveo.net":
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
