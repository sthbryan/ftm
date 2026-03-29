package process

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/sthbryan/ftm/internal/config"
	"github.com/sthbryan/ftm/internal/providers"
	"github.com/sthbryan/ftm/internal/providers/cloudflared"
	"github.com/sthbryan/ftm/internal/providers/pinggy"
	"github.com/sthbryan/ftm/internal/providers/ssh"
	"github.com/sthbryan/ftm/internal/providers/tunnelmole"
)

type Manager struct {
	mu                  sync.RWMutex
	processes           map[string]*ManagedProcess
	providers           map[config.Provider]providers.Provider
	DownloadProgress    chan providers.DownloadProgress
	StatusChannel       chan config.TunnelStatus
	NotificationHandler func(status config.TunnelStatus)
	ExpirationCallbacks struct {
		OnStart func(tunnelID, name, provider string, startedAt time.Time)
		OnStop  func(tunnelID string)
	}
}

func (m *Manager) SetProgressChannel(ch chan providers.DownloadProgress) {
	m.DownloadProgress = ch
	for _, p := range m.providers {
		if installer, ok := p.(interface {
			SetProgressChannel(chan providers.DownloadProgress)
		}); ok {
			installer.SetProgressChannel(ch)
		}
	}
}

func (m *Manager) SetNotificationHandler(handler func(config.TunnelStatus)) {
	m.NotificationHandler = handler
}

func (m *Manager) callNotificationHandler(status config.TunnelStatus) {
	if m.NotificationHandler != nil {
		m.NotificationHandler(status)
	}
}

func (m *Manager) SetStatusChannel(ch chan config.TunnelStatus) {
	m.StatusChannel = ch
}

func (m *Manager) callStatusUpdate(status config.TunnelStatus) {
	if m.StatusChannel != nil {
		select {
		case m.StatusChannel <- status:
		default:
		}
	}
}

func (m *Manager) SetExpirationCallbacks(start func(string, string, string, time.Time), stop func(string)) {
	m.ExpirationCallbacks.OnStart = start
	m.ExpirationCallbacks.OnStop = stop
}

func (m *Manager) callExpirationStart(tunnelID, name, provider string, startedAt time.Time) {
	if m.ExpirationCallbacks.OnStart != nil {
		m.ExpirationCallbacks.OnStart(tunnelID, name, provider, startedAt)
	}
}

func (m *Manager) callExpirationStop(tunnelID string) {
	if m.ExpirationCallbacks.OnStop != nil {
		m.ExpirationCallbacks.OnStop(tunnelID)
	}
}

func NewManager() *Manager {
	return &Manager{
		processes: make(map[string]*ManagedProcess),
		providers: map[config.Provider]providers.Provider{
			config.ProviderCloudflared:  cloudflared.New(),
			config.ProviderTunnelmole:   tunnelmole.New(),
			config.ProviderLocalhostRun: ssh.NewLocalhostRun(),
			config.ProviderServeo:       ssh.NewServeo(),
			config.ProviderPinggy:       pinggy.New(),
		},
	}
}

func (m *Manager) GetProvider(p config.Provider) (providers.Provider, bool) {
	provider, ok := m.providers[p]
	return provider, ok
}

func (m *Manager) ListProviders() []providers.Provider {
	result := make([]providers.Provider, 0, len(m.providers))
	for _, p := range m.providers {
		result = append(result, p)
	}
	return result
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

	return installer.Install(m.DownloadProgress)
}

func (m *Manager) Start(tunnel config.TunnelConfig, onUpdate func(config.TunnelStatus)) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if existing, ok := m.processes[tunnel.ID]; ok && existing.Process != nil {
		return fmt.Errorf("tunnel %s is already running", tunnel.ID)
	}

	provider, ok := m.providers[tunnel.Provider]
	if !ok {
		return fmt.Errorf("unknown provider: %s", tunnel.Provider)
	}

	if installer, ok := provider.(providers.AutoInstaller); ok && !installer.IsInstalled() {
		return fmt.Errorf("installing")
	}

	logBuffer := NewLogBuffer()
	mp := &ManagedProcess{
		Config:         tunnel,
		Provider:       provider,
		LogBuffer:      logBuffer,
		OnUpdate:       onUpdate,
		Status:         tunnel.Status(),
		logSubscribers: make(map[chan string]struct{}),
	}
	logBuffer.OnNewLine = func(line string) {
		mp.publishLog(line)
	}

	urlCapture := newURLCapture(provider, func(url string) { m.updateURL(tunnel.ID, url) })
	writer := io.MultiWriter(logBuffer, urlCapture)

	ctx := context.Background()
	proc, err := provider.Start(ctx, tunnel, writer)
	if err != nil {
		return err
	}
	mp.Process = proc
	mp.Status.State = config.TunnelStateStarting

	m.processes[tunnel.ID] = mp

	if onUpdate != nil {
		onUpdate(mp.Status)
	}

	go m.startupTimeoutMonitor(tunnel.ID)
	m.callExpirationStart(tunnel.ID, tunnel.Name, string(tunnel.Provider), time.Now())

	return nil
}

func (m *Manager) startupTimeoutMonitor(tunnelID string) {
	time.Sleep(5 * time.Second)
	m.mu.Lock()
	if mp, ok := m.processes[tunnelID]; ok {
		if mp.Status.PublicURL == "" && mp.Status.State != config.TunnelStateOnline {
			mp.Status.State = config.TunnelStateConnecting
			if mp.OnUpdate != nil {
				mp.OnUpdate(mp.Status)
			}
			m.callStatusUpdate(mp.Status)
		}
	}
	m.mu.Unlock()

	time.Sleep(25 * time.Second)
	m.mu.Lock()
	defer m.mu.Unlock()

	if mp, ok := m.processes[tunnelID]; ok {
		if mp.Status.State == config.TunnelStateConnecting || mp.Status.State == config.TunnelStateStarting {
			mp.Status.State = config.TunnelStateTimeout
			mp.Status.ErrorMessage = "Connection timed out after 30 seconds"
			if mp.Process != nil && mp.Process.Cancel != nil {
				mp.Process.Cancel()
				mp.Process = nil
			}
			if mp.OnUpdate != nil {
				mp.OnUpdate(mp.Status)
			}

			m.callNotificationHandler(mp.Status)
			m.callStatusUpdate(mp.Status)
		}
	}
}

func (m *Manager) Stop(tunnelID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	mp, ok := m.processes[tunnelID]
	if !ok {
		return fmt.Errorf("tunnel %s is not running", tunnelID)
	}

	if mp.Process != nil && mp.Process.Cancel != nil {
		mp.Process.Cancel()
	}

	mp.Status.State = config.TunnelStateStopping
	mp.Status.PublicURL = ""

	mp.closeLogSubscribers()

	if mp.OnUpdate != nil {
		mp.OnUpdate(mp.Status)
	}

	m.callNotificationHandler(mp.Status)
	m.callStatusUpdate(mp.Status)
	m.callExpirationStop(tunnelID)
	delete(m.processes, tunnelID)

	return nil
}

func (m *Manager) Get(tunnelID string) (*ManagedProcess, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	mp, ok := m.processes[tunnelID]
	return mp, ok
}

func (m *Manager) GetStatus(tunnelID string) (config.TunnelStatus, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	mp, ok := m.processes[tunnelID]
	if !ok {
		return config.TunnelStatus{}, false
	}
	return mp.Status, true
}

func (m *Manager) updateURL(tunnelID, url string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	mp, ok := m.processes[tunnelID]
	if !ok {
		return
	}

	mp.PublicURL = url
	mp.Status.PublicURL = url
	mp.Status.State = config.TunnelStateOnline

	if mp.OnUpdate != nil {
		mp.OnUpdate(mp.Status)
	}

	m.callNotificationHandler(mp.Status)
	m.callStatusUpdate(mp.Status)
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
	defer m.mu.Unlock()

	for tunnelID := range m.processes {
		if mp, ok := m.processes[tunnelID]; ok && mp.Process != nil && mp.Process.Cancel != nil {
			mp.Process.Cancel()
		}
	}
	m.processes = make(map[string]*ManagedProcess)
}
