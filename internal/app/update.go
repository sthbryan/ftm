package app

import (
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/sthbryan/ftm/internal/app/ui"
	"github.com/sthbryan/ftm/internal/i18n"
	"github.com/sthbryan/ftm/internal/providers"
)

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		return m.handleKey(msg)

	case tea.MouseMsg:
		return m.handleMouse(msg)

	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		return m, nil

	case tea.BackgroundColorMsg:
		ui.SetDarkBackground(msg.IsDark())
		return m, nil

	case tickMsg:
		return m.handleTick()

	case downloadProgressMsg:
		return m.handleDownloadProgress(msg)

	case statusUpdateMsg:
		return m.handleStatusUpdate(msg)

	case updateCheckMsg:
		return m.handleUpdateCheck(msg)

	case updateApplyMsg:
		return m.handleUpdateApply(msg)

	case providerSummaryMsg:
		return m.handleProviderSummary(msg)
	}

	return m, nil
}

func (m *Model) handleTick() (tea.Model, tea.Cmd) {
	m.refreshItems()

	if m.Message != "" && time.Now().After(m.messageUntil) {
		m.Message = ""
	}
	return m, tickCmd()
}

func (m *Model) handleDownloadProgress(msg downloadProgressMsg) (tea.Model, tea.Cmd) {
	m.DownloadProgress = providers.DownloadProgress(msg)
	if m.DownloadProgress.Done {
		m.State = viewList
		if m.PendingTunnel != nil {
			for _, item := range m.Items {
				if ti, ok := item.(TunnelItem); ok && ti.Tunnel.ID == m.PendingTunnel.ID {
					m.PendingTunnel = nil
					m.showMessage(i18n.T("install_complete"))
					return m, m.startTunnel(ti)
				}
			}
			m.PendingTunnel = nil
		}
		m.showMessage(i18n.T("download_complete"))
	}
	return m, m.checkDownloadProgress()
}

func (m *Model) handleStatusUpdate(msg statusUpdateMsg) (tea.Model, tea.Cmd) {
	m.refreshItems()
	if msg.status.ErrorMessage != "" {
		m.playBeep()
		m.showMessage(i18n.TF("error_state", msg.status.ErrorMessage))
	}
	return m, m.waitForStatus()
}

func (m *Model) handleUpdateCheck(msg updateCheckMsg) (tea.Model, tea.Cmd) {
	if msg.err == nil && msg.info != nil && msg.info.HasUpdate {
		m.UpdateAvailable = msg.info
	}
	return m, scheduleUpdateRecheck()
}

func (m *Model) handleUpdateApply(msg updateApplyMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.showMessage(i18n.TF("update_apply_failed", msg.err.Error()))
		return m, nil
	}
	m.RelaunchOnExit = true
	return m, tea.Quit
}

func scheduleUpdateRecheck() tea.Cmd {
	return tea.Tick(6*time.Hour, func(time.Time) tea.Msg {
		return checkUpdateCmd()()
	})
}
