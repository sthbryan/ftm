package service

import (
	"net"
	"slices"
	"strconv"
	"sync"
	"time"

	"github.com/sthbryan/ftm/internal/core"
)

// statusReader is implemented by a supervisor that knows the live status of
// the tunnels it runs.
type statusReader interface {
	GetStatus(tunnelID string) (core.TunnelStatus, bool)
}

// logReader is implemented by a supervisor that keeps the recent output of
// each tunnel it runs.
type logReader interface {
	GetLogs(tunnelID string) []string
}

// TunnelView is a saved connection together with its live status.
type TunnelView struct {
	Tunnel core.TunnelConfig
	// Status is the live status. Running is false and Status is zero when the
	// supervisor has never run the tunnel.
	Status  core.TunnelStatus
	Running bool
	// NeedsInstall is set for a stopped tunnel whose provider has to be
	// downloaded before it can start.
	NeedsInstall bool
}

// View merges a saved connection with what the supervisor knows about it.
func (s *Service) View(t core.TunnelConfig) TunnelView {
	v := TunnelView{Tunnel: t}
	if r, ok := s.sup.(statusReader); ok {
		v.Status, v.Running = r.GetStatus(t.ID)
	}
	if !v.Running || v.Status.State == core.TunnelStateStopped {
		needsInstall, canInstall := s.sup.CheckInstallation(t.Provider)
		v.NeedsInstall = needsInstall && canInstall
	}
	return v
}

// TunnelViews returns every saved connection with its live status, in the
// saved order.
func (s *Service) TunnelViews() []TunnelView {
	tunnels := s.store.Tunnels()
	views := make([]TunnelView, 0, len(tunnels))
	for _, t := range tunnels {
		views = append(views, s.View(t))
	}
	return views
}

// TunnelView returns one saved connection with its live status.
func (s *Service) TunnelView(id string) (TunnelView, bool) {
	t, ok := s.store.Tunnel(id)
	if !ok {
		return TunnelView{}, false
	}
	return s.View(t), true
}

// Status returns the live status of a tunnel the supervisor has run since ftm
// started.
func (s *Service) Status(tunnelID string) (core.TunnelStatus, bool) {
	if r, ok := s.sup.(statusReader); ok {
		return r.GetStatus(tunnelID)
	}
	return core.TunnelStatus{}, false
}

// Running reports whether a tunnel is starting, connecting or online.
func (s *Service) Running(tunnelID string) bool {
	st, ok := s.Status(tunnelID)
	if !ok {
		return false
	}
	switch st.State {
	case core.TunnelStateStarting, core.TunnelStateConnecting, core.TunnelStateOnline:
		return true
	}
	return false
}

// Logs returns the recent output of a tunnel, oldest first. It is empty for a
// tunnel that has not run since ftm started.
func (s *Service) Logs(tunnelID string) []string {
	if r, ok := s.sup.(logReader); ok {
		if lines := r.GetLogs(tunnelID); lines != nil {
			return lines
		}
	}
	return []string{}
}

// foundryPorts are the ports Foundry VTT usually listens on.
var foundryPorts = []int{30000, 30001, 30002, 30003, 30004, 30005, 30006, 30007, 30008, 30009}

// DetectFoundryPorts reports which of the usual Foundry ports, plus any extra
// ports asked for, are taken, and the one to suggest for a new connection.
// scanned lists every port it checked.
func (s *Service) DetectFoundryPorts(extra ...int) (inUse, scanned []int, suggested int) {
	inUse = []int{}
	scanned = append([]int{}, foundryPorts...)
	for _, port := range extra {
		if port > 0 && port <= 65535 && !slices.Contains(scanned, port) {
			scanned = append(scanned, port)
		}
	}
	busy := make([]bool, len(scanned))
	var wg sync.WaitGroup
	for i, port := range scanned {
		wg.Add(1)
		go func(i, port int) {
			defer wg.Done()
			busy[i] = listening(port)
		}(i, port)
	}
	wg.Wait()
	for i, port := range scanned {
		if busy[i] {
			inUse = append(inUse, port)
		}
	}

	suggested = foundryPorts[0]
	for _, port := range inUse {
		if slices.Contains(foundryPorts, port) {
			suggested = port
			break
		}
	}
	return inUse, scanned, suggested
}

// listening reports whether something accepts connections on the port on this
// computer. Binding the port instead would miss a server that listens only on
// 127.0.0.1: macOS still lets the wildcard address bind next to it.
func listening(port int) bool {
	for _, host := range []string{"127.0.0.1", "::1"} {
		conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, strconv.Itoa(port)), 200*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			return true
		}
	}
	return false
}
