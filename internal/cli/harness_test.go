package cli

import (
	"bytes"
	"context"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/sthbryan/ftm/internal/client"
	"github.com/sthbryan/ftm/internal/core"
	"github.com/sthbryan/ftm/internal/events"
	"github.com/sthbryan/ftm/internal/service"
	"github.com/sthbryan/ftm/internal/store"
	"github.com/sthbryan/ftm/internal/transport/httpapi"
)

// fakeSupervisor brings every tunnel online at once, the way a working
// provider would, and remembers its output.
type fakeSupervisor struct {
	bus *events.Bus

	mu     sync.Mutex
	status map[string]core.TunnelStatus
	fail   map[string]string
}

func newFakeSupervisor(bus *events.Bus) *fakeSupervisor {
	return &fakeSupervisor{bus: bus, status: map[string]core.TunnelStatus{}, fail: map[string]string{}}
}

func (f *fakeSupervisor) Start(t core.TunnelConfig) error {
	f.mu.Lock()
	st := t.Status()
	st.State = core.TunnelStateConnecting
	f.status[t.ID] = st
	reason, fails := f.fail[t.Name]
	f.mu.Unlock()
	f.bus.Publish(events.TunnelStatus{Status: st})

	go func() {
		time.Sleep(20 * time.Millisecond)
		f.mu.Lock()
		if fails {
			st.State = core.TunnelStateError
			st.ErrorMessage = reason
		} else {
			st.State = core.TunnelStateOnline
			st.PublicURL = "https://" + t.ID + ".example.test"
		}
		f.status[t.ID] = st
		f.mu.Unlock()
		f.bus.Publish(events.LogLine{TunnelID: t.ID, Line: "tunnel ready"})
		f.bus.Publish(events.TunnelStatus{Status: st})
	}()
	return nil
}

func (f *fakeSupervisor) Stop(id string) error {
	f.mu.Lock()
	st, ok := f.status[id]
	if ok {
		st.State = core.TunnelStateStopped
		st.PublicURL = ""
		f.status[id] = st
	}
	f.mu.Unlock()
	if ok {
		f.bus.Publish(events.TunnelStatus{Status: st})
	}
	return nil
}

func (f *fakeSupervisor) StopAll() {
	f.mu.Lock()
	ids := make([]string, 0, len(f.status))
	for id := range f.status {
		ids = append(ids, id)
	}
	f.mu.Unlock()
	for _, id := range ids {
		_ = f.Stop(id)
	}
}

func (f *fakeSupervisor) CheckInstallation(core.Provider) (bool, bool) { return false, false }
func (f *fakeSupervisor) InstallProvider(core.Provider) error          { return nil }

func (f *fakeSupervisor) GetStatus(id string) (core.TunnelStatus, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	st, ok := f.status[id]
	return st, ok
}

func (f *fakeSupervisor) GetLogs(id string) []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	if st, ok := f.status[id]; ok && st.State == core.TunnelStateOnline {
		return []string{"tunnel ready"}
	}
	return nil
}

// fakeCore is a core run "in this process" for the tests.
type fakeCore struct {
	svc      *service.Service
	bus      *events.Bus
	sup      *fakeSupervisor
	web      *httpapi.Server
	restart  chan struct{}
	onServe  func()
	shutdown int
}

func newFakeCore(t *testing.T) *fakeCore {
	t.Helper()
	root := t.TempDir()
	st := store.New(store.Dirs{Config: filepath.Join(root, "config"), Legacy: filepath.Join(root, "legacy")}, core.DefaultConfig())
	bus := events.New()
	t.Cleanup(bus.Close)
	sup := newFakeSupervisor(bus)
	svc := service.New(st, sup, bus)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	svc.Run(ctx)
	return &fakeCore{svc: svc, bus: bus, sup: sup, restart: make(chan struct{})}
}

func (c *fakeCore) Service() *service.Service { return c.svc }
func (c *fakeCore) Bus() *events.Bus          { return c.bus }
func (c *fakeCore) RunTUI() error             { return nil }
func (c *fakeCore) Restart() <-chan struct{}  { return c.restart }
func (c *fakeCore) Relaunch() error           { return nil }
func (c *fakeCore) Shutdown()                 { c.shutdown++; c.svc.StopAll() }

func (c *fakeCore) ServeWeb(int) (string, error) {
	if c.onServe != nil {
		c.onServe()
	}
	return "http://localhost:0", nil
}

// serve runs the real API over the core, the way a running ftm does, and
// returns a client for it.
func (c *fakeCore) serve(t *testing.T) *client.Client {
	t.Helper()
	c.web = httpapi.NewServer(c.svc, c.bus)
	c.web.SetToken("token")
	if err := c.web.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = c.web.Stop() })
	return client.New(c.web.URL(), "token")
}

// mode is how a command reaches ftm: another process running it, or the core
// started by the command.
type mode struct {
	name     string
	attached bool
}

var modes = []mode{{"standalone", false}, {"attached", true}}

type harness struct {
	t    *testing.T
	core *fakeCore
	deps Deps
	out  *bytes.Buffer
	err  *bytes.Buffer
	// copied is what the last command put on the clipboard.
	copied string
	opened string
}

func newHarness(t *testing.T, m mode) *harness {
	t.Helper()
	h := &harness{t: t, core: newFakeCore(t), out: &bytes.Buffer{}, err: &bytes.Buffer{}}
	dir := t.TempDir()
	h.deps = Deps{
		Dirs: Dirs{Config: dir, ConfigFile: filepath.Join(dir, "config.yaml"), Runtime: dir},
		Peek: func() (core.Config, error) {
			cfg := core.DefaultConfig()
			cfg.Language = "en"
			return *cfg, nil
		},
		Interactive: func() bool { return false },
		OpenURL:     func(u string) error { h.opened = u; return nil },
		Copy:        func(s string) error { h.copied = s; return nil },
		Out:         h.out,
		Err:         h.err,
	}
	if m.attached {
		c := h.core.serve(t)
		h.deps.Attach = func() (*client.Client, error) { return c, nil }
		h.deps.Open = func(bool) (Core, error) {
			t.Fatal("an attached command must not start its own core")
			return nil, nil
		}
	} else {
		h.deps.Attach = func() (*client.Client, error) { return nil, nil }
		h.deps.Open = func(bool) (Core, error) { return h.core, nil }
	}
	return h
}

// run executes one command line and returns its exit code. A command that
// keeps tunnels open ends when it starts serving.
func (h *harness) run(args ...string) int {
	h.t.Helper()
	h.out.Reset()
	h.err.Reset()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	h.core.onServe = cancel
	return Execute(ctx, args, h.deps)
}

func (h *harness) mustRun(args ...string) string {
	h.t.Helper()
	if code := h.run(args...); code != 0 {
		h.t.Fatalf("ftm %v exited %d\nstdout: %s\nstderr: %s", args, code, h.out, h.err)
	}
	return h.out.String()
}
