package supervisor

import (
	"context"
	"fmt"
	"io"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sthbryan/ftm/internal/core"
	"github.com/sthbryan/ftm/internal/events"
	"github.com/sthbryan/ftm/internal/provider"
	"github.com/sthbryan/ftm/internal/provider/registry"
	"github.com/sthbryan/ftm/internal/supervisor/proxy"
)

type Manager struct {
	mu                 sync.RWMutex
	processes          map[string]*ManagedProcess
	providers          map[core.Provider]provider.Provider
	providerExpiration map[string]int
	visitorTracking    bool
	binaries           *binaryState

	// handlersMu guards everything below. These are set while the manager is
	// running and read from the process watchers, so they cannot share mu: a
	// blocking callback would stall the tunnel map.
	handlersMu          sync.RWMutex
	events              events.Publisher
	notificationHandler func(status core.TunnelStatus)
	visitorHandler      func(status core.TunnelStatus)
	expirationStart     func(tunnelID, name, provider string, startedAt time.Time)
	expirationStop      func(tunnelID string)
}

// SetEvents sets where the manager publishes tunnel states, stats, log lines
// and install progress. Publishing never waits for a reader.
func (m *Manager) SetEvents(p events.Publisher) {
	m.handlersMu.Lock()
	m.events = p
	m.handlersMu.Unlock()
}

func (m *Manager) publish(e events.Event) {
	m.handlersMu.RLock()
	p := m.events
	m.handlersMu.RUnlock()

	if p != nil {
		p.Publish(e)
	}
}

func (m *Manager) SetNotificationHandler(handler func(core.TunnelStatus)) {
	m.handlersMu.Lock()
	m.notificationHandler = handler
	m.handlersMu.Unlock()
}

func (m *Manager) callNotificationHandler(status core.TunnelStatus) {
	m.handlersMu.RLock()
	handler := m.notificationHandler
	m.handlersMu.RUnlock()

	if handler != nil {
		handler(status)
	}
}

func (m *Manager) SetVisitorHandler(handler func(core.TunnelStatus)) {
	m.handlersMu.Lock()
	m.visitorHandler = handler
	m.handlersMu.Unlock()
}

func (m *Manager) reportNewVisitor(tunnelID string) {
	m.handlersMu.RLock()
	handler := m.visitorHandler
	m.handlersMu.RUnlock()

	if handler == nil {
		return
	}

	if status, ok := m.GetStatus(tunnelID); ok && status.State == core.TunnelStateOnline {
		handler(status)
	}
}

const visitorWindow = 5 * time.Minute

func (m *Manager) SetVisitorTracking(enabled bool) {
	m.mu.Lock()
	m.visitorTracking = enabled
	m.mu.Unlock()
}

func (m *Manager) SetProviderExpiration(minutes map[string]int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.providerExpiration = make(map[string]int, len(minutes))
	for provider, mins := range minutes {
		m.providerExpiration[provider] = mins
	}
}

func (m *Manager) expiresAtLocked(provider core.Provider, startedAt time.Time) int64 {
	mins, ok := m.providerExpiration[string(provider)]
	if !ok || mins <= 0 {
		return 0
	}
	return startedAt.Add(time.Duration(mins) * time.Minute).UnixMilli()
}

func (m *Manager) callStatusUpdate(status core.TunnelStatus) {
	m.publish(events.TunnelStatus{Status: status})
}

func (m *Manager) SetExpirationCallbacks(start func(string, string, string, time.Time), stop func(string)) {
	m.handlersMu.Lock()
	m.expirationStart = start
	m.expirationStop = stop
	m.handlersMu.Unlock()
}

func (m *Manager) callExpirationStart(tunnelID, name, provider string, startedAt time.Time) {
	m.handlersMu.RLock()
	start := m.expirationStart
	m.handlersMu.RUnlock()

	if start != nil {
		start(tunnelID, name, provider, startedAt)
	}
}

func (m *Manager) callExpirationStop(tunnelID string) {
	m.handlersMu.RLock()
	stop := m.expirationStop
	m.handlersMu.RUnlock()

	if stop != nil {
		stop(tunnelID)
	}
}

func NewManager() *Manager {
	provider.MigrateLegacyBinDir()

	return &Manager{
		processes:       make(map[string]*ManagedProcess),
		visitorTracking: true,
		binaries:        newBinaryState(),
		providers:       registry.Default(),
	}
}

func (m *Manager) CheckInstallation(providerType core.Provider) (needsInstall bool, autoInstall bool) {
	p, ok := m.providers[providerType]
	if !ok {
		return false, false
	}

	installer, ok := p.(provider.AutoInstaller)
	if !ok {
		return false, false
	}

	if installer.IsInstalled() {
		return false, true
	}

	return true, true
}

// InstallProvider downloads a provider's binary and publishes its progress.
// The last event of the install has Done set, or Err when it failed.
func (m *Manager) InstallProvider(providerType core.Provider) error {
	p, ok := m.providers[providerType]
	if !ok {
		return fmt.Errorf("unknown provider: %s", providerType)
	}

	installer, ok := p.(provider.AutoInstaller)
	if !ok {
		return fmt.Errorf("provider %s does not support auto-install", providerType)
	}

	progress, finish := m.progressSink(string(providerType))
	err := installer.Install(progress)
	finish(err)
	return err
}

// progressSink returns a channel a provider can report download progress on
// and a finish to call once the provider returned. Progress is drained by its
// own goroutine, so a provider never waits on a reader; finish publishes the
// failure, if any, after every progress event.
func (m *Manager) progressSink(name string) (chan<- provider.DownloadProgress, func(error)) {
	ch := make(chan provider.DownloadProgress, 16)
	drained := make(chan struct{})

	go func() {
		defer close(drained)
		for p := range ch {
			m.publish(installProgress(name, p))
		}
	}()

	finish := func(err error) {
		close(ch)
		<-drained
		if err != nil {
			m.publish(events.InstallProgress{Provider: name, Err: err.Error()})
		}
	}
	return ch, finish
}

func installProgress(id string, p provider.DownloadProgress) events.InstallProgress {
	name := p.Name
	if name == "" {
		name = id
	}
	e := events.InstallProgress{
		Provider: name,
		Percent:  p.Percent,
		Current:  p.Current,
		Total:    p.Total,
		Done:     p.Done,
	}
	if p.Error != nil {
		e.Err = p.Error.Error()
	}
	return e
}

// Start runs a tunnel. Its states are published as events.TunnelStatus.
func (m *Manager) Start(tunnel core.TunnelConfig) error {
	m.mu.Lock()
	if existing, ok := m.processes[tunnel.ID]; ok && existing.Process != nil {
		m.mu.Unlock()
		return fmt.Errorf("tunnel %s is already running", tunnel.ID)
	}

	prov, ok := m.providers[tunnel.Provider]
	if !ok {
		m.mu.Unlock()
		return fmt.Errorf("unknown provider: %s", tunnel.Provider)
	}

	if installer, ok := prov.(provider.AutoInstaller); ok && !installer.IsInstalled() {
		m.mu.Unlock()
		return fmt.Errorf("installing")
	}

	if m.IsUpdating(tunnel.Provider) {
		m.mu.Unlock()
		return fmt.Errorf("%s is being updated, try again in a moment", tunnel.Provider)
	}

	visitorTracking := m.visitorTracking
	m.mu.Unlock()

	// Everything from here on can block on the filesystem, the network or a
	// process spawn, so it must not run while holding the manager lock. Holding
	// it froze every status read and every other tunnel start.
	logBuffer := NewLogBuffer()
	mp := &ManagedProcess{
		Config:         tunnel,
		Provider:       prov,
		LogBuffer:      logBuffer,
		Status:         tunnel.Status(),
		logSubscribers: make(map[chan string]struct{}),
	}

	exposed := tunnel
	if visitorTracking {
		opts := proxy.Options{
			Window:       visitorWindow,
			OnNewVisitor: func() { m.reportNewVisitor(tunnel.ID) },
		}

		if counter, err := proxy.Start(tunnel.LocalPort, opts); err != nil {
			log.Printf("visitor counting disabled for %s: %v", tunnel.Name, err)
		} else {
			mp.Proxy = counter
			exposed.LocalPort = counter.Port()
		}
	}
	logBuffer.OnNewLine = func(line string) {
		mp.publishLog(line)
		m.publish(events.LogLine{TunnelID: tunnel.ID, Line: line})
	}

	// A provider can print its public URL before Start returns, while the
	// process is still not registered. Keep the first URL seen so it is applied
	// once the process is in the map instead of being dropped.
	var earlyURL atomic.Value
	urlCapture := newURLCapture(prov, func(url string) {
		earlyURL.Store(url)
		m.updateURL(tunnel.ID, url)
	})
	writer := io.MultiWriter(logBuffer, urlCapture)

	ctx := context.Background()
	proc, err := prov.Start(ctx, exposed, writer)
	if err != nil {
		mp.closeProxy()
		return err
	}
	startedAt := time.Now()

	m.mu.Lock()
	// A concurrent Start for the same tunnel may have won the race while this
	// one was spawning.
	if existing, ok := m.processes[tunnel.ID]; ok && existing.Process != nil {
		m.mu.Unlock()
		mp.closeProxy()
		proc.Stop()
		return fmt.Errorf("tunnel %s is already running", tunnel.ID)
	}

	mp.Process = proc
	mp.Status.State = core.TunnelStateStarting
	mp.Status.ExpiresAt = m.expiresAtLocked(tunnel.Provider, startedAt)
	mp.Status.StartedAt = startedAt.UnixMilli()

	if url, ok := earlyURL.Load().(string); ok && url != "" {
		mp.PublicURL = url
		mp.Status.PublicURL = url
		mp.Status.State = core.TunnelStateOnline
	}

	m.processes[tunnel.ID] = mp
	status := mp.Status
	m.mu.Unlock()

	m.publishStatus(status)

	go m.startupTimeoutMonitor(tunnel.ID, proc)
	go m.watchExit(tunnel.ID, proc)
	m.callExpirationStart(tunnel.ID, tunnel.Name, string(tunnel.Provider), startedAt)

	return nil
}

// publishStatus publishes a state change. Callers hold no manager lock.
func (m *Manager) publishStatus(status core.TunnelStatus) {
	m.callStatusUpdate(status)
}

func (m *Manager) watchExit(tunnelID string, proc *provider.Process) {
	<-proc.Exited()

	m.mu.Lock()
	mp, ok := m.processes[tunnelID]
	if !ok || mp.Process != proc {
		m.mu.Unlock()
		return
	}

	mp.Process = nil
	mp.closeLogSubscribers()
	mp.closeProxy()

	mp.Status.PublicURL = ""
	if err := proc.Err(); err != nil {
		mp.Status.State = core.TunnelStateError
		mp.Status.ErrorMessage = failureReason(mp.LogBuffer.GetLines(), fmt.Sprintf("tunnel process exited unexpectedly: %v", err))
	} else {
		mp.Status.State = core.TunnelStateStopped
		mp.Status.ErrorMessage = ""
	}

	status := mp.Status
	m.mu.Unlock()

	m.publishStatus(status)
	m.callNotificationHandler(status)
	m.callExpirationStop(tunnelID)
}

const (
	connectingAfter = 5 * time.Second
	giveUpAfter     = 25 * time.Second
)

func (m *Manager) startupTimeoutMonitor(tunnelID string, proc *provider.Process) {
	select {
	case <-proc.Exited():
		return
	case <-time.After(connectingAfter):
	}

	m.mu.Lock()
	mp, ok := m.processes[tunnelID]
	if !ok || mp.Process != proc {
		m.mu.Unlock()
		return
	}
	changed := false
	if mp.Status.PublicURL == "" && mp.Status.State != core.TunnelStateOnline {
		mp.Status.State = core.TunnelStateConnecting
		changed = true
	}
	status := mp.Status
	m.mu.Unlock()

	if changed {
		m.publishStatus(status)
	}

	select {
	case <-proc.Exited():
		return
	case <-time.After(giveUpAfter):
	}

	m.mu.Lock()
	mp, ok = m.processes[tunnelID]
	if !ok || mp.Process != proc {
		m.mu.Unlock()
		return
	}
	if mp.Status.State != core.TunnelStateConnecting && mp.Status.State != core.TunnelStateStarting {
		m.mu.Unlock()
		return
	}

	mp.Process = nil
	mp.closeLogSubscribers()
	mp.closeProxy()

	mp.Status.State = core.TunnelStateTimeout
	mp.Status.ErrorMessage = "Connection timed out after 30 seconds"
	if reported := reportedFailure(mp.LogBuffer.GetLines()); reported != "" {
		mp.Status.ErrorMessage = reported
	}
	status = mp.Status
	m.mu.Unlock()

	m.publishStatus(status)
	m.callNotificationHandler(status)
	m.callExpirationStop(tunnelID)

	proc.Stop()
}

func (m *Manager) Stop(tunnelID string) error {
	m.mu.Lock()

	mp, ok := m.processes[tunnelID]
	if !ok {
		m.mu.Unlock()
		return fmt.Errorf("tunnel %s is not running", tunnelID)
	}

	delete(m.processes, tunnelID)
	mp.closeLogSubscribers()
	mp.closeProxy()

	proc := mp.Process
	mp.Status.State = core.TunnelStateStopping
	mp.Status.PublicURL = ""
	stopping := mp.Status
	m.mu.Unlock()

	m.publishStatus(stopping)
	m.callNotificationHandler(stopping)

	if proc != nil {
		proc.Stop()
	}

	m.mu.Lock()
	mp.Status.State = core.TunnelStateStopped
	mp.Status.ErrorMessage = ""
	stopped := mp.Status
	m.mu.Unlock()

	m.publishStatus(stopped)

	m.callExpirationStop(tunnelID)

	return nil
}

func (m *Manager) GetStatus(tunnelID string) (core.TunnelStatus, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	mp, ok := m.processes[tunnelID]
	if !ok {
		return core.TunnelStatus{}, false
	}

	status := mp.Status
	if mp.Proxy != nil {
		stats := mp.Proxy.Stats()
		status.ActiveSessions = stats.ActiveSessions
		status.Visitors = stats.Visitors
	}

	return status, true
}

func (m *Manager) updateURL(tunnelID, url string) {
	m.mu.Lock()

	mp, ok := m.processes[tunnelID]
	if !ok {
		m.mu.Unlock()
		return
	}

	if mp.Status.PublicURL != "" {
		m.mu.Unlock()
		return
	}

	mp.PublicURL = url
	mp.Status.PublicURL = url
	mp.Status.State = core.TunnelStateOnline

	status := mp.Status
	m.mu.Unlock()

	m.publishStatus(status)
	m.callNotificationHandler(status)
}

func (m *Manager) GetLogs(tunnelID string) []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	mp, ok := m.processes[tunnelID]
	if !ok {
		return []string{}
	}

	return mp.LogBuffer.GetLines()
}

func (m *Manager) SubscribeLogs(tunnelID string) (<-chan string, func()) {
	m.mu.RLock()
	mp, ok := m.processes[tunnelID]
	m.mu.RUnlock()
	if !ok {
		return nil, func() {}
	}

	ch, cancel := mp.addLogSubscriber()
	return ch, cancel
}

func (m *Manager) IsRunning(tunnelID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	mp, ok := m.processes[tunnelID]
	return ok && (mp.Status.State == core.TunnelStateOnline || mp.Status.State == core.TunnelStateConnecting || mp.Status.State == core.TunnelStateStarting)
}

func (m *Manager) StopAll() {
	m.mu.Lock()
	procs := make([]*provider.Process, 0, len(m.processes))
	for _, mp := range m.processes {
		mp.closeLogSubscribers()
		if mp.Process != nil {
			procs = append(procs, mp.Process)
		}
	}
	m.processes = make(map[string]*ManagedProcess)
	m.mu.Unlock()

	var wg sync.WaitGroup
	for _, proc := range procs {
		wg.Add(1)
		go func(p *provider.Process) {
			defer wg.Done()
			p.Stop()
		}(proc)
	}
	wg.Wait()
}

// PublishStats publishes the visitor numbers of every online tunnel whose
// numbers changed since the last call.
func (m *Manager) PublishStats() {
	m.mu.Lock()
	var changed []events.TunnelStats
	for id, mp := range m.processes {
		if mp.Proxy == nil || mp.Status.State != core.TunnelStateOnline {
			continue
		}
		stats := mp.Proxy.Stats()
		if stats.Visitors == mp.lastStats.Visitors && stats.ActiveSessions == mp.lastStats.ActiveSessions {
			continue
		}
		mp.lastStats = stats
		changed = append(changed, events.TunnelStats{
			ID:             id,
			Visitors:       stats.Visitors,
			ActiveSessions: stats.ActiveSessions,
		})
	}
	m.mu.Unlock()

	for _, e := range changed {
		m.publish(e)
	}
}

// RunStats calls PublishStats every interval until ctx is done.
func (m *Manager) RunStats(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.PublishStats()
		}
	}
}
