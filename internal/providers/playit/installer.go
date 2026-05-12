package playit

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

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

func (i *Installer) IsInstalled() bool {
	_, err := os.Stat(i.PlayitBin())
	return err == nil
}

func (i *Installer) IsClaimed() bool {
	configPath := i.ConfigPath()
	if _, err := os.Stat(configPath); err != nil {
		return false
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return false
	}
	return len(data) > 100
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

	if err := i.downloader.Download(url, binPath, progress, "playit"); err != nil {
		return fmt.Errorf("download failed: %w", err)
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
			return base + "/playit-linux-aarch64", nil
		case "amd64":
			return base + "/playit-linux-amd64", nil
		case "arm":
			return base + "/playit-linux-armv7", nil
		case "386":
			return base + "/playit-linux-i686", nil
		}
		return "", fmt.Errorf("unsupported architecture for Linux: %s", arch)
	case "windows":
		return base + "/playit-windows-x86_64.exe", nil
	case "darwin":
		return "", fmt.Errorf("macOS builds are discontinued")
	default:
		return "", fmt.Errorf("unsupported OS: %s", os)
	}
}
