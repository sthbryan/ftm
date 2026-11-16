package provider

import (
	"context"
	"io"

	"github.com/sthbryan/ftm/internal/core"
)

type Provider interface {
	Name() string
	BinaryName() string

	Start(ctx context.Context, tunnel core.TunnelConfig, logWriter io.Writer) (*Process, error)
	ParseURL(line string) string
}

type AutoInstaller interface {
	Provider
	IsInstalled() bool
	Install(progress chan<- DownloadProgress) error
}

type LogLine struct {
	Line        string
	IsError     bool
	ContainsURL bool
}
