package web

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/sthbryan/ftm/internal/config"
	"github.com/sthbryan/ftm/internal/process"
)

// providerUpdateService rechecks the provider binaries on the same schedule as
// the app update check and tells the dashboard when one falls behind.
type providerUpdateService struct {
	manager   *process.Manager
	broadcast func(any)

	mu       sync.Mutex
	notified map[config.Provider]string
}

func newProviderUpdateService(manager *process.Manager, broadcast func(any)) *providerUpdateService {
	return &providerUpdateService{
		manager:   manager,
		broadcast: broadcast,
		notified:  map[config.Provider]string{},
	}
}

func (s *providerUpdateService) Start(ctx context.Context) {
	go func() {
		s.Check(ctx, false)

		t := time.NewTicker(process.LatestCheckInterval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				s.Check(ctx, true)
			}
		}
	}()
}

// Check refreshes the latest releases and broadcasts every provider that has
// an update it has not announced yet. It returns the full report.
func (s *providerUpdateService) Check(ctx context.Context, force bool) []process.ProviderBinary {
	s.manager.CheckLatest(ctx, force)
	report := s.manager.ProviderBinaries(ctx)
	s.announce(report)
	return report
}

func (s *providerUpdateService) announce(report []process.ProviderBinary) {
	var fresh []providerUpdateNotice

	s.mu.Lock()
	for _, b := range report {
		if !b.HasUpdate {
			delete(s.notified, b.ID)
			continue
		}
		if s.notified[b.ID] == b.Latest {
			continue
		}
		s.notified[b.ID] = b.Latest
		fresh = append(fresh, providerUpdateNotice{
			ID:         string(b.ID),
			Name:       b.Name,
			Version:    b.Version,
			Latest:     b.Latest,
			Source:     string(b.Source),
			Updatable:  b.Updatable,
			ReleaseURL: b.ReleaseURL,
		})
	}
	s.mu.Unlock()

	if len(fresh) == 0 {
		return
	}

	log.Printf("web: provider updates available for %d provider(s)", len(fresh))
	s.broadcast(providerUpdateAvailableMessage{
		Type:      "provider_update_available",
		Providers: fresh,
	})
}

// forget clears the announcement for a provider after it was updated, so a
// later release is announced again.
func (s *providerUpdateService) forget(id config.Provider) {
	s.mu.Lock()
	delete(s.notified, id)
	s.mu.Unlock()
}
