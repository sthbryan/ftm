package httpapi

import (
	"crypto/subtle"
	"net"
	"net/http"
	"net/url"
	"strings"
)

func isLoopbackHost(host string) bool {
	if host == "localhost" || strings.HasSuffix(host, ".localhost") {
		return true
	}
	if ip := net.ParseIP(host); ip != nil {
		return ip.IsLoopback()
	}
	return false
}

func isWailsHost(host string) bool {
	return host == "wails" || host == "wails.localhost" || strings.HasSuffix(host, ".wails.localhost")
}

func allowedOrigin(r *http.Request) (origin string, ok bool) {
	origin = r.Header.Get("Origin")
	if origin == "" {
		return "", true
	}

	u, err := url.Parse(origin)
	if err != nil {
		return "", false
	}

	host := u.Hostname()
	if isLoopbackHost(host) || isWailsHost(host) {
		return origin, true
	}

	return "", false
}

func allowedHost(r *http.Request) bool {
	host := r.Host
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	host = strings.Trim(host, "[]")

	if host == "" {
		return false
	}

	return isLoopbackHost(host) || isWailsHost(host)
}

// TokenHeader carries the runtime token for clients that are not a browser.
// "Authorization: Bearer <token>" works too.
const TokenHeader = "X-FTM-Token"

func isGuardedPath(path string) bool {
	return strings.HasPrefix(path, "/api/") || path == "/api" || strings.HasPrefix(path, "/ws/")
}

// fromBrowserPage reports whether a request without an Origin header comes
// from a page this server served: browsers mark their own same-origin
// requests (and address-bar navigations) with Sec-Fetch-Site, and leave
// Origin off same-origin GETs such as EventSource.
func fromBrowserPage(r *http.Request) bool {
	switch r.Header.Get("Sec-Fetch-Site") {
	case "same-origin", "none":
		return true
	}
	return false
}

func requestToken(r *http.Request) string {
	if tok := r.Header.Get(TokenHeader); tok != "" {
		return tok
	}
	if auth := r.Header.Get("Authorization"); len(auth) > 7 && strings.EqualFold(auth[:7], "bearer ") {
		return strings.TrimSpace(auth[7:])
	}
	return ""
}

// guard keeps the API to this machine: the Host must be a loopback or Wails
// name (against DNS rebinding), a browser Origin must be local, and a client
// that is not a browser page must present the runtime token.
func (s *Server) guard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !allowedHost(r) {
			writeError(w, http.StatusForbidden, "forbidden_host", "forbidden host")
			return
		}
		if !isGuardedPath(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		origin, ok := allowedOrigin(r)
		if !ok {
			writeError(w, http.StatusForbidden, "forbidden_origin", "forbidden origin")
			return
		}
		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Add("Vary", "Origin")
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Last-Event-ID, "+TokenHeader)

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		if origin == "" && !fromBrowserPage(r) && !s.validToken(requestToken(r)) {
			writeError(w, http.StatusUnauthorized, "unauthorized", "missing or wrong runtime token")
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) validToken(tok string) bool {
	want := s.token()
	if want == "" {
		// No runtime file was published (tests, or the lock is missing), so
		// there is no token a client could have read.
		return true
	}
	return subtle.ConstantTimeCompare([]byte(tok), []byte(want)) == 1
}
