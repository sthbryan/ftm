package service

import (
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/sthbryan/ftm/internal/core"
	"github.com/sthbryan/ftm/internal/events"
	"github.com/sthbryan/ftm/internal/store"
	"github.com/sthbryan/ftm/internal/updater"
)

type fakeSupervisor struct {
	mu         sync.Mutex
	installed  map[core.Provider]bool
	installErr error
	installs   int
	release    chan struct{} // when set, InstallProvider waits on it
	started    []core.TunnelConfig
	stopped    []string
	stopAll    int
	startErr   error
}

func newFake() *fakeSupervisor {
	return &fakeSupervisor{installed: map[core.Provider]bool{}}
}

func (f *fakeSupervisor) Start(t core.TunnelConfig) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.startErr != nil {
		return f.startErr
	}
	f.started = append(f.started, t)
	return nil
}

func (f *fakeSupervisor) Stop(id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.stopped = append(f.stopped, id)
	return nil
}

func (f *fakeSupervisor) StopAll() {
	f.mu.Lock()
	f.stopAll++
	f.mu.Unlock()
}

func (f *fakeSupervisor) CheckInstallation(p core.Provider) (bool, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return !f.installed[p], true
}

func (f *fakeSupervisor) InstallProvider(p core.Provider) error {
	f.mu.Lock()
	f.installs++
	release := f.release
	f.mu.Unlock()

	if release != nil {
		<-release
	}

	f.mu.Lock()
	defer f.mu.Unlock()
	if f.installErr != nil {
		return f.installErr
	}
	f.installed[p] = true
	return nil
}

func (f *fakeSupervisor) startedIDs() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	ids := make([]string, len(f.started))
	for i, t := range f.started {
		ids[i] = t.ID
	}
	return ids
}

type fixture struct {
	svc *Service
	sup *fakeSupervisor
	st  *store.Store
	ch  <-chan events.Event
}

func newFixture(t *testing.T, cfg *core.Config) fixture {
	t.Helper()
	if cfg == nil {
		cfg = core.DefaultConfig()
	}

	root := t.TempDir()
	st := store.New(store.Dirs{
		Config: filepath.Join(root, "config"),
		Legacy: filepath.Join(root, "legacy"),
	}, cfg)

	bus := events.New()
	ch, cancel := bus.Subscribe()
	t.Cleanup(func() {
		cancel()
		bus.Close()
	})

	sup := newFake()
	svc := New(st, sup, bus)
	svc.applySettings = nil
	svc.checkUpdate = func() (*updater.Info, error) { return &updater.Info{}, nil }
	svc.applyUpdate = func(*updater.Info) error { return nil }

	return fixture{svc: svc, sup: sup, st: st, ch: ch}
}

func next(t *testing.T, ch <-chan events.Event) events.Event {
	t.Helper()
	select {
	case e := <-ch:
		return e
	case <-time.After(2 * time.Second):
		t.Fatal("no event published")
	}
	return nil
}

func waitFor(t *testing.T, ch <-chan events.Event, match func(events.Event) bool) events.Event {
	t.Helper()
	deadline := time.After(2 * time.Second)
	for {
		select {
		case e := <-ch:
			if match(e) {
				return e
			}
		case <-deadline:
			t.Fatal("expected event never published")
		}
	}
}

func cfgWith(tunnels ...core.TunnelConfig) *core.Config {
	cfg := core.DefaultConfig()
	for _, t := range tunnels {
		cfg.AddTunnel(t)
	}
	return cfg
}

func TestCreateTunnelSavesAndPublishes(t *testing.T) {
	f := newFixture(t, nil)

	tunnel, err := f.svc.CreateTunnel(TunnelInput{Name: "My World", Provider: core.ProviderCloudflared, LocalPort: 30000})
	if err != nil {
		t.Fatalf("CreateTunnel: %v", err)
	}
	if tunnel.ID == "" {
		t.Fatal("no id assigned")
	}

	saved, err := store.Peek(f.st.Dirs())
	if err != nil {
		t.Fatalf("Peek: %v", err)
	}
	if len(saved.Tunnels) != 1 || saved.Tunnels[0].ID != tunnel.ID {
		t.Fatalf("persisted tunnels = %+v", saved.Tunnels)
	}

	e, ok := next(t, f.ch).(events.TunnelSaved)
	if !ok || !e.Created || e.Tunnel.ID != tunnel.ID {
		t.Fatalf("event = %+v, want TunnelSaved{Created} for %s", e, tunnel.ID)
	}
}

func TestCreateTunnelRejectsInvalidInput(t *testing.T) {
	f := newFixture(t, nil)

	for _, in := range []TunnelInput{
		{Provider: core.ProviderBore, LocalPort: 30000},
		{Name: "x", LocalPort: 30000},
		{Name: "x", Provider: core.ProviderBore, LocalPort: 0},
		{Name: "x", Provider: core.ProviderBore, LocalPort: 70000},
	} {
		if _, err := f.svc.CreateTunnel(in); !errors.Is(err, ErrInvalidTunnel) {
			t.Errorf("CreateTunnel(%+v) err = %v, want ErrInvalidTunnel", in, err)
		}
	}
	if n := len(f.st.Tunnels()); n != 0 {
		t.Fatalf("%d tunnels saved, want none", n)
	}
}

func TestUpdateTunnelChangesOnlyPatchedFields(t *testing.T) {
	f := newFixture(t, cfgWith(core.TunnelConfig{ID: "a", Name: "Old", Provider: core.ProviderBore, LocalPort: 30000}))

	name := "New"
	got, err := f.svc.UpdateTunnel("a", TunnelPatch{Name: &name})
	if err != nil {
		t.Fatalf("UpdateTunnel: %v", err)
	}
	if got.Name != "New" || got.Provider != core.ProviderBore || got.LocalPort != 30000 {
		t.Fatalf("got %+v", got)
	}

	e, ok := next(t, f.ch).(events.TunnelSaved)
	if !ok || e.Created || e.Tunnel.Name != "New" {
		t.Fatalf("event = %+v, want TunnelSaved for the edit", e)
	}
}

func TestUpdateTunnelErrors(t *testing.T) {
	f := newFixture(t, cfgWith(core.TunnelConfig{ID: "a", Name: "A", Provider: core.ProviderBore, LocalPort: 30000}))

	name := "x"
	if _, err := f.svc.UpdateTunnel("missing", TunnelPatch{Name: &name}); !errors.Is(err, ErrTunnelNotFound) {
		t.Errorf("missing id err = %v, want ErrTunnelNotFound", err)
	}
	port := 0
	if _, err := f.svc.UpdateTunnel("a", TunnelPatch{LocalPort: &port}); !errors.Is(err, ErrInvalidTunnel) {
		t.Errorf("port 0 err = %v, want ErrInvalidTunnel", err)
	}
}

func TestDeleteTunnelStopsRemovesAndPublishes(t *testing.T) {
	f := newFixture(t, cfgWith(core.TunnelConfig{ID: "a", Name: "A", Provider: core.ProviderBore, LocalPort: 30000}))

	if err := f.svc.DeleteTunnel("a"); err != nil {
		t.Fatalf("DeleteTunnel: %v", err)
	}
	if len(f.sup.stopped) != 1 || f.sup.stopped[0] != "a" {
		t.Errorf("stopped = %v, want [a]", f.sup.stopped)
	}
	if _, ok := f.st.Tunnel("a"); ok {
		t.Error("tunnel still saved")
	}
	if e, ok := next(t, f.ch).(events.TunnelDeleted); !ok || e.ID != "a" {
		t.Fatalf("event = %+v, want TunnelDeleted a", e)
	}
}

func TestClearTunnelsPublishesEachDeletion(t *testing.T) {
	f := newFixture(t, cfgWith(
		core.TunnelConfig{ID: "a", Name: "A", Provider: core.ProviderBore, LocalPort: 30000},
		core.TunnelConfig{ID: "b", Name: "B", Provider: core.ProviderBore, LocalPort: 30001},
	))

	n, err := f.svc.ClearTunnels()
	if err != nil || n != 2 {
		t.Fatalf("ClearTunnels = %d, %v; want 2, nil", n, err)
	}
	got := map[string]bool{}
	for i := 0; i < 2; i++ {
		got[next(t, f.ch).(events.TunnelDeleted).ID] = true
	}
	if !got["a"] || !got["b"] {
		t.Fatalf("deleted events = %v", got)
	}
}

func TestStartTunnelWhenInstalledStartsAtOnce(t *testing.T) {
	f := newFixture(t, cfgWith(core.TunnelConfig{ID: "a", Name: "A", Provider: core.ProviderBore, LocalPort: 30000}))
	f.sup.installed[core.ProviderBore] = true

	installing, err := f.svc.StartTunnel("a")
	if err != nil || installing {
		t.Fatalf("StartTunnel = %v, %v; want started, nil", installing, err)
	}
	if ids := f.sup.startedIDs(); len(ids) != 1 || ids[0] != "a" {
		t.Fatalf("started = %v", ids)
	}
}

func TestStartTunnelUnknownID(t *testing.T) {
	f := newFixture(t, nil)
	if _, err := f.svc.StartTunnel("nope"); !errors.Is(err, ErrTunnelNotFound) {
		t.Fatalf("err = %v, want ErrTunnelNotFound", err)
	}
}

func TestStartTunnelInstallsThenStarts(t *testing.T) {
	f := newFixture(t, cfgWith(core.TunnelConfig{ID: "a", Name: "A", Provider: core.ProviderCloudflared, LocalPort: 30000}))

	installing, err := f.svc.StartTunnel("a")
	if err != nil || !installing {
		t.Fatalf("StartTunnel = %v, %v; want installing, nil", installing, err)
	}

	e := next(t, f.ch).(events.TunnelStatus)
	if e.Status.ID != "a" || e.Status.State != core.TunnelStateInstall {
		t.Fatalf("first event = %+v, want the installing state", e.Status)
	}

	deadline := time.Now().Add(2 * time.Second)
	for len(f.sup.startedIDs()) == 0 {
		if time.Now().After(deadline) {
			t.Fatal("the tunnel was never started after the install")
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func TestConcurrentStartsShareOneInstall(t *testing.T) {
	f := newFixture(t, cfgWith(
		core.TunnelConfig{ID: "a", Name: "A", Provider: core.ProviderCloudflared, LocalPort: 30000},
		core.TunnelConfig{ID: "b", Name: "B", Provider: core.ProviderCloudflared, LocalPort: 30001},
	))
	f.sup.release = make(chan struct{})

	for _, id := range []string{"a", "b"} {
		if installing, err := f.svc.StartTunnel(id); err != nil || !installing {
			t.Fatalf("StartTunnel(%s) = %v, %v", id, installing, err)
		}
	}
	time.Sleep(20 * time.Millisecond)
	close(f.sup.release)

	deadline := time.Now().Add(2 * time.Second)
	for len(f.sup.startedIDs()) < 2 {
		if time.Now().After(deadline) {
			t.Fatalf("started = %v, want both", f.sup.startedIDs())
		}
		time.Sleep(5 * time.Millisecond)
	}
	if f.sup.installs != 1 {
		t.Fatalf("installs = %d, want one shared download", f.sup.installs)
	}
}

func TestStartTunnelPublishesInstallFailure(t *testing.T) {
	f := newFixture(t, cfgWith(core.TunnelConfig{ID: "a", Name: "A", Provider: core.ProviderCloudflared, LocalPort: 30000}))
	f.sup.installErr = errors.New("network down")

	if _, err := f.svc.StartTunnel("a"); err != nil {
		t.Fatalf("StartTunnel: %v", err)
	}

	e := waitFor(t, f.ch, func(e events.Event) bool {
		s, ok := e.(events.TunnelStatus)
		return ok && s.Status.State == core.TunnelStateError
	}).(events.TunnelStatus)
	if e.Status.ID != "a" || e.Status.ErrorMessage == "" {
		t.Fatalf("failure = %+v", e.Status)
	}
	if ids := f.sup.startedIDs(); len(ids) != 0 {
		t.Fatalf("started %v after a failed install", ids)
	}
}

func TestStartTunnelPublishesStartFailureAfterInstall(t *testing.T) {
	f := newFixture(t, cfgWith(core.TunnelConfig{ID: "a", Name: "A", Provider: core.ProviderCloudflared, LocalPort: 30000}))
	f.sup.startErr = errors.New("port in use")

	if _, err := f.svc.StartTunnel("a"); err != nil {
		t.Fatalf("StartTunnel: %v", err)
	}

	e := waitFor(t, f.ch, func(e events.Event) bool {
		s, ok := e.(events.TunnelStatus)
		return ok && s.Status.State == core.TunnelStateError
	}).(events.TunnelStatus)
	if e.Status.ErrorMessage != "port in use" {
		t.Fatalf("message = %q", e.Status.ErrorMessage)
	}
}

func TestStopAndStopAllGoThroughTheSupervisor(t *testing.T) {
	f := newFixture(t, nil)
	_ = f.svc.StopTunnel("a")
	f.svc.StopAll()
	if len(f.sup.stopped) != 1 || f.sup.stopAll != 1 {
		t.Fatalf("stopped = %v, stopAll = %d", f.sup.stopped, f.sup.stopAll)
	}
}

func TestUpdateSettingsSavesAppliesAndPublishes(t *testing.T) {
	f := newFixture(t, nil)
	var applied core.Config
	f.svc.applySettings = func(c core.Config) { applied = c }

	sound := false
	lang := "es"
	cfg, err := f.svc.UpdateSettings(SettingsPatch{NotificationSound: &sound, Language: &lang})
	if err != nil {
		t.Fatalf("UpdateSettings: %v", err)
	}
	if cfg.NotificationSound || cfg.Language != "es" {
		t.Fatalf("cfg = %+v", cfg)
	}
	if applied.Language != "es" {
		t.Error("settings were not applied")
	}
	if e, ok := next(t, f.ch).(events.SettingsChanged); !ok || e.Config.Language != "es" {
		t.Fatalf("event = %+v, want SettingsChanged", e)
	}
}

func TestSetWebPort(t *testing.T) {
	f := newFixture(t, nil)
	if err := f.svc.SetWebPort(40510); err != nil {
		t.Fatalf("SetWebPort: %v", err)
	}
	if got := f.svc.Config().WebPort; got != 40510 {
		t.Fatalf("WebPort = %d", got)
	}
	if err := f.svc.SetWebPort(-1); err == nil {
		t.Fatal("SetWebPort(-1) succeeded")
	}
}

func TestUpdateAvailableIsPublishedOnce(t *testing.T) {
	f := newFixture(t, nil)
	f.svc.checkUpdate = func() (*updater.Info, error) {
		return &updater.Info{HasUpdate: true, LatestVersion: "9.9.9"}, nil
	}

	for i := 0; i < 2; i++ {
		if _, err := f.svc.CheckForUpdate(); err != nil {
			t.Fatalf("CheckForUpdate: %v", err)
		}
	}

	if e, ok := next(t, f.ch).(events.UpdateAvailable); !ok || e.Latest != "9.9.9" {
		t.Fatalf("event = %+v, want UpdateAvailable 9.9.9", e)
	}
	select {
	case e := <-f.ch:
		t.Fatalf("second check published %+v, want nothing", e)
	case <-time.After(50 * time.Millisecond):
	}
	if info := f.svc.UpdateInfo(); info == nil || !info.HasUpdate {
		t.Fatal("UpdateInfo does not remember the check")
	}
}

func TestApplyUpdateWithoutOneIsANoop(t *testing.T) {
	f := newFixture(t, nil)
	called := false
	f.svc.applyUpdate = func(*updater.Info) error { called = true; return nil }

	if err := f.svc.ApplyUpdate(); err != nil || called {
		t.Fatalf("ApplyUpdate = %v, called = %v; want a no-op", err, called)
	}
}
