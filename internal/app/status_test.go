package app

import (
	"testing"
	"time"

	"github.com/sthbryan/ftm/internal/config"
)

func TestPublishStatusReachesTheModel(t *testing.T) {
	m := &Model{statusUpdates: make(chan config.TunnelStatus, 4)}

	want := config.TunnelStatus{ID: "t1", State: config.TunnelStateOnline}
	m.publishStatus(want)

	msg := m.waitForStatus()()

	got, ok := msg.(statusUpdateMsg)
	if !ok {
		t.Fatalf("waitForStatus produced %T, want statusUpdateMsg", msg)
	}
	if got.tunnelID != want.ID || got.status.State != want.State {
		t.Fatalf("got %+v, want %+v", got.status, want)
	}
}

func TestPublishStatusNeverBlocksTheManager(t *testing.T) {
	m := &Model{statusUpdates: make(chan config.TunnelStatus, 1)}

	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 100; i++ {
			m.publishStatus(config.TunnelStatus{ID: "t1"})
		}
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("publishStatus blocked once the buffer filled")
	}
}

func TestMessageExpiresOnItsOwnSchedule(t *testing.T) {
	m := &Model{App: &App{Store: testStore(t, config.DefaultConfig())}}

	m.showMessage("URL copied")
	if m.Message == "" {
		t.Fatal("showMessage did not set the message")
	}

	m.handleTick()
	if m.Message == "" {
		t.Error("the message expired immediately instead of lasting its timeout")
	}

	m.messageUntil = time.Now().Add(-time.Second)
	m.handleTick()

	if m.Message != "" {
		t.Error("the message outlived its timeout")
	}
}
