package ssh

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/sthbryan/ftm/internal/core"
	"github.com/sthbryan/ftm/internal/provider"
)

type SSHProvider struct {
	host       string
	name       string
	sshKeyPath string
}

func NewLocalhostRun() provider.Provider {
	return &SSHProvider{
		host: "localhost.run",
		name: "localhost.run",
	}
}

func NewServeo() provider.Provider {
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

func (p *SSHProvider) Start(ctx context.Context, tunnel core.TunnelConfig, logWriter io.Writer) (*provider.Process, error) {
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

	proc, err := provider.StartProcess(cmd, cancel)
	if err != nil {
		return nil, fmt.Errorf("failed to start ssh tunnel: %w", err)
	}

	return proc, nil
}

func (p *SSHProvider) ParseURL(line string) string {
	switch p.host {
	case "localhost.run":
		return provider.ExtractURL(line, func(host string) bool {
			return provider.IsSubdomainOf(host, "lhr.life")
		})
	case "serveo.net":
		return provider.ExtractURL(line, func(host string) bool {
			return provider.IsSubdomainOf(host, "serveousercontent.com", "serveo.net")
		})
	}

	return ""
}
