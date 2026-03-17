package providers

import (
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"foundry-tunnel/internal/config"
)

type Installer struct {
	binDir string
}

func NewInstaller() *Installer {
	return &Installer{
		binDir: filepath.Join(config.ConfigDir(), "bin"),
	}
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
	url := i.playitggURL()
	binPath := filepath.Join(i.binDir, "playit")
	
	if err := i.downloadBinary(url, binPath); err != nil {
		return "", err
	}
	
	return binPath, nil
}

func (i *Installer) installCloudflared() (string, error) {
	url := i.cloudflaredURL()
	binPath := filepath.Join(i.binDir, "cloudflared")
	
	tmpFile := binPath + ".tmp"
	if err := i.downloadFile(url, tmpFile); err != nil {
		return "", err
	}
	defer os.Remove(tmpFile)
	
	if strings.HasSuffix(url, ".tgz") {
		if err := i.extractTgz(tmpFile, binPath); err != nil {
			return "", err
		}
	} else if strings.HasSuffix(url, ".zip") {
		if err := i.extractZip(tmpFile, binPath); err != nil {
			return "", err
		}
	}
	
	return binPath, nil
}

func (i *Installer) installTunnelmole() (string, error) {
	return "", fmt.Errorf("tunnelmole requires npm. Run: npm install -g tunnelmole")
}

func (i *Installer) playitggURL() string {
	os := runtime.GOOS
	arch := runtime.GOARCH
	
	switch os {
	case "darwin":
		if arch == "arm64" {
			return "https://github.com/playit-cloud/playit-agent/releases/latest/download/playit-darwin-aarch64"
		}
		return "https://github.com/playit-cloud/playit-agent/releases/latest/download/playit-darwin-amd64"
	case "linux":
		if arch == "arm64" {
			return "https://github.com/playit-cloud/playit-agent/releases/latest/download/playit-linux-aarch64"
		}
		return "https://github.com/playit-cloud/playit-agent/releases/latest/download/playit-linux-amd64"
	case "windows":
		return "https://github.com/playit-cloud/playit-agent/releases/latest/download/playit-windows-x86_64.exe"
	default:
		return ""
	}
}

func (i *Installer) cloudflaredURL() string {
	os := runtime.GOOS
	arch := runtime.GOARCH
	
	base := "https://github.com/cloudflare/cloudflared/releases/latest/download"
	
	switch os {
	case "darwin":
		if arch == "arm64" {
			return base + "/cloudflared-darwin-arm64.tgz"
		}
		return base + "/cloudflared-darwin-amd64.tgz"
	case "linux":
		if arch == "arm64" {
			return base + "/cloudflared-linux-arm64"
		}
		return base + "/cloudflared-linux-amd64"
	case "windows":
		return base + "/cloudflared-windows-amd64.exe"
	default:
		return ""
	}
}

func (i *Installer) downloadBinary(url, dest string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != 200 {
		return fmt.Errorf("download failed: %s", resp.Status)
	}
	
	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()
	
	if _, err := io.Copy(out, resp.Body); err != nil {
		return err
	}
	
	return os.Chmod(dest, 0755)
}

func (i *Installer) downloadFile(url, dest string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != 200 {
		return fmt.Errorf("download failed: %s", resp.Status)
	}
	
	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()
	
	_, err = io.Copy(out, resp.Body)
	return err
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
