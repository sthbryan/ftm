package process

import (
	"context"
	"fmt"
	"io"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sthbryan/ftm/internal/config"
	"github.com/sthbryan/ftm/internal/providers"
	"github.com/sthbryan/ftm/internal/providers/bore"
	"github.com/sthbryan/ftm/internal/providers/cloudflared"
	"github.com/sthbryan/ftm/internal/providers/pinggy"
	"github.com/sthbryan/ftm/internal/providers/ssh"
	"github.com/sthbryan/ftm/internal/providers/tunnelmole"
	"github.com/sthbryan/ftm/internal/proxy"
)

type Manager struct {
	mu                 sync.RWMutex
	processes          map[string]*ManagedProcess
	providers          map[config.Provider]providers.Provider
	providerExpiration map[string]int
	visitorTracking    bool
	binaries           *binaryState

	// handlersMu guards everything below. These are set while the manager is
	// running and read from the process watchers, so they cannot share mu: a
	// blocking callback would stall the tunnel map.
	handlersMu          sync.RWMutex
	downloadProgress    chan providers.DownloadProgress
	statusChannel       chan config.TunnelStatus
	notificationHandler func(status config.TunnelStatus)
	visitorHandler      func(status config.TunnelStatus)
	expirationStart     func(tunnelID, name, provider string, startedAt time.Time)
	expirationStop      func(tunnelID string)
}

func (m *Manager) SetProgressChannel(ch chan providers.DownloadProgress) {
	m.handlersMu.Lock()
	m.downloadProgress = ch
	m.handlersMu.Unlock()

	for _, p := range m.providers {
		if installer, ok := p.(interface {
			SetProgressChannel(chan providers.DownloadProgress)
		}); ok {
			installer.SetProgressChannel(ch)
		}
	}
}

// ProgressEvents returns the stream of install progress updates.
func (m *Manager) ProgressEvents() chan providers.DownloadProgress {
	m.handlersMu.RLock()
	defer m.handlersMu.RUnlock()
	return m.downloadProgress
}

func (m *Manager) SetNotificationHandler(handler func(config.TunnelStatus)) {
	m.handlersMu.Lock()
	m.notificationHandler = handler
	m.handlersMu.Unlock()
}

func (m *Manager) callNotificationHandler(status config.TunnelStatus) {
	m.handlersMu.RLock()
	handler := m.notificationHandler
	m.handlersMu.RUnlock()

	if handler != nil {
		handler(status)
	}
}

func (m *Manager) SetVisitorHandler(handler func(config.TunnelStatus)) {
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

	if status, ok := m.GetStatus(tunnelID); ok && status.State == config.TunnelStateOnline {
		handler(status)
	}
}

func (m *Manager) SetStatusChannel(ch chan config.TunnelStatus) {
	m.handlersMu.Lock()
	m.statusChannel = ch
	m.handlersMu.Unlock()
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

func (m *Manager) expiresAtLocked(provider config.Provider, startedAt time.Time) int64 {
	mins, ok := m.providerExpiration[string(provider)]
	if !ok || mins <= 0 {
		return 0
	}
	return startedAt.Add(time.Duration(mins) * time.Minute).UnixMilli()
}

func (m *Manager) callStatusUpdate(status config.TunnelStatus) {
	m.handlersMu.RLock()
	ch := m.statusChannel
	m.handlersMu.RUnlock()

	if ch == nil {
		return
	}

	select {
	case ch <- status:
	default:
		log.Printf("process: status channel full, dropped %q update for tunnel %s", status.State, status.ID)
	}
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
	providers.MigrateLegacyBinDir()

	return &Manager{
		processes:       make(map[string]*ManagedProcess),
		visitorTracking: true,
		binaries:        newBinaryState(),
		providers: map[config.Provider]providers.Provider{
			config.ProviderCloudflared:  cloudflared.New(),
			config.ProviderTunnelmole:   tunnelmole.New(),
			config.ProviderLocalhostRun: ssh.NewLocalhostRun(),
			config.ProviderServeo:       ssh.NewServeo(),
			config.ProviderPinggy:       pinggy.New(),
			config.ProviderBore:         bore.New(),
		},
	}
}

func (m *Manager) CheckInstallation(providerType config.Provider) (needsInstall bool, autoInstall bool) {
	provider, ok := m.providers[providerType]
	if !ok {
		return false, false
	}

	installer, ok := provider.(providers.AutoInstaller)
	if !ok {
		return false, false
	}

	if installer.IsInstalled() {
		return false, true
	}

	return true, true
}

func (m *Manager) InstallProvider(providerType config.Provider) error {
	provider, ok := m.providers[providerType]
	if !ok {
		return fmt.Errorf("unknown provider: %s", providerType)
	}

	installer, ok := provider.(providers.AutoInstaller)
	if !ok {
		return fmt.Errorf("provider %s does not support auto-install", providerType)
	}

	return installer.Install(m.ProgressEvents())
}

func (m *Manager) Start(tunnel config.TunnelConfig, onUpdate func(config.TunnelStatus)) error {
	m.mu.Lock()
	if existing, ok := m.processes[tunnel.ID]; ok && existing.Process != nil {
		m.mu.Unlock()
		return fmt.Errorf("tunnel %s is already running", tunnel.ID)
	}

	provider, ok := m.providers[tunnel.Provider]
	if !ok {
		m.mu.Unlock()
		return fmt.Errorf("unknown provider: %s", tunnel.Provider)
	}

	if installer, ok := provider.(providers.AutoInstaller); ok && !installer.IsInstalled() {
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
		Provider:       provider,
		LogBuffer:      logBuffer,
		OnUpdate:       onUpdate,
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
	}

	// A provider can print its public URL before Start returns, while the
	// process is still not registered. Keep the first URL seen so it is applied
	// once the process is in the map instead of being dropped.
	var earlyURL atomic.Value
	urlCapture := newURLCapture(provider, func(url string) {
		earlyURL.Store(url)
		m.updateURL(tunnel.ID, url)
	})
	writer := io.MultiWriter(logBuffer, urlCapture)

	ctx := context.Background()
	proc, err := provider.Start(ctx, exposed, writer)
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
	mp.Status.State = config.TunnelStateStarting
	mp.Status.ExpiresAt = m.expiresAtLocked(tunnel.Provider, startedAt)
	mp.Status.StartedAt = startedAt.UnixMilli()

	if url, ok := earlyURL.Load().(string); ok && url != "" {
		mp.PublicURL = url
		mp.Status.PublicURL = url
		mp.Status.State = config.TunnelStateOnline
	}

	m.processes[tunnel.ID] = mp
	status := mp.Status
	m.mu.Unlock()

	m.publishStatus(mp, status)

	go m.startupTimeoutMonitor(tunnel.ID, proc)
	go m.watchExit(tunnel.ID, proc)
	m.callExpirationStart(tunnel.ID, tunnel.Name, string(tunnel.Provider), startedAt)

	return nil
}

// publishStatus notifies the owner and the status channel outside the manager
// lock, so a slow callback can never freeze the tunnel map.
func (m *Manager) publishStatus(mp *ManagedProcess, status config.TunnelStatus) {
	if mp.OnUpdate != nil {
		mp.OnUpdate(status)
	}
	m.callStatusUpdate(status)
}

func (m *Manager) watchExit(tunnelID string, proc *providers.Process) {
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
		mp.Status.State = config.TunnelStateError
		mp.Status.ErrorMessage = failureReason(mp.LogBuffer.GetLines(), fmt.Sprintf("tunnel process exited unexpectedly: %v", err))
	} else {
		mp.Status.State = config.TunnelStateStopped
		mp.Status.ErrorMessage = ""
	}

	status := mp.Status
	m.mu.Unlock()

	m.publishStatus(mp, status)
	m.callNotificationHandler(status)
	m.callExpirationStop(tunnelID)
}

const (
	connectingAfter = 5 * time.Second
	giveUpAfter     = 25 * time.Second
)

func (m *Manager) startupTimeoutMonitor(tunnelID string, proc *providers.Process) {
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
	if mp.Status.PublicURL == "" && mp.Status.State != config.TunnelStateOnline {
		mp.Status.State = config.TunnelStateConnecting
		changed = true
	}
	status := mp.Status
	m.mu.Unlock()

	if changed {
		m.publishStatus(mp, status)
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
	if mp.Status.State != config.TunnelStateConnecting && mp.Status.State != config.TunnelStateStarting {
		m.mu.Unlock()
		return
	}

	mp.Process = nil
	mp.closeLogSubscribers()
	mp.closeProxy()

	mp.Status.State = config.TunnelStateTimeout
	mp.Status.ErrorMessage = "Connection timed out after 30 seconds"
	if reported := reportedFailure(mp.LogBuffer.GetLines()); reported != "" {
		mp.Status.ErrorMessage = reported
	}
	status = mp.Status
	m.mu.Unlock()

	m.publishStatus(mp, status)
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
	mp.Status.State = config.TunnelStateStopping
	mp.Status.PublicURL = ""
	stopping := mp.Status
	m.mu.Unlock()

	m.publishStatus(mp, stopping)
	m.callNotificationHandler(stopping)

	if proc != nil {
		proc.Stop()
	}

	m.mu.Lock()
	mp.Status.State = config.TunnelStateStopped
	mp.Status.ErrorMessage = ""
	stopped := mp.Status
	m.mu.Unlock()

	m.publishStatus(mp, stopped)

	m.callExpirationStop(tunnelID)

	return nil
}

func (m *Manager) GetStatus(tunnelID string) (config.TunnelStatus, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	mp, ok := m.processes[tunnelID]
	if !ok {
		return config.TunnelStatus{}, false
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
	mp.Status.State = config.TunnelStateOnline

	status := mp.Status
	m.mu.Unlock()

	m.publishStatus(mp, status)
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
	return ok && (mp.Status.State == config.TunnelStateOnline || mp.Status.State == config.TunnelStateConnecting || mp.Status.State == config.TunnelStateStarting)
}

func (m *Manager) StopAll() {
	m.mu.Lock()
	procs := make([]*providers.Process, 0, len(m.processes))
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
		go func(p *providers.Process) {
			defer wg.Done()
			p.Stop()
		}(proc)
	}
	wg.Wait()
}
