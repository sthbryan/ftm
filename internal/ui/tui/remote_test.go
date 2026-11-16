package tui

import (
	"context"
	"net/http"
	"path/filepath"
	"sync"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/sthbryan/ftm/internal/client"
	"github.com/sthbryan/ftm/internal/core"
	"github.com/sthbryan/ftm/internal/events"
	"github.com/sthbryan/ftm/internal/service"
	"github.com/sthbryan/ftm/internal/store"
	"github.com/sthbryan/ftm/internal/transport/httpapi"
)

// realtime is how soon a change on one surface must show on the other.
const realtime = time.Second

type fakeSupervisor struct {
	mu      sync.Mutex
	started []string
	stopped []string
}

func (f *fakeSupervisor) Start(t core.TunnelConfig) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.started = append(f.started, t.ID)
	return nil
}

func (f *fakeSupervisor) Stop(id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.stopped = append(f.stopped, id)
	return nil
}

func (f *fakeSupervisor) StopAll()                                     {}
func (f *fakeSupervisor) CheckInstallation(core.Provider) (bool, bool) { return false, false }
func (f *fakeSupervisor) InstallProvider(core.Provider) error          { return nil }

func (f *fakeSupervisor) calls() (started, stopped []string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.started...), append([]string(nil), f.stopped...)
}

type instance struct {
	svc *service.Service
	bus *events.Bus
	sup *fakeSupervisor
	api *httpapi.Server
}

// runInstance runs the real API on a loopback port, the way ftm-desktop or
// `ftm serve` serves it.
func runInstance(t *testing.T) *instance {
	t.Helper()
	root := t.TempDir()
	st := store.New(store.Dirs{
		Config: filepath.Join(root, "config"),
		Legacy: filepath.Join(root, "legacy"),
	}, core.DefaultConfig())
	bus := events.New()
	t.Cleanup(bus.Close)
	sup := &fakeSupervisor{}
	svc := service.New(st, sup, bus)

	api := httpapi.NewServer(svc, bus)
	api.SetToken("secret")
	if err := api.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = api.Stop() })
	return &instance{svc: svc, bus: bus, sup: sup, api: api}
}

func (in *instance) client() *client.Client { return client.New(in.api.URL(), "secret") }

// attachedModel is the TUI attached to in, with its event stream open.
func attachedModel(t *testing.T, in *instance) *Model {
	t.Helper()
	m := NewModel(NewRemote(in.client(), t.TempDir(), nil))
	t.Cleanup(func() {
		m.Close()
		// A keep-alive connection left by an ended stream would hold up the
		// server's shutdown.
		http.DefaultTransport.(*http.Transport).CloseIdleConnections()
	})
	m.Width, m.Height = 100, 30
	waitFor(t, m, "the event stream opens", func() bool { return true })
	return m
}

// waitFor feeds the model its events until cond holds, failing after
// realtime. The model has no timer of its own: only events move it.
func waitFor(t *testing.T, m *Model, what string, cond func() bool) {
	t.Helper()
	deadline := time.After(realtime)
	next := make(chan tea.Msg, 1)
	for {
		go func() { next <- m.waitForEvent()() }()
		select {
		case msg := <-next:
			if msg != nil {
				m.Update(msg)
			}
			if cond() {
				return
			}
		case <-deadline:
			t.Fatalf("%s: not within %v", what, realtime)
		}
	}
}

// webEvents follows /api/events the way an open web view does.
func webEvents(t *testing.T, in *instance) <-chan client.Event {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	ch := make(chan client.Event, 64)
	opened := make(chan struct{})
	go func() {
		_ = in.client().Events(ctx, func(e client.Event) bool {
			if e.Name == client.EventOpen {
				close(opened)
				return true
			}
			select {
			case ch <- e:
			default:
			}
			return true
		})
	}()
	select {
	case <-opened:
	case <-time.After(realtime):
		t.Fatal("the web view's event stream did not open")
	}
	return ch
}

func expectWebEvent(t *testing.T, ch <-chan client.Event, name string, match func(client.Event) bool) {
	t.Helper()
	deadline := time.After(realtime)
	for {
		select {
		case e := <-ch:
			if e.Name == name && match(e) {
				return
			}
		case <-deadline:
			t.Fatalf("the web view got no %s event within %v", name, realtime)
		}
	}
}

func tunnelID(id string) func(client.Event) bool {
	return func(e client.Event) bool {
		var t client.Tunnel
		return e.Decode(&t) == nil && t.ID == id
	}
}

func TestAttachedTUIShowsWebChangesAsTheyHappen(t *testing.T) {
	in := runInstance(t)
	m := attachedModel(t, in)

	created, err := in.svc.CreateTunnel(service.TunnelInput{Name: "Ravenloft", Provider: core.ProviderBore, LocalPort: 30000})
	if err != nil {
		t.Fatal(err)
	}
	waitFor(t, m, "a connection added in the web view", func() bool { return len(m.Items) == 1 })

	name := "Barovia"
	if _, err := in.svc.UpdateTunnel(created.ID, service.TunnelPatch{Name: &name}); err != nil {
		t.Fatal(err)
	}
	waitFor(t, m, "a rename in the web view", func() bool {
		item, ok := m.selectedItem()
		return ok && item.Tunnel.Name == "Barovia"
	})

	in.bus.Publish(events.TunnelStatus{Status: core.TunnelStatus{
		ID: created.ID, Name: name, State: core.TunnelStateOnline, PublicURL: "https://barovia.example",
	}})
	waitFor(t, m, "a connection going online", func() bool {
		item, ok := m.selectedItem()
		return ok && item.Status.State == core.TunnelStateOnline
	})

	if err := in.svc.DeleteTunnel(created.ID); err != nil {
		t.Fatal(err)
	}
	waitFor(t, m, "a connection removed in the web view", func() bool { return len(m.Items) == 0 })
}

func TestAttachedTUIChangesReachTheWebView(t *testing.T) {
	in := runInstance(t)
	web := webEvents(t, in)
	m := attachedModel(t, in)

	m.Draft = TunnelDraft{Name: "Ravenloft", Provider: string(core.ProviderBore), Port: "30000"}
	m.createTunnel()
	tunnels := in.svc.Tunnels()
	if len(tunnels) != 1 {
		t.Fatalf("the running ftm holds %d connections, want the one added in the TUI", len(tunnels))
	}
	id := tunnels[0].ID
	expectWebEvent(t, web, "tunnel_state", tunnelID(id))

	m.State = viewEditTunnel
	m.editingTunnelID = id
	m.Draft = TunnelDraft{Name: "Barovia", Provider: string(core.ProviderBore), Port: "30001"}
	m.updateTunnel()
	if got, _ := in.svc.Tunnel(id); got.Name != "Barovia" || got.LocalPort != 30001 {
		t.Fatalf("the running ftm has %+v after the edit", got)
	}
	expectWebEvent(t, web, "tunnel_state", func(e client.Event) bool {
		var t client.Tunnel
		return e.Decode(&t) == nil && t.ID == id && t.Name == "Barovia"
	})

	m.refreshItems()
	item, _ := m.selectedItem()
	m.startTunnel(item)()
	if started, _ := in.sup.calls(); len(started) != 1 || started[0] != id {
		t.Fatalf("started = %v, want the connection started from the TUI", started)
	}
	m.stopTunnel(item)()
	if _, stopped := in.sup.calls(); len(stopped) == 0 || stopped[len(stopped)-1] != id {
		t.Fatalf("stopped = %v, want the connection stopped from the TUI", stopped)
	}

	m.pendingDeleteID = id
	m.confirmDelete()
	if len(in.svc.Tunnels()) != 0 {
		t.Fatal("the connection deleted in the TUI is still saved")
	}
	expectWebEvent(t, web, "tunnel_deleted", func(e client.Event) bool {
		var t struct {
			ID string `json:"id"`
		}
		return e.Decode(&t) == nil && t.ID == id
	})
}

func TestAttachedTUISavesSettingsInTheRunningFtm(t *testing.T) {
	in := runInstance(t)
	m := attachedModel(t, in)

	m.openSettings()
	m.SettingsView.NotificationSound = !m.prefs.Sound
	want := m.SettingsView.NotificationSound
	m.saveSettings()

	if got := in.svc.Config().NotificationSound; got != want {
		t.Fatalf("NotificationSound = %v in the running ftm, want %v", got, want)
	}
}

func TestAttachedTUIFollowsTheSelectedConnectionsOutput(t *testing.T) {
	in := runInstance(t)
	if _, err := in.svc.CreateTunnel(service.TunnelInput{Name: "Ravenloft", Provider: core.ProviderBore, LocalPort: 30000}); err != nil {
		t.Fatal(err)
	}
	m := attachedModel(t, in)
	id := in.svc.Tunnels()[0].ID
	if m.logsID != id {
		t.Fatalf("logsID = %q, want the selected connection %q", m.logsID, id)
	}

	// The log stream subscribes asynchronously; publish until a line lands.
	deadline := time.Now().Add(realtime)
	for len(m.logs) == 0 && time.Now().Before(deadline) {
		in.bus.Publish(events.LogLine{TunnelID: id, Line: "connected"})
		next := make(chan tea.Msg, 1)
		go func() { next <- m.waitForEvent()() }()
		select {
		case msg := <-next:
			m.Update(msg)
		case <-time.After(50 * time.Millisecond):
		}
	}
	if len(m.logs) == 0 || m.logs[len(m.logs)-1] != "connected" {
		t.Fatalf("logs = %v, want the line the running ftm printed", m.logs)
	}
}

func TestAttachedTUISaysWhenTheRunningFtmGoesAway(t *testing.T) {
	in := runInstance(t)
	m := attachedModel(t, in)

	http.DefaultTransport.(*http.Transport).CloseIdleConnections()
	go func() { _ = in.api.Stop() }()
	waitFor(t, m, "the running ftm closing", func() bool { return m.lost })
}

func TestMergeLogsDropsLinesReadTwice(t *testing.T) {
	got := mergeLogs([]string{"a", "b", "c"}, []string{"b", "c", "d"})
	want := []string{"a", "b", "c", "d"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}
