package tui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/sthbryan/ftm/internal/updater"
)

type updateCheckMsg struct {
	info *updater.Info
	err  error
}

type updateApplyMsg struct {
	err error
}

// checkUpdateCmd shows what the service already knows. The service checks on
// its own schedule and publishes UpdateAvailable when it finds one.
func (m *Model) checkUpdateCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := m.call()
		defer cancel()
		if info, err := m.App.UpdateInfo(ctx); err == nil && info != nil {
			return updateCheckMsg{info: info}
		}
		return nil
	}
}

func (m *Model) applyUpdateCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := m.call()
		defer cancel()
		return updateApplyMsg{err: m.App.ApplyUpdate(ctx)}
	}
}
