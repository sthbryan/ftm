package providers

import (
	"fmt"
	"io"
	"net/http"
	"os"
)

type BaseDownloader struct {
	client *http.Client
}

func NewBaseDownloader() *BaseDownloader {
	return &BaseDownloader{
		client: &http.Client{},
	}
}

func (d *BaseDownloader) Download(url, dest string, progress chan<- DownloadProgress, name string) error {
	resp, err := d.client.Get(url)
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
				progress <- DownloadProgress{
					Percent: percent,
					Current: downloaded,
					Total:   total,
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

	return nil
}
