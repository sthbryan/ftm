package providers

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
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

func ExtractTarGz(src, destDir, binaryName string) error {
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
			if name == binaryName || name == binaryName+".exe" {
				return extractFileToDir(tr, destDir, name)
			}
		}
	}

	return fmt.Errorf("executable %s not found in archive", binaryName)
}

func ExtractZip(src, destDir, binaryName string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		if !f.Mode().IsRegular() {
			continue
		}
		name := filepath.Base(f.Name)
		if name == binaryName || name == binaryName+".exe" {
			rc, err := f.Open()
			if err != nil {
				return err
			}
			defer rc.Close()
			return extractFileToDir(rc, destDir, name)
		}
	}

	return fmt.Errorf("executable %s not found in archive", binaryName)
}

func extractFileToDir(src io.Reader, destDir, filename string) error {
	destPath := filepath.Join(destDir, filename)
	tmpPath := destPath + ".tmp"

	out, err := os.Create(tmpPath)
	if err != nil {
		return err
	}

	_, err = io.Copy(out, src)
	if closeErr := out.Close(); closeErr != nil && err == nil {
		err = closeErr
	}

	if err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to write %s: %w", filename, err)
	}

	if err := os.Chmod(tmpPath, 0755); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to chmod %s: %w", filename, err)
	}

	if err := os.Rename(tmpPath, destPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to rename %s: %w", filename, err)
	}

	return nil
}

func ExtractArchive(src, destDir, binaryName string) error {
	lowerSrc := strings.ToLower(src)
	if strings.HasSuffix(lowerSrc, ".zip") {
		return ExtractZip(src, destDir, binaryName)
	}
	if strings.HasSuffix(lowerSrc, ".tar.gz") || strings.HasSuffix(lowerSrc, ".tgz") {
		return ExtractTarGz(src, destDir, binaryName)
	}
	return fmt.Errorf("unsupported archive format: %s", src)
}
