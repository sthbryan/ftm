package supervisor

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/sthbryan/ftm/internal/core"
	"github.com/sthbryan/ftm/internal/provider"
)

type fakeProvider struct{}

func (fakeProvider) Name() string       { return "Fake" }
func (fakeProvider) BinaryName() string { return "fake" }

func (fakeProvider) Start(context.Context, core.TunnelConfig, io.Writer) (*provider.Process, error) {
	return &provider.Process{}, nil
}

func (fakeProvider) ParseURL(line string) string {
	idx := strings.Index(line, "https://")
	if idx == -1 {
		return ""
	}
	return strings.Fields(line[idx:])[0]
}

func collectURLs(t *testing.T, writes []string) []string {
	t.Helper()

	var got []string
	w := newURLCapture(fakeProvider{}, func(url string) { got = append(got, url) })

	for _, chunk := range writes {
		n, err := w.Write([]byte(chunk))
		if err != nil {
			t.Fatalf("Write(%q) failed: %v", chunk, err)
		}
		if n != len(chunk) {
			t.Fatalf("Write(%q) = %d bytes, want %d", chunk, n, len(chunk))
		}
	}

	return got
}

func TestURLCaptureFindsURLInCompleteLine(t *testing.T) {
	got := collectURLs(t, []string{"tunnel ready at https://abc.trycloudflare.com\n"})

	if len(got) != 1 || got[0] != "https://abc.trycloudflare.com" {
		t.Fatalf("captured %v, want one trycloudflare URL", got)
	}
}

func TestURLCaptureJoinsSplitWrites(t *testing.T) {
	got := collectURLs(t, []string{"tunnel ready at https://abc.tryclo", "udflare.com\n"})

	if len(got) != 1 || got[0] != "https://abc.trycloudflare.com" {
		t.Fatalf("captured %v, want the URL reassembled across writes", got)
	}
}

func TestURLCaptureDoesNotEmitIncompleteLine(t *testing.T) {
	got := collectURLs(t, []string{"tunnel ready at https://abc.trycloudflare.com"})

	if len(got) != 0 {
		t.Fatalf("captured %v from a line with no newline yet, want nothing", got)
	}
}

func TestURLCaptureHandlesMultipleLinesInOneWrite(t *testing.T) {
	got := collectURLs(t, []string{
		"starting\nurl https://one.example.com\nnoise\nurl https://two.example.com\n",
	})

	want := []string{"https://one.example.com", "https://two.example.com"}
	if len(got) != len(want) {
		t.Fatalf("captured %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("URL %d = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestURLCaptureIgnoresLinesWithoutURL(t *testing.T) {
	got := collectURLs(t, []string{"INF connecting\nINF retrying\n"})

	if len(got) != 0 {
		t.Fatalf("captured %v, want nothing", got)
	}
}

func TestLogBufferTrimsToMaxLength(t *testing.T) {
	lb := NewLogBuffer()

	for i := 0; i < lb.maxLen+250; i++ {
		if _, err := lb.Write([]byte("line\n")); err != nil {
			t.Fatalf("Write failed: %v", err)
		}
	}

	if got := len(lb.GetLines()); got > lb.maxLen {
		t.Fatalf("buffer holds %d lines, want at most %d", got, lb.maxLen)
	}
}

func TestLogBufferSkipsBlankLines(t *testing.T) {
	lb := NewLogBuffer()

	if _, err := lb.Write([]byte("first\n\n   \nsecond\n")); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	got := lb.GetLines()
	want := []string{"first", "second"}

	if len(got) != len(want) {
		t.Fatalf("lines = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("line %d = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestLogBufferPublishesLinesInOrder(t *testing.T) {
	lb := NewLogBuffer()

	var got []string
	lb.OnNewLine = func(line string) { got = append(got, line) }

	if _, err := lb.Write([]byte("one\ntwo\nthree\n")); err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if _, err := lb.Write([]byte("four\nfive\n")); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	want := []string{"one", "two", "three", "four", "five"}
	if len(got) != len(want) {
		t.Fatalf("published %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("published %v, want %v", got, want)
		}
	}
}

func TestLogBufferWriteDoesNotHoldLockDuringPublish(t *testing.T) {
	lb := NewLogBuffer()
	lb.OnNewLine = func(string) {

		lb.GetLines()
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		lb.Write([]byte("line\n")) //nolint:errcheck
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Write deadlocked while publishing to a subscriber")
	}
}

func TestLogBufferGetLinesReturnsCopy(t *testing.T) {
	lb := NewLogBuffer()
	if _, err := lb.Write([]byte("original\n")); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	lines := lb.GetLines()
	lines[0] = "mutated"

	if got := lb.GetLines()[0]; got != "original" {
		t.Fatalf("buffer line = %q, want the caller's mutation not to leak in", got)
	}
}
