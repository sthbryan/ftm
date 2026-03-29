package web

import (
	"fmt"
	"net/http"
	"strings"
)

func (h *Handlers) handleLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	id := h.extractLogID(r.URL.Path)
	if id == "" {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}

	logs := h.manager.GetLogs(id)
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(strings.Join(logs, "\n")))
}

func (h *Handlers) handleLogsStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/logs/")
	path = strings.TrimSuffix(path, "/stream")
	id := strings.Split(path, "/")[0]
	if id == "" {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	for _, line := range h.manager.GetLogs(id) {
		fmt.Fprintf(w, "data: %s\n\n", line)
		flusher.Flush()
	}

	logChan, unsubscribe := h.manager.SubscribeLogs(id)
	if logChan == nil {
		fmt.Fprintf(w, "data: [tunnel stopped]\n\n")
		flusher.Flush()
		return
	}
	defer unsubscribe()

	for {
		select {
		case <-r.Context().Done():
			return
		case line, ok := <-logChan:
			if !ok {
				return
			}
			fmt.Fprintf(w, "data: %s\n\n", line)
			flusher.Flush()
		}
	}
}

func (h *Handlers) extractLogID(path string) string {
	path = strings.TrimPrefix(path, "/api/logs/")
	parts := strings.Split(path, "/")
	if len(parts) > 0 && parts[0] != "" {
		return parts[0]
	}
	return ""
}
