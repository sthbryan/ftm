package tui

import (
	"errors"

	tea "charm.land/bubbletea/v2"

	"github.com/sthbryan/ftm/internal/platform/open"
)

// restarter is a backend whose process restarts after an update applied
// from the web view.
type restarter interface {
	RestartRequested() <-chan struct{}
}

// Run shows the TUI until the user quits. relaunch is true when an update was
// applied to this process and the caller should shut down and start the new
// binary.
func Run(b Backend) (relaunch bool, err error) {
	m := NewModel(b)
	defer m.Close()
	p := tea.NewProgram(m)

	// An update applied from the web view ends the TUI the same way one
	// applied from the TUI does: quit, shut down, relaunch.
	if r, ok := b.(restarter); ok {
		done := make(chan struct{})
		defer close(done)
		go func() {
			select {
			case <-r.RestartRequested():
				p.Send(updateApplyMsg{})
			case <-done:
			}
		}()
	}

	final, err := p.Run()
	if err != nil {
		return false, err
	}

	fm, ok := final.(*Model)
	return ok && fm.RelaunchOnExit, nil
}

// openDashboard opens the web view in the default browser.
func openDashboard(b Backend) error {
	url := b.DashboardURL()
	if url == "" {
		return errors.New("web server not started")
	}
	return open.URL(url)
}
