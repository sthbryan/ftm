package cloudflared

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

func (i *Installer) CloudflaredBin() string {
	if runtime.GOOS == "windows" {
		return filepath.Join(i.BaseDir, "cloudflared.exe")
	}
	return filepath.Join(i.BaseDir, "cloudflared")
}

func (i *Installer) IsInstalled() bool {
	_, err := os.Stat(i.CloudflaredBin())
	return err == nil
}

func (i *Installer) Install(progress chan<- providers.DownloadProgress) error {
	if err := os.MkdirAll(i.BaseDir, 0755); err != nil {
		return fmt.Errorf("failed to create base dir: %w", err)
	}

	if i.IsInstalled() {
		return nil
	}

	url, err := i.cloudflaredURL()
	if err != nil {
		return err
	}

	binPath := i.CloudflaredBin()

	if progress != nil {
		progress <- providers.DownloadProgress{
			Percent: 10,
			Current: 0,
			Total:   100,
			Name:    "cloudflared",
		}
	}

	if strings.HasSuffix(url, ".tgz") {
		tmpFile := binPath + ".tgz"
		if err := i.downloader.Download(url, tmpFile, progress, "cloudflared"); err != nil {
			os.Remove(tmpFile)
			return fmt.Errorf("download failed: %w", err)
		}
		defer os.Remove(tmpFile)

		if err := i.extractTgz(tmpFile, binPath); err != nil {
			return fmt.Errorf("extract failed: %w", err)
		}
	} else {
		if err := i.downloader.Download(url, binPath, progress, "cloudflared"); err != nil {
			return fmt.Errorf("download failed: %w", err)
		}
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
			Name:    "cloudflared",
		}
	}

	return nil
}

func (i *Installer) cloudflaredURL() (string, error) {
	os := runtime.GOOS
	arch := runtime.GOARCH

	base := "https://github.com/cloudflare/cloudflared/releases/latest/download"

	switch os {
	case "darwin":
		if arch == "arm64" {
			return base + "/cloudflared-darwin-arm64.tgz", nil
		}
		return base + "/cloudflared-darwin-amd64.tgz", nil
	case "linux":
		if arch == "arm64" {
			return base + "/cloudflared-linux-arm64", nil
		}
		return base + "/cloudflared-linux-amd64", nil
	case "windows":
		return base + "/cloudflared-windows-amd64.exe", nil
	default:
		return "", fmt.Errorf("unsupported OS: %s", os)
	}
}

func (i *Installer) extractTgz(src, dest string) error {
	cmd := exec.Command("tar", "-xzf", src, "-C", filepath.Dir(dest))
	if err := cmd.Run(); err != nil {
		return err
	}

	entries, err := os.ReadDir(filepath.Dir(dest))
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() && entry.Name() == "cloudflared" {
			extractedPath := filepath.Join(filepath.Dir(dest), entry.Name())
			if extractedPath != dest {
				return os.Rename(extractedPath, dest)
			}
			return os.Chmod(dest, 0755)
		}
	}

	return fmt.Errorf("cloudflared binary not found in extracted archive")
}
