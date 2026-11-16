package supervisor

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os/exec"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sthbryan/ftm/internal/core"
	"github.com/sthbryan/ftm/internal/provider"
)

type portRecordingProvider struct {
	port atomic.Int64
}

func (p *portRecordingProvider) Name() string       { return "Recorder" }
func (p *portRecordingProvider) BinaryName() string { return "sh" }

func (p *portRecordingProvider) Start(ctx context.Context, tunnel core.TunnelConfig, logWriter io.Writer) (*provider.Process, error) {
	p.port.Store(int64(tunnel.LocalPort))

	ctx, cancel := context.WithCancel(ctx)
	cmd := exec.CommandContext(ctx, "sh", "-c", "echo https://recorder.example; sleep 2")
	cmd.Stdout = logWriter
	cmd.Stderr = logWriter

	return provider.StartProcess(cmd, cancel)
}

func (p *portRecordingProvider) ParseURL(line string) string {
	if len(line) > 8 && line[:8] == "https://" {
		return line
	}
	return ""
}

func TestTunnelIsExposedThroughTheVisitorProxy(t *testing.T) {
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, "foundry")
	}))
	t.Cleanup(origin.Close)

	parsed, err := url.Parse(origin.URL)
	if err != nil {
		t.Fatalf("parse origin: %v", err)
	}
	originPort, err := strconv.Atoi(parsed.Port())
	if err != nil {
		t.Fatalf("parse origin port: %v", err)
	}

	provider := &portRecordingProvider{}
	m := NewManager()
	m.providers[core.ProviderCloudflared] = provider
	updates := make(chan core.TunnelStatus, 64)
	forwardStatuses(t, m, updates)
	t.Cleanup(m.StopAll)

	tunnel := tunnelFixture()
	tunnel.LocalPort = originPort

	if err := m.Start(tunnel); err != nil {
		t.Fatalf("Start: %v", err)
	}
	waitForState(t, updates, core.TunnelStateOnline)

	exposed := int(provider.port.Load())
	if exposed == originPort {
		t.Fatalf("provider was pointed straight at the origin port %d", originPort)
	}

	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/", exposed))
	if err != nil {
		t.Fatalf("get through the proxy: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "foundry" {
		t.Fatalf("proxy returned %q, want %q", body, "foundry")
	}

	status, ok := m.GetStatus(tunnel.ID)
	if !ok {
		t.Fatal("no status for the running tunnel")
	}
	if status.Visitors != 1 {
		t.Errorf("Visitors = %d, want 1", status.Visitors)
	}
	if status.LocalPort != originPort {
		t.Errorf("LocalPort = %d, want the origin port %d", status.LocalPort, originPort)
	}
}

func TestVisitorTrackingCanBeDisabled(t *testing.T) {
	provider := &portRecordingProvider{}
	m := NewManager()
	m.providers[core.ProviderCloudflared] = provider
	m.SetVisitorTracking(false)
	updates := make(chan core.TunnelStatus, 64)
	forwardStatuses(t, m, updates)
	t.Cleanup(m.StopAll)

	if err := m.Start(tunnelFixture()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	waitForState(t, updates, core.TunnelStateOnline)

	if got := int(provider.port.Load()); got != tunnelFixture().LocalPort {
		t.Errorf("provider port = %d, want the local port %d", got, tunnelFixture().LocalPort)
	}
}

func TestProxyStopsWithTheTunnel(t *testing.T) {
	provider := &portRecordingProvider{}
	m := NewManager()
	m.providers[core.ProviderCloudflared] = provider
	updates := make(chan core.TunnelStatus, 64)
	forwardStatuses(t, m, updates)

	if err := m.Start(tunnelFixture()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	waitForState(t, updates, core.TunnelStateOnline)

	exposed := int(provider.port.Load())
	if err := m.Stop("t1"); err != nil {
		t.Fatalf("Stop: %v", err)
	}

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/", exposed))
		if err != nil {
			return
		}
		resp.Body.Close()

		time.Sleep(20 * time.Millisecond)
	}

	t.Fatal("the proxy port is still accepting requests after the tunnel stopped")
}
