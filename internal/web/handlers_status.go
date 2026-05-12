package web

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"

	"github.com/sthbryan/ftm/internal/version"
)

func (h *Handlers) handleStatus(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"port":    h.server.Port(),
		"version": version.Version,
	})
}

func (h *Handlers) handleProviders(w http.ResponseWriter) {
	providers := []map[string]string{
		{"id": "cloudflared", "name": "Cloudflared"},
		{"id": "tunnelmole", "name": "Tunnelmole"},
		{"id": "localhostrun", "name": "localhost.run"},
		{"id": "serveo", "name": "Serveo"},
		{"id": "pinggy", "name": "Pinggy"},
		{"id": "bore", "name": "bore"},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(providers)
}

func (h *Handlers) handleDetectPort(w http.ResponseWriter) {
	commonPorts := []int{30000, 30001, 30002, 30003, 30004, 30005, 30006, 30007, 30008, 30009}
	found := []int{}

	for _, port := range commonPorts {
		if ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port)); err == nil {
			ln.Close()
		} else {
			found = append(found, port)
		}
	}

	suggested := 30000
	if len(found) > 0 {
		suggested = found[0]
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"ports":     found,
		"suggested": suggested,
	})
}
