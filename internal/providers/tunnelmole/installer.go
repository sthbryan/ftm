package tunnelmole

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/sthbryan/ftm/internal/providers"
)

// npmLatest names the newest published client. The prebuilt binaries on
// tunnelmole.com are not versioned, so the npm version is only a label; whether
// the download changed is decided by its ETag.
const npmLatest = "https://registry.npmjs.org/tunnelmole/latest"

var httpClient = &http.Client{Timeout: 15 * time.Second}

type Installer struct{}

func NewInstaller() *Installer {
	return &Installer{}
}

func binaryFile() string {
	if runtime.GOOS == "windows" {
		return "tmole.exe"
	}
	return "tmole"
}

func (i *Installer) TunnelmoleBin() string {
	return providers.BundledPath(binaryFile())
}

func (i *Installer) IsInstalled() bool {
	_, err := os.Stat(i.TunnelmoleBin())
	return err == nil
}

func (i *Installer) Install(progress chan<- providers.DownloadProgress) error {
	if i.IsInstalled() {
		return nil
	}

	url := downloadURL(runtime.GOOS)
	if url == "" {
		return fmt.Errorf("unsupported platform: %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	ctx := context.Background()
	rel, err := i.LatestRelease(ctx)
	if err != nil {
		rel = &providers.Release{Asset: providers.Asset{Name: filepath.Base(url), URL: url}}
	}
	return i.InstallRelease(ctx, rel, progress)
}

func (i *Installer) LatestRelease(ctx context.Context) (*providers.Release, error) {
	url := downloadURL(runtime.GOOS)
	if url == "" {
		return nil, fmt.Errorf("tunnelmole: unsupported platform %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	etag, err := headETag(ctx, url)
	if err != nil {
		return nil, err
	}

	// The npm version is a nice-to-have label; the ETag alone is enough to
	// tell whether there is something new to download.
	version, _ := npmVersion(ctx)

	return &providers.Release{
		Version:    version,
		ReleaseURL: "https://www.npmjs.com/package/tunnelmole",
		ETag:       etag,
		Asset:      providers.Asset{Name: filepath.Base(url), URL: url},
	}, nil
}

func (i *Installer) InstallRelease(ctx context.Context, rel *providers.Release, progress chan<- providers.DownloadProgress) error {
	sendProgress(progress, 10, false)

	staging, err := providers.StagingDir()
	if err != nil {
		return err
	}
	defer os.RemoveAll(staging)

	downloaded := filepath.Join(staging, rel.Asset.Name)
	etag, err := providers.Download(ctx, rel.Asset.URL, downloaded, rel.Asset.SHA256, progress, "tunnelmole")
	if err != nil {
		return fmt.Errorf("failed to download tunnelmole: %w", err)
	}

	binary := downloaded
	if strings.HasSuffix(downloaded, ".gz") {
		binary = filepath.Join(staging, binaryFile())
		if err := unpack(downloaded, binary); err != nil {
			return fmt.Errorf("failed to unpack tunnelmole: %w", err)
		}
	}

	dest := filepath.Join(providers.BinDir(), binaryFile())
	if err := providers.ReplaceBinary(binary, dest); err != nil {
		return fmt.Errorf("install tunnelmole: %w", err)
	}

	installed := *rel
	if version, err := providers.DetectVersion(ctx, dest); err == nil {
		installed.Version = version
	}
	if err := providers.RecordRelease("tunnelmole", dest, &installed, etag); err != nil {
		return fmt.Errorf("record tunnelmole install: %w", err)
	}

	sendProgress(progress, 100, true)
	return nil
}

func sendProgress(progress chan<- providers.DownloadProgress, percent float64, done bool) {
	if progress != nil {
		progress <- providers.DownloadProgress{Percent: percent, Total: 100, Done: done, Name: "tunnelmole"}
	}
}

func downloadURL(goos string) string {
	switch goos {
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

func headETag(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
	if err != nil {
		return "", err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("tunnelmole: %w", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("tunnelmole: %s returned %s", url, resp.Status)
	}

	if etag := resp.Header.Get("ETag"); etag != "" {
		return etag, nil
	}
	if modified := resp.Header.Get("Last-Modified"); modified != "" {
		return modified, nil
	}
	return "", fmt.Errorf("tunnelmole: %s has no ETag or Last-Modified header", url)
}

func npmVersion(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, npmLatest, nil)
	if err != nil {
		return "", err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("npm registry returned %s", resp.Status)
	}

	var pkg struct {
		Version string `json:"version"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&pkg); err != nil {
		return "", err
	}
	return providers.NormalizeVersion(pkg.Version), nil
}

// unpack writes the tmole executable inside a .gz download to dest. The file
// has shipped both as a bare gzip of the binary and as a gzipped tarball.
func unpack(src, dest string) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()

	br := bufio.NewReaderSize(gz, 1024)
	header, err := br.Peek(512)
	if err != nil && err != io.EOF {
		return err
	}

	var body io.Reader = br
	if isTar(header) {
		tr := tar.NewReader(br)
		for {
			h, err := tr.Next()
			if err == io.EOF {
				return fmt.Errorf("tmole binary not found in archive")
			}
			if err != nil {
				return err
			}
			if h.Typeflag == tar.TypeReg && strings.HasPrefix(filepath.Base(h.Name), "tmole") {
				body = tr
				break
			}
		}
	}

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, body); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

// isTar checks for the ustar magic at offset 257 of the first block.
func isTar(block []byte) bool {
	return len(block) >= 262 && string(block[257:262]) == "ustar"
}
