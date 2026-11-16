// Command fakefoundry stands in for Foundry VTT while you work on ftm, so you
// can try detection, the world name and player counting without installing
// Foundry.
//
//	go run ./tools/fakefoundry                      # Foundry on :30000 with a world launched
//	go run ./tools/fakefoundry -port 30004 -world tomb-of-annihilation
//	go run ./tools/fakefoundry -setup               # open, but no world launched
//	go run ./tools/fakefoundry players <link> -n 4  # four players join a table
//
// The server answers what ftm looks at: a page titled "Foundry Virtual
// Tabletop" (a real page you can open), /api/status with the same fields as
// Foundry 14, and WebSocket upgrades it keeps open. Every open WebSocket is a
// player, which is how a browser in a Foundry world behaves; opening the
// table's public link in a browser or phone joins as one. A small terminal
// view shows the world and who is connected: s toggles the setup screen, w
// switches worlds, q quits.
package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/coder/websocket"
)

//go:embed page.html
var page []byte

var worlds = []struct{ id, system string }{
	{"curse-of-strahd", "dnd5e"},
	{"tomb-of-annihilation", "dnd5e"},
	{"the-enemy-within", "wfrp4e"},
	{"masks-of-nyarlathotep", "CoC7"},
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "players" {
		players(os.Args[2:])
		return
	}
	serve(os.Args[1:])
}

// ── server ──────────────────────────────────────────────────────────────

type player struct {
	id     int
	from   string
	tunnel bool
	since  time.Time
}

type table struct {
	mu      sync.Mutex
	version string
	world   int
	setup   bool
	started time.Time
	players map[int]player
	next    int
	log     []string
	notify  func()
}

func (t *table) event(format string, args ...any) {
	t.log = append(t.log, time.Now().Format("15:04:05")+"  "+fmt.Sprintf(format, args...))
	if len(t.log) > 8 {
		t.log = t.log[len(t.log)-8:]
	}
}

func (t *table) changed() {
	if t.notify != nil {
		t.notify()
	}
}

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
		"users":         len(t.players) + 1, // the GM, at this computer
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
	t.players[id] = player{id: id, from: from, tunnel: tunnel, since: time.Now()}
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

func serve(args []string) {
	flags := flag.NewFlagSet("fakefoundry", flag.ExitOnError)
	port := flags.Int("port", 30000, "port to listen on, like Foundry's own")
	world := flags.String("world", worlds[0].id, "world id to report")
	system := flags.String("system", "", "game system to report (default: the world's)")
	version := flags.String("version", "14.368", "Foundry version to report")
	setup := flags.Bool("setup", false, "start on the setup screen, with no world launched")
	plain := flags.Bool("plain", false, "print events instead of the terminal view")
	_ = flags.Parse(args)

	t := &table{version: *version, setup: *setup, started: time.Now(), players: map[int]player{}}
	t.world = 0
	for i, w := range worlds {
		if w.id == *world {
			t.world = i
		}
	}
	if t.world == 0 && *world != worlds[0].id {
		worlds = append([]struct{ id, system string }{{*world, "dnd5e"}}, worlds...)
	}
	if *system != "" {
		worlds[t.world].system = *system
	}

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		fmt.Fprintf(os.Stderr, "fakefoundry: port %d: %v\n", *port, err)
		os.Exit(1)
	}
	server := &http.Server{Handler: t.handler(), ReadHeaderTimeout: 10 * time.Second}
	go func() { _ = server.Serve(listener) }()
	defer server.Close()

	if *plain {
		t.notify = func() {
			t.mu.Lock()
			fmt.Println(t.log[len(t.log)-1])
			t.mu.Unlock()
		}
		fmt.Printf("fake Foundry v%s on http://localhost:%d\n", t.version, *port)
		stop := make(chan os.Signal, 1)
		signal.Notify(stop, os.Interrupt)
		<-stop
		return
	}

	program := tea.NewProgram(&view{t: t, port: *port})
	t.notify = func() { program.Send(refresh{}) }
	if _, err := program.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// ── terminal view ───────────────────────────────────────────────────────

type refresh struct{}
type tick struct{}

type view struct {
	t    *table
	port int
}

func (v *view) Init() tea.Cmd { return every() }

func every() tea.Cmd {
	return tea.Tick(time.Second, func(time.Time) tea.Msg { return tick{} })
}

func (v *view) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return v, tea.Quit
		case "s":
			v.t.mu.Lock()
			v.t.setup = !v.t.setup
			if v.t.setup {
				v.t.event("closed the world: setup screen")
			} else {
				v.t.event("launched %s", worlds[v.t.world].id)
			}
			v.t.mu.Unlock()
		case "w":
			v.t.mu.Lock()
			v.t.world = (v.t.world + 1) % len(worlds)
			v.t.event("switched to %s", worlds[v.t.world].id)
			v.t.mu.Unlock()
		}
	case tick:
		return v, every()
	}
	return v, nil
}

var (
	gold  = lipgloss.NewStyle().Foreground(lipgloss.Color("#d4a84a")).Bold(true)
	muted = lipgloss.NewStyle().Foreground(lipgloss.Color("#8a8494"))
	ok    = lipgloss.NewStyle().Foreground(lipgloss.Color("#6fd3a0"))
	warn  = lipgloss.NewStyle().Foreground(lipgloss.Color("#f0bc5e"))
)

func (v *view) View() tea.View {
	t := v.t
	t.mu.Lock()
	defer t.mu.Unlock()

	var b strings.Builder
	fmt.Fprintf(&b, "\n  %s %s\n\n", gold.Render("fake Foundry"), muted.Render(fmt.Sprintf("v%s · localhost:%d", t.version, v.port)))
	if t.setup {
		fmt.Fprintf(&b, "  %s no world launched\n", warn.Render("●"))
	} else {
		w := worlds[t.world]
		fmt.Fprintf(&b, "  %s %s %s   %d at the table\n", ok.Render("●"), w.id, muted.Render("("+w.system+")"), len(t.players))
	}
	for id := 1; id <= t.next; id++ {
		p, here := t.players[id]
		if !here {
			continue
		}
		via := muted.Render("direct")
		if p.tunnel {
			via = ok.Render("via tunnel")
		}
		fmt.Fprintf(&b, "    player %d  %s  %s\n", p.id, p.from, via)
	}
	fmt.Fprintf(&b, "\n  %s\n", muted.Render("s setup · w world · q quit"))
	return tea.NewView(b.String())
}

// ── players ─────────────────────────────────────────────────────────────

func players(args []string) {
	flags := flag.NewFlagSet("players", flag.ExitOnError)
	count := flags.Int("n", 3, "how many players join")
	gap := flags.Duration("every", 3*time.Second, "time between joins, and between leaves")
	leave := flags.Duration("leave", 0, "start leaving after this long (default: stay until Ctrl+C)")
	flags.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: go run ./tools/fakefoundry players <link> [-n 3] [-every 3s] [-leave 20s]")
		flags.PrintDefaults()
	}
	var link string
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		link, args = args[0], args[1:]
	}
	_ = flags.Parse(args)
	if link == "" && flags.NArg() > 0 {
		link = flags.Arg(0)
	}
	if link == "" {
		flags.Usage()
		os.Exit(2)
	}

	target := strings.TrimRight(link, "/")
	target = strings.Replace(target, "https://", "wss://", 1)
	target = strings.Replace(target, "http://", "ws://", 1)
	if !strings.HasPrefix(target, "ws") {
		target = "wss://" + target
	}
	target += "/socket.io/?EIO=4&transport=websocket"

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	var wg sync.WaitGroup
	cancels := []context.CancelFunc{}
	for i := 1; i <= *count; i++ {
		playerCtx, cancel := context.WithCancel(ctx)
		cancels = append(cancels, cancel)
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			join(playerCtx, n, target)
		}(i)
		if !sleep(ctx, *gap) {
			break
		}
	}

	if *leave > 0 && sleep(ctx, *leave) {
		for i := len(cancels) - 1; i >= 0; i-- {
			cancels[i]()
			if !sleep(ctx, *gap) {
				break
			}
		}
	} else if *leave == 0 {
		fmt.Println("everyone is at the table; Ctrl+C to leave")
		<-ctx.Done()
	}
	for _, cancel := range cancels {
		cancel()
	}
	wg.Wait()
}

func join(ctx context.Context, n int, target string) {
	dial, cancel := context.WithTimeout(ctx, 15*time.Second)
	conn, _, err := websocket.Dial(dial, target, nil)
	cancel()
	if err != nil {
		fmt.Printf("player %d could not join: %v\n", n, err)
		return
	}
	defer conn.CloseNow()
	fmt.Printf("player %d joined\n", n)

	// A ping every 20 s keeps proxies (Cloudflare closes idle sockets) from
	// dropping the connection.
	ticker := time.NewTicker(20 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			_ = conn.Close(websocket.StatusNormalClosure, "left the table")
			fmt.Printf("player %d left\n", n)
			return
		case <-ticker.C:
			pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			err := conn.Ping(pingCtx)
			cancel()
			if err != nil && ctx.Err() == nil {
				fmt.Printf("player %d was disconnected: %v\n", n, err)
				return
			}
		}
	}
}

func sleep(ctx context.Context, d time.Duration) bool {
	select {
	case <-ctx.Done():
		return false
	case <-time.After(d):
		return true
	}
}
