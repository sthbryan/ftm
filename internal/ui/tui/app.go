package tui

import (
	"errors"

	tea "charm.land/bubbletea/v2"

	"github.com/sthbryan/ftm/internal/core"
	"github.com/sthbryan/ftm/internal/events"
	"github.com/sthbryan/ftm/internal/platform/open"
	"github.com/sthbryan/ftm/internal/service"
)

// Dashboard is the web dashboard the TUI links to.
type Dashboard interface {
	URL() string
	ClientCount() int
}

// App is what the TUI drives. Every change goes through Service and every
// update arrives on Bus; cmd/ftm wires them.
type App struct {
	Service *service.Service
	Bus     *events.Bus
	Web     Dashboard
}

// Run shows the TUI until the user quits. relaunch is true when an update was
// applied and the caller should shut down and start the new binary.
func Run(a *App) (relaunch bool, err error) {
	p := tea.NewProgram(NewModel(a))

	// An update applied from the dashboard ends the TUI the same way one
	// applied from the TUI does: quit, shut down, relaunch.
	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-a.Service.RestartRequested():
			p.Send(updateApplyMsg{})
		case <-done:
		}
	}()

	final, err := p.Run()
	if err != nil {
		return false, err
	}

	m, ok := final.(*Model)
	return ok && m.RelaunchOnExit, nil
}

// OpenDashboard opens the web dashboard in the default browser.
func (a *App) OpenDashboard() error {
	if a.Web == nil {
		return errors.New("web server not started")
	}
	return open.URL(a.Web.URL())
}

// OpenConfigDir opens the folder that holds config.yaml.
func (a *App) OpenConfigDir() error {
	return open.Dir(a.Service.Dirs().Config)
}

// Config returns a copy of the current config.
func (a *App) Config() core.Config {
	return a.Service.Config()
}
