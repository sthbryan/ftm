package supervisor

import (
	"testing"
	"time"

	"github.com/sthbryan/ftm/internal/core"
)

func TestNotificationHandlerDoesNotHoldTheManagerLock(t *testing.T) {
	m, updates := newScriptManager(t, "echo https://example.trycloudflare.com; sleep 5")
	t.Cleanup(m.StopAll)

	reading := make(chan []string, 1)

	m.SetNotificationHandler(func(status core.TunnelStatus) {
		if status.State != core.TunnelStateOnline {
			return
		}
		reading <- m.GetLogs("t1")
	})

	if err := m.Start(tunnelFixture()); err != nil {
		t.Fatalf("Start: %v", err)
	}

	select {
	case <-reading:
	case <-time.After(5 * time.Second):
		t.Fatal("the notification handler deadlocked against the manager lock")
	}

	waitForState(t, updates, core.TunnelStateOnline)
}

func TestStatusReadsStayResponsiveWhileNotifying(t *testing.T) {
	m, _ := newScriptManager(t, "echo https://example.trycloudflare.com; sleep 5")
	t.Cleanup(m.StopAll)

	release := make(chan struct{})
	notified := make(chan struct{})

	m.SetNotificationHandler(func(status core.TunnelStatus) {
		if status.State != core.TunnelStateOnline {
			return
		}
		close(notified)
		<-release
	})

	if err := m.Start(tunnelFixture()); err != nil {
		t.Fatalf("Start: %v", err)
	}

	<-notified

	done := make(chan struct{})
	go func() {
		defer close(done)
		m.GetStatus("t1")
		m.GetLogs("t1")
		m.IsRunning("t1")
	}()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Error("a slow notification froze every read on the manager")
	}

	close(release)
}
