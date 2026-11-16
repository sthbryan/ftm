package notify

import (
	"fmt"
	"sync"
	"time"
)

type ExpirationConfig struct {
	Thresholds []int

	ProviderExpirationMinutes map[string]int
}

type ExpirationMonitor struct {
	mu       sync.RWMutex
	timers   map[string]*time.Timer
	config   ExpirationConfig
	onNotify func(name string, minutes int)
}

func NewExpirationMonitor(config ExpirationConfig, onNotify func(name string, minutes int)) *ExpirationMonitor {
	return &ExpirationMonitor{
		timers:   make(map[string]*time.Timer),
		config:   config,
		onNotify: onNotify,
	}
}

func (m *ExpirationMonitor) Start(tunnelID, name, provider string, startedAt time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()

	maxMinutes, ok := m.config.ProviderExpirationMinutes[provider]
	if !ok || maxMinutes == 0 {
		return
	}

	maxDuration := time.Duration(maxMinutes) * time.Minute
	deadline := startedAt.Add(maxDuration)

	for _, threshold := range m.config.Thresholds {
		thresholdDuration := time.Duration(threshold) * time.Minute
		notificationTime := deadline.Add(-thresholdDuration)
		wait := time.Until(notificationTime)

		if wait <= 0 {

			continue
		}

		key := m.timerKey(tunnelID, threshold)

		if existing, ok := m.timers[key]; ok {
			existing.Stop()
		}

		m.timers[key] = time.AfterFunc(wait, func() {
			m.mu.RLock()
			callback := m.onNotify
			m.mu.RUnlock()

			if callback != nil {
				callback(name, threshold)
			}

			m.mu.Lock()
			delete(m.timers, key)
			m.mu.Unlock()
		})
	}

	expireWait := time.Until(deadline)
	if expireWait > 0 {
		key := m.timerKey(tunnelID, 0)
		if existing, ok := m.timers[key]; ok {
			existing.Stop()
		}

		m.timers[key] = time.AfterFunc(expireWait, func() {
			m.mu.RLock()
			callback := m.onNotify
			m.mu.RUnlock()

			if callback != nil {
				callback(name, 0)
			}

			m.mu.Lock()
			delete(m.timers, key)
			m.mu.Unlock()
		})
	}
}

func (m *ExpirationMonitor) Stop(tunnelID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for key, timer := range m.timers {
		if len(key) >= len(tunnelID) && key[:len(tunnelID)] == tunnelID {
			timer.Stop()
			delete(m.timers, key)
		}
	}
}

func (m *ExpirationMonitor) StopAll() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, timer := range m.timers {
		timer.Stop()
	}
	m.timers = make(map[string]*time.Timer)
}

func (m *ExpirationMonitor) timerKey(tunnelID string, threshold int) string {
	return fmt.Sprintf("%s-%d", tunnelID, threshold)
}
