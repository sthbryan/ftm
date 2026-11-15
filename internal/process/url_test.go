package process

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/sthbryan/ftm/internal/config"
)

func TestRepeatedURLInLogsNotifiesOnce(t *testing.T) {
	script := `echo "https://example.trycloudflare.com"
echo "https://example.trycloudflare.com"
echo "https://example.trycloudflare.com"
sleep 1`

	m, updates := newScriptManager(t, script)
	t.Cleanup(m.StopAll)

	var online int32
	m.SetNotificationHandler(func(status config.TunnelStatus) {
		if status.State == config.TunnelStateOnline {
			atomic.AddInt32(&online, 1)
		}
	})

	if err := m.Start(tunnelFixture()); err != nil {
		t.Fatalf("Start: %v", err)
	}

	waitForState(t, updates, config.TunnelStateOnline)
	time.Sleep(500 * time.Millisecond)

	if got := atomic.LoadInt32(&online); got != 1 {
		t.Errorf("every request line notified: got %d online notifications, want 1", got)
	}
}

func TestFirstURLOfARunWins(t *testing.T) {
	script := `echo "https://kolfa-1-2-3-4.run.pinggy-free.link"
sleep 0.3
echo "https://uqehf-1-2-3-4.free.pinggy.net"
sleep 1`

	m, updates := newScriptManager(t, script)
	t.Cleanup(m.StopAll)

	seen := make(chan string, 8)
	m.SetNotificationHandler(func(status config.TunnelStatus) {
		if status.State == config.TunnelStateOnline {
			seen <- status.PublicURL
		}
	})

	if err := m.Start(tunnelFixture()); err != nil {
		t.Fatalf("Start: %v", err)
	}

	waitForState(t, updates, config.TunnelStateOnline)

	select {
	case got := <-seen:
		if got != "https://kolfa-1-2-3-4.run.pinggy-free.link" {
			t.Fatalf("notified with %q, want the first URL of the run", got)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("never notified")
	}

	select {
	case got := <-seen:
		t.Errorf("alternate host re-notified with %q", got)
	case <-time.After(700 * time.Millisecond):
	}
}
