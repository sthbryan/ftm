package app

import (
	tea "charm.land/bubbletea/v2"

	"github.com/sthbryan/ftm/internal/app/ui"
	"github.com/sthbryan/ftm/internal/app/ui/views"
	"github.com/sthbryan/ftm/internal/config"
	"github.com/sthbryan/ftm/internal/i18n"
)

func (m *Model) View() tea.View {
	var content string
	var cursor *tea.Cursor
	if m.Width == 0 || m.Height == 0 {
		content = i18n.T("loading")
	} else {
		switch m.State {
		case viewList:
			content = m.viewList()
		case viewLogs:
			content = ui.Overlay(m.viewList(), m.viewLogs(), m.Width, m.Height)
		case viewNewTunnel, viewEditTunnel:
			content, cursor = m.overlayTunnelEditor()
		case viewDownloading:
			content = m.viewDownloading()
		case viewSettings:
			content = ui.Overlay(m.viewList(), m.viewSettings(), m.Width, m.Height)
		case viewConfirm:
			content = ui.Overlay(m.confirmBackdrop(), m.viewConfirm(), m.Width, m.Height)
		default:
			content = m.viewList()
		}
	}

	v := tea.NewView(ui.Fill(content, m.Width, m.Height))
	v.Cursor = cursor
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	return v
}

func (m *Model) viewList() string {
	if len(m.Items) == 0 && !m.App.Config().Onboarded {
		return m.viewEmptyState()
	}

	view := views.NewListView()
	view.Width = m.Width
	view.Height = m.Height
	view.Items = m.collectTunnelData()
	view.Cursor = m.Cursor
	view.Message = m.Message
	view.Dashboard = m.App.WebServer.URL()
	view.Sessions = m.App.WebServer.ClientCount()
	view.TwoColumnLimit = TwoColumnThreshold
	view.Shortcuts = m.Keys.listShortcuts()
	if m.UpdateAvailable != nil {
		view.UpdateBadge = i18n.TF("update_tui_badge", m.UpdateAvailable.LatestVersion)
	}

	rendered := view.Render()
	m.listTop = view.ListTop
	m.listFirst = view.FirstItem

	return rendered
}

func (m *Model) collectTunnelData() []views.TunnelViewData {
	data := make([]views.TunnelViewData, 0, len(m.Items))
	for i, item := range m.Items {
		if ti, ok := item.(TunnelItem); ok {
			var logs []string
			if i == m.Cursor {
				logs = m.App.Manager.GetLogs(ti.Tunnel.ID)
			}

			data = append(data, views.TunnelViewData{
				Name:           ti.Tunnel.Name,
				Provider:       string(ti.Tunnel.Provider),
				LocalPort:      ti.Tunnel.LocalPort,
				StatusState:    statusStateIndex(ti.Status.State),
				PublicURL:      ti.Status.PublicURL,
				Visitors:       ti.Status.Visitors,
				ActiveSessions: ti.Status.ActiveSessions,
				ErrorMsg:       ti.Status.ErrorMessage,
				Logs:           logs,
			})
		}
	}
	return data
}

func statusStateIndex(state config.TunnelState) int {
	switch state {
	case config.TunnelStateStarting:
		return 1
	case config.TunnelStateConnecting:
		return 2
	case config.TunnelStateOnline:
		return 3
	case config.TunnelStateError:
		return 4
	case config.TunnelStateTimeout:
		return 5
	case config.TunnelStateStopped:
		return 6
	default:
		return 0
	}
}

func (m *Model) viewEmptyState() string {
	view := views.NewEmptyState()
	view.Height = m.Height
	view.Width = m.Width
	view.Dashboard = m.App.WebServer.URL()
	view.Sessions = m.App.WebServer.ClientCount()

	return view.Render()
}

func (m *Model) viewLogs() string {
	width, height := views.LogsBox(m.Width, m.Height)

	m.LogViewport.SetWidth(ui.PanelInner(width))
	m.LogViewport.SetHeight(height - ui.PanelChrome - 1)
	m.updateLogViewport()

	view := views.NewLogsView()
	view.Width = width
	view.Height = height
	view.TunnelName = m.getTunnelName(m.SelectedTunnel)
	view.Content = m.LogViewport.View()

	return view.Render()
}

func (m *Model) getTunnelName(id string) string {
	if item, ok := m.selectedItem(); ok && item.Tunnel.ID == id {
		return item.Tunnel.Name
	}

	for _, t := range m.App.Store.Tunnels() {
		if t.ID == id {
			return t.Name
		}
	}
	return ""
}

func (m *Model) overlayTunnelEditor() (string, *tea.Cursor) {
	view := views.NewTunnelEditor()
	view.Providers = providerNames()
	view.Focus = m.EditorFocus
	view.IsEditMode = m.State == viewEditTunnel
	view.Name = m.Draft.Name
	view.Provider = m.Draft.Provider
	view.Port = m.Draft.Port

	editor := view.Render()
	content := ui.Overlay(m.viewList(), editor, m.Width, m.Height)

	if !view.HasCursor {
		return content, nil
	}

	x, y := ui.OverlayOrigin(editor, m.Width, m.Height)

	return content, tea.NewCursor(x+view.CursorColumn, y+view.CursorRow)
}

func providerNames() []string {
	all := config.AllProviders()

	names := make([]string, 0, len(all))
	for _, provider := range all {
		names = append(names, i18n.ProviderText(string(provider)))
	}

	return names
}

func (m *Model) viewDownloading() string {
	view := views.NewDownloadingView()
	view.Width = m.Width
	view.Percent = m.DownloadProgress.Percent
	view.Name = m.DownloadProgress.Name
	view.Current = m.DownloadProgress.Current
	view.Total = m.DownloadProgress.Total

	progressView := m.ProgressBar.ViewAs(view.Percent / 100)

	return view.Render(progressView)
}

func (m *Model) viewConfirm() string {
	view := views.NewConfirmView()
	view.Danger = true

	if m.pendingConfirm == confirmClearTunnels {
		view.Title = i18n.T("confirm_reset_title")
		view.Message = m.clearTunnelsMessage()
		return view.Render()
	}

	view.Title = i18n.T("confirm_delete_title")
	view.Message = i18n.TF("confirm_delete_body", m.pendingDeleteName)

	return view.Render()
}

func (m *Model) confirmBackdrop() string {
	if m.pendingConfirm == confirmClearTunnels {
		return ui.Overlay(m.viewList(), m.viewSettings(), m.Width, m.Height)
	}
	return m.viewList()
}

func (m *Model) clearTunnelsMessage() string {
	count := len(m.App.Store.Tunnels())
	if count == 1 {
		return i18n.T("confirm_reset_body_one")
	}
	return i18n.TF("confirm_reset_body_other", count)
}

func (m *Model) viewSettings() string {
	view := views.NewSettingsView()
	if m.SettingsView != nil {
		view.NotificationsEnabled = m.SettingsView.NotificationsEnabled
		view.NotificationSound = m.SettingsView.NotificationSound
		view.Focused = m.SettingsView.Focused
	}
	view.TunnelCount = len(m.App.Store.Tunnels())
	return view.Render()
}
