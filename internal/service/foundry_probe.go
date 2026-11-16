package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/sthbryan/ftm/internal/supervisor/proxy"
)

// foundryProbeTimeout bounds one look at a busy port. Foundry answers its own
// pages in a few milliseconds on the same machine; anything slower is not it.
const foundryProbeTimeout = 600 * time.Millisecond

// foundryMarker is in the <title> of every page Foundry VTT serves before a
// world loads (setup, license, join) and in the game page itself.
var foundryMarker = []byte("Foundry Virtual Tabletop")

// foundryClient reaches Foundry on either loopback, the way the tunnel does.
var foundryClient = &http.Client{
	Timeout:   foundryProbeTimeout,
	Transport: &http.Transport{DialContext: proxy.DialLoopbackAddr, DisableKeepAlives: true},
}

// FoundryStatus is what a running Foundry says about itself on /api/status:
// whether a world is launched, which one, its game system, how many users are
// connected and how long the world has been up. Older builds or anything that
// only matched the page title leave the details empty.
type FoundryStatus struct {
	Port          int    `json:"port"`
	Active        bool   `json:"active"`
	Version       string `json:"version,omitempty"`
	World         string `json:"world,omitempty"`
	System        string `json:"system,omitempty"`
	SystemVersion string `json:"systemVersion,omitempty"`
	Users         int    `json:"users"`
	UptimeMs      int64  `json:"uptimeMs,omitempty"`
}

// FoundryOn reports which of the given ports have Foundry VTT behind them, and
// what each one says about itself. The ports are probed at the same time.
func (s *Service) FoundryOn(ctx context.Context, ports []int) []FoundryStatus {
	var (
		mu    sync.Mutex
		found = []FoundryStatus{}
		wg    sync.WaitGroup
	)
	for _, port := range ports {
		wg.Add(1)
		go func(port int) {
			defer wg.Done()
			if status, ok := probeFoundry(ctx, port); ok {
				mu.Lock()
				found = append(found, status)
				mu.Unlock()
			}
		}(port)
	}
	wg.Wait()
	sort.Slice(found, func(i, j int) bool { return found[i].Port < found[j].Port })
	return found
}

// statusBody is Foundry's /api/status answer. It needs no login and also
// answers on the setup screen, with active false and no world.
type statusBody struct {
	Active        bool    `json:"active"`
	Version       string  `json:"version"`
	World         string  `json:"world"`
	System        string  `json:"system"`
	SystemVersion string  `json:"systemVersion"`
	Users         int     `json:"users"`
	Uptime        float64 `json:"uptime"`
}

func probeFoundry(ctx context.Context, port int) (FoundryStatus, bool) {
	if body, ok := get(ctx, port, "/api/status"); ok {
		var status statusBody
		if json.Unmarshal(body, &status) == nil && status.Version != "" {
			return FoundryStatus{
				Port:          port,
				Active:        status.Active,
				Version:       status.Version,
				World:         status.World,
				System:        status.System,
				SystemVersion: status.SystemVersion,
				Users:         status.Users,
				UptimeMs:      int64(status.Uptime),
			}, true
		}
	}
	if body, ok := get(ctx, port, "/"); ok && bytes.Contains(body, foundryMarker) {
		return FoundryStatus{Port: port}, true
	}
	return FoundryStatus{}, false
}

func get(ctx context.Context, port int, path string) ([]byte, bool) {
	ctx, cancel := context.WithTimeout(ctx, foundryProbeTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("http://127.0.0.1:%d%s", port, path), nil)
	if err != nil {
		return nil, false
	}
	resp, err := foundryClient.Do(req)
	if err != nil {
		return nil, false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, false
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	if err != nil && len(body) == 0 {
		return nil, false
	}
	return body, true
}
