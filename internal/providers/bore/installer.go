package bore

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/sthbryan/ftm/internal/providers"
)

type Installer struct {
	BaseDir string
}

func NewInstaller(baseDir string) *Installer {
	return &Installer{BaseDir: baseDir}
}

func (i *Installer) BoreBin() string {
	if runtime.GOOS == "windows" {
		return filepath.Join(i.BaseDir, "bore.exe")
	}
	return filepath.Join(i.BaseDir, "bore")
}

func (i *Installer) IsInstalled() bool {
	if _, err := os.Stat(i.BoreBin()); err == nil {
		return true
	}
	return false
}

func (i *Installer) Install(progress chan<- providers.DownloadProgress) error {
	if err := os.MkdirAll(i.BaseDir, 0755); err != nil {
		return fmt.Errorf("failed to create base dir: %w", err)
	}

	if i.IsInstalled() {
		return nil
	}

	platform := runtime.GOOS
	arch := runtime.GOARCH

	var url string
	var filename string

	switch platform {
	case "darwin":
		if arch == "arm64" {
			filename = "bore-v0.6.0-aarch64-apple-darwin.tar.gz"
			url = "https://github.com/ekzhang/bore/releases/download/v0.6.0/" + filename
		} else {
			filename = "bore-v0.6.0-x86_64-apple-darwin.tar.gz"
			url = "https://github.com/ekzhang/bore/releases/download/v0.6.0/" + filename
		}
	case "linux":
		if arch == "arm64" {
			filename = "bore-v0.6.0-aarch64-unknown-linux-musl.tar.gz"
			url = "https://github.com/ekzhang/bore/releases/download/v0.6.0/" + filename
		} else {
			filename = "bore-v0.6.0-x86_64-unknown-linux-musl.tar.gz"
			url = "https://github.com/ekzhang/bore/releases/download/v0.6.0/" + filename
		}
	case "windows":
		filename = "bore-v0.6.0-x86_64-pc-windows-msvc.zip"
		url = "https://github.com/ekzhang/bore/releases/download/v0.6.0/" + filename
	default:
		return fmt.Errorf("unsupported platform: %s", platform)
	}

	destArchive := filepath.Join(i.BaseDir, filename)

	if progress != nil {
		progress <- providers.DownloadProgress{
			Percent: 10,
			Current: 0,
			Total:   100,
			Name:    "bore",
		}
	}

	if err := providers.DownloadWithProgress(url, destArchive, progress, "bore"); err != nil {
		return fmt.Errorf("failed to download bore: %w", err)
	}

	defer os.Remove(destArchive)

	if err := providers.ExtractTarGz(destArchive, i.BaseDir); err != nil {
		return fmt.Errorf("failed to extract bore: %w", err)
	}

	if _, err := os.Stat(i.BoreBin()); err != nil {
		return fmt.Errorf("binary not found after install: %w", err)
	}

	if progress != nil {
		progress <- providers.DownloadProgress{
			Percent: 100,
			Done:    true,
			Name:    "bore",
		}
	}

	return nil
}
