package process

import (
	"context"
	"io"
	"os/exec"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/sthbryan/ftm/internal/config"
	"github.com/sthbryan/ftm/internal/events"
	"github.com/sthbryan/ftm/internal/providers"
)

type scriptProvider struct {
	script string
}

func (scriptProvider) Name() string       { return "Script" }
func (scriptProvider) BinaryName() string { return "sh" }

func (p scriptProvider) Start(ctx context.Context, _ config.TunnelConfig, logWriter io.Writer) (*providers.Process, error) {
	ctx, cancel := context.WithCancel(ctx)
	cmd := exec.CommandContext(ctx, "sh", "-c", p.script)
	cmd.Stdout = logWriter
	cmd.Stderr = logWriter

	return providers.StartProcess(cmd, cancel)
}

func (scriptProvider) ParseURL(line string) string {
	if len(line) > 8 && line[:8] == "https://" {
		return line
	}
	return ""
}

func newScriptManager(t *testing.T, script string) (*Manager, chan config.TunnelStatus) {
	t.Helper()

	if runtime.GOOS == "windows" {
		t.Skip("uses a POSIX shell")
	}

	m := NewManager()
	m.providers[config.ProviderCloudflared] = scriptProvider{script: script}

	updates := make(chan config.TunnelStatus, 64)
	forwardStatuses(t, m, updates)

	return m, updates
}

func tunnelFixture() config.TunnelConfig {
	return config.TunnelConfig{
		ID:        "t1",
		Name:      "Foundry VTT",
		Provider:  config.ProviderCloudflared,
		LocalPort: 30000,
	}
}

func waitForState(t *testing.T, updates <-chan config.TunnelStatus, want config.TunnelState) config.TunnelStatus {
	t.Helper()

	deadline := time.After(15 * time.Second)
	for {
		select {
		case status := <-updates:
			if status.State == want {
				return status
			}
		case <-deadline:
			t.Fatalf("never saw state %q", want)
		}
	}
}

func TestManagerReportsCrashedProvider(t *testing.T) {
	m, updates := newScriptManager(t, "echo https://example.trycloudflare.com; sleep 0.2; exit 7")

	if err := m.Start(tunnelFixture()); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	status := waitForState(t, updates, config.TunnelStateError)
	if status.ErrorMessage == "" {
		t.Error("error state carries no message")
	}
	if status.PublicURL != "" {
		t.Errorf("PublicURL = %q after the process died, want it cleared", status.PublicURL)
	}

	if m.IsRunning("t1") {
		t.Error("crashed tunnel is still reported as running")
	}

	persisted, ok := m.GetStatus("t1")
	if !ok {
		t.Fatal("crashed tunnel has no status left for the REST API to report")
	}
	if persisted.State != config.TunnelStateError {
		t.Errorf("persisted state = %q, want %q", persisted.State, config.TunnelStateError)
	}
	if persisted.ErrorMessage == "" {
		t.Error("persisted status lost the error message")
	}
}

func TestManagerReportsCleanExitAsStopped(t *testing.T) {
	m, updates := newScriptManager(t, "exit 0")

	if err := m.Start(tunnelFixture()); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	waitForState(t, updates, config.TunnelStateStopped)
}

func TestManagerStopIsNotReportedAsError(t *testing.T) {
	m, updates := newScriptManager(t, "sleep 60")

	if err := m.Start(tunnelFixture()); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	if err := m.Stop("t1"); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	waitForState(t, updates, config.TunnelStateStopped)

	deadline := time.After(time.Second)
	for {
		select {
		case status := <-updates:
			if status.State == config.TunnelStateError {
				t.Fatalf("Stop produced an error state: %q", status.ErrorMessage)
			}
		case <-deadline:
			return
		}
	}
}

func TestManagerStopWaitsForExit(t *testing.T) {
	m, _ := newScriptManager(t, "sleep 60")

	if err := m.Start(tunnelFixture()); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	m.mu.RLock()
	proc := m.processes["t1"].Process
	m.mu.RUnlock()

	if err := m.Stop("t1"); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	select {
	case <-proc.Exited():
	default:
		t.Fatal("Stop returned while the process was still running")
	}
}

func TestManagerStopAllWaitsForEveryProcess(t *testing.T) {
	m, _ := newScriptManager(t, "sleep 60")

	for _, id := range []string{"t1", "t2", "t3"} {
		tunnel := tunnelFixture()
		tunnel.ID = id
		if err := m.Start(tunnel); err != nil {
			t.Fatalf("Start(%s) failed: %v", id, err)
		}
	}

	m.mu.RLock()
	procs := make([]*providers.Process, 0, len(m.processes))
	for _, mp := range m.processes {
		procs = append(procs, mp.Process)
	}
	m.mu.RUnlock()

	if len(procs) != 3 {
		t.Fatalf("started %d processes, want 3", len(procs))
	}

	m.StopAll()

	for i, proc := range procs {
		select {
		case <-proc.Exited():
		default:
			t.Fatalf("process %d still running after StopAll", i)
		}
	}
}

func TestManagerRestartIsNotTimedOutByTheOldMonitor(t *testing.T) {
	m, updates := newScriptManager(t, "sleep 60")

	if err := m.Start(tunnelFixture()); err != nil {
		t.Fatalf("first Start failed: %v", err)
	}
	if err := m.Stop("t1"); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}
	if err := m.Start(tunnelFixture()); err != nil {
		t.Fatalf("second Start failed: %v", err)
	}

	deadline := time.After(9 * time.Second)
	for {
		select {
		case status := <-updates:
			if status.State == config.TunnelStateTimeout {
				t.Fatal("the stale monitor timed out the restarted tunnel")
			}
		case <-deadline:
			if _, ok := m.GetStatus("t1"); !ok {
				t.Fatal("restarted tunnel is no longer registered")
			}
			m.StopAll()
			return
		}
	}
}

func TestManagerPublishesExpiryForExpiringProviders(t *testing.T) {
	m, updates := newScriptManager(t, "sleep 5")
	m.SetProviderExpiration(map[string]int{string(config.ProviderCloudflared): 60})

	before := time.Now()
	if err := m.Start(tunnelFixture()); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer m.StopAll()

	status := <-updates
	if status.ExpiresAt == 0 {
		t.Fatal("ExpiresAt = 0 for a provider that expires after 60 minutes")
	}

	got := time.UnixMilli(status.ExpiresAt)
	wantMin := before.Add(59 * time.Minute)
	wantMax := before.Add(61 * time.Minute)
	if got.Before(wantMin) || got.After(wantMax) {
		t.Fatalf("ExpiresAt = %v, want roughly 60 minutes after %v", got, before)
	}
}

func TestManagerLeavesExpiryUnsetForProvidersThatDoNotExpire(t *testing.T) {
	m, updates := newScriptManager(t, "sleep 5")
	m.SetProviderExpiration(map[string]int{string(config.ProviderCloudflared): 0})

	if err := m.Start(tunnelFixture()); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer m.StopAll()

	if status := <-updates; status.ExpiresAt != 0 {
		t.Fatalf("ExpiresAt = %d for a provider that does not expire, want 0", status.ExpiresAt)
	}
}

func TestLogLinesArePublished(t *testing.T) {
	m, _ := newScriptManager(t, "echo hello from the provider; sleep 5")
	t.Cleanup(m.StopAll)

	bus := events.New()
	defer bus.Close()
	m.SetEvents(bus)
	ch, cancel := bus.Subscribe(events.Only(events.KindLogLine))
	defer cancel()

	if err := m.Start(tunnelFixture()); err != nil {
		t.Fatalf("Start: %v", err)
	}

	deadline := time.After(5 * time.Second)
	for {
		select {
		case e := <-ch:
			line := e.(events.LogLine)
			if line.TunnelID == "t1" && strings.Contains(line.Line, "hello from the provider") {
				return
			}
		case <-deadline:
			t.Fatal("the provider's output was never published")
		}
	}
}
