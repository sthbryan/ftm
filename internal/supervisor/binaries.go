package supervisor

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os/exec"
	"sync"
	"time"

	"github.com/sthbryan/ftm/internal/core"
	"github.com/sthbryan/ftm/internal/provider"
)

// BinarySource says where the binary a provider runs comes from.
type BinarySource string

const (
	// SourceBundled is a copy ftm downloaded into its own bin directory.
	SourceBundled BinarySource = "bundled"
	// SourcePath is a copy the user installed (on PATH or a well-known
	// location). ftm reports its version but never replaces it.
	SourcePath BinarySource = "path"
	// SourceSystem is a tool the OS provides, such as ssh.
	SourceSystem BinarySource = "system"
	// SourceNone means the provider is not installed yet.
	SourceNone BinarySource = "none"
)

// LatestCheckInterval is how long a latest-release lookup stays fresh.
const LatestCheckInterval = 6 * time.Hour

// failedCheckRetry is how long a failed lookup waits before it is retried, so a
// rate-limited GitHub API is not hammered on every request.
const failedCheckRetry = 15 * time.Minute

var (
	ErrNotUpdatable     = errors.New("provider binary is not managed by ftm")
	ErrExternalBinary   = errors.New("provider binary was installed outside ftm; update it with the tool that installed it")
	ErrUpdateInProgress = errors.New("an update for this provider is already running")
	ErrUnknownProvider  = errors.New("unknown provider")
)

// ProviderBinary describes the installed binary of one provider.
type ProviderBinary struct {
	ID         core.Provider
	Name       string
	Installed  bool
	Version    string
	Latest     string
	Source     BinarySource
	Path       string
	HasUpdate  bool
	Updatable  bool
	Updating   bool
	ReleaseURL string
	CheckedAt  time.Time
	Error      string
}

type latestEntry struct {
	release   *provider.Release
	err       error
	checkedAt time.Time
}

func (e latestEntry) fresh(now time.Time) bool {
	if e.checkedAt.IsZero() {
		return false
	}
	ttl := LatestCheckInterval
	if e.err != nil {
		ttl = failedCheckRetry
	}
	return now.Sub(e.checkedAt) < ttl
}

type binaryState struct {
	mu       sync.Mutex
	latest   map[core.Provider]latestEntry
	updating map[core.Provider]bool
}

func newBinaryState() *binaryState {
	return &binaryState{
		latest:   map[core.Provider]latestEntry{},
		updating: map[core.Provider]bool{},
	}
}

func (m *Manager) updatable(id core.Provider) (provider.Updatable, bool) {
	p, ok := m.providers[id]
	if !ok {
		return nil, false
	}
	u, ok := p.(provider.Updatable)
	return u, ok
}

// CheckLatest refreshes the latest-release lookup of every downloadable
// provider whose cached answer is stale, or all of them when force is set.
func (m *Manager) CheckLatest(ctx context.Context, force bool) {
	now := time.Now()

	var wg sync.WaitGroup
	for _, id := range core.AllProviders() {
		u, ok := m.updatable(id)
		if !ok {
			continue
		}

		m.binaries.mu.Lock()
		entry := m.binaries.latest[id]
		m.binaries.mu.Unlock()
		if !force && entry.fresh(now) {
			continue
		}

		wg.Add(1)
		go func(id core.Provider, u provider.Updatable) {
			defer wg.Done()
			rel, err := u.LatestRelease(ctx)
			if err != nil {
				log.Printf("providers: latest %s lookup failed: %v", id, err)
			}

			m.binaries.mu.Lock()
			prev := m.binaries.latest[id]
			next := latestEntry{release: rel, err: err, checkedAt: time.Now()}
			// Keep the last good answer through a transient failure.
			if err != nil && prev.release != nil {
				next.release = prev.release
			}
			m.binaries.latest[id] = next
			m.binaries.mu.Unlock()
		}(id, u)
	}
	wg.Wait()
}

// ProviderBinaries reports every provider's installed binary against the
// cached latest release. It looks up releases only when the cache is stale.
func (m *Manager) ProviderBinaries(ctx context.Context) []ProviderBinary {
	m.CheckLatest(ctx, false)

	all := core.AllProviders()
	out := make([]ProviderBinary, len(all))

	var wg sync.WaitGroup
	for i, id := range all {
		wg.Add(1)
		go func(i int, id core.Provider) {
			defer wg.Done()
			out[i] = m.ProviderBinary(ctx, id)
		}(i, id)
	}
	wg.Wait()

	return out
}

// ProviderBinary reports one provider using the cached latest release.
func (m *Manager) ProviderBinary(ctx context.Context, id core.Provider) ProviderBinary {
	info := ProviderBinary{ID: id, Name: string(id), Source: SourceNone}

	p, ok := m.providers[id]
	if !ok {
		info.Error = ErrUnknownProvider.Error()
		return info
	}
	info.Name = p.Name()

	u, ok := p.(provider.Updatable)
	if !ok {
		// The SSH providers run the system ssh client.
		info.Source = SourceSystem
		if path, err := exec.LookPath("ssh"); err == nil {
			info.Installed = true
			info.Path = path
		}
		return info
	}

	m.binaries.mu.Lock()
	entry := m.binaries.latest[id]
	info.Updating = m.binaries.updating[id]
	m.binaries.mu.Unlock()

	info.CheckedAt = entry.checkedAt
	if entry.err != nil {
		info.Error = entry.err.Error()
	}

	if path := u.FindBinary(); path != "" {
		info.Installed = true
		info.Path = path
		info.Source = SourcePath
		if provider.IsBundled(path) {
			info.Source = SourceBundled
		}

		version, err := provider.DetectVersion(ctx, path)
		if err != nil && info.Source == SourceBundled {
			if rec, ok := provider.LookupInstall(string(id)); ok && rec.Version != "" && rec.Matches(path) {
				version, err = rec.Version, nil
			}
		}
		if err != nil && info.Error == "" {
			info.Error = err.Error()
		}
		info.Version = version
	}

	info.Updatable = info.Source == SourceBundled || info.Source == SourceNone

	if rel := entry.release; rel != nil {
		info.ReleaseURL = rel.ReleaseURL
		info.Latest, info.HasUpdate = compareRelease(string(id), info, rel)
	}

	return info
}

// compareRelease decides whether rel is newer than what is installed and what
// to show as the latest version.
func compareRelease(name string, info ProviderBinary, rel *provider.Release) (latest string, hasUpdate bool) {
	if !info.Installed {
		return rel.Version, false
	}

	// An unversioned download (tunnelmole) is new when its ETag changed since
	// ftm installed it. Then the copy on disk is the latest one there is.
	if rel.ETag != "" && info.Source == SourceBundled {
		if rec, ok := provider.LookupInstall(name); ok && rec.ETag != "" {
			if rec.ETag == rel.ETag {
				if info.Version != "" {
					return info.Version, false
				}
				return rel.Version, false
			}
			return rel.Version, true
		}
	}

	// A bundled binary whose version cannot be read (wrong architecture, a
	// truncated download) is worth replacing; one the user installed is not
	// ftm's to judge.
	if info.Version == "" {
		return rel.Version, info.Source == SourceBundled && rel.Version != ""
	}

	return rel.Version, provider.IsNewer(rel.Version, info.Version)
}

// UpdateResult is what an update did.
type UpdateResult struct {
	Provider  core.Provider
	Version   string
	Restarted []string
}

type runningTunnel struct {
	config core.TunnelConfig
}

// UpdateProvider downloads the latest release of a provider and swaps it in
// for ftm's bundled copy. Tunnels running on that provider are stopped first,
// since Windows will not replace a running executable, and restarted after.
// A binary installed outside ftm is never touched.
func (m *Manager) UpdateProvider(ctx context.Context, id core.Provider, progress chan<- provider.DownloadProgress) (*UpdateResult, error) {
	if _, ok := m.providers[id]; !ok {
		return nil, ErrUnknownProvider
	}
	u, ok := m.updatable(id)
	if !ok {
		return nil, ErrNotUpdatable
	}

	if path := u.FindBinary(); path != "" && !provider.IsBundled(path) {
		return nil, ErrExternalBinary
	}

	m.binaries.mu.Lock()
	if m.binaries.updating[id] {
		m.binaries.mu.Unlock()
		return nil, ErrUpdateInProgress
	}
	m.binaries.updating[id] = true
	m.binaries.mu.Unlock()

	var finish func(error)
	if progress == nil {
		progress, finish = m.progressSink(string(id))
	}

	rel, stopped, installErr := m.replaceBinary(ctx, id, u, progress)
	if finish != nil {
		finish(installErr)
	}

	// Start refuses while the flag is set, so clear it before restarting.
	m.binaries.mu.Lock()
	delete(m.binaries.updating, id)
	m.binaries.mu.Unlock()

	result := &UpdateResult{Provider: id}
	for _, t := range stopped {
		if err := m.Start(t.config); err != nil {
			log.Printf("providers: could not restart %s after updating %s: %v", t.config.Name, id, err)
			continue
		}
		result.Restarted = append(result.Restarted, t.config.ID)
	}

	if installErr != nil {
		return result, installErr
	}

	if version, err := provider.DetectVersion(ctx, u.BundledBinary()); err == nil {
		result.Version = version
	} else {
		result.Version = rel.Version
	}
	return result, nil
}

// replaceBinary looks up the release, stops the provider's tunnels and installs
// the new binary. It returns the tunnels it stopped so the caller restarts
// them whether or not the install worked: a failed swap leaves the old binary.
func (m *Manager) replaceBinary(ctx context.Context, id core.Provider, u provider.Updatable, progress chan<- provider.DownloadProgress) (*provider.Release, []runningTunnel, error) {
	rel, err := u.LatestRelease(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("look up the latest %s: %w", id, err)
	}

	m.binaries.mu.Lock()
	m.binaries.latest[id] = latestEntry{release: rel, checkedAt: time.Now()}
	m.binaries.mu.Unlock()

	stopped := m.stopProviderTunnels(id)
	return rel, stopped, u.InstallRelease(ctx, rel, progress)
}

// IsUpdating reports whether a provider's binary is being replaced.
func (m *Manager) IsUpdating(id core.Provider) bool {
	m.binaries.mu.Lock()
	defer m.binaries.mu.Unlock()
	return m.binaries.updating[id]
}

func (m *Manager) stopProviderTunnels(id core.Provider) []runningTunnel {
	m.mu.RLock()
	var running []runningTunnel
	for _, mp := range m.processes {
		if mp.Config.Provider == id && mp.Process != nil {
			running = append(running, runningTunnel{config: mp.Config})
		}
	}
	m.mu.RUnlock()

	for _, t := range running {
		if err := m.Stop(t.config.ID); err != nil {
			log.Printf("providers: could not stop %s before updating %s: %v", t.config.Name, id, err)
		}
	}
	return running
}
