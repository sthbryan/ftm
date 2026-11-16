package service

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/sthbryan/ftm/internal/core"
	"github.com/sthbryan/ftm/internal/events"
	"github.com/sthbryan/ftm/internal/provider"
	"github.com/sthbryan/ftm/internal/supervisor"
)

// ProviderBinary describes the installed binary of one provider.
type ProviderBinary = events.ProviderBinary

// Binary sources, as ProviderBinary.Source reports them.
const (
	SourceBundled = string(supervisor.SourceBundled)
	SourcePath    = string(supervisor.SourcePath)
	SourceSystem  = string(supervisor.SourceSystem)
	SourceNone    = string(supervisor.SourceNone)
)

var (
	// ErrUnknownProvider is returned for a provider id ftm does not know.
	ErrUnknownProvider = supervisor.ErrUnknownProvider
	// ErrNotUpdatable is returned for a provider that runs on a system tool.
	ErrNotUpdatable = supervisor.ErrNotUpdatable
	// ErrExternalBinary is returned for a binary installed outside ftm.
	ErrExternalBinary = supervisor.ErrExternalBinary
	// ErrUpdateInProgress is returned while the same provider is updating.
	ErrUpdateInProgress = supervisor.ErrUpdateInProgress
)

// binaryManager is implemented by a supervisor that can report and replace
// provider binaries.
type binaryManager interface {
	CheckLatest(ctx context.Context, force bool)
	ProviderBinaries(ctx context.Context) []supervisor.ProviderBinary
	ProviderBinary(ctx context.Context, id core.Provider) supervisor.ProviderBinary
	UpdateProvider(ctx context.Context, id core.Provider, progress chan<- provider.DownloadProgress) (*supervisor.UpdateResult, error)
	IsUpdating(id core.Provider) bool
}

func providerBinaryFrom(b supervisor.ProviderBinary) ProviderBinary {
	return ProviderBinary{
		ID:         string(b.ID),
		Name:       b.Name,
		Installed:  b.Installed,
		Version:    b.Version,
		Latest:     b.Latest,
		Source:     string(b.Source),
		Path:       b.Path,
		HasUpdate:  b.HasUpdate,
		Updatable:  b.Updatable,
		Updating:   b.Updating,
		ReleaseURL: b.ReleaseURL,
		CheckedAt:  b.CheckedAt,
		Error:      b.Error,
	}
}

// ProviderBinaries reports every provider binary in config.AllProviders
// order. refresh skips the cache of the latest releases. Providers with a
// release not announced yet are published as events.ProviderUpdatesAvailable.
func (s *Service) ProviderBinaries(ctx context.Context, refresh bool) []ProviderBinary {
	bm, ok := s.sup.(binaryManager)
	if !ok {
		return []ProviderBinary{}
	}

	bm.CheckLatest(ctx, refresh)
	raw := bm.ProviderBinaries(ctx)
	report := make([]ProviderBinary, 0, len(raw))
	for _, b := range raw {
		report = append(report, providerBinaryFrom(b))
	}
	s.announceProviderUpdates(report)
	return report
}

// ProviderBinary reports one provider binary.
func (s *Service) ProviderBinary(ctx context.Context, id core.Provider) (ProviderBinary, error) {
	bm, ok := s.sup.(binaryManager)
	if !ok || !knownProvider(id) {
		return ProviderBinary{}, ErrUnknownProvider
	}
	return providerBinaryFrom(bm.ProviderBinary(ctx, id)), nil
}

// UpdateProvider starts replacing ftm's copy of a provider binary with the
// latest release, or installs it when it is missing. It returns at once:
// progress is published as events.InstallProgress and the outcome once as
// events.ProviderUpdated.
func (s *Service) UpdateProvider(ctx context.Context, id core.Provider) error {
	bm, ok := s.sup.(binaryManager)
	if !ok || !knownProvider(id) {
		return ErrUnknownProvider
	}

	info := bm.ProviderBinary(ctx, id)
	switch {
	case info.Source == supervisor.SourceSystem:
		return ErrNotUpdatable
	case info.Source == supervisor.SourcePath:
		return ErrExternalBinary
	case info.Updating || bm.IsUpdating(id):
		return ErrUpdateInProgress
	}

	go s.runProviderUpdate(bm, id)
	return nil
}

func (s *Service) runProviderUpdate(bm binaryManager, id core.Provider) {
	ctx := s.background()

	result, err := bm.UpdateProvider(ctx, id, nil)
	if errors.Is(err, ErrUpdateInProgress) {
		// Another request won the race and reports the outcome.
		return
	}

	e := events.ProviderUpdated{Provider: string(id), OK: err == nil, Restarted: []string{}}
	if result != nil {
		e.Version = result.Version
		if result.Restarted != nil {
			e.Restarted = result.Restarted
		}
	}
	if err != nil {
		log.Printf("service: updating %s failed: %v", id, err)
		e.Error = err.Error()
	} else {
		s.forgetProviderUpdate(id)
	}
	e.Binary = providerBinaryFrom(bm.ProviderBinary(ctx, id))

	s.publish(e)
}

// announceProviderUpdates publishes the providers whose latest release has
// not been announced yet.
func (s *Service) announceProviderUpdates(report []ProviderBinary) {
	var fresh []ProviderBinary

	s.providerMu.Lock()
	for _, b := range report {
		id := core.Provider(b.ID)
		if !b.HasUpdate {
			delete(s.announced, id)
			continue
		}
		if s.announced[id] == b.Latest {
			continue
		}
		s.announced[id] = b.Latest
		fresh = append(fresh, b)
	}
	s.providerMu.Unlock()

	if len(fresh) == 0 {
		return
	}
	s.publish(events.ProviderUpdatesAvailable{Providers: fresh})
}

// forgetProviderUpdate clears the announcement for a provider after it was
// updated, so a later release is announced again.
func (s *Service) forgetProviderUpdate(id core.Provider) {
	s.providerMu.Lock()
	delete(s.announced, id)
	s.providerMu.Unlock()
}

// runProviderChecks looks for newer provider binaries at start and then on
// the same schedule as the release cache.
func (s *Service) runProviderChecks(ctx context.Context) {
	if _, ok := s.sup.(binaryManager); !ok {
		return
	}

	s.ProviderBinaries(ctx, false)

	t := time.NewTicker(supervisor.LatestCheckInterval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			s.ProviderBinaries(ctx, true)
		}
	}
}

func knownProvider(p core.Provider) bool {
	for _, known := range core.AllProviders() {
		if known == p {
			return true
		}
	}
	return false
}
