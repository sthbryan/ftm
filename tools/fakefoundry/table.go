package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
)

//go:embed page.html
var page []byte

// worlds the w key steps through: id as Foundry reports it, and its system.
var worlds = []struct{ id, system string }{
	{"curse-of-strahd", "dnd5e"},
	{"tomb-of-annihilation", "dnd5e"},
	{"lost-mine-of-phandelver", "dnd5e"},
	{"abomination-vaults", "pf2e"},
	{"kingmaker", "pf2e"},
	{"masks-of-nyarlathotep", "CoC7"},
	{"the-enemy-within", "wfrp4e"},
	{"the-one-ring", "tor2e"},
	{"blades-in-the-dark", "blades-in-the-dark"},
	{"alien-chariot-of-the-gods", "alienrpg"},
}

type player struct {
	id     int
	from   string
	tunnel bool
}

// table is the fake Foundry: which world runs, who is connected, and the
// players this program itself sent through a link (bots).
type table struct {
	mu      sync.Mutex
	version string
	port    int
	world   int
	setup   bool
	started time.Time
	players map[int]player
	next    int
	bots    []context.CancelFunc
	log     []string
	notify  func()
}

func newTable(version, world string, setup bool, port int) *table {
	t := &table{version: version, port: port, setup: setup, started: time.Now(), players: map[int]player{}}
	found := false
	for i, w := range worlds {
		if w.id == world {
			t.world, found = i, true
		}
	}
	if !found {
		worlds = append([]struct{ id, system string }{{world, "dnd5e"}}, worlds...)
	}
	return t
}

func (t *table) event(format string, args ...any) {
	t.log = append(t.log, time.Now().Format("15:04:05")+"  "+fmt.Sprintf(format, args...))
	if len(t.log) > 4 {
		t.log = t.log[len(t.log)-4:]
	}
}

func (t *table) changed() {
	if t.notify != nil {
		t.notify()
	}
}

// status is Foundry's /api/status. users counts everyone connected, the GM
// (at this computer) included, as Foundry does.
func (t *table) status() map[string]any {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.setup {
		return map[string]any{"active": false, "version": t.version}
	}
	w := worlds[t.world]
	return map[string]any{
		"active":        true,
		"version":       t.version,
		"world":         w.id,
		"system":        w.system,
		"systemVersion": "4.1.2",
		"users":         len(t.players) + 1,
		"uptime":        time.Since(t.started).Milliseconds(),
	}
}

func (t *table) handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.EqualFold(r.Header.Get("Upgrade"), "websocket") {
			t.hold(w, r)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/api/status") {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(t.status())
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(page)
	})
}

// hold accepts a player's WebSocket and keeps it until they leave.
func (t *table) hold(w http.ResponseWriter, r *http.Request) {
	t.mu.Lock()
	setup := t.setup
	t.mu.Unlock()
	if setup {
		http.Error(w, "no world launched", http.StatusServiceUnavailable)
		return
	}
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
	if err != nil {
		return
	}
	defer conn.CloseNow()

	from, tunnel := source(r)
	t.mu.Lock()
	t.next++
	id := t.next
	t.players[id] = player{id: id, from: from, tunnel: tunnel}
	t.event("player %d joined from %s", id, from)
	t.mu.Unlock()
	t.changed()

	for {
		if _, _, err := conn.Read(r.Context()); err != nil {
			break
		}
	}

	t.mu.Lock()
	delete(t.players, id)
	t.event("player %d left", id)
	t.mu.Unlock()
	t.changed()
}

// addBot sends one more player, through the link when there is one (so ftm's
// proxy sees and counts them) or straight to this server otherwise.
func (t *table) addBot(link string) {
	target := wsTarget(link)
	if target == "" {
		target = fmt.Sprintf("ws://127.0.0.1:%d/socket.io/?EIO=4&transport=websocket", t.port)
	}
	ctx, cancel := context.WithCancel(context.Background())
	t.mu.Lock()
	t.bots = append(t.bots, cancel)
	n := len(t.bots)
	t.mu.Unlock()
	go join(ctx, n, target, func(format string, args ...any) {
		t.mu.Lock()
		t.event(format, args...)
		t.mu.Unlock()
		t.changed()
	})
}

// dropBot sends the last bot home.
func (t *table) dropBot() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if len(t.bots) == 0 {
		return
	}
	t.bots[len(t.bots)-1]()
	t.bots = t.bots[:len(t.bots)-1]
}

func (t *table) dropBots() {
	t.mu.Lock()
	defer t.mu.Unlock()
	for _, cancel := range t.bots {
		cancel()
	}
	t.bots = nil
}

// source says where a player came from, and whether through a tunnel: ftm's
// proxy passes on the provider's client-address headers.
func source(r *http.Request) (string, bool) {
	for _, header := range []string{"Cf-Connecting-Ip", "X-Real-Ip", "X-Forwarded-For"} {
		if value := r.Header.Get(header); value != "" {
			first, _, _ := strings.Cut(value, ",")
			return strings.TrimSpace(first), true
		}
	}
	host, _, _ := net.SplitHostPort(r.RemoteAddr)
	return host, false
}
