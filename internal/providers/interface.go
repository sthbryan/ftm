package providers

import (
	"context"
	"io"

	"foundry-tunnel/internal/config"
)

type Provider interface {
	Name() string
	BinaryName() string
	InstallURL() string
	RequiresAuth() bool
	
	Start(ctx context.Context, tunnel config.TunnelConfig, logWriter io.Writer) (*Process, error)
	ParseURL(line string) string
	IsReady(line string) bool
}

// AutoInstaller is implemented by providers that can auto-install themselves
type AutoInstaller interface {
	Provider
	IsInstalled() bool
	Install(progress chan<- DownloadProgress) error
}

type Process struct {
	Cancel context.CancelFunc
	PublicURL string
}

type LogLine struct {
	Line      string
	IsError   bool
	ContainsURL bool
}
