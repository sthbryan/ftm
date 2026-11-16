package tui

import (
	"errors"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/sthbryan/ftm/internal/core"
	"github.com/sthbryan/ftm/internal/events"
	"github.com/sthbryan/ftm/internal/provider"
)

type (
	statusUpdateMsg struct {
		tunnelID string
		status   core.TunnelStatus
	}
	tickMsg struct{}

	// installStartedMsg says a start is waiting for its provider download.
	installStartedMsg struct {
		provider core.Provider
	}

	// busMsg wraps one event from the bus; handling it re-arms the wait for
	// the next one.
	busMsg struct {
		msg tea.Msg
	}

	// tunnelsChangedMsg is any change to the saved connections.
	tunnelsChangedMsg struct{}
)

const (
	refreshInterval = time.Second
	messageTimeout  = 3 * time.Second
)

func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		tea.RequestBackgroundColor,
		tickCmd(),
		m.waitForEvent(),
		m.checkUpdateCmd(),
	)
}

func tickCmd() tea.Cmd {
	return tea.Every(refreshInterval, func(t time.Time) tea.Msg {
		return tickMsg{}
	})
}

// waitForEvent turns the next bus event into a model message.
func (m *Model) waitForEvent() tea.Cmd {
	if m.events == nil {
		return nil
	}
	ch := m.events
	return func() tea.Msg {
		for e := range ch {
			if msg := m.eventMsg(e); msg != nil {
				return busMsg{msg: msg}
			}
		}
		return nil
	}
}

func (m *Model) eventMsg(e events.Event) tea.Msg {
	switch e := e.(type) {
	case events.TunnelStatus:
		return statusUpdateMsg{tunnelID: e.Status.ID, status: e.Status}
	case events.InstallProgress:
		p := provider.DownloadProgress{
			Name:    e.Provider,
			Percent: e.Percent,
			Current: e.Current,
			Total:   e.Total,
			Done:    e.Done,
		}
		if e.Err != "" {
			p.Error = errors.New(e.Err)
		}
		return downloadProgressMsg(p)
	case events.TunnelSaved, events.TunnelDeleted:
		return tunnelsChangedMsg{}
	case events.UpdateAvailable:
		return updateCheckMsg{info: m.App.Service.UpdateInfo()}
	}
	return nil
}

type downloadProgressMsg provider.DownloadProgress

func (m *Model) showMessage(msg string) {
	m.Message = msg
	m.messageUntil = time.Now().Add(messageTimeout)
}

func (m *Model) startTunnel(item TunnelItem) tea.Cmd {
	return func() tea.Msg {
		installing, err := m.App.Service.StartTunnel(item.Tunnel.ID)
		if err != nil {
			return statusUpdateMsg{
				tunnelID: item.Tunnel.ID,
				status:   core.TunnelStatus{ErrorMessage: err.Error(), State: core.TunnelStateError},
			}
		}
		if installing {
			return installStartedMsg{provider: item.Tunnel.Provider}
		}
		return nil
	}
}

func (m *Model) stopTunnel(item TunnelItem) tea.Cmd {
	return func() tea.Msg {
		_ = m.App.Service.StopTunnel(item.Tunnel.ID)
		return nil
	}
}
