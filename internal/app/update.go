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

	case busMsg:
		model, cmd := m.Update(msg.msg)
		return model, tea.Batch(cmd, m.waitForEvent())

	case tunnelsChangedMsg:
		m.refreshItems()
		return m, nil

	case installStartedMsg:
		m.DownloadingProvider = string(msg.provider)
		m.DownloadProgress = providers.DownloadProgress{}
		m.State = viewDownloading
		return m, nil

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

// handleDownloadProgress follows a provider download. The service starts the
// waiting tunnel itself once it finishes; the model only leaves the progress
// screen if it is still on it.
func (m *Model) handleDownloadProgress(msg downloadProgressMsg) (tea.Model, tea.Cmd) {
	m.DownloadProgress = providers.DownloadProgress(msg)

	failed := m.DownloadProgress.Error != nil
	if !m.DownloadProgress.Done && !failed {
		return m, nil
	}
	if m.State != viewDownloading {
		return m, nil
	}

	m.State = viewList
	if !failed {
		m.showMessage(i18n.T("install_complete"))
	}
	return m, nil
}

func (m *Model) handleStatusUpdate(msg statusUpdateMsg) (tea.Model, tea.Cmd) {
	m.refreshItems()
	if msg.status.ErrorMessage != "" {
		if m.State == viewDownloading {
			m.State = viewList
		}
		m.playBeep()
		m.showMessage(i18n.TF("error_state", msg.status.ErrorMessage))
	}
	return m, nil
}

func (m *Model) handleUpdateCheck(msg updateCheckMsg) (tea.Model, tea.Cmd) {
	if msg.err == nil && msg.info != nil && msg.info.HasUpdate {
		m.UpdateAvailable = msg.info
	}
	return m, nil
}

func (m *Model) handleUpdateApply(msg updateApplyMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.showMessage(i18n.TF("update_apply_failed", msg.err.Error()))
		return m, nil
	}
	m.RelaunchOnExit = true
	return m, tea.Quit
}
