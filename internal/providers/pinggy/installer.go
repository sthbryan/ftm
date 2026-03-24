package pinggy

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/deadbryam/ftm/internal/providers"
)

type Installer struct {
	BaseDir string
}

func NewInstaller(baseDir string) *Installer {
	return &Installer{BaseDir: baseDir}
}

func (i *Installer) PinggyBin() string {
	if runtime.GOOS == "windows" {
		return filepath.Join(i.BaseDir, "pinggy.exe")
	}
	return filepath.Join(i.BaseDir, "pinggy")
}

func (i *Installer) IsInstalled() bool {
	_, err := os.Stat(i.PinggyBin())
	return err == nil
}

func (i *Installer) Install(progress chan<- providers.DownloadProgress) error {
	if err := os.MkdirAll(i.BaseDir, 0755); err != nil {
		return fmt.Errorf("failed to create base dir: %w", err)
	}

	if i.IsInstalled() {
		return nil
	}

	version, err := i.getLatestVersion()
	if err != nil {
		return fmt.Errorf("failed to get latest version: %w", err)
	}

	url := i.pinggyURL(version)
	if url == "" {
		return fmt.Errorf("unsupported platform: %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	if progress != nil {
		progress <- providers.DownloadProgress{
			Percent: 10,
			Current: 0,
			Total:   100,
			Name:    "pinggy",
		}
	}

	dest := i.PinggyBin() + ".tmp"
	if err := i.download(url, dest, progress); err != nil {
		os.Remove(dest)
		return fmt.Errorf("failed to download pinggy: %w", err)
	}

	if err := os.Chmod(dest, 0755); err != nil {
		os.Remove(dest)
		return fmt.Errorf("failed to set executable permission: %w", err)
	}

	if err := os.Rename(dest, i.PinggyBin()); err != nil {
		os.Remove(dest)
		return fmt.Errorf("failed to move binary to final location: %w", err)
	}

	if progress != nil {
		progress <- providers.DownloadProgress{
			Percent: 100,
			Done:    true,
			Name:    "pinggy",
		}
	}

	return nil
}

func (i *Installer) getLatestVersion() (string, error) {
	resp, err := http.Get("https://api.github.com/repos/Pinggy-io/cli-js/releases/latest")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned status: %s", resp.Status)
	}

	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", err
	}

	return strings.TrimPrefix(release.TagName, "v"), nil
}

func (i *Installer) pinggyURL(version string) string {
	switch runtime.GOOS {
	case "darwin":
		if runtime.GOARCH == "arm64" {
			return fmt.Sprintf("https://github.com/Pinggy-io/cli-js/releases/download/v%s/pinggy-macos-arm64", version)
		}
		return fmt.Sprintf("https://github.com/Pinggy-io/cli-js/releases/download/v%s/pinggy-macos-x64", version)
	case "linux":
		if runtime.GOARCH == "arm64" {
			return fmt.Sprintf("https://github.com/Pinggy-io/cli-js/releases/download/v%s/pinggy-linux-arm64", version)
		}
		return fmt.Sprintf("https://github.com/Pinggy-io/cli-js/releases/download/v%s/pinggy-linux-x64", version)
	case "windows":
		if runtime.GOARCH == "arm64" {
			return fmt.Sprintf("https://github.com/Pinggy-io/cli-js/releases/download/v%s/pinggy-win-arm64.exe", version)
		}
		return fmt.Sprintf("https://github.com/Pinggy-io/cli-js/releases/download/v%s/pinggy-win-x64.exe", version)
	default:
		return ""
	}
}

func (i *Installer) download(url, dest string, progress chan<- providers.DownloadProgress) error {
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("http.Get failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	total := resp.ContentLength
	downloaded := int64(0)
	buf := make([]byte, 32*1024)

	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			out.Write(buf[:n])
			downloaded += int64(n)
			if total > 0 && progress != nil {
				percent := 10 + float64(downloaded)/float64(total)*80
				progress <- providers.DownloadProgress{
					Percent: percent,
					Current: downloaded,
					Total:   total,
					Name:    "pinggy",
				}
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}

	return nil
}
