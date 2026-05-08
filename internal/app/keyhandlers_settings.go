package app

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/sthbryan/ftm/internal/app/ui/views"
	"github.com/sthbryan/ftm/internal/config"
	"github.com/sthbryan/ftm/internal/i18n"
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
		if sv.Focused < 2 {
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
	}
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
	m.SettingsView.NotificationsEnabled = m.App.Config.NotificationsStatus == config.NotificationGranted
	m.SettingsView.NotificationSound = m.App.Config.NotificationSound
	m.SettingsView.Language = i18n.GetCurrentLang()
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
	m.App.Config.Language = sv.Language

	i18n.SetLanguage(sv.Language)

	m.App.SaveConfig()
}
