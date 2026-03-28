package app

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/sthbryan/ftm/internal/config"
	"github.com/sthbryan/ftm/internal/providers"
)

type (
	statusUpdateMsg struct {
		tunnelID string
		status   config.TunnelStatus
	}
	tickMsg struct{}
)

func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		tickCmd(),
		m.checkDownloadProgress(),
	)
}

func tickCmd() tea.Cmd {
	return tea.Every(100*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg{}
	})
}

type downloadProgressMsg providers.DownloadProgress

func (m *Model) showMessage(msg string) {
	m.Message = msg
	m.MessageTimer = 30
}

func (m *Model) startTunnel(item TunnelItem) tea.Cmd {
	return func() tea.Msg {

		if needsInstall, canInstall := m.App.Manager.CheckInstallation(item.Tunnel.Provider); needsInstall && canInstall {
			m.DownloadingProvider = string(item.Tunnel.Provider)
			m.State = viewDownloading
			m.PendingTunnel = &item.Tunnel
			return m.installProvider(item.Tunnel.Provider)()
		}

		err := m.App.Manager.Start(item.Tunnel, func(status config.TunnelStatus) {
		})

		if err != nil {
			return statusUpdateMsg{
				tunnelID: item.Tunnel.ID,
				status:   config.TunnelStatus{ErrorMessage: err.Error(), State: config.TunnelStateError},
			}
		}

		return nil
	}
}

func (m *Model) installProvider(providerType config.Provider) tea.Cmd {
	return func() tea.Msg {
		err := m.App.Manager.InstallProvider(providerType)
		if err != nil {
			return statusUpdateMsg{
				tunnelID: "",
				status:   config.TunnelStatus{ErrorMessage: "Install failed: " + err.Error(), State: config.TunnelStateError},
			}
		}
		return downloadProgressMsg{Done: true, Percent: 100}
	}
}

func (m *Model) stopTunnel(item TunnelItem) tea.Cmd {
	return func() tea.Msg {
		m.App.Manager.Stop(item.Tunnel.ID)
		return nil
	}
}

func (m *Model) checkDownloadProgress() tea.Cmd {
	return func() tea.Msg {
		for p := range m.App.DownloadProgress {
			return downloadProgressMsg(p)
		}
		return nil
	}
}
