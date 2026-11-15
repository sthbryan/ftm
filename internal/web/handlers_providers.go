package web

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/sthbryan/ftm/internal/config"
	"github.com/sthbryan/ftm/internal/process"
)

// providerBinaryMessage is one row of the "Installed" section.
type providerBinaryMessage struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Installed  bool   `json:"installed"`
	Version    string `json:"version"`
	Latest     string `json:"latest"`
	Source     string `json:"source"`
	Path       string `json:"path"`
	HasUpdate  bool   `json:"hasUpdate"`
	Updatable  bool   `json:"updatable"`
	Updating   bool   `json:"updating"`
	ReleaseURL string `json:"releaseUrl"`
	CheckedAt  int64  `json:"checkedAt"`
	Error      string `json:"error,omitempty"`
}

type providerBinariesResponse struct {
	Providers []providerBinaryMessage `json:"providers"`
}

type providerUpdateNotice struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Version    string `json:"version"`
	Latest     string `json:"latest"`
	Source     string `json:"source"`
	Updatable  bool   `json:"updatable"`
	ReleaseURL string `json:"releaseUrl"`
}

type providerUpdateAvailableMessage struct {
	Type      string                 `json:"type"`
	Providers []providerUpdateNotice `json:"providers"`
}

type providerUpdatedMessage struct {
	Type      string                 `json:"type"`
	Provider  string                 `json:"provider"`
	OK        bool                   `json:"ok"`
	Version   string                 `json:"version,omitempty"`
	Error     string                 `json:"error,omitempty"`
	Restarted []string               `json:"restarted"`
	Binary    *providerBinaryMessage `json:"binary,omitempty"`
}

func providerBinaryFrom(b process.ProviderBinary) providerBinaryMessage {
	msg := providerBinaryMessage{
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
		Error:      b.Error,
	}
	if !b.CheckedAt.IsZero() {
		msg.CheckedAt = b.CheckedAt.UnixMilli()
	}
	return msg
}

// handleInstalledProviders serves GET /api/providers/installed. ?refresh=1
// skips the 6-hour cache of the latest releases.
func (h *Handlers) handleInstalledProviders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	force := r.URL.Query().Get("refresh") == "1" || r.URL.Query().Get("refresh") == "true"
	report := h.server.providerSvc.Check(r.Context(), force)

	resp := providerBinariesResponse{Providers: make([]providerBinaryMessage, 0, len(report))}
	for _, b := range report {
		resp.Providers = append(resp.Providers, providerBinaryFrom(b))
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// handleProviderActions serves POST /api/providers/{id}/update. The update
// runs in the background: progress arrives as "install" events and the result
// as a "provider_updated" event.
func (h *Handlers) handleProviderActions(w http.ResponseWriter, r *http.Request, path string) {
	rest := strings.TrimPrefix(path, "/api/providers/")
	id, action, ok := strings.Cut(rest, "/")
	if !ok || action != "update" || id == "" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	provider := config.Provider(id)
	if !knownProvider(provider) {
		http.Error(w, "unknown provider", http.StatusNotFound)
		return
	}

	info := h.manager.ProviderBinary(r.Context(), provider)
	switch {
	case info.Source == process.SourceSystem:
		http.Error(w, process.ErrNotUpdatable.Error(), http.StatusBadRequest)
		return
	case info.Source == process.SourcePath:
		http.Error(w, process.ErrExternalBinary.Error(), http.StatusConflict)
		return
	case info.Updating || h.manager.IsUpdating(provider):
		http.Error(w, process.ErrUpdateInProgress.Error(), http.StatusConflict)
		return
	}

	go h.runProviderUpdate(provider)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"ok":       true,
		"provider": id,
		"state":    "updating",
	})
}

func (h *Handlers) runProviderUpdate(provider config.Provider) {
	ctx := h.server.updateCtx

	result, err := h.manager.UpdateProvider(ctx, provider, nil)
	if errors.Is(err, process.ErrUpdateInProgress) {
		// Another request won the race and will report the outcome.
		return
	}

	msg := providerUpdatedMessage{
		Type:      "provider_updated",
		Provider:  string(provider),
		OK:        err == nil,
		Restarted: []string{},
	}
	if result != nil {
		msg.Version = result.Version
		if result.Restarted != nil {
			msg.Restarted = result.Restarted
		}
	}
	if err != nil {
		log.Printf("web: updating %s failed: %v", provider, err)
		msg.Error = err.Error()
	} else {
		h.server.providerSvc.forget(provider)
	}

	binary := providerBinaryFrom(h.manager.ProviderBinary(ctx, provider))
	msg.Binary = &binary

	h.server.broadcastJSON(msg)
}

func knownProvider(p config.Provider) bool {
	for _, known := range config.AllProviders() {
		if known == p {
			return true
		}
	}
	return false
}
