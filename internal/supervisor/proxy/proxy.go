package proxy

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"
)

type Stats struct {
	ActiveSessions int
	Visitors       int
	Requests       int64
}

type Options struct {
	Window       time.Duration
	OnNewVisitor func()
}

type Proxy struct {
	listener net.Listener
	server   *http.Server
	counter  *counter
	port     int
	onNew    func()
}

func Start(targetPort int, opts Options) (*Proxy, error) {
	target, err := url.Parse(fmt.Sprintf("http://127.0.0.1:%d", targetPort))
	if err != nil {
		return nil, err
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("failed to listen: %w", err)
	}

	c := newCounter(opts.Window)
	reverse := httputil.NewSingleHostReverseProxy(target)
	reverse.ErrorHandler = func(w http.ResponseWriter, _ *http.Request, _ error) {
		w.WriteHeader(http.StatusBadGateway)
	}

	p := &Proxy{
		listener: listener,
		counter:  c,
		port:     listener.Addr().(*net.TCPAddr).Port,
		onNew:    opts.OnNewVisitor,
	}
	p.server = &http.Server{
		Handler:           p.handler(reverse),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() { _ = p.server.Serve(listener) }()

	return p, nil
}

func (p *Proxy) handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if p.counter.seen(clientKey(r)) && p.onNew != nil {
			go p.onNew()
		}

		if isUpgrade(r) {
			p.counter.openSession()
			defer p.counter.closeSession()
		}

		next.ServeHTTP(w, r)
	})
}

func (p *Proxy) Port() int {
	return p.port
}

func (p *Proxy) Stats() Stats {
	return p.counter.stats()
}

func (p *Proxy) Close() error {
	return p.server.Close()
}

func isUpgrade(r *http.Request) bool {
	return strings.EqualFold(r.Header.Get("Upgrade"), "websocket")
}

func clientKey(r *http.Request) string {
	for _, header := range []string{"Cf-Connecting-Ip", "X-Real-Ip", "X-Forwarded-For"} {
		if value := r.Header.Get(header); value != "" {
			if first, _, ok := strings.Cut(value, ","); ok {
				return strings.TrimSpace(first)
			}
			return strings.TrimSpace(value)
		}
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}

	return host
}
