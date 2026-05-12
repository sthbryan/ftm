package playit

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/sthbryan/ftm/internal/providers"
)

type Installer struct {
	BaseDir    string
	downloader *providers.BaseDownloader
}

func NewInstaller(baseDir string) *Installer {
	return &Installer{
		BaseDir:    baseDir,
		downloader: providers.NewBaseDownloader(),
	}
}

func (i *Installer) PlayitBin() string {
	if runtime.GOOS == "windows" {
		return filepath.Join(i.BaseDir, "playit.exe")
	}
	return filepath.Join(i.BaseDir, "playit")
}

func (i *Installer) ConfigPath() string {
	if runtime.GOOS == "windows" {
		return filepath.Join(os.Getenv("APPDATA"), "playit", "playit.toml")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "playit", "playit.toml")
}

// IsInstalled checks if playit binary exists
func (i *Installer) IsInstalled() bool {
	_, err := os.Stat(i.PlayitBin())
	return err == nil
}

// IsClaimed checks if the agent has been claimed (playit.toml exists with secret key)
func (i *Installer) IsClaimed() bool {
	configPath := i.ConfigPath()
	if _, err := os.Stat(configPath); err != nil {
		return false
	}
	// Check if file has content (contains secret key after claiming)
	data, err := os.ReadFile(configPath)
	if err != nil {
		return false
	}
	return strings.Contains(string(data), "secret")
}

func (i *Installer) Install(progress chan<- providers.DownloadProgress) error {
	if err := os.MkdirAll(i.BaseDir, 0755); err != nil {
		return fmt.Errorf("failed to create base dir: %w", err)
	}

	if i.IsInstalled() {
		if progress != nil {
			progress <- providers.DownloadProgress{
				Percent: 100,
				Done:    true,
				Name:    "playit",
			}
		}
		return nil
	}

	url, err := i.playitURL()
	if err != nil {
		return err
	}

	binPath := i.PlayitBin()

	if progress != nil {
		progress <- providers.DownloadProgress{
			Percent: 10,
			Current: 0,
			Total:   100,
			Name:    "playit",
		}
	}

	tmpFile := binPath + ".tar.gz"
	if err := i.downloader.Download(url, tmpFile, progress, "playit"); err != nil {
		os.Remove(tmpFile)
		return fmt.Errorf("download failed: %w", err)
	}
	defer os.Remove(tmpFile)

	if err := extractTarGz(tmpFile, binPath); err != nil {
		return fmt.Errorf("extract failed: %w", err)
	}

	if _, err := os.Stat(binPath); err != nil {
		return fmt.Errorf("binary not found after install: %w", err)
	}

	if err := os.Chmod(binPath, 0755); err != nil {
		return fmt.Errorf("chmod failed: %w", err)
	}

	if progress != nil {
		progress <- providers.DownloadProgress{
			Percent: 100,
			Done:    true,
			Name:    "playit",
		}
	}

	return nil
}

func (i *Installer) playitURL() (string, error) {
	os := runtime.GOOS
	arch := runtime.GOARCH

	const version = "v0.17.1"
	base := fmt.Sprintf("https://github.com/playit-cloud/playit-agent/releases/download/%s", version)

	switch os {
	case "linux":
		switch arch {
		case "arm64":
			return base + "/playit-agent_" + strings.TrimPrefix(version, "v") + "_linux_arm64.tar.gz", nil
		case "amd64":
			return base + "/playit-agent_" + strings.TrimPrefix(version, "v") + "_linux_x86_64.tar.gz", nil
		}
		return "", fmt.Errorf("unsupported architecture for Linux: %s", arch)
	case "windows":
		return base + "/playit-agent_" + strings.TrimPrefix(version, "v") + "_windows_x86_64.zip", nil
	case "darwin":
		// macOS builds have been discontinued, but we provide the download URL structure
		// Users on Mac need to compile from source or use Docker
		return "", fmt.Errorf("macOS builds are discontinued. Please use Docker or compile from source")
	default:
		return "", fmt.Errorf("unsupported OS: %s", os)
	}
}

func extractTarGz(src, dest string) error {
	cmd := exec.Command("tar", "-xzf", src, "-C", filepath.Dir(dest))
	if err := cmd.Run(); err != nil {
		return err
	}

	entries, err := os.ReadDir(filepath.Dir(dest))
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			name := entry.Name()
			if name == "playit" || name == "playit.exe" || name == "playit-agent" {
				extractedPath := filepath.Join(filepath.Dir(dest), name)
				if extractedPath != dest {
					if err := os.Rename(extractedPath, dest); err != nil {
						return err
					}
				}
				return os.Chmod(dest, 0755)
			}
		}
	}

	return fmt.Errorf("playit binary not found in extracted archive")
}
