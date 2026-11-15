package app

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/sthbryan/ftm/internal/app/ui/views"
	"github.com/sthbryan/ftm/internal/config"
	"github.com/sthbryan/ftm/internal/i18n"
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
		if len(m.App.Store.Tunnels()) == 0 {
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

	tunnels := m.App.Store.Tunnels()
	ids := make([]string, 0, len(tunnels))
	for _, t := range tunnels {
		ids = append(ids, t.ID)
	}

	for _, id := range ids {
		_ = m.App.Manager.Stop(id)
	}

	saveErr := m.App.UpdateConfig(func(c *config.Config) error {
		c.ClearTunnels()
		return nil
	})
	m.refreshItems()

	if m.App.WebServer != nil {
		for _, id := range ids {
			m.App.WebServer.BroadcastTunnelDeleted(id)
		}
	}

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

	err := m.App.UpdateConfig(func(c *config.Config) error {
		if sv.NotificationsEnabled {
			c.NotificationsStatus = config.NotificationGranted
		} else {
			c.NotificationsStatus = config.NotificationRejected
		}
		c.NotificationSound = sv.NotificationSound
		c.Language = sv.Language
		return nil
	})

	i18n.SetLanguage(sv.Language)

	if err != nil {
		m.showMessage(i18n.T("error_saving_config") + ": " + err.Error())
	}
}
