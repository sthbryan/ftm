package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sthbryan/ftm/internal/config"
)

func TestListTunnelsReturnsAnArrayWhenThereAreNone(t *testing.T) {
	h := newTestServer(t, nil, &config.Config{}).handlers

	w := httptest.NewRecorder()
	h.listTunnels(w)

	if got := w.Body.String(); got != "[]\n" {
		t.Fatalf("body = %q, want an empty JSON array", got)
	}

	var decoded []map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&decoded); err != nil && err != http.ErrBodyNotAllowed {
		t.Fatalf("response does not decode as an array: %v", err)
	}
}
