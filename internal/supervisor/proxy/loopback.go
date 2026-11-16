package proxy

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// loopbacks are the addresses a local server may listen on. Foundry on Windows
// can end up on ::1 alone, so IPv4 by itself is not enough.
var loopbacks = []string{"127.0.0.1", "::1"}

// DialLoopback connects to port on this machine through whichever loopback
// answers. Both are tried at once: on Windows a refused loopback connection
// takes about two seconds, so trying them in turn would stall on the wrong one.
func DialLoopback(ctx context.Context, port int) (net.Conn, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	type result struct {
		conn net.Conn
		err  error
	}
	results := make(chan result, len(loopbacks))
	var dialer net.Dialer
	for _, host := range loopbacks {
		go func(host string) {
			conn, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(host, strconv.Itoa(port)))
			results <- result{conn, err}
		}(host)
	}

	var errs []error
	for range loopbacks {
		r := <-results
		if r.err == nil {
			// The other attempt is cancelled; close it too if it also won.
			go func(left int) {
				for ; left > 0; left-- {
					if extra := <-results; extra.conn != nil {
						_ = extra.conn.Close()
					}
				}
			}(len(loopbacks) - 1 - len(errs))
			return r.conn, nil
		}
		errs = append(errs, r.err)
	}
	return nil, errors.Join(errs...)
}

// DialLoopbackAddr is DialLoopback for an http.Transport: it keeps the port of
// addr and ignores its host.
func DialLoopbackAddr(ctx context.Context, _, addr string) (net.Conn, error) {
	_, portText, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}
	port, err := strconv.Atoi(portText)
	if err != nil {
		return nil, err
	}
	return DialLoopback(ctx, port)
}

// upstreamRecheck is how long the proxy trusts the loopback it picked before
// asking again, so a Foundry restarted on the other address is found.
const upstreamRecheck = 30 * time.Second

// upstream is where the proxy sends traffic: the loopback on which an HTTP
// server really answers. Something else can share Foundry's port on one of
// them (a Windows port forward) and accept connections without answering, so
// the first address to accept is not enough.
type upstream struct {
	port int

	mu      sync.Mutex
	host    string
	checked time.Time
}

func (u *upstream) dial(ctx context.Context, _, _ string) (net.Conn, error) {
	if host := u.pick(ctx); host != "" {
		var dialer net.Dialer
		conn, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(host, strconv.Itoa(u.port)))
		if err == nil {
			return conn, nil
		}
		u.forget()
	}
	return DialLoopback(ctx, u.port)
}

func (u *upstream) pick(ctx context.Context) string {
	u.mu.Lock()
	defer u.mu.Unlock()
	if u.host != "" && time.Since(u.checked) < upstreamRecheck {
		return u.host
	}
	u.host = answeringLoopback(ctx, u.port)
	u.checked = time.Now()
	return u.host
}

func (u *upstream) forget() {
	u.mu.Lock()
	u.host = ""
	u.mu.Unlock()
}

// answeringLoopback asks both loopbacks for a page at once and returns the
// first that answers with any HTTP status, or "" when neither does in time.
func answeringLoopback(ctx context.Context, port int) string {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	client := &http.Client{Transport: &http.Transport{Proxy: nil, DisableKeepAlives: true}}

	answers := make(chan string, len(loopbacks))
	for _, host := range loopbacks {
		go func(host string) {
			url := "http://" + net.JoinHostPort(host, strconv.Itoa(port)) + "/api/status"
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
			if err != nil {
				answers <- ""
				return
			}
			resp, err := client.Do(req)
			if err != nil {
				answers <- ""
				return
			}
			_ = resp.Body.Close()
			answers <- host
		}(host)
	}
	for range loopbacks {
		if host := <-answers; host != "" {
			return host
		}
	}
	return ""
}
