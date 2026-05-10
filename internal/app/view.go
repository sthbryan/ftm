package app

import (
	"github.com/sthbryan/ftm/internal/app/ui/views"
	"github.com/sthbryan/ftm/internal/config"
	"github.com/sthbryan/ftm/internal/i18n"
)

func (m *Model) View() string {
	if m.Width == 0 || m.Height == 0 {
		return i18n.T("loading")
	}

	switch m.State {
	case viewList:
		return m.viewList()
	case viewLogs:
		return m.viewLogs()
	case viewAddForm:
		return m.viewAddForm(false)
	case viewEditForm:
		return m.viewAddForm(true)
	case viewDownloading:
		return m.viewDownloading()
	case viewSettings:
		return m.viewSettings()
	default:
		return m.viewList()
	}
}

func (m *Model) viewList() string {
	if len(m.Items) == 0 {
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

	return view.Render()
}

func (m *Model) collectTunnelData() []views.TunnelViewData {
	data := make([]views.TunnelViewData, 0, len(m.Items))
	for _, item := range m.Items {
		if ti, ok := item.(TunnelItem); ok {
			data = append(data, views.TunnelViewData{
				Name:        ti.Tunnel.Name,
				Provider:    string(ti.Tunnel.Provider),
				LocalPort:   ti.Tunnel.LocalPort,
				StatusState: statusStateIndex(ti.Status.State),
				StatusMsg:   statusMsg(ti.Status.State),
				PublicURL:   ti.Status.PublicURL,
				ErrorMsg:    ti.Status.ErrorMessage,
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

func statusMsg(state config.TunnelState) string {
	switch state {
	case config.TunnelStateStarting:
		return i18n.T("starting")
	case config.TunnelStateConnecting:
		return i18n.T("connecting")
	case config.TunnelStateOnline:
		return i18n.T("online")
	case config.TunnelStateError:
		return i18n.T("error")
	case config.TunnelStateTimeout:
		return i18n.T("timeout")
	default:
		return i18n.T("offline")
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
	view := views.NewLogsView()
	view.Width = m.Width
	view.TunnelName = m.getTunnelName(m.SelectedTunnel)

	logs := m.App.Manager.GetLogs(m.SelectedTunnel)
	var content string
	for _, log := range logs {
		content += log + "\n"
	}
	view.Content = content

	m.updateLogViewport()

	return view.Render()
}

func (m *Model) getTunnelName(id string) string {
	if item, ok := m.selectedItem(); ok && item.Tunnel.ID == id {
		return item.Tunnel.Name
	}

	for _, t := range m.App.Config.Tunnels {
		if t.ID == id {
			return t.Name
		}
	}
	return ""
}

func (m *Model) viewAddForm(isEdit bool) string {
	view := views.NewFormView()
	view.Width = m.Width
	view.Focus = m.FormFocus
	view.IsEditMode = isEdit
	view.Name = m.FormValues.Name
	view.Provider = m.FormValues.Provider
	view.Port = m.FormValues.Port

	return view.Render()
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

func (m *Model) viewSettings() string {
	view := views.NewSettingsView()
	view.Width = m.Width
	if m.SettingsView != nil {
		view.NotificationsEnabled = m.SettingsView.NotificationsEnabled
		view.NotificationSound = m.SettingsView.NotificationSound
		view.Focused = m.SettingsView.Focused
	}
	return view.Render()
}
