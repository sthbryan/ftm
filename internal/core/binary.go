package core

import (
	"errors"
	"time"
)

// BinarySource says where the binary a provider runs comes from.
type BinarySource string

const (
	// SourceBundled is a copy ftm downloaded into its own bin directory.
	SourceBundled BinarySource = "bundled"
	// SourcePath is a copy the user installed (on PATH or a well-known
	// location). ftm reports its version but never replaces it.
	SourcePath BinarySource = "path"
	// SourceSystem is a tool the OS provides, such as ssh.
	SourceSystem BinarySource = "system"
	// SourceNone means the provider is not installed yet.
	SourceNone BinarySource = "none"
)

var (
	ErrNotUpdatable     = errors.New("provider binary is not managed by ftm")
	ErrExternalBinary   = errors.New("provider binary was installed outside ftm; update it with the tool that installed it")
	ErrUpdateInProgress = errors.New("an update for this provider is already running")
	ErrUnknownProvider  = errors.New("unknown provider")
)

// ProviderBinary describes the installed binary of one provider.
type ProviderBinary struct {
	ID         Provider
	Name       string
	Installed  bool
	Version    string
	Latest     string
	Source     BinarySource
	Path       string
	HasUpdate  bool
	Updatable  bool
	Updating   bool
	ReleaseURL string
	CheckedAt  time.Time
	Error      string
}

// UpdateResult is what a provider update did: the version now installed and
// the tunnels it restarted.
type UpdateResult struct {
	Provider  Provider
	Version   string
	Restarted []string
}
