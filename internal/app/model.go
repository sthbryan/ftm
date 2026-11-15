package app

import (
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/sthbryan/ftm/internal/config"
	"github.com/sthbryan/ftm/internal/i18n"
	"github.com/sthbryan/ftm/internal/providers"
)

type (
	statusUpdateMsg struct {
		tunnelID string
		status   config.TunnelStatus
	}
	tickMsg struct{}
)

const (
	refreshInterval = time.Second
	messageTimeout  = 3 * time.Second
)

func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		tea.RequestBackgroundColor,
		tickCmd(),
		m.waitForStatus(),
		m.checkDownloadProgress(),
		checkUpdateCmd(),
	)
}

func tickCmd() tea.Cmd {
	return tea.Every(refreshInterval, func(t time.Time) tea.Msg {
		return tickMsg{}
	})
}

func (m *Model) waitForStatus() tea.Cmd {
	return func() tea.Msg {
		status, ok := <-m.statusUpdates
		if !ok {
			return nil
		}
		return statusUpdateMsg{tunnelID: status.ID, status: status}
	}
}

func (m *Model) publishStatus(status config.TunnelStatus) {
	select {
	case m.statusUpdates <- status:
	default:
	}
}

type downloadProgressMsg providers.DownloadProgress

func (m *Model) showMessage(msg string) {
	m.Message = msg
	m.messageUntil = time.Now().Add(messageTimeout)
}

func (m *Model) startTunnel(item TunnelItem) tea.Cmd {
	return func() tea.Msg {

		if needsInstall, canInstall := m.App.Manager.CheckInstallation(item.Tunnel.Provider); needsInstall && canInstall {
			m.DownloadingProvider = string(item.Tunnel.Provider)
			m.State = viewDownloading
			m.PendingTunnel = &item.Tunnel
			return m.installProvider(item.Tunnel.Provider)()
		}

		err := m.App.Manager.Start(item.Tunnel, m.publishStatus)

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
				status:   config.TunnelStatus{ErrorMessage: i18n.TF("error_install_failed", err.Error()), State: config.TunnelStateError},
			}
		}
		return downloadProgressMsg{Done: true, Percent: 100}
	}
}

func (m *Model) stopTunnel(item TunnelItem) tea.Cmd {
	return func() tea.Msg {
		_ = m.App.Manager.Stop(item.Tunnel.ID)
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
