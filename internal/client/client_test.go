package client

import (
	"context"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/sthbryan/ftm/internal/core"
	"github.com/sthbryan/ftm/internal/events"
	"github.com/sthbryan/ftm/internal/service"
	"github.com/sthbryan/ftm/internal/store"
	"github.com/sthbryan/ftm/internal/transport/httpapi"
)

type fakeSupervisor struct {
	mu      sync.Mutex
	started []string
}

func (f *fakeSupervisor) Start(t core.TunnelConfig) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.started = append(f.started, t.ID)
	return nil
}
func (f *fakeSupervisor) Stop(string) error                            { return nil }
func (f *fakeSupervisor) StopAll()                                     {}
func (f *fakeSupervisor) CheckInstallation(core.Provider) (bool, bool) { return false, false }
func (f *fakeSupervisor) InstallProvider(core.Provider) error          { return nil }

const testToken = "secret"

// newInstance runs the real API on a loopback port, the way a running ftm
// serves it.
func newInstance(t *testing.T) (*Client, *fakeSupervisor) {
	t.Helper()
	root := t.TempDir()
	st := store.New(store.Dirs{Config: filepath.Join(root, "config"), Legacy: filepath.Join(root, "legacy")}, core.DefaultConfig())
	bus := events.New()
	t.Cleanup(bus.Close)
	sup := &fakeSupervisor{}
	svc := service.New(st, sup, bus)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	svc.Run(ctx)

	api := httpapi.NewServer(svc, bus)
	api.SetToken(testToken)
	if err := api.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = api.Stop() })
	return New(api.URL(), testToken), sup
}

func TestTunnelLifecycleOverTheAPI(t *testing.T) {
	c, sup := newInstance(t)
	ctx := context.Background()

	created, err := c.CreateTunnel(ctx, TunnelInput{Name: "Friday game", Provider: "cloudflared", LocalPort: 30000})
	if err != nil {
		t.Fatalf("CreateTunnel: %v", err)
	}
	if created.ID == "" || created.Name != "Friday game" || created.Port != 30000 || created.State != "stopped" {
		t.Fatalf("created = %+v", created)
	}

	edited, err := c.UpdateTunnel(ctx, created.ID, TunnelInput{LocalPort: 30001})
	if err != nil {
		t.Fatalf("UpdateTunnel: %v", err)
	}
	if edited.Port != 30001 || edited.Name != "Friday game" {
		t.Fatalf("edited = %+v", edited)
	}

	list, err := c.Tunnels(ctx)
	if err != nil || len(list) != 1 || list[0].ID != created.ID {
		t.Fatalf("Tunnels = %+v, %v", list, err)
	}

	if _, err := c.StartTunnel(ctx, created.ID); err != nil {
		t.Fatalf("StartTunnel: %v", err)
	}
	if len(sup.started) != 1 || sup.started[0] != created.ID {
		t.Fatalf("started = %v", sup.started)
	}
	if err := c.StopTunnel(ctx, created.ID); err != nil {
		t.Fatalf("StopTunnel: %v", err)
	}
	if _, err := c.Logs(ctx, created.ID); err != nil {
		t.Fatalf("Logs: %v", err)
	}

	if err := c.DeleteTunnel(ctx, created.ID); err != nil {
		t.Fatalf("DeleteTunnel: %v", err)
	}
	if _, err := c.Tunnel(ctx, created.ID); !IsNotFound(err) {
		t.Fatalf("Tunnel after delete: err = %v, want not found", err)
	}
}

func TestWrongTokenIsRefused(t *testing.T) {
	c, _ := newInstance(t)
	c.token = "nope"
	_, err := c.Tunnels(context.Background())
	apiErr, ok := err.(*Error)
	if !ok || apiErr.Status != http.StatusUnauthorized {
		t.Fatalf("err = %v, want 401", err)
	}
}

func TestSettingsRoundTrip(t *testing.T) {
	c, _ := newInstance(t)
	ctx := context.Background()
	off := false
	got, err := c.UpdateSettings(ctx, SettingsPatch{NotificationSound: &off})
	if err != nil {
		t.Fatalf("UpdateSettings: %v", err)
	}
	if got.NotificationSound {
		t.Fatal("NotificationSound still on")
	}
	again, err := c.Settings(ctx)
	if err != nil || again.NotificationSound {
		t.Fatalf("Settings = %+v, %v", again, err)
	}
}

func TestEventsDeliverChanges(t *testing.T) {
	c, _ := newInstance(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ready := make(chan struct{})
	got := make(chan Event, 1)
	go func() {
		_ = c.Events(ctx, func(e Event) bool {
			if e.Name == EventOpen {
				close(ready)
			}
			if e.Name == "tunnel_state" {
				got <- e
				return false
			}
			return true
		})
	}()
	<-ready
	tunnel, err := c.CreateTunnel(ctx, TunnelInput{Name: "x", Provider: "bore", LocalPort: 30000})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.StartTunnel(ctx, tunnel.ID); err != nil {
		t.Fatal(err)
	}

	select {
	case e := <-got:
		var state Tunnel
		if err := e.Decode(&state); err != nil || state.ID != tunnel.ID {
			t.Fatalf("event = %s, %v", e.Data, err)
		}
	case <-ctx.Done():
		t.Fatal("no tunnel_state event")
	}
}

func TestReadEventsParsesTheStream(t *testing.T) {
	stream := "retry: 3000\n\n: ping\n\nid: 4\nevent: log\ndata: {\"line\":\"a\"}\n\nevent: log_snapshot\ndata: {\"lines\":\n data: []}\n\n"
	var got []Event
	err := readEvents(strings.NewReader(stream), func(e Event) bool {
		got = append(got, e)
		return true
	})
	if err == nil {
		t.Fatal("a stream that ends must report it")
	}
	if len(got) != 2 || got[0].Name != "log" || string(got[0].Data) != `{"line":"a"}` || got[1].Name != "log_snapshot" {
		t.Fatalf("got %+v", got)
	}
}

func TestReadEventsStopsWhenAsked(t *testing.T) {
	stream := "event: a\ndata: 1\n\nevent: b\ndata: 2\n\n"
	n := 0
	err := readEvents(strings.NewReader(stream), func(Event) bool { n++; return false })
	if err != nil || n != 1 {
		t.Fatalf("n = %d, err = %v", n, err)
	}
}
