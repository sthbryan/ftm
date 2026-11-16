package proxy

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"sync/atomic"
	"testing"
	"time"
)

func startOrigin(t *testing.T, handler http.Handler) int {
	t.Helper()

	origin := httptest.NewServer(handler)
	t.Cleanup(origin.Close)

	parsed, err := url.Parse(origin.URL)
	if err != nil {
		t.Fatalf("parse origin url: %v", err)
	}

	port, err := strconv.Atoi(parsed.Port())
	if err != nil {
		t.Fatalf("parse origin port: %v", err)
	}

	return port
}

func TestProxyForwardsRequests(t *testing.T) {
	port := startOrigin(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "origin:%s", r.URL.Path)
	}))

	p, err := Start(port, Options{Window: time.Minute})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() { _ = p.Close() })

	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/join", p.Port()))
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "origin:/join" {
		t.Errorf("body = %q, want %q", body, "origin:/join")
	}
}

func TestProxyCountsDistinctVisitors(t *testing.T) {
	port := startOrigin(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	p, err := Start(port, Options{Window: time.Minute})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() { _ = p.Close() })

	for _, ip := range []string{"1.1.1.1", "1.1.1.1", "2.2.2.2"} {
		req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://127.0.0.1:%d/", p.Port()), nil)
		if err != nil {
			t.Fatalf("new request: %v", err)
		}
		req.Header.Set("X-Forwarded-For", ip)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("do: %v", err)
		}
		resp.Body.Close()
	}

	stats := p.Stats()
	if stats.Visitors != 2 {
		t.Errorf("Visitors = %d, want 2", stats.Visitors)
	}
	if stats.Requests != 3 {
		t.Errorf("Requests = %d, want 3", stats.Requests)
	}
}

func TestProxyTracksLiveWebsocketSessions(t *testing.T) {
	released := make(chan struct{})
	port := startOrigin(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, _, err := w.(http.Hijacker).Hijack()
		if err != nil {
			t.Errorf("hijack: %v", err)
			return
		}
		defer conn.Close()

		fmt.Fprint(conn, "HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\n\r\n")
		<-released
	}))

	p, err := Start(port, Options{Window: time.Minute})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() { _ = p.Close() })

	conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", p.Port()))
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	fmt.Fprintf(conn, "GET /socket.io/ HTTP/1.1\r\nHost: localhost\r\nUpgrade: websocket\r\nConnection: Upgrade\r\n\r\n")

	waitFor(t, func() bool { return p.Stats().ActiveSessions == 1 })

	close(released)
	conn.Close()

	waitFor(t, func() bool { return p.Stats().ActiveSessions == 0 })
}

func waitFor(t *testing.T, cond func() bool) {
	t.Helper()

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}

	t.Fatal("condition not met before the deadline")
}

func TestNewVisitorFiresOncePerClient(t *testing.T) {
	port := startOrigin(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	var announced int32
	p, err := Start(port, Options{
		Window:       time.Minute,
		OnNewVisitor: func() { atomic.AddInt32(&announced, 1) },
	})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() { _ = p.Close() })

	for _, ip := range []string{"1.1.1.1", "1.1.1.1", "1.1.1.1", "2.2.2.2"} {
		req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://127.0.0.1:%d/", p.Port()), nil)
		if err != nil {
			t.Fatalf("new request: %v", err)
		}
		req.Header.Set("X-Forwarded-For", ip)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("do: %v", err)
		}
		resp.Body.Close()
	}

	waitFor(t, func() bool { return atomic.LoadInt32(&announced) == 2 })

	time.Sleep(200 * time.Millisecond)
	if got := atomic.LoadInt32(&announced); got != 2 {
		t.Errorf("announced %d visitors, want 2", got)
	}
}

func TestKnownVisitorIsNotAnnouncedAfterTheWindowExpires(t *testing.T) {
	c := newCounter(time.Minute)
	base := time.Now()
	c.now = func() time.Time { return base }

	if !c.seen("1.1.1.1") {
		t.Fatal("the first request was not treated as a new visitor")
	}

	c.now = func() time.Time { return base.Add(10 * time.Minute) }

	if c.seen("1.1.1.1") {
		t.Error("a returning visitor was announced as new")
	}
}

func TestVisitorsExpireAfterTheWindow(t *testing.T) {
	c := newCounter(time.Minute)
	base := time.Now()
	c.now = func() time.Time { return base }

	c.seen("1.1.1.1")
	c.now = func() time.Time { return base.Add(2 * time.Minute) }
	c.seen("2.2.2.2")

	if got := c.stats().Visitors; got != 1 {
		t.Errorf("Visitors = %d, want 1", got)
	}
}
