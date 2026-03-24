package process

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/deadbryam/ftm/internal/config"
	"github.com/deadbryam/ftm/internal/providers"
	"github.com/deadbryam/ftm/internal/providers/cloudflared"
	"github.com/deadbryam/ftm/internal/providers/pinggy"
	"github.com/deadbryam/ftm/internal/providers/playitgg"
	"github.com/deadbryam/ftm/internal/providers/ssh"
	"github.com/deadbryam/ftm/internal/providers/tunnelmole"
)

type Manager struct {
	mu        sync.RWMutex
	processes map[string]*ManagedProcess
	providers map[config.Provider]providers.Provider

	DownloadProgress chan providers.DownloadProgress
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

type ManagedProcess struct {
	Config    config.TunnelConfig
	Provider  providers.Provider
	Process   *providers.Process
	LogBuffer *LogBuffer
	Status    config.TunnelStatus
	PublicURL string
	OnUpdate  func(config.TunnelStatus)
}

type LogBuffer struct {
	mu     sync.RWMutex
	lines  []string
	maxLen int
}

func NewLogBuffer() *LogBuffer {
	return &LogBuffer{
		lines:  make([]string, 0, 100),
		maxLen: 500,
	}
}

func (lb *LogBuffer) Write(p []byte) (n int, err error) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	lines := strings.Split(string(p), "\n")
	for _, line := range lines {
		if line = strings.TrimSpace(line); line != "" {
			lb.lines = append(lb.lines, line)
		}
	}

	if len(lb.lines) > lb.maxLen {
		lb.lines = lb.lines[len(lb.lines)-lb.maxLen:]
	}

	return len(p), nil
}

func (lb *LogBuffer) GetLines() []string {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	result := make([]string, len(lb.lines))
	copy(result, lb.lines)
	return result
}

func NewManager() *Manager {
	return &Manager{
		processes: make(map[string]*ManagedProcess),
		providers: map[config.Provider]providers.Provider{
			config.ProviderPlayitgg:     playitgg.New(),
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

	urlCapture := &urlCaptureWriter{
		provider: provider,
		onURL:    func(url string) { m.updateURL(tunnel.ID, url) },
	}
	writer := io.MultiWriter(logBuffer, urlCapture)

	ctx := context.Background()
	proc, err := provider.Start(ctx, tunnel, writer)
	if err != nil {
		return err
	}

	mp := &ManagedProcess{
		Config:    tunnel,
		Provider:  provider,
		Process:   proc,
		LogBuffer: logBuffer,
		OnUpdate:  onUpdate,
		Status:    tunnel.Status(),
	}
	mp.Status.State = config.TunnelStateStarting

	m.processes[tunnel.ID] = mp

	if onUpdate != nil {
		onUpdate(mp.Status)
	}

	go func() {
		time.Sleep(5 * time.Second)
		m.mu.Lock()
		if mp, ok := m.processes[tunnel.ID]; ok {
			if mp.Status.PublicURL == "" && mp.Status.State != config.TunnelStateOnline {
				mp.Status.State = config.TunnelStateConnecting
				if mp.OnUpdate != nil {
					mp.OnUpdate(mp.Status)
				}
			}
		}
		m.mu.Unlock()
	}()

	go func() {
		time.Sleep(30 * time.Second)
		m.mu.Lock()
		if mp, ok := m.processes[tunnel.ID]; ok {
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
			}
		}
		m.mu.Unlock()
	}()

	return nil
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

	delete(m.processes, tunnelID)

	if mp.OnUpdate != nil {
		mp.OnUpdate(mp.Status)
	}

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

func (m *Manager) IsRunning(tunnelID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	mp, ok := m.processes[tunnelID]
	return ok && (mp.Status.State == config.TunnelStateOnline || mp.Status.State == config.TunnelStateConnecting || mp.Status.State == config.TunnelStateStarting)
}

type urlCaptureWriter struct {
	provider providers.Provider
	onURL    func(string)
	buf      bytes.Buffer
}

func (w *urlCaptureWriter) Write(p []byte) (n int, err error) {
	w.buf.Write(p)

	lines := strings.Split(w.buf.String(), "\n")
	w.buf.Reset()

	if len(lines) > 0 && !strings.HasSuffix(string(p), "\n") {
		w.buf.WriteString(lines[len(lines)-1])
		lines = lines[:len(lines)-1]
	}

	for _, line := range lines {
		if url := w.provider.ParseURL(line); url != "" {
			w.onURL(url)
		}
	}

	return len(p), nil
}
