package playit

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/sthbryan/ftm/internal/config"
	"github.com/sthbryan/ftm/internal/providers"
)

var claimURLRegex = regexp.MustCompile(`https://playit\.gg/claim/[a-zA-Z0-9-]+`)

var tunnelURLRegex = regexp.MustCompile(`[a-zA-Z0-9][a-zA-Z0-9-]*\.(gl\.joinmc\.link|at\.ply\.gg|ply\.gg|tunnel\.playit\.gg)`)

var domainRegex = regexp.MustCompile(`([a-zA-Z0-9-]+\.(?:gl\.joinmc\.link|at\.ply\.gg|ply\.gg|tunnel\.playit\.gg))`)

type PlayitProvider struct {
	installer *Installer
	baseDir   string
}

func New() providers.Provider {
	configDir, _ := os.UserHomeDir()
	if configDir == "" {
		configDir = "."
	}
	baseDir := filepath.Join(configDir, ".config", "foundry-tunnel", "bin")

	return &PlayitProvider{
		installer: NewInstaller(baseDir),
		baseDir:   baseDir,
	}
}

func (p *PlayitProvider) Name() string {
	return "Playit.gg"
}

func (p *PlayitProvider) BinaryName() string {
	if runtime.GOOS == "windows" {
		return "playit.exe"
	}
	return "playit"
}

func (p *PlayitProvider) FindBinary() string {

	if path, err := exec.LookPath(p.BinaryName()); err == nil {
		return path
	}

	if p.installer.IsInstalled() {
		return p.installer.PlayitBin()
	}

	home, _ := os.UserHomeDir()
	candidates := []string{
		filepath.Join(home, ".config", "playit", p.BinaryName()),
		filepath.Join(home, ".local", "bin", p.BinaryName()),
		"/usr/local/bin/" + p.BinaryName(),
		"/usr/bin/" + p.BinaryName(),
		"./" + p.BinaryName(),
	}

	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return ""
}

func (p *PlayitProvider) Start(ctx context.Context, tunnel config.TunnelConfig, logWriter io.Writer) (*providers.Process, error) {
	binary := p.FindBinary()
	if binary == "" {
		if err := p.installer.Install(nil); err != nil {
			return nil, fmt.Errorf("failed to install playit: %w", err)
		}
		binary = p.installer.PlayitBin()
	}

	ctx, cancel := context.WithCancel(ctx)

	cmd := exec.CommandContext(ctx, binary)

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

func (p *PlayitProvider) ParseURL(line string) string {

	if matches := tunnelURLRegex.FindStringSubmatch(line); len(matches) > 0 {
		return matches[0]
	}

	if matches := domainRegex.FindStringSubmatch(line); len(matches) > 0 {
		return matches[0]
	}

	return ""
}

func (p *PlayitProvider) ParseClaimURL(line string) string {
	if matches := claimURLRegex.FindStringSubmatch(line); len(matches) > 0 {
		return matches[0]
	}
	return ""
}

func (p *PlayitProvider) IsReady(line string) bool {
	lineLower := strings.ToLower(line)

	if tunnelURLRegex.MatchString(line) {
		return true
	}

	readyIndicators := []string{
		"newclient",
		"agent registered",
		"connection established",
		".gl.joinmc.link",
		".at.ply.gg",
		".ply.gg",
		".tunnel.playit.gg",
	}

	for _, indicator := range readyIndicators {
		if strings.Contains(lineLower, indicator) {
			return true
		}
	}

	return false
}

func (p *PlayitProvider) IsClaimed() bool {
	return p.installer.IsClaimed()
}

func (p *PlayitProvider) IsInstalled() bool {
	return p.installer.IsInstalled()
}

func (p *PlayitProvider) Install(progress chan<- providers.DownloadProgress) error {
	return p.installer.Install(progress)
}
