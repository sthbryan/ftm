package tunnelmole

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

type TunnelmoleProvider struct {
	installer *BunInstaller
}

func New() providers.Provider {
	configDir, _ := os.UserHomeDir()
	if configDir == "" {
		configDir = "."
	}
	baseDir := filepath.Join(configDir, ".config", "foundry-tunnel")

	return &TunnelmoleProvider{
		installer: NewBunInstaller(baseDir),
	}
}

func (p *TunnelmoleProvider) Name() string {
	return "tunnelmole"
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

func (p *TunnelmoleProvider) IsInstalled() bool {

	if _, err := exec.LookPath("tmole"); err == nil {
		return true
	}
	if _, err := exec.LookPath("tunnelmole"); err == nil {
		return true
	}

	return p.installer.IsInstalled()
}

func (p *TunnelmoleProvider) Install(progress chan<- providers.DownloadProgress) error {
	return p.installer.Install(progress)
}

func (p *TunnelmoleProvider) FindBinary() string {

	if path, err := exec.LookPath("tmole"); err == nil {
		return path
	}
	if path, err := exec.LookPath("tunnelmole"); err == nil {
		return path
	}

	if p.installer.IsInstalled() {
		return p.installer.TunnelmoleBin()
	}

	home, _ := os.UserHomeDir()
	candidates := []string{
		filepath.Join(home, ".npm-global", "bin", "tmole"),
		filepath.Join(home, ".npm-global", "bin", "tunnelmole"),
		"/usr/local/bin/tmole",
		"/usr/local/bin/tunnelmole",
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
		return nil, fmt.Errorf("installing")
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

var tunnelmoleURLRegex = regexp.MustCompile(`https?://[a-zA-Z0-9-]+-ip-[0-9-]+\.tunnelmole\.net`)

func (p *TunnelmoleProvider) ParseURL(line string) string {

	if strings.Contains(line, "dashboard.tunnelmole.com") {
		return ""
	}

	matches := tunnelmoleURLRegex.FindStringSubmatch(line)
	if len(matches) > 0 {
		return matches[0]
	}

	lineLower := strings.ToLower(line)
	if idx := strings.Index(lineLower, "https://"); idx != -1 {
		rest := line[idx:]
		if endIdx := strings.IndexAny(rest, " \t\n\r)"); endIdx != -1 {
			rest = rest[:endIdx]
		}
		lowerRest := strings.ToLower(rest)
		if strings.Contains(lowerRest, "tunnelmole.net") &&
			!strings.Contains(lowerRest, "dashboard") {
			return rest
		}
	}

	return ""
}

func (p *TunnelmoleProvider) IsReady(line string) bool {
	lineLower := strings.ToLower(line)

	if tunnelmoleURLRegex.MatchString(line) {
		return true
	}
	return strings.Contains(lineLower, "your site is available at") ||
		strings.Contains(lineLower, "tunnelmole.net")
}
