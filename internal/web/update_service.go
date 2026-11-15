package web

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/sthbryan/ftm/internal/updater"
	"github.com/sthbryan/ftm/internal/version"
)

const (
	updateRepo          = "sthbryan/ftm"
	updateCheckInterval = 6 * time.Hour
)

type updateService struct {
	mu        sync.RWMutex
	info      *updater.Info
	broadcast func(string)
	repo      string
	current   string
}

func newUpdateService(broadcast func(string)) *updateService {
	return &updateService{
		broadcast: broadcast,
		repo:      updateRepo,
		current:   version.Version,
	}
}

func (s *updateService) Start(ctx context.Context) {
	if err := s.Check(); err != nil {
		log.Printf("update check failed: %v", err)
	}
	go s.loop(ctx)
}

func (s *updateService) loop(ctx context.Context) {
	t := time.NewTicker(updateCheckInterval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if err := s.Check(); err != nil {
				log.Printf("update check failed: %v", err)
			}
		}
	}
}

func (s *updateService) Check() error {
	info, err := updater.New(s.repo).Check(s.current)
	if err != nil {
		return err
	}
	s.mu.Lock()
	hadUpdate := s.info != nil && s.info.HasUpdate
	s.info = info
	s.mu.Unlock()
	if info.HasUpdate && !hadUpdate {
		s.broadcastUpdate(info)
	}
	return nil
}

func (s *updateService) Info() *updater.Info {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.info
}

func (s *updateService) Apply() error {
	s.mu.RLock()
	info := s.info
	s.mu.RUnlock()
	if info == nil || !info.HasUpdate {
		return nil
	}
	return updater.New(s.repo).Apply(info)
}

func (s *updateService) broadcastUpdate(info *updater.Info) {
	payload := map[string]interface{}{
		"type":       "update_available",
		"current":    s.current,
		"latest":     info.LatestVersion,
		"tag":        info.Tag,
		"assetName":  info.AssetName,
		"releaseUrl": info.ReleaseURL,
		"method":     string(info.Method),
	}
	data, err := json.Marshal(payload)
	if err != nil {
		log.Printf("web: could not encode the update notification: %v", err)
		return
	}
	s.broadcast(string(data))
}
