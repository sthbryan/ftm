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

type PinggyProvider struct {
	sshKeyPath string
}

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

func (p *PinggyProvider) ensureSSHKey() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	keyPath := filepath.Join(home, ".ssh", "id_rsa")

	if _, err := os.Stat(keyPath); err == nil {
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

	return keyPath, nil
}

func (p *PinggyProvider) Start(ctx context.Context, tunnel config.TunnelConfig, logWriter io.Writer) (*providers.Process, error) {
	binary := p.FindBinary()
	if binary == "" {
		return nil, fmt.Errorf("ssh not found. Install OpenSSH")
	}

	keyPath, err := p.ensureSSHKey()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(ctx)

	args := []string{
		"-p", "443",
		"-i", keyPath,
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "IdentitiesOnly=yes",
		"-o", "ServerAliveInterval=30",
		"-o", "ServerAliveCountMax=3",
		"-o", "ConnectTimeout=10",
		"-o", "LogLevel=ERROR",
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

var pinggyRegex = regexp.MustCompile(`https?://[a-z0-9-]+\.[a-z]+\.pinggy\.(io|link)`)

func (p *PinggyProvider) ParseURL(line string) string {
	clean := stripANSI(line)
	cleanLower := strings.ToLower(clean)

	if strings.Contains(cleanLower, "dashboard.pinggy.io") {
		return ""
	}

	matches := pinggyRegex.FindStringSubmatch(cleanLower)
	if len(matches) > 0 {
		return matches[0]
	}

	if strings.Contains(cleanLower, "pinggy.link") || 
	   (strings.Contains(cleanLower, ".pinggy.io") && !strings.Contains(cleanLower, "dashboard")) {
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

	return strings.Contains(cleanLower, "pinggy.link") ||
		(strings.Contains(cleanLower, ".pinggy.io") && !strings.Contains(cleanLower, "dashboard")) ||
		strings.Contains(cleanLower, "connected")
}
