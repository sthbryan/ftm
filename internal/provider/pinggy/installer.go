package pinggy

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/sthbryan/ftm/internal/provider"
)

const githubRepo = "Pinggy-io/cli-js"

type Installer struct{}

func NewInstaller() *Installer {
	return &Installer{}
}

func binaryFile() string {
	if runtime.GOOS == "windows" {
		return "pinggy.exe"
	}
	return "pinggy"
}

func (i *Installer) PinggyBin() string {
	return provider.BundledPath(binaryFile())
}

func (i *Installer) IsInstalled() bool {
	_, err := os.Stat(i.PinggyBin())
	return err == nil
}

func (i *Installer) Install(progress chan<- provider.DownloadProgress) error {
	if i.IsInstalled() {
		return nil
	}

	ctx := context.Background()
	rel, err := i.LatestRelease(ctx)
	if err != nil {
		return fmt.Errorf("failed to get latest version: %w", err)
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
	etag, err := provider.InstallAsset(ctx, rel.Asset, dest, "pinggy", progress, "pinggy")
	if err != nil {
		return fmt.Errorf("install pinggy: %w", err)
	}

	if err := provider.RecordRelease("pinggy", dest, rel, etag); err != nil {
		return fmt.Errorf("record pinggy install: %w", err)
	}

	sendProgress(progress, 100, true)
	return nil
}

func sendProgress(progress chan<- provider.DownloadProgress, percent float64, done bool) {
	if progress != nil {
		progress <- provider.DownloadProgress{Percent: percent, Total: 100, Done: done, Name: "pinggy"}
	}
}

func assetName(goos, goarch string) (string, error) {
	arch := "x64"
	if goarch == "arm64" {
		arch = "arm64"
	}

	switch goos {
	case "darwin":
		return "pinggy-macos-" + arch, nil
	case "linux":
		return "pinggy-linux-" + arch, nil
	case "windows":
		return "pinggy-win-" + arch + ".exe", nil
	default:
		return "", fmt.Errorf("pinggy: unsupported platform %s/%s", goos, goarch)
	}
}
