package proxy

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"testing"
	"time"
)

// serveOn starts a server on one loopback address only, as Foundry on Windows
// may listen on ::1 alone.
func serveOn(t *testing.T, host string) int {
	t.Helper()
	listener, err := net.Listen("tcp", net.JoinHostPort(host, "0"))
	if err != nil {
		t.Skipf("no %s loopback here: %v", host, err)
	}
	server := &http.Server{
		Handler:           http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = io.WriteString(w, "foundry") }),
		ReadHeaderTimeout: time.Second,
	}
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(func() { _ = server.Close() })
	return listener.Addr().(*net.TCPAddr).Port
}

func TestProxyReachesAServerOnEitherLoopback(t *testing.T) {
	for _, host := range []string{"127.0.0.1", "::1"} {
		t.Run(host, func(t *testing.T) {
			p, err := Start(serveOn(t, host), Options{Window: time.Minute})
			if err != nil {
				t.Fatal(err)
			}
			defer p.Close()

			resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/", p.Port()))
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()
			body, _ := io.ReadAll(resp.Body)
			if resp.StatusCode != http.StatusOK || string(body) != "foundry" {
				t.Fatalf("got %d %q, want 200 \"foundry\"", resp.StatusCode, body)
			}
		})
	}
}

func TestDialLoopbackFailsWhenNothingListens(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	_ = listener.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if conn, err := DialLoopback(ctx, port); err == nil {
		_ = conn.Close()
		t.Fatal("dialed a port nothing listens on")
	}
}

// A Windows port forward can hold 127.0.0.1 on Foundry's port, accept every
// connection and never answer, while Foundry answers on ::1. Every request
// through the proxy must still reach Foundry.
func TestProxySkipsALoopbackThatNeverAnswers(t *testing.T) {
	port := serveOn(t, "::1")
	silent, err := net.Listen("tcp", net.JoinHostPort("127.0.0.1", fmt.Sprint(port)))
	if err != nil {
		t.Skipf("can't share the port on 127.0.0.1: %v", err)
	}
	t.Cleanup(func() { _ = silent.Close() })
	go func() {
		for {
			conn, err := silent.Accept()
			if err != nil {
				return
			}
			t.Cleanup(func() { _ = conn.Close() })
		}
	}()

	p, err := Start(port, Options{Window: time.Minute})
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()

	client := &http.Client{Timeout: 3 * time.Second, Transport: &http.Transport{DisableKeepAlives: true}}
	for i := range 20 {
		resp, err := client.Get(fmt.Sprintf("http://127.0.0.1:%d/", p.Port()))
		if err != nil {
			t.Fatalf("request %d: %v", i, err)
		}
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusOK || string(body) != "foundry" {
			t.Fatalf("request %d: got %d %q", i, resp.StatusCode, body)
		}
	}
}
