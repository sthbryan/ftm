package process

import (
	"bytes"
	"strings"

	"github.com/sthbryan/ftm/internal/config"
	"github.com/sthbryan/ftm/internal/providers"
)

type ManagedProcess struct {
	Config    config.TunnelConfig
	Provider  providers.Provider
	Process   *providers.Process
	LogBuffer *LogBuffer
	Status    config.TunnelStatus
	PublicURL string
	OnUpdate  func(config.TunnelStatus)
	LogStream chan string
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
