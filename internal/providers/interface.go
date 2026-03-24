package providers

import (
	"context"
	"io"

	"github.com/sthbryan/ftm/internal/config"
)

type Provider interface {
	Name() string
	BinaryName() string

	Start(ctx context.Context, tunnel config.TunnelConfig, logWriter io.Writer) (*Process, error)
	ParseURL(line string) string
}

type AutoInstaller interface {
	Provider
	IsInstalled() bool
	Install(progress chan<- DownloadProgress) error
}

type Process struct {
	Cancel    context.CancelFunc
	PublicURL string
}

type LogLine struct {
	Line        string
	IsError     bool
	ContainsURL bool
}
