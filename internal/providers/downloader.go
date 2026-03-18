package providers

import (
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
}

func downloadWithProgress(url, dest string, progress chan<- DownloadProgress) error {
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
		}
	}

	return nil
}
