package httpapi

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/sthbryan/ftm/internal/config"
)

func TestListTunnelsReturnsAnArrayWhenThereAreNone(t *testing.T) {
	s := newTestServer(t, nil, &config.Config{})

	w := call(t, s, http.MethodGet, "/api/tunnels", "")

	if got := w.Body.String(); got != "[]\n" {
		t.Fatalf("body = %q, want an empty JSON array", got)
	}

	var decoded []map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&decoded); err != nil && err != http.ErrBodyNotAllowed {
		t.Fatalf("response does not decode as an array: %v", err)
	}
}
