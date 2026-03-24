package tunnelmole

import (
	"archive/tar"
	"compress/gzip"
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

func (i *Installer) TunnelmoleBin() string {
	if runtime.GOOS == "windows" {
		return filepath.Join(i.BaseDir, "tmole.exe")
	}
	return filepath.Join(i.BaseDir, "tmole")
}

func (i *Installer) IsInstalled() bool {
	_, err := os.Stat(i.TunnelmoleBin())
	return err == nil
}

func (i *Installer) Install(progress chan<- providers.DownloadProgress) error {
	if err := os.MkdirAll(i.BaseDir, 0755); err != nil {
		return fmt.Errorf("failed to create base dir: %w", err)
	}

	if i.IsInstalled() {
		return nil
	}

	url := i.tunnelmoleURL()
	if url == "" {
		return fmt.Errorf("unsupported platform: %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	if progress != nil {
		progress <- providers.DownloadProgress{
			Percent: 10,
			Current: 0,
			Total:   100,
			Name:    "tunnelmole",
		}
	}

	dest := i.TunnelmoleBin() + ".tmp"
	if err := i.download(url, dest, progress); err != nil {
		os.Remove(dest)
		return fmt.Errorf("failed to download tunnelmole: %w", err)
	}

	if err := os.Chmod(dest, 0755); err != nil {
		os.Remove(dest)
		return fmt.Errorf("failed to set executable permission: %w", err)
	}

	if err := os.Rename(dest, i.TunnelmoleBin()); err != nil {
		os.Remove(dest)
		return fmt.Errorf("failed to move binary to final location: %w", err)
	}

	if progress != nil {
		progress <- providers.DownloadProgress{
			Percent: 100,
			Done:    true,
			Name:    "tunnelmole",
		}
	}

	return nil
}

func (i *Installer) tunnelmoleURL() string {
	switch runtime.GOOS {
	case "darwin":
		return "https://tunnelmole.com/downloads/tmole-mac.gz"
	case "linux":
		return "https://tunnelmole.com/downloads/tmole-linux.gz"
	case "windows":
		return "https://tunnelmole.com/downloads/tmole.exe"
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
					Name:    "tunnelmole",
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

	if runtime.GOOS == "windows" {
		return nil
	}

	out.Close()

	headerBytes := make([]byte, 2)
	f, err := os.Open(dest)
	if err != nil {
		return err
	}
	f.Read(headerBytes)
	f.Close()

	if headerBytes[0] != 0x1f || headerBytes[1] != 0x8b {
		return fmt.Errorf("not a valid gzip archive")
	}

	gzFile, err := os.Open(dest)
	if err != nil {
		return err
	}
	defer gzFile.Close()

	gzReader, err := gzip.NewReader(gzFile)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzReader.Close()

	testTarReader := tar.NewReader(gzReader)
	_, err = testTarReader.Next()

	if err != nil && strings.Contains(err.Error(), "invalid tar header") {
		rawDest := dest + ".raw"

		gzFile.Close()
		gzFile, err = os.Open(dest)
		if err != nil {
			return err
		}
		defer gzFile.Close()

		gzReader, err = gzip.NewReader(gzFile)
		if err != nil {
			return err
		}
		defer gzReader.Close()

		outFile, err := os.Create(rawDest)
		if err != nil {
			return err
		}
		io.Copy(outFile, gzReader)
		outFile.Close()

		os.Remove(dest)
		os.Rename(rawDest, dest)
		return nil
	}

	gzFile.Close()
	gzFile, err = os.Open(dest)
	if err != nil {
		return err
	}
	defer gzFile.Close()

	gzReader, err = gzip.NewReader(gzFile)
	if err != nil {
		return err
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar: %w", err)
		}

		name := filepath.Base(header.Name)
		if strings.HasPrefix(name, "tmole") {
			outFile, err := os.Create(dest)
			if err != nil {
				return err
			}
			io.Copy(outFile, tarReader)
			outFile.Close()
			return nil
		}
	}

	return fmt.Errorf("tmole binary not found in archive")
}
