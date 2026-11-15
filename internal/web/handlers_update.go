package web

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/sthbryan/ftm/internal/updater"
	"github.com/sthbryan/ftm/internal/version"
)

func (h *Handlers) handleUpdate(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.getUpdate(w)
	case http.MethodPost:
		h.postUpdate(w)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handlers) getUpdate(w http.ResponseWriter) {
	resp := map[string]interface{}{
		"current":    version.Version,
		"latest":     "",
		"tag":        "",
		"assetName":  "",
		"releaseUrl": "",
		"hasUpdate":  false,
		"method":     string(updater.DetectMethod()),
	}
	if info := h.svc.UpdateInfo(); info != nil {
		resp["latest"] = info.LatestVersion
		resp["tag"] = info.Tag
		resp["assetName"] = info.AssetName
		resp["releaseUrl"] = info.ReleaseURL
		resp["hasUpdate"] = info.HasUpdate
		resp["method"] = string(info.Method)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *Handlers) postUpdate(w http.ResponseWriter) {
	info := h.svc.UpdateInfo()
	if info == nil || !info.HasUpdate {
		http.Error(w, "no update available", http.StatusBadRequest)
		return
	}
	if err := h.svc.ApplyUpdate(); err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, updater.ErrNotSelfUpdatable) {
			status = http.StatusConflict
		}
		http.Error(w, err.Error(), status)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"ok": true})
	go func() {
		time.Sleep(500 * time.Millisecond)
		h.svc.StopAll()

		if err := updater.Relaunch(); err != nil {
			log.Printf("relaunch after update failed: %v", err)
		}
		os.Exit(0)
	}()
}
