package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/sthbryan/ftm/internal/core"
	"github.com/sthbryan/ftm/internal/events"
	"github.com/sthbryan/ftm/internal/provider"
	"github.com/sthbryan/ftm/internal/supervisor"
	"github.com/sthbryan/ftm/internal/updater"
)

// The server starts right after Run; a slow or hanging GitHub must not hold
// it up.
func TestRunDoesNotWaitForTheUpdateCheck(t *testing.T) {
	f := newFixture(t, nil)
	answer := make(chan struct{})
	defer close(answer)
	f.svc.checkUpdate = func() (*updater.Info, error) {
		<-answer
		return &updater.Info{}, nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	returned := make(chan struct{})
	go func() {
		f.svc.Run(ctx)
		close(returned)
	}()
	select {
	case <-returned:
	case <-time.After(time.Second):
		t.Fatal("Run waited for the update check")
	}
}

func TestRequestRestartClosesTheChannelOnce(t *testing.T) {
	f := newFixture(t, nil)
	select {
	case <-f.svc.RestartRequested():
		t.Fatal("restart requested before anyone asked")
	default:
	}

	f.svc.RequestRestart()
	f.svc.RequestRestart()

	select {
	case <-f.svc.RestartRequested():
	default:
		t.Fatal("RestartRequested not closed")
	}
}

type statusFake struct {
	*fakeSupervisor
	status map[string]core.TunnelStatus
	logs   map[string][]string
}

func (s statusFake) GetStatus(id string) (core.TunnelStatus, bool) {
	st, ok := s.status[id]
	return st, ok
}

func (s statusFake) GetLogs(id string) []string { return s.logs[id] }

func TestTunnelViewsMergeLiveStatus(t *testing.T) {
	f := newFixture(t, cfgWith(
		core.TunnelConfig{ID: "a", Name: "A", Provider: core.ProviderBore, LocalPort: 30000},
		core.TunnelConfig{ID: "b", Name: "B", Provider: core.ProviderBore, LocalPort: 30001},
	))
	sup := statusFake{
		fakeSupervisor: f.sup,
		status:         map[string]core.TunnelStatus{"a": {ID: "a", State: core.TunnelStateOnline}},
		logs:           map[string][]string{"a": {"line"}},
	}
	f.svc.sup = sup

	views := f.svc.TunnelViews()
	if len(views) != 2 || !views[0].Running || views[0].Status.State != core.TunnelStateOnline || views[0].NeedsInstall {
		t.Fatalf("views[0] = %+v", views[0])
	}
	if views[1].Running || !views[1].NeedsInstall {
		t.Fatalf("views[1] = %+v, want a stopped tunnel that needs its provider", views[1])
	}
	if _, ok := f.svc.TunnelView("nope"); ok {
		t.Fatal("TunnelView found an unknown id")
	}
	if got := f.svc.Logs("a"); len(got) != 1 {
		t.Fatalf("Logs = %v", got)
	}
	if got := f.svc.Logs("b"); got == nil || len(got) != 0 {
		t.Fatalf("Logs = %#v, want an empty slice", got)
	}
}

type binaryFake struct {
	*fakeSupervisor
	report   []supervisor.ProviderBinary
	updated  chan core.Provider
	updating bool
	err      error
}

func (b *binaryFake) CheckLatest(context.Context, bool) {}
func (b *binaryFake) ProviderBinaries(context.Context) []supervisor.ProviderBinary {
	return b.report
}
func (b *binaryFake) ProviderBinary(_ context.Context, id core.Provider) supervisor.ProviderBinary {
	for _, r := range b.report {
		if r.ID == id {
			return r
		}
	}
	return supervisor.ProviderBinary{ID: id, Source: supervisor.SourceNone, Updatable: true}
}
func (b *binaryFake) UpdateProvider(_ context.Context, id core.Provider, _ chan<- provider.DownloadProgress) (*supervisor.UpdateResult, error) {
	b.updated <- id
	return &supervisor.UpdateResult{Provider: id, Version: "2"}, b.err
}
func (b *binaryFake) IsUpdating(core.Provider) bool { return b.updating }

func TestProviderUpdateNoticeIsPublishedOncePerVersion(t *testing.T) {
	f := newFixture(t, nil)
	sup := &binaryFake{fakeSupervisor: f.sup, report: []supervisor.ProviderBinary{
		{ID: core.ProviderCloudflared, Version: "2024.6.1", Latest: "2026.11.0", HasUpdate: true, Updatable: true, Source: supervisor.SourceBundled},
		{ID: core.ProviderBore, Version: "0.6.0", Latest: "0.6.0", Source: supervisor.SourceBundled},
	}}
	f.svc.sup = sup
	ctx := context.Background()

	f.svc.ProviderBinaries(ctx, false)
	f.svc.ProviderBinaries(ctx, false)
	e, ok := next(t, f.ch).(events.ProviderUpdatesAvailable)
	if !ok || len(e.Providers) != 1 || e.Providers[0].ID != "cloudflared" {
		t.Fatalf("event = %+v, want one notice for cloudflared", e)
	}
	select {
	case e := <-f.ch:
		t.Fatalf("second check published %+v", e)
	case <-time.After(50 * time.Millisecond):
	}

	sup.report[0].Latest = "2026.12.0"
	f.svc.ProviderBinaries(ctx, true)
	if e, ok := next(t, f.ch).(events.ProviderUpdatesAvailable); !ok || e.Providers[0].Latest != "2026.12.0" {
		t.Fatalf("a newer release was not announced: %+v", e)
	}
}

func TestUpdateProviderChecksThenReportsTheOutcome(t *testing.T) {
	f := newFixture(t, nil)
	sup := &binaryFake{fakeSupervisor: f.sup, updated: make(chan core.Provider, 1), report: []supervisor.ProviderBinary{
		{ID: core.ProviderServeo, Source: supervisor.SourceSystem},
		{ID: core.ProviderBore, Source: supervisor.SourcePath},
	}}
	f.svc.sup = sup
	ctx := context.Background()

	for id, want := range map[core.Provider]error{
		"nope":              ErrUnknownProvider,
		core.ProviderServeo: ErrNotUpdatable,
		core.ProviderBore:   ErrExternalBinary,
	} {
		if err := f.svc.UpdateProvider(ctx, id); !errors.Is(err, want) {
			t.Errorf("UpdateProvider(%s) = %v, want %v", id, err, want)
		}
	}

	sup.updating = true
	if err := f.svc.UpdateProvider(ctx, core.ProviderCloudflared); !errors.Is(err, ErrUpdateInProgress) {
		t.Fatalf("UpdateProvider while updating = %v", err)
	}
	sup.updating = false

	sup.err = errors.New("checksum mismatch")
	if err := f.svc.UpdateProvider(ctx, core.ProviderCloudflared); err != nil {
		t.Fatalf("UpdateProvider = %v", err)
	}
	<-sup.updated
	e, ok := next(t, f.ch).(events.ProviderUpdated)
	if !ok || e.OK || e.Error != "checksum mismatch" || e.Provider != "cloudflared" || e.Restarted == nil {
		t.Fatalf("event = %+v, want a failed ProviderUpdated", e)
	}
}

// Without a supervisor that manages binaries there is nothing to report.
func TestProviderBinariesWithoutSupport(t *testing.T) {
	f := newFixture(t, nil)
	if got := f.svc.ProviderBinaries(context.Background(), false); got == nil || len(got) != 0 {
		t.Fatalf("ProviderBinaries = %#v, want empty", got)
	}
	if err := f.svc.UpdateProvider(context.Background(), core.ProviderCloudflared); !errors.Is(err, ErrUnknownProvider) {
		t.Fatalf("UpdateProvider = %v", err)
	}
}
