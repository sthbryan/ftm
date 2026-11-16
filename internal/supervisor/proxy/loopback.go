package proxy

import (
	"context"
	"errors"
	"net"
	"strconv"
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
