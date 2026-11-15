package process

import (
	"errors"
	"testing"
	"time"

	"github.com/sthbryan/ftm/internal/config"
	"github.com/sthbryan/ftm/internal/events"
	"github.com/sthbryan/ftm/internal/providers"
)

// forwardStatuses gives m a bus and copies every tunnel status it publishes
// into ch, the way the tests read state changes.
func forwardStatuses(t testing.TB, m *Manager, ch chan config.TunnelStatus) {
	t.Helper()

	bus := events.New()
	m.SetEvents(bus)
	sub, cancel := bus.Subscribe(events.Only(events.KindTunnelStatus))
	done := make(chan struct{})
	t.Cleanup(func() {
		close(done)
		cancel()
		bus.Close()
	})

	go func() {
		for e := range sub {
			select {
			case ch <- e.(events.TunnelStatus).Status:
			case <-done:
				return
			}
		}
	}()
}

func TestStatusUpdatesNeverWaitForAReader(t *testing.T) {
	m := NewManager()
	bus := events.New()
	defer bus.Close()
	m.SetEvents(bus)
	_, cancel := bus.Subscribe(events.WithQueue(2)) // never read
	defer cancel()

	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 1000; i++ {
			m.callStatusUpdate(config.TunnelStatus{ID: "t1", State: config.TunnelStateOnline})
		}
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("a status update waited for a subscriber that never reads")
	}
}

func TestStatusUpdateWithoutEventsIsNoop(t *testing.T) {
	m := NewManager()

	done := make(chan struct{})
	go func() {
		defer close(done)
		m.callStatusUpdate(config.TunnelStatus{ID: "t1", State: config.TunnelStateStopped})
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("callStatusUpdate blocked with no bus set")
	}
}

func TestStatusUpdateIsPublished(t *testing.T) {
	m := NewManager()
	ch := make(chan config.TunnelStatus, 4)
	forwardStatuses(t, m, ch)

	want := config.TunnelStatus{ID: "t1", Name: "Foundry", State: config.TunnelStateOnline}
	m.callStatusUpdate(want)

	select {
	case got := <-ch:
		if got.ID != want.ID || got.State != want.State {
			t.Fatalf("got %+v, want %+v", got, want)
		}
	case <-time.After(time.Second):
		t.Fatal("status update was not published")
	}
}

func TestInstallProgressReachesEverySubscriber(t *testing.T) {
	m := NewManager()
	bus := events.New()
	defer bus.Close()
	m.SetEvents(bus)

	a, cancelA := bus.Subscribe(events.Only(events.KindInstallProgress))
	defer cancelA()
	b, cancelB := bus.Subscribe(events.Only(events.KindInstallProgress))
	defer cancelB()

	progress, finish := m.progressSink("cloudflared")
	for i := 1; i <= 3; i++ {
		progress <- providers.DownloadProgress{Percent: float64(i * 30), Name: "cloudflared"}
	}
	progress <- providers.DownloadProgress{Percent: 100, Done: true, Name: "cloudflared"}
	finish(nil)

	for name, ch := range map[string]<-chan events.Event{"a": a, "b": b} {
		for i := 0; i < 4; i++ {
			select {
			case e := <-ch:
				p := e.(events.InstallProgress)
				if p.Provider != "cloudflared" {
					t.Fatalf("%s: provider = %q", name, p.Provider)
				}
				if i == 3 && !p.Done {
					t.Fatalf("%s: last event %+v, want Done", name, p)
				}
			case <-time.After(2 * time.Second):
				t.Fatalf("%s: got %d of 4 progress events", name, i)
			}
		}
	}
}

func TestInstallFailureIsPublishedAfterProgress(t *testing.T) {
	m := NewManager()
	bus := events.New()
	defer bus.Close()
	m.SetEvents(bus)
	ch, cancel := bus.Subscribe()
	defer cancel()

	progress, finish := m.progressSink("bore")
	progress <- providers.DownloadProgress{Percent: 10}
	finish(errors.New("checksum mismatch"))

	first := (<-ch).(events.InstallProgress)
	if first.Provider != "bore" || first.Percent != 10 {
		t.Fatalf("first = %+v, want bore at 10%%", first)
	}
	last := (<-ch).(events.InstallProgress)
	if last.Err != "checksum mismatch" {
		t.Fatalf("last = %+v, want the failure", last)
	}
}

// A slow subscriber must not hold up a download: the provider writes progress
// as fast as it likes and the install returns.
func TestSlowSubscriberDoesNotBlockADownload(t *testing.T) {
	m := NewManager()
	bus := events.New()
	defer bus.Close()
	m.SetEvents(bus)
	_, cancel := bus.Subscribe(events.WithQueue(1)) // never read
	defer cancel()

	done := make(chan struct{})
	go func() {
		defer close(done)
		progress, finish := m.progressSink("cloudflared")
		for i := 0; i < 5000; i++ {
			progress <- providers.DownloadProgress{Percent: float64(i) / 50}
		}
		finish(nil)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("the download waited on a subscriber that never reads")
	}
}
