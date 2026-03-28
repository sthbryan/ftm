package tunnelmole

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

const errRosettaNeeded = "tunnelmole requires Rosetta 2 to run on Apple Silicon. Install it with: softwareupdate --install-rosetta"

func RosettaInstalled() bool {
	_, err := os.Stat("/Library/Apple/usr/libexec/oah")
	return err == nil
}

func needsRosetta(err string) bool {
	return strings.Contains(err, "bad CPU type") || strings.Contains(err, "executable")
}

type TunnelmoleProvider struct {
	installer *Installer
}

func New() providers.Provider {
	configDir, _ := os.UserHomeDir()
	if configDir == "" {
		configDir = "."
	}
	baseDir := filepath.Join(configDir, ".config", "foundry-tunnel", "bin")

	return &TunnelmoleProvider{
		installer: NewInstaller(baseDir),
	}
}

func (p *TunnelmoleProvider) Name() string {
	return "tunnelmole"
}

func (p *TunnelmoleProvider) BinaryName() string {
	return "tunnelmole"
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

	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Stdout = logWriter
	cmd.Stderr = logWriter

	if err := cmd.Start(); err != nil {
		cancel()
		if needsRosetta(err.Error()) && runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" {
			if !RosettaInstalled() {
				return nil, fmt.Errorf("%w: %s", fmt.Errorf("rosetta required"), errRosettaNeeded)
			}
			cmd = exec.CommandContext(ctx, "/Library/Apple/usr/libexec/oah", append([]string{"-r", binary}, args...)...)
			cmd.Stdout = logWriter
			cmd.Stderr = logWriter
			if err := cmd.Start(); err != nil {
				return nil, fmt.Errorf("failed to start tunnelmole: %w", err)
			}
			return &providers.Process{Cancel: cancel}, nil
		}
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
