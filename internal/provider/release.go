package provider

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	githubAPI       = "https://api.github.com"
	apiTimeout      = 15 * time.Second
	downloadTimeout = 10 * time.Minute
	userAgent       = "ftm-provider-updater"
)

var apiClient = &http.Client{Timeout: apiTimeout}

// Release is one downloadable version of a provider binary for this platform.
type Release struct {
	// Version is the release version without a "v" prefix. It can be empty for
	// a provider that does not version its download (see ETag).
	Version string
	// ReleaseURL is a page with the release notes, when there is one.
	ReleaseURL string
	// Asset is the file to download for runtime.GOOS/GOARCH.
	Asset Asset
	// ETag identifies an unversioned download. A provider that sets it is
	// considered out of date when the recorded ETag differs.
	ETag string
}

// Asset is a single download.
type Asset struct {
	Name string
	URL  string
	// SHA256 is the expected hex digest, or "" when the publisher does not
	// provide one.
	SHA256 string
}

// Updatable is a provider whose binary ftm downloads and can replace.
type Updatable interface {
	AutoInstaller

	// BundledBinary is the path of ftm's own copy, which may not exist yet.
	BundledBinary() string
	// FindBinary is the binary the provider will actually run: a copy on PATH
	// wins over the bundled one.
	FindBinary() string
	// LatestRelease looks up the newest release for this platform.
	LatestRelease(ctx context.Context) (*Release, error)
	// InstallRelease downloads rel and swaps it in for the bundled copy.
	InstallRelease(ctx context.Context, rel *Release, progress chan<- DownloadProgress) error
}

// GitHubRelease is the subset of the GitHub release API ftm reads.
type GitHubRelease struct {
	TagName    string        `json:"tag_name"`
	HTMLURL    string        `json:"html_url"`
	Prerelease bool          `json:"prerelease"`
	Draft      bool          `json:"draft"`
	Assets     []GitHubAsset `json:"assets"`
}

type GitHubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	// Digest is "sha256:<hex>"; GitHub computes it for every uploaded asset.
	Digest string `json:"digest"`
}

// FetchLatestGitHubRelease returns the latest stable release of repo
// ("owner/name").
func FetchLatestGitHubRelease(ctx context.Context, repo string) (*GitHubRelease, error) {
	url := fmt.Sprintf("%s/repos/%s/releases/latest", githubAPI, repo)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", userAgent)
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := apiClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("github: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("github %s: status %d: %s", repo, resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var rel GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, fmt.Errorf("github %s: decode: %w", repo, err)
	}
	if rel.Draft || rel.Prerelease || rel.TagName == "" {
		return nil, fmt.Errorf("github %s: latest release is not a stable tagged release", repo)
	}
	return &rel, nil
}

// Asset finds the named asset and carries over its published digest.
func (r *GitHubRelease) Asset(name string) (Asset, error) {
	for _, a := range r.Assets {
		if a.Name == name {
			return Asset{
				Name:   a.Name,
				URL:    a.BrowserDownloadURL,
				SHA256: digestHex(a.Digest),
			}, nil
		}
	}
	return Asset{}, fmt.Errorf("release %s has no asset %q", r.TagName, name)
}

// Release converts a GitHub release into a Release for the named asset.
func (r *GitHubRelease) Release(assetName string) (*Release, error) {
	asset, err := r.Asset(assetName)
	if err != nil {
		return nil, err
	}
	return &Release{
		Version:    NormalizeVersion(r.TagName),
		ReleaseURL: r.HTMLURL,
		Asset:      asset,
	}, nil
}

func digestHex(digest string) string {
	algo, value, ok := strings.Cut(digest, ":")
	if !ok || !strings.EqualFold(algo, "sha256") {
		return ""
	}
	return strings.ToLower(value)
}

// ErrChecksumMismatch means a download did not match its published digest.
var ErrChecksumMismatch = errors.New("checksum mismatch")

// Download fetches url into dest, reporting progress between 10% and 90% and
// checking the SHA-256 digest when one is given. It returns the response ETag.
func Download(ctx context.Context, url, dest, wantSHA256 string, progress chan<- DownloadProgress, name string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, downloadTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download %s: %s", url, resp.Status)
	}

	out, err := os.Create(dest)
	if err != nil {
		return "", err
	}

	hash := sha256.New()
	total := resp.ContentLength
	var current int64
	buf := make([]byte, 64*1024)

	for {
		n, rerr := resp.Body.Read(buf)
		if n > 0 {
			if _, werr := out.Write(buf[:n]); werr != nil {
				out.Close()
				return "", fmt.Errorf("write %s: %w", dest, werr)
			}
			hash.Write(buf[:n])
			current += int64(n)
			if progress != nil && total > 0 {
				progress <- DownloadProgress{
					Percent: 10 + float64(current)/float64(total)*80,
					Current: current,
					Total:   total,
					Name:    name,
				}
			}
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			out.Close()
			return "", rerr
		}
	}

	if err := out.Close(); err != nil {
		return "", err
	}

	if wantSHA256 != "" {
		got := hex.EncodeToString(hash.Sum(nil))
		if !strings.EqualFold(got, wantSHA256) {
			return "", fmt.Errorf("%w for %s: got %s, want %s", ErrChecksumMismatch, filepath.Base(url), got, wantSHA256)
		}
	}

	return resp.Header.Get("ETag"), nil
}

// StagingDir creates a scratch directory next to the bundled binaries, so the
// final rename stays on one volume.
func StagingDir() (string, error) {
	if err := os.MkdirAll(BinDir(), 0755); err != nil {
		return "", err
	}
	return os.MkdirTemp(BinDir(), ".ftm-update-*")
}

// InstallAsset downloads asset, verifies it, extracts binaryName when the
// asset is an archive, and atomically replaces dest with the result.
func InstallAsset(ctx context.Context, asset Asset, dest, binaryName string, progress chan<- DownloadProgress, name string) (etag string, err error) {
	staging, err := StagingDir()
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(staging)

	downloaded := filepath.Join(staging, asset.Name)
	if asset.Name == "" {
		downloaded = filepath.Join(staging, "download")
	}

	etag, err = Download(ctx, asset.URL, downloaded, asset.SHA256, progress, name)
	if err != nil {
		return "", err
	}

	binary := downloaded
	if isArchive(downloaded) {
		extractDir := filepath.Join(staging, "extract")
		if err := os.MkdirAll(extractDir, 0755); err != nil {
			return "", err
		}
		if err := ExtractArchive(downloaded, extractDir, binaryName); err != nil {
			return "", err
		}
		binary = filepath.Join(extractDir, binaryName)
		if runtime.GOOS == "windows" {
			if _, err := os.Stat(binary); err != nil {
				binary += ".exe"
			}
		}
	}

	if err := ReplaceBinary(binary, dest); err != nil {
		return "", err
	}
	return etag, nil
}

func isArchive(path string) bool {
	lower := strings.ToLower(path)
	return strings.HasSuffix(lower, ".zip") || strings.HasSuffix(lower, ".tar.gz") || strings.HasSuffix(lower, ".tgz")
}

// ReplaceBinary makes src executable and moves it over dest. Windows will not
// replace a file another handle holds, so an existing dest is first moved
// aside and restored if the swap fails.
func ReplaceBinary(src, dest string) error {
	if err := os.Chmod(src, 0755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return err
	}

	if err := os.Rename(src, dest); err == nil {
		return nil
	} else if _, statErr := os.Stat(dest); statErr != nil {
		return err
	}

	old := dest + ".old"
	_ = os.Remove(old)
	if err := os.Rename(dest, old); err != nil {
		return fmt.Errorf("replace %s (is it still running?): %w", filepath.Base(dest), err)
	}
	if err := os.Rename(src, dest); err != nil {
		_ = os.Rename(old, dest)
		return fmt.Errorf("replace %s: %w", filepath.Base(dest), err)
	}
	// Windows can refuse to delete an image that is still mapped; the next
	// update removes it.
	_ = os.Remove(old)
	return nil
}
