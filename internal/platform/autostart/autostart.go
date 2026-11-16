package autostart

import (
	"errors"

	"github.com/sthbryan/ftm/internal/buildinfo"
)

var (
	ErrUnsupported    = errors.New("autostart is not available on this build")
	ErrDisabledByUser = errors.New("autostart was turned off outside the app and must be re-enabled there")
)

type Manager interface {
	Supported() bool
	Enabled() (bool, error)
	Enable() error
	Disable() error
	Repair() error
}

func New() Manager {
	if !buildinfo.IsDesktop() {
		return unsupported{}
	}
	return newPlatformManager()
}

type unsupported struct{}

func (unsupported) Supported() bool        { return false }
func (unsupported) Enabled() (bool, error) { return false, nil }
func (unsupported) Enable() error          { return ErrUnsupported }
func (unsupported) Disable() error         { return ErrUnsupported }
func (unsupported) Repair() error          { return nil }

type pathBackend interface {
	Enabled() (bool, error)
	Enable() error
	registeredPath() (string, error)
}

func repairPath(b pathBackend) error {
	enabled, err := b.Enabled()
	if err != nil || !enabled {
		return err
	}

	registered, err := b.registeredPath()
	if err != nil {
		return err
	}

	want, err := execPath()
	if err != nil {
		return err
	}

	if samePath(registered, want) {
		return nil
	}

	return b.Enable()
}
