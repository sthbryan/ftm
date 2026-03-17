package providers

import (
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"foundry-tunnel/internal/config"
)

type Installer struct {
	binDir   string
	progress chan<- DownloadProgress
}

func NewInstaller() *Installer {
	return &Installer{
		binDir: filepath.Join(config.ConfigDir(), "bin"),
	}
}

func (i *Installer) SetProgressChannel(ch chan<- DownloadProgress) {
	i.progress = ch
}

func (i *Installer) EnsureInstalled(p Provider) (string, error) {
	binPath := filepath.Join(i.binDir, p.BinaryName())
	
	if _, err := os.Stat(binPath); err == nil {
		return binPath, nil
	}
	
	if err := os.MkdirAll(i.binDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create bin dir: %w", err)
	}
	
	switch p.Name() {
	case "Playit.gg":
		return i.installPlayitgg()
	case "Cloudflare Tunnel":
		return i.installCloudflared()
	case "Tunnelmole":
		return i.installTunnelmole()
	default:
		return "", fmt.Errorf("auto-install not supported for %s", p.Name())
	}
}

func (i *Installer) installPlayitgg() (string, error) {
	url, err := i.playitggURL()
	if err != nil {
		return "", err
	}
	
	binPath := filepath.Join(i.binDir, "playit")
	
	if err := i.downloadBinary(url, binPath); err != nil {
		return "", fmt.Errorf("download failed: %w", err)
	}
	
	return binPath, nil
}

func (i *Installer) installCloudflared() (string, error) {
	url, err := i.cloudflaredURL()
	if err != nil {
		return "", err
	}
	
	binPath := filepath.Join(i.binDir, "cloudflared")
	
	if strings.HasSuffix(url, ".tgz") {
		tmpFile := binPath + ".tgz"
		if err := i.downloadFile(url, tmpFile); err != nil {
			return "", fmt.Errorf("download failed: %w", err)
		}
		defer os.Remove(tmpFile)
		
		if err := i.extractTgz(tmpFile, binPath); err != nil {
			return "", fmt.Errorf("extract failed: %w", err)
		}
	} else {
		if err := i.downloadBinary(url, binPath); err != nil {
			return "", fmt.Errorf("download failed: %w", err)
		}
	}
	
	if _, err := os.Stat(binPath); err != nil {
		return "", fmt.Errorf("binary not found after install: %w", err)
	}
	
	if err := os.Chmod(binPath, 0755); err != nil {
		return "", fmt.Errorf("chmod failed: %w", err)
	}
	
	return binPath, nil
}

func (i *Installer) installTunnelmole() (string, error) {
	return "", fmt.Errorf("tunnelmole requires npm. Run: npm install -g tunnelmole")
}

func (i *Installer) playitggURL() (string, error) {
	os := runtime.GOOS
	arch := runtime.GOARCH
	
	switch os {
	case "darwin":
		return "", fmt.Errorf("playit.gg no tiene build para macOS. Usa Cloudflared o instala manualmente con: brew install playit")
	case "linux":
		if arch == "arm64" {
			return "https://github.com/playit-cloud/playit-agent/releases/latest/download/playit-linux-aarch64", nil
		}
		return "https://github.com/playit-cloud/playit-agent/releases/latest/download/playit-linux-amd64", nil
	case "windows":
		return "https://github.com/playit-cloud/playit-agent/releases/latest/download/playit-windows-x86_64.exe", nil
	default:
		return "", fmt.Errorf("unsupported OS: %s", os)
	}
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

func (i *Installer) downloadBinary(url, dest string) error {
	return downloadWithProgress(url, dest, i.progress)
}

func (i *Installer) downloadFile(url, dest string) error {
	return downloadWithProgress(url, dest, i.progress)
}

func (i *Installer) extractTgz(src, dest string) error {
	file, err := os.Open(src)
	if err != nil {
		return err
	}
	defer file.Close()
	
	gzr, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzr.Close()
	
	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()
	
	if _, err := io.Copy(out, gzr); err != nil {
		return err
	}
	
	return os.Chmod(dest, 0755)
}

func (i *Installer) extractZip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()
	
	for _, f := range r.File {
		if f.Name == "cloudflared" || f.Name == "cloudflared.exe" {
			rc, err := f.Open()
			if err != nil {
				return err
			}
			defer rc.Close()
			
			out, err := os.Create(dest)
			if err != nil {
				return err
			}
			defer out.Close()
			
			if _, err := io.Copy(out, rc); err != nil {
				return err
			}
			
			return os.Chmod(dest, 0755)
		}
	}
	
	return fmt.Errorf("binary not found in zip")
}

func (i *Installer) BinDir() string {
	return i.binDir
}
