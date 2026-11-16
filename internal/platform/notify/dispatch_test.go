package notify

import (
	"testing"
	"time"
)

type blockingNotifier struct {
	entered chan struct{}
	release chan struct{}
}

func (b *blockingNotifier) IsAvailable() bool { return true }

func (b *blockingNotifier) Notify(string, string) error {
	close(b.entered)
	<-b.release
	return nil
}

func (b *blockingNotifier) PlaySound(SoundType) error {
	close(b.entered)
	<-b.release
	return nil
}

func withBlocking(t *testing.T) *blockingNotifier {
	t.Helper()

	b := &blockingNotifier{entered: make(chan struct{}), release: make(chan struct{})}

	prevNotifier, prevSound, prevAvailable := notifier, soundPlayer, available
	notifier, soundPlayer, available = b, b, true
	SetNotificationsEnabled(true)
	SetSoundEnabled(true)

	t.Cleanup(func() {
		close(b.release)
		notifier, soundPlayer, available = prevNotifier, prevSound, prevAvailable
	})

	return b
}

func TestNotifyReturnsWithoutWaitingForTheDesktop(t *testing.T) {
	b := withBlocking(t)

	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = Notify("Tunnel Active", "Foundry VTT")
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Notify blocked on the desktop notifier")
	}

	select {
	case <-b.entered:
	case <-time.After(2 * time.Second):
		t.Error("the notifier was never actually called")
	}
}

func TestPlaySoundReturnsWithoutWaitingForThePlayer(t *testing.T) {
	b := withBlocking(t)

	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = PlaySound(SoundSuccess)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("PlaySound blocked on the audio player")
	}

	select {
	case <-b.entered:
	case <-time.After(2 * time.Second):
		t.Error("the sound player was never actually called")
	}
}
