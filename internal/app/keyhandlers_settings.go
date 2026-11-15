package app

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/sthbryan/ftm/internal/app/ui/views"
	"github.com/sthbryan/ftm/internal/config"
	"github.com/sthbryan/ftm/internal/i18n"
	"github.com/sthbryan/ftm/internal/service"
)

func (m *Model) handleSettingsKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.SettingsView == nil {
		m.State = viewList
		return m, nil
	}

	sv := m.SettingsView

	switch {
	case key.Matches(msg, m.Keys.Up):
		if sv.Focused > 0 {
			sv.Focused--
		}

	case key.Matches(msg, m.Keys.Down):
		if sv.Focused < views.SettingsRows-1 {
			sv.Focused++
		}

	case key.Matches(msg, m.Keys.Left):
		if sv.Focused == 2 {
			sv.Language = cycleLanguage(sv.Language, -1)
			i18n.SetLanguage(sv.Language)
		}

	case key.Matches(msg, m.Keys.Right):
		if sv.Focused == 2 {
			sv.Language = cycleLanguage(sv.Language, 1)
			i18n.SetLanguage(sv.Language)
		}

	case key.Matches(msg, m.Keys.Enter), key.Matches(msg, m.Keys.Toggle):
		m.handleSettingsSelect()

	case key.Matches(msg, m.Keys.Back), key.Matches(msg, m.Keys.Quit):
		m.saveSettings()
		m.SettingsView = nil
		m.State = viewList
	}

	return m, nil
}

func (m *Model) handleSettingsSelect() {
	if m.SettingsView == nil {
		return
	}

	sv := m.SettingsView

	switch sv.Focused {
	case 0:
		sv.NotificationsEnabled = !sv.NotificationsEnabled
	case 1:
		sv.NotificationSound = !sv.NotificationSound
	case 2:

		sv.Language = cycleLanguage(sv.Language, 1)
		i18n.SetLanguage(sv.Language)
	case 3:
		if len(m.App.Service.Tunnels()) == 0 {
			return
		}
		m.pendingConfirm = confirmClearTunnels
		m.State = viewConfirm
	}
}

func (m *Model) confirmClearTunnels() (tea.Model, tea.Cmd) {
	m.pendingConfirm = confirmDeleteTunnel
	m.saveSettings()
	m.SettingsView = nil

	_, saveErr := m.App.Service.ClearTunnels()
	m.refreshItems()

	m.Cursor = 0
	m.SelectedTunnel = ""
	m.State = viewList

	if saveErr != nil {
		m.showMessage(i18n.T("error_saving_config") + ": " + saveErr.Error())
	} else {
		m.showMessage(i18n.T("connections_cleared"))
	}

	return m, nil
}

func cycleLanguage(current string, dir int) string {
	langs := i18n.SupportedLanguages()
	for i, lang := range langs {
		if lang == current {
			next := i + dir
			if next < 0 {
				next = len(langs) - 1
			} else if next >= len(langs) {
				next = 0
			}
			return langs[next]
		}
	}
	return langs[0]
}

func (m *Model) openSettings() {
	m.SettingsView = views.NewSettingsView()
	cfg := m.App.Config()
	m.SettingsView.NotificationsEnabled = cfg.NotificationsStatus == config.NotificationGranted
	m.SettingsView.NotificationSound = cfg.NotificationSound
	m.SettingsView.Language = i18n.GetCurrentLang()
	m.SettingsView.TunnelCount = len(cfg.Tunnels)
	m.State = viewSettings
}

func (m *Model) saveSettings() {
	if m.SettingsView == nil {
		return
	}

	sv := m.SettingsView

	status := config.NotificationRejected
	if sv.NotificationsEnabled {
		status = config.NotificationGranted
	}
	sound, lang := sv.NotificationSound, sv.Language

	_, err := m.App.Service.UpdateSettings(service.SettingsPatch{
		NotificationsStatus: &status,
		NotificationSound:   &sound,
		Language:            &lang,
	})

	if err != nil {
		m.showMessage(i18n.T("error_saving_config") + ": " + err.Error())
	}
}
