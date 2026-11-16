package tui

import (
	"context"
	"slices"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/sthbryan/ftm/internal/core"
	"github.com/sthbryan/ftm/internal/provider"
)

type (
	// installStartedMsg says a start is waiting for its provider download.
	installStartedMsg struct {
		provider core.Provider
	}

	// busMsg wraps one message from the backend's events; handling it
	// re-arms the wait for the next one.
	busMsg struct {
		msg tea.Msg
	}

	// messageExpiredMsg clears the status message it was scheduled for.
	messageExpiredMsg struct {
		seq int
	}

	// logsLoadedMsg carries the recent output of the tunnel being watched.
	logsLoadedMsg struct {
		tunnelID string
		lines    []string
	}
)

const (
	messageTimeout = 3 * time.Second
	// requestTimeout bounds one call to the backend, which is a request to
	// the running ftm when attached.
	requestTimeout = 5 * time.Second
	// maxLogLines is how much of the watched tunnel's output the model keeps.
	maxLogLines = 1000
)

func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		tea.RequestBackgroundColor,
		m.waitForEvent(),
		m.checkUpdateCmd(),
	)
}

// call returns a context for one backend call.
func (m *Model) call() (context.Context, context.CancelFunc) {
	parent := m.ctx
	if parent == nil {
		parent = context.Background()
	}
	return context.WithTimeout(parent, requestTimeout)
}

// waitForEvent delivers the next message from the backend.
func (m *Model) waitForEvent() tea.Cmd {
	if m.events == nil {
		return nil
	}
	ch, ctx := m.events, m.ctx
	return func() tea.Msg {
		select {
		case msg := <-ch:
			if msg == nil {
				return nil
			}
			return busMsg{msg: msg}
		case <-ctx.Done():
			return nil
		}
	}
}

type downloadProgressMsg provider.DownloadProgress

// showMessage puts msg in the status line for messageTimeout. Update
// schedules the timer that clears it.
func (m *Model) showMessage(msg string) {
	m.Message = msg
	m.messageSeq++
	m.messageUntil = time.Now().Add(messageTimeout)
}

// expireMessageCmd schedules clearing the current message once, the first
// time Update sees it.
func (m *Model) expireMessageCmd() tea.Cmd {
	if m.Message == "" || m.messageTimer == m.messageSeq {
		return nil
	}
	m.messageTimer = m.messageSeq
	seq := m.messageSeq
	return tea.Tick(time.Until(m.messageUntil), func(time.Time) tea.Msg {
		return messageExpiredMsg{seq: seq}
	})
}

func (m *Model) startTunnel(item TunnelItem) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := m.call()
		defer cancel()
		installing, err := m.App.StartTunnel(ctx, item.Tunnel.ID)
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
		ctx, cancel := m.call()
		defer cancel()
		_ = m.App.StopTunnel(ctx, item.Tunnel.ID)
		return nil
	}
}

// watchedTunnel is the tunnel whose output is on screen: the one in the
// logs view, or the selected one in the list.
func (m *Model) watchedTunnel() string {
	if m.State == viewLogs {
		return m.SelectedTunnel
	}
	if item, ok := m.selectedItem(); ok {
		return item.Tunnel.ID
	}
	return ""
}

// followSelection starts watching the output of the tunnel now on screen
// when it changed.
func (m *Model) followSelection() tea.Cmd {
	id := m.watchedTunnel()
	if id == m.logsID {
		return nil
	}
	m.logsID = id
	m.logs = nil
	m.App.WatchLogs(id)
	if id == "" {
		return nil
	}
	return func() tea.Msg {
		ctx, cancel := m.call()
		defer cancel()
		lines, _ := m.App.Logs(ctx, id)
		return logsLoadedMsg{tunnelID: id, lines: lines}
	}
}

func (m *Model) appendLog(line string) {
	m.logs = append(m.logs, line)
	if extra := len(m.logs) - maxLogLines; extra > 0 {
		m.logs = append([]string(nil), m.logs[extra:]...)
	}
}

// mergeLogs joins the recent lines read from the backend with the lines that
// arrived while the read was in flight, dropping the ones both hold.
func mergeLogs(recent, arrived []string) []string {
	overlap := 0
	for k := min(len(recent), len(arrived)); k > 0; k-- {
		if slices.Equal(recent[len(recent)-k:], arrived[:k]) {
			overlap = k
			break
		}
	}
	out := make([]string, 0, len(recent)+len(arrived)-overlap)
	out = append(out, recent...)
	out = append(out, arrived[overlap:]...)
	if extra := len(out) - maxLogLines; extra > 0 {
		out = out[extra:]
	}
	return out
}
