package tunnelmole

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"foundry-tunnel/internal/providers"
)

const bunVersion = "1.2.0"

type BunInstaller struct {
	BaseDir string
}

func NewBunInstaller(baseDir string) *BunInstaller {
	return &BunInstaller{BaseDir: baseDir}
}

func (bi *BunInstaller) BunDir() string {
	return filepath.Join(bi.BaseDir, "bun")
}

func (bi *BunInstaller) BunBin() string {
	bunDir := bi.BunDir()
	if runtime.GOOS == "windows" {
		return filepath.Join(bunDir, "bun.exe")
	}
	return filepath.Join(bunDir, "bun")
}

func (bi *BunInstaller) TunnelmoleBin() string {
	bunDir := bi.BunDir()
	if runtime.GOOS == "windows" {
		return filepath.Join(bunDir, "bin", "tunnelmole.exe")
	}
	return filepath.Join(bunDir, "bin", "tunnelmole")
}

func (bi *BunInstaller) IsInstalled() bool {
	_, err := os.Stat(bi.TunnelmoleBin())
	return err == nil
}

func (bi *BunInstaller) bunURL() string {
	var osName, arch string

	switch runtime.GOOS {
	case "darwin":
		osName = "darwin"
		if runtime.GOARCH == "arm64" {
			arch = "aarch64"
		} else {
			arch = "x64"
		}
	case "linux":
		osName = "linux"
		if runtime.GOARCH == "arm64" {
			arch = "aarch64"
		} else {
			arch = "x64"
		}
	case "windows":
		osName = "windows"
		arch = "x64"
	default:
		return ""
	}

	return fmt.Sprintf("https://github.com/oven-sh/bun/releases/download/bun-v%s/bun-%s-%s.zip",
		bunVersion, osName, arch)
}

func (bi *BunInstaller) Install(progress chan<- providers.DownloadProgress) error {
	if err := os.MkdirAll(bi.BaseDir, 0755); err != nil {
		return fmt.Errorf("failed to create base dir: %w", err)
	}

	if bi.IsInstalled() {
		return nil
	}

	bunURL := bi.bunURL()
	if bunURL == "" {
		return fmt.Errorf("unsupported platform: %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	archivePath := filepath.Join(bi.BaseDir, "bun-archive.zip")
	if err := bi.download(bunURL, archivePath, progress); err != nil {
		return fmt.Errorf("failed to download Bun: %w", err)
	}
	defer os.Remove(archivePath)

	if progress != nil {
		progress <- providers.DownloadProgress{
			Percent: 45,
			Current: 0,
			Total:   100,
		}
	}

	if err := bi.extract(archivePath, bi.BunDir()); err != nil {
		return fmt.Errorf("failed to extract Bun: %w", err)
	}

	if err := bi.installTunnelmole(progress); err != nil {
		return fmt.Errorf("failed to install tunnelmole: %w", err)
	}

	return nil
}

func (bi *BunInstaller) download(url, dest string, progress chan<- providers.DownloadProgress) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
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
				progress <- providers.DownloadProgress{
					Percent: float64(downloaded) / float64(total) * 50,
					Current: downloaded,
					Total:   total,
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

func (bi *BunInstaller) extract(archivePath, destDir string) error {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer r.Close()

	bunExe := "bun"
	if runtime.GOOS == "windows" {
		bunExe = "bun.exe"
	}

	for _, f := range r.File {
		if strings.HasSuffix(f.Name, "/"+bunExe) || strings.HasSuffix(f.Name, "\\"+bunExe) {
			rc, err := f.Open()
			if err != nil {
				return fmt.Errorf("failed to open zip entry %s: %w", f.Name, err)
			}
			defer rc.Close()

			if err := os.MkdirAll(destDir, 0755); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", destDir, err)
			}

			outPath := filepath.Join(destDir, bunExe)
			out, err := os.Create(outPath)
			if err != nil {
				return fmt.Errorf("failed to create file %s: %w", outPath, err)
			}

			_, err = io.Copy(out, rc)
			out.Close()

			if err != nil {
				return fmt.Errorf("failed to extract bun: %w", err)
			}

			if runtime.GOOS != "windows" {
				if err := os.Chmod(outPath, 0755); err != nil {
					return fmt.Errorf("failed to make bun executable: %w", err)
				}
			}

			return nil
		}
	}

	return fmt.Errorf("bun executable not found in archive")
}

func (bi *BunInstaller) installTunnelmole(progress chan<- providers.DownloadProgress) error {
	bun := bi.BunBin()

	if _, err := os.Stat(bun); err != nil {
		return fmt.Errorf("bun binary not found at %s: %w", bun, err)
	}

	if progress != nil {
		progress <- providers.DownloadProgress{
			Percent: 50,
			Current: 0,
			Total:   100,
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	bunDir := bi.BunDir()
	env := os.Environ()
	env = append(env, fmt.Sprintf("BUN_INSTALL=%s", bunDir))

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, bun, "install", "-g", "tunnelmole")
	} else {
		cmd = exec.CommandContext(ctx, bun, "install", "-g", "tunnelmole")
	}

	cmd.Env = env
	cmd.Dir = bi.BaseDir

	done := make(chan error, 1)
	go func() {
		output, err := cmd.CombinedOutput()
		if err != nil {
			done <- fmt.Errorf("bun install failed: %w\nOutput: %s", err, string(output))
			return
		}
		done <- nil
	}()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	percent := 50.0
	for {
		select {
		case err := <-done:
			if err != nil {
				return err
			}

			if _, err := os.Stat(bi.TunnelmoleBin()); err != nil {
				return fmt.Errorf("tunnelmole binary not found after install at %s", bi.TunnelmoleBin())
			}

			if progress != nil {
				progress <- providers.DownloadProgress{
					Percent: 100,
					Done:    true,
				}
			}
			return nil

		case <-ticker.C:
			percent += 1
			if percent > 95 {
				percent = 95
			}
			if progress != nil {
				progress <- providers.DownloadProgress{
					Percent: percent,
					Current: 0,
					Total:   100,
				}
			}

		case <-ctx.Done():
			return fmt.Errorf("installation timed out after 5 minutes")
		}
	}
}
