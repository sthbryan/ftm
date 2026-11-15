package process

import (
	"context"
	"io"
	"os/exec"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sthbryan/ftm/internal/config"
	"github.com/sthbryan/ftm/internal/providers"
)

// blockingProvider parks inside Start until the test releases it, which is the
// window where the manager used to hold its lock.
type blockingProvider struct {
	entered chan struct{}
	release chan struct{}
}

func (p *blockingProvider) Name() string       { return "Blocker" }
func (p *blockingProvider) BinaryName() string { return "sh" }

func (p *blockingProvider) Start(ctx context.Context, tunnel config.TunnelConfig, logWriter io.Writer) (*providers.Process, error) {
	close(p.entered)
	<-p.release

	ctx, cancel := context.WithCancel(ctx)
	cmd := exec.CommandContext(ctx, "sh", "-c", "echo https://blocker.example; sleep 5")
	cmd.Stdout = logWriter
	cmd.Stderr = logWriter

	return providers.StartProcess(cmd, cancel)
}

func (p *blockingProvider) ParseURL(line string) string {
	if len(line) > 8 && line[:8] == "https://" {
		return line
	}
	return ""
}

func TestStartDoesNotBlockReadsWhileSpawning(t *testing.T) {
	provider := &blockingProvider{
		entered: make(chan struct{}),
		release: make(chan struct{}),
	}

	m := NewManager()
	m.providers[config.ProviderCloudflared] = provider
	m.SetVisitorTracking(false)
	t.Cleanup(m.StopAll)

	started := make(chan error, 1)
	go func() { started <- m.Start(tunnelFixture()) }()

	<-provider.entered

	reads := make(chan struct{})
	go func() {
		m.GetStatus("t1")
		m.GetLogs("t1")
		m.IsRunning("t1")
		close(reads)
	}()

	select {
	case <-reads:
	case <-time.After(2 * time.Second):
		t.Fatal("status reads blocked while a tunnel was spawning")
	}

	close(provider.release)
	if err := <-started; err != nil {
		t.Fatalf("Start: %v", err)
	}
}

// countingProvider waits long enough for two Starts to overlap.
type countingProvider struct {
	started atomic.Int32
}

func (p *countingProvider) Name() string       { return "Counter" }
func (p *countingProvider) BinaryName() string { return "sh" }

func (p *countingProvider) Start(ctx context.Context, tunnel config.TunnelConfig, logWriter io.Writer) (*providers.Process, error) {
	p.started.Add(1)
	time.Sleep(50 * time.Millisecond)

	ctx, cancel := context.WithCancel(ctx)
	cmd := exec.CommandContext(ctx, "sh", "-c", "echo https://counter.example; sleep 5")
	cmd.Stdout = logWriter
	cmd.Stderr = logWriter

	return providers.StartProcess(cmd, cancel)
}

func (p *countingProvider) ParseURL(line string) string {
	if len(line) > 8 && line[:8] == "https://" {
		return line
	}
	return ""
}

func TestConcurrentStartsReportOneWinner(t *testing.T) {
	provider := &countingProvider{}

	m := NewManager()
	m.providers[config.ProviderCloudflared] = provider
	m.SetVisitorTracking(false)
	t.Cleanup(m.StopAll)

	results := make(chan error, 2)
	for i := 0; i < 2; i++ {
		go func() { results <- m.Start(tunnelFixture()) }()
	}

	successes := 0
	for i := 0; i < 2; i++ {
		if err := <-results; err == nil {
			successes++
		}
	}

	if successes != 1 {
		t.Fatalf("%d concurrent Starts reported success, want exactly 1", successes)
	}
	if got := provider.started.Load(); got != 2 {
		t.Fatalf("provider spawned %d times, want both attempts to reach the provider", got)
	}
}
