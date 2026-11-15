package events

import (
	"sync"
	"testing"
	"time"
)

func recv(t *testing.T, ch <-chan Event) Event {
	t.Helper()
	select {
	case e, ok := <-ch:
		if !ok {
			t.Fatal("channel closed, want an event")
		}
		return e
	case <-time.After(2 * time.Second):
		t.Fatal("no event delivered")
	}
	return nil
}

func TestEverySubscriberReceivesEveryEvent(t *testing.T) {
	b := New()
	defer b.Close()

	const subs, n = 3, 50
	chans := make([]<-chan Event, subs)
	for i := range chans {
		ch, cancel := b.Subscribe()
		defer cancel()
		chans[i] = ch
	}

	for i := 0; i < n; i++ {
		b.Publish(InstallProgress{Provider: "cloudflared", Percent: float64(i)})
	}

	for i, ch := range chans {
		for want := 0; want < n; want++ {
			got, ok := recv(t, ch).(InstallProgress)
			if !ok || got.Percent != float64(want) {
				t.Fatalf("subscriber %d event %d = %+v, want percent %d in order", i, want, got, want)
			}
		}
	}
}

func TestPublishNeverWaitsForASlowSubscriber(t *testing.T) {
	b := New()
	defer b.Close()

	_, cancel := b.Subscribe(WithQueue(4)) // never read
	defer cancel()
	fast, cancelFast := b.Subscribe()
	defer cancelFast()

	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 10_000; i++ {
			b.Publish(LogLine{TunnelID: "t1", Line: "x"})
		}
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Publish blocked on a subscriber that never reads")
	}

	if b.Dropped() == 0 {
		t.Error("Dropped() = 0, want the stuck subscriber to have shed old events")
	}
	if _, ok := recv(t, fast).(LogLine); !ok {
		t.Error("the reading subscriber got nothing")
	}
}

func TestSlowSubscriberKeepsTheNewestEvents(t *testing.T) {
	b := New()
	defer b.Close()

	ch, cancel := b.Subscribe(WithQueue(2))
	defer cancel()

	// The pump may hold one event in hand while it waits for the reader, so
	// the reader sees at most queue+1 events and the last one is the newest.
	for i := 0; i < 100; i++ {
		b.Publish(InstallProgress{Percent: float64(i)})
	}

	var last InstallProgress
	for {
		select {
		case e := <-ch:
			last = e.(InstallProgress)
			continue
		case <-time.After(100 * time.Millisecond):
		}
		break
	}
	if last.Percent != 99 {
		t.Fatalf("last delivered = %v, want the newest (99)", last.Percent)
	}
}

func TestCancelClosesTheChannelAndStopsDelivery(t *testing.T) {
	b := New()
	defer b.Close()

	ch, cancel := b.Subscribe()
	cancel()
	cancel() // twice is fine

	select {
	case _, ok := <-ch:
		if ok {
			t.Fatal("got an event after cancel")
		}
	case <-time.After(time.Second):
		t.Fatal("channel not closed after cancel")
	}

	b.Publish(TunnelDeleted{ID: "x"}) // must not panic on a cancelled subscriber
}

func TestOnlyFiltersByKind(t *testing.T) {
	b := New()
	defer b.Close()

	ch, cancel := b.Subscribe(Only(KindTunnelDeleted))
	defer cancel()

	b.Publish(LogLine{Line: "noise"})
	b.Publish(TunnelDeleted{ID: "a"})

	if got, ok := recv(t, ch).(TunnelDeleted); !ok || got.ID != "a" {
		t.Fatalf("got %+v, want TunnelDeleted a", got)
	}
}

func TestCloseEndsEverySubscription(t *testing.T) {
	b := New()
	ch, _ := b.Subscribe()
	b.Close()

	select {
	case _, ok := <-ch:
		if ok {
			t.Fatal("got an event, want a closed channel")
		}
	case <-time.After(time.Second):
		t.Fatal("Close did not close the subscription")
	}

	late, _ := b.Subscribe()
	if _, ok := <-late; ok {
		t.Fatal("a subscription after Close should be closed")
	}
	b.Publish(TunnelDeleted{ID: "x"})
}

func TestNilBusPublishIsANoop(t *testing.T) {
	var b *Bus
	b.Publish(TunnelDeleted{ID: "x"})
}

func TestConcurrentPublishSubscribeCancel(t *testing.T) {
	b := New()
	defer b.Close()

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				b.Publish(TunnelStats{ID: "t", Visitors: j})
			}
		}()
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				ch, cancel := b.Subscribe(WithQueue(8))
				select {
				case <-ch:
				default:
				}
				cancel()
			}
		}()
	}
	wg.Wait()
}
