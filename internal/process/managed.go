package process

import (
	"bytes"
	"strings"
	"sync"

	"github.com/sthbryan/ftm/internal/config"
	"github.com/sthbryan/ftm/internal/providers"
)

type ManagedProcess struct {
	Config         config.TunnelConfig
	Provider       providers.Provider
	Process        *providers.Process
	LogBuffer      *LogBuffer
	Status         config.TunnelStatus
	PublicURL      string
	OnUpdate       func(config.TunnelStatus)
	logsMu         sync.RWMutex
	logSubscribers map[chan string]struct{}
}

func (mp *ManagedProcess) addLogSubscriber() (chan string, func()) {
	ch := make(chan string, 100)
	mp.logsMu.Lock()
	if mp.logSubscribers == nil {
		mp.logSubscribers = make(map[chan string]struct{})
	}
	mp.logSubscribers[ch] = struct{}{}
	mp.logsMu.Unlock()

	cancel := func() {
		mp.logsMu.Lock()
		if _, ok := mp.logSubscribers[ch]; ok {
			delete(mp.logSubscribers, ch)
			close(ch)
		}
		mp.logsMu.Unlock()
	}

	return ch, cancel
}

func (mp *ManagedProcess) publishLog(line string) {
	mp.logsMu.RLock()
	for ch := range mp.logSubscribers {
		select {
		case ch <- line:
		default:
		}
	}
	mp.logsMu.RUnlock()
}

func (mp *ManagedProcess) closeLogSubscribers() {
	mp.logsMu.Lock()
	for ch := range mp.logSubscribers {
		close(ch)
		delete(mp.logSubscribers, ch)
	}
	mp.logsMu.Unlock()
}

type urlCaptureWriter struct {
	provider providers.Provider
	onURL    func(string)
	buf      bytes.Buffer
}

func newURLCapture(provider providers.Provider, onURL func(string)) *urlCaptureWriter {
	return &urlCaptureWriter{
		provider: provider,
		onURL:    onURL,
	}
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
