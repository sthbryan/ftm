package providers

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

type DownloadProgress struct {
	Total   int64
	Current int64
	Percent float64
	Done    bool
	Error   error
	Name    string
}

func downloadWithProgress(url, dest string, progress chan<- DownloadProgress, name string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	total := resp.ContentLength

	tmpDir := filepath.Dir(dest)
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return err
	}

	out, err := os.Create(dest + ".tmp")
	if err != nil {
		return err
	}
	defer out.Close()
	defer os.Remove(dest + ".tmp")

	buf := make([]byte, 32*1024)
	var current int64

	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			out.Write(buf[:n])
			current += int64(n)

			if progress != nil && total > 0 {
				progress <- DownloadProgress{
					Total:   total,
					Current: current,
					Percent: float64(current) / float64(total) * 100,
					Done:    false,
					Name:    name,
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

	out.Close()

	if err := os.Rename(dest+".tmp", dest); err != nil {
		return err
	}

	if err := os.Chmod(dest, 0755); err != nil {
		return err
	}

	if progress != nil {
		progress <- DownloadProgress{
			Total:   total,
			Current: current,
			Percent: 100,
			Done:    true,
			Name:    name,
		}
	}

	return nil
}

func DownloadWithProgress(url, dest string, progress chan<- DownloadProgress, name string) error {
	return downloadWithProgress(url, dest, progress, name)
}

func ExtractTarGz(src, destDir string) error {
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

	tr := tar.NewReader(gzr)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		if header.Typeflag == tar.TypeReg {
			name := filepath.Base(header.Name)
			if name == "bore" || name == "bore.exe" || name == "cloudflared" || name == "cloudflared.exe" {
				destPath := filepath.Join(destDir, name)
				out, err := os.Create(destPath)
				if err != nil {
					return err
				}
				defer out.Close()

				if _, err := io.Copy(out, tr); err != nil {
					return err
				}

				if err := os.Chmod(destPath, 0755); err != nil {
					return err
				}
				return nil
			}
		}
	}

	return fmt.Errorf("executable not found in archive")
}
