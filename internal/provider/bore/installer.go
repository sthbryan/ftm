package bore

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/sthbryan/ftm/internal/provider"
)

const (
	githubRepo = "ekzhang/bore"
	// fallbackVersion is installed when the GitHub API cannot be reached, so a
	// first install still works while rate limited.
	fallbackVersion = "v0.6.0"
)

type Installer struct{}

func NewInstaller() *Installer {
	return &Installer{}
}

func binaryFile() string {
	if runtime.GOOS == "windows" {
		return "bore.exe"
	}
	return "bore"
}

func (i *Installer) BoreBin() string {
	return provider.BundledPath(binaryFile())
}

func (i *Installer) IsInstalled() bool {
	_, err := os.Stat(i.BoreBin())
	return err == nil
}

func (i *Installer) Install(progress chan<- provider.DownloadProgress) error {
	if i.IsInstalled() {
		return nil
	}

	ctx := context.Background()
	rel, err := i.LatestRelease(ctx)
	if err != nil {
		name, nameErr := assetName(fallbackVersion, runtime.GOOS, runtime.GOARCH)
		if nameErr != nil {
			return nameErr
		}
		rel = &provider.Release{
			Version: provider.NormalizeVersion(fallbackVersion),
			Asset: provider.Asset{
				Name: name,
				URL:  "https://github.com/" + githubRepo + "/releases/download/" + fallbackVersion + "/" + name,
			},
		}
	}

	return i.InstallRelease(ctx, rel, progress)
}

func (i *Installer) LatestRelease(ctx context.Context) (*provider.Release, error) {
	gh, err := provider.FetchLatestGitHubRelease(ctx, githubRepo)
	if err != nil {
		return nil, err
	}

	name, err := assetName(gh.TagName, runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return nil, err
	}
	return gh.Release(name)
}

func (i *Installer) InstallRelease(ctx context.Context, rel *provider.Release, progress chan<- provider.DownloadProgress) error {
	sendProgress(progress, 10, false)

	dest := filepath.Join(provider.BinDir(), binaryFile())
	etag, err := provider.InstallAsset(ctx, rel.Asset, dest, "bore", progress, "bore")
	if err != nil {
		return fmt.Errorf("install bore: %w", err)
	}

	if err := provider.RecordRelease("bore", dest, rel, etag); err != nil {
		return fmt.Errorf("record bore install: %w", err)
	}

	sendProgress(progress, 100, true)
	return nil
}

func sendProgress(progress chan<- provider.DownloadProgress, percent float64, done bool) {
	if progress != nil {
		progress <- provider.DownloadProgress{Percent: percent, Total: 100, Done: done, Name: "bore"}
	}
}

// assetName is the release archive for a platform, e.g.
// bore-v0.6.0-aarch64-apple-darwin.tar.gz.
func assetName(tag, goos, goarch string) (string, error) {
	var target string
	switch goos {
	case "darwin":
		switch goarch {
		case "arm64":
			target = "aarch64-apple-darwin.tar.gz"
		case "amd64":
			target = "x86_64-apple-darwin.tar.gz"
		}
	case "linux":
		switch goarch {
		case "arm64":
			target = "aarch64-unknown-linux-musl.tar.gz"
		case "amd64":
			target = "x86_64-unknown-linux-musl.tar.gz"
		case "386":
			target = "i686-unknown-linux-musl.tar.gz"
		}
	case "windows":
		switch goarch {
		case "amd64":
			target = "x86_64-pc-windows-msvc.zip"
		case "386":
			target = "i686-pc-windows-msvc.zip"
		}
	}

	if target == "" {
		return "", fmt.Errorf("bore: unsupported platform %s/%s", goos, goarch)
	}
	return "bore-" + tag + "-" + target, nil
}
