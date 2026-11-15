package web

import (
	"encoding/json"
	"errors"
	"net/http"
)

var ErrNoPiP = errors.New("floating panel is not available on this build")

type pipRequest struct {
	ID string `json:"id"`
}

func (h *Handlers) handlePiP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req pipRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.ID == "" {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}

	if _, ok := h.svc.Tunnel(req.ID); !ok {
		http.Error(w, "tunnel not found", http.StatusNotFound)
		return
	}

	if err := h.server.OpenPiP(req.ID); err != nil {
		if errors.Is(err, ErrNoPiP) {
			http.Error(w, err.Error(), http.StatusNotImplemented)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
