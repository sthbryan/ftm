package app

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/sthbryan/ftm/internal/app/ui/views"
	"github.com/sthbryan/ftm/internal/config"
)

func (m *Model) handleSettingsKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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
		if sv.Focused < 4 {
			sv.Focused++
		}

	case key.Matches(msg, m.Keys.Enter), key.Matches(msg, m.Keys.Toggle):
		m.handleSettingsSelect()
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
		sv.Theme = "light"
	case 3:
		sv.Theme = "dark"
	case 4:
		sv.Theme = "system"
	}
}

func (m *Model) openSettings() {
	m.SettingsView = views.NewSettingsView()
	m.SettingsView.NotificationsEnabled = m.App.Config.NotificationsStatus == config.NotificationGranted
	m.SettingsView.NotificationSound = m.App.Config.NotificationSound
	m.SettingsView.Theme = m.App.Config.Theme
	m.State = viewSettings
}

func (m *Model) saveSettings() {
	if m.SettingsView == nil {
		return
	}

	sv := m.SettingsView

	if sv.NotificationsEnabled {
		m.App.Config.NotificationsStatus = config.NotificationGranted
	} else {
		m.App.Config.NotificationsStatus = config.NotificationRejected
	}

	m.App.Config.NotificationSound = sv.NotificationSound
	m.App.Config.Theme = sv.Theme

	m.App.SaveConfig()
}
