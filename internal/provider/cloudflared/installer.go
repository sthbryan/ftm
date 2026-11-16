package cloudflared

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/sthbryan/ftm/internal/provider"
)

const (
	githubRepo = "cloudflare/cloudflared"
	// latestDownload serves the newest asset without the API, for when the
	// GitHub API is rate limited during a first install.
	latestDownload = "https://github.com/cloudflare/cloudflared/releases/latest/download/"
)

type Installer struct{}

func NewInstaller() *Installer {
	return &Installer{}
}

func binaryFile() string {
	if runtime.GOOS == "windows" {
		return "cloudflared.exe"
	}
	return "cloudflared"
}

func (i *Installer) CloudflaredBin() string {
	return provider.BundledPath(binaryFile())
}

func (i *Installer) IsInstalled() bool {
	_, err := os.Stat(i.CloudflaredBin())
	return err == nil
}

// Install downloads cloudflared once; updates go through InstallRelease.
func (i *Installer) Install(progress chan<- provider.DownloadProgress) error {
	if i.IsInstalled() {
		return nil
	}

	ctx := context.Background()
	rel, err := i.LatestRelease(ctx)
	if err != nil {
		name, nameErr := assetName(runtime.GOOS, runtime.GOARCH)
		if nameErr != nil {
			return nameErr
		}
		rel = &provider.Release{Asset: provider.Asset{Name: name, URL: latestDownload + name}}
	}

	return i.InstallRelease(ctx, rel, progress)
}

func (i *Installer) LatestRelease(ctx context.Context) (*provider.Release, error) {
	name, err := assetName(runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return nil, err
	}

	gh, err := provider.FetchLatestGitHubRelease(ctx, githubRepo)
	if err != nil {
		return nil, err
	}
	return gh.Release(name)
}

func (i *Installer) InstallRelease(ctx context.Context, rel *provider.Release, progress chan<- provider.DownloadProgress) error {
	sendProgress(progress, 10, false)

	dest := filepath.Join(provider.BinDir(), binaryFile())
	etag, err := provider.InstallAsset(ctx, rel.Asset, dest, "cloudflared", progress, "cloudflared")
	if err != nil {
		return fmt.Errorf("install cloudflared: %w", err)
	}

	if err := provider.RecordRelease("cloudflared", dest, rel, etag); err != nil {
		return fmt.Errorf("record cloudflared install: %w", err)
	}

	sendProgress(progress, 100, true)
	return nil
}

func sendProgress(progress chan<- provider.DownloadProgress, percent float64, done bool) {
	if progress != nil {
		progress <- provider.DownloadProgress{Percent: percent, Total: 100, Done: done, Name: "cloudflared"}
	}
}

// assetName is the release file for a platform. macOS ships a tarball; Linux
// and Windows ship the bare executable.
func assetName(goos, goarch string) (string, error) {
	switch goos {
	case "darwin":
		if goarch == "arm64" {
			return "cloudflared-darwin-arm64.tgz", nil
		}
		return "cloudflared-darwin-amd64.tgz", nil
	case "linux":
		switch goarch {
		case "arm64":
			return "cloudflared-linux-arm64", nil
		case "arm":
			return "cloudflared-linux-arm", nil
		case "386":
			return "cloudflared-linux-386", nil
		}
		return "cloudflared-linux-amd64", nil
	case "windows":
		if goarch == "386" {
			return "cloudflared-windows-386.exe", nil
		}
		return "cloudflared-windows-amd64.exe", nil
	default:
		return "", fmt.Errorf("cloudflared: unsupported OS %s", goos)
	}
}
