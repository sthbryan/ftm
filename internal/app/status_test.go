package app

import (
	"testing"
	"time"

	"github.com/sthbryan/ftm/internal/config"
	"github.com/sthbryan/ftm/internal/events"
	"github.com/sthbryan/ftm/internal/service"
)

func TestBusEventsReachTheModel(t *testing.T) {
	app := testApp(t, testStore(t, config.DefaultConfig()))
	m := NewModel(app)
	defer m.stopEvents()

	want := config.TunnelStatus{ID: "t1", State: config.TunnelStateOnline}
	app.Bus.Publish(events.TunnelStatus{Status: want})

	msg, ok := m.waitForEvent()().(busMsg)
	if !ok {
		t.Fatal("waitForEvent did not produce a bus message")
	}
	got, ok := msg.msg.(statusUpdateMsg)
	if !ok {
		t.Fatalf("waitForEvent produced %T, want statusUpdateMsg", msg.msg)
	}
	if got.tunnelID != want.ID || got.status.State != want.State {
		t.Fatalf("got %+v, want %+v", got.status, want)
	}
}

func TestChangesFromAnotherSurfaceRefreshTheList(t *testing.T) {
	app := testApp(t, testStore(t, config.DefaultConfig()))
	m := NewModel(app)
	defer m.stopEvents()

	// The web dashboard adds a connection through the same service.
	if _, err := app.Service.CreateTunnel(service.TunnelInput{Name: "World", Provider: config.ProviderBore, LocalPort: 30000}); err != nil {
		t.Fatalf("CreateTunnel: %v", err)
	}

	msg := m.waitForEvent()()
	m.Update(msg)

	if len(m.Items) != 1 {
		t.Fatalf("Items = %d, want the new connection listed", len(m.Items))
	}
}

func TestInstallProgressLeavesTheDownloadScreenWhenDone(t *testing.T) {
	m := &Model{App: testApp(t, testStore(t, config.DefaultConfig())), State: viewList}

	m.Update(installStartedMsg{provider: config.ProviderCloudflared})
	if m.State != viewDownloading {
		t.Fatalf("State = %v, want the download screen", m.State)
	}

	m.Update(downloadProgressMsg{Name: "cloudflared", Percent: 40})
	if m.State != viewDownloading || m.DownloadProgress.Percent != 40 {
		t.Fatalf("State = %v, progress = %v", m.State, m.DownloadProgress.Percent)
	}

	m.Update(downloadProgressMsg{Name: "cloudflared", Percent: 100, Done: true})
	if m.State != viewList {
		t.Fatalf("State = %v, want the list after the download", m.State)
	}
}

func TestInstallFromAnotherSurfaceDoesNotMoveTheTUI(t *testing.T) {
	m := &Model{App: testApp(t, testStore(t, config.DefaultConfig())), State: viewSettings}

	m.Update(downloadProgressMsg{Name: "cloudflared", Percent: 100, Done: true})
	if m.State != viewSettings {
		t.Fatalf("State = %v, want the settings screen left alone", m.State)
	}
}

func TestInstallFailureLeavesTheDownloadScreen(t *testing.T) {
	m := &Model{App: testApp(t, testStore(t, config.DefaultConfig())), State: viewDownloading}

	m.Update(statusUpdateMsg{tunnelID: "t1", status: config.TunnelStatus{ID: "t1", State: config.TunnelStateError, ErrorMessage: "network down"}})
	if m.State != viewList {
		t.Fatalf("State = %v, want the list after a failed install", m.State)
	}
}

func TestMessageExpiresOnItsOwnSchedule(t *testing.T) {
	m := &Model{App: testApp(t, testStore(t, config.DefaultConfig()))}

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
