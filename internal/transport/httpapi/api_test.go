package httpapi

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/sthbryan/ftm/internal/core"
	"github.com/sthbryan/ftm/internal/events"
	"github.com/sthbryan/ftm/internal/provider"
	"github.com/sthbryan/ftm/internal/supervisor"
)

// fakeSupervisor runs nothing but behaves like the process manager: a
// provider may need a download first, and Start/Stop publish states.
type fakeSupervisor struct {
	mu        sync.Mutex
	bus       *events.Bus
	installed map[core.Provider]bool
	release   chan struct{}
	status    map[string]core.TunnelStatus
	started   []string
	stopped   []string
	logs      map[string][]string
}

func newFakeSupervisor() *fakeSupervisor {
	return &fakeSupervisor{
		installed: map[core.Provider]bool{},
		status:    map[string]core.TunnelStatus{},
		logs:      map[string][]string{},
	}
}

func (f *fakeSupervisor) Start(t core.TunnelConfig) error {
	f.mu.Lock()
	st := t.Status()
	st.State = core.TunnelStateOnline
	st.PublicURL = "https://" + t.ID + ".example"
	f.status[t.ID] = st
	f.started = append(f.started, t.ID)
	bus := f.bus
	f.mu.Unlock()
	bus.Publish(events.TunnelStatus{Status: st})
	return nil
}

func (f *fakeSupervisor) Stop(id string) error {
	f.mu.Lock()
	st, ok := f.status[id]
	st.State = core.TunnelStateStopped
	st.PublicURL = ""
	f.status[id] = st
	f.stopped = append(f.stopped, id)
	bus := f.bus
	f.mu.Unlock()
	if ok {
		bus.Publish(events.TunnelStatus{Status: st})
	}
	return nil
}

func (f *fakeSupervisor) StopAll() {}

func (f *fakeSupervisor) CheckInstallation(p core.Provider) (bool, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return !f.installed[p], true
}

func (f *fakeSupervisor) InstallProvider(p core.Provider) error {
	f.mu.Lock()
	release := f.release
	bus := f.bus
	f.mu.Unlock()

	bus.Publish(events.InstallProgress{Provider: string(p), Percent: 40, Current: 4, Total: 10})
	if release != nil {
		<-release
	}
	bus.Publish(events.InstallProgress{Provider: string(p), Percent: 100, Current: 10, Total: 10, Done: true})

	f.mu.Lock()
	f.installed[p] = true
	f.mu.Unlock()
	return nil
}

func (f *fakeSupervisor) GetStatus(id string) (core.TunnelStatus, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	st, ok := f.status[id]
	return st, ok
}

func (f *fakeSupervisor) GetLogs(id string) []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.logs[id]...)
}

func (f *fakeSupervisor) startedIDs() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.started...)
}

// fakeBinaries adds provider binary management to fakeSupervisor.
type fakeBinaries struct {
	*fakeSupervisor
	updated chan core.Provider
}

func (f *fakeBinaries) CheckLatest(context.Context, bool) {}
func (f *fakeBinaries) ProviderBinaries(ctx context.Context) []supervisor.ProviderBinary {
	return []supervisor.ProviderBinary{f.ProviderBinary(ctx, core.ProviderCloudflared)}
}
func (f *fakeBinaries) ProviderBinary(_ context.Context, id core.Provider) supervisor.ProviderBinary {
	return supervisor.ProviderBinary{ID: id, Name: string(id), Installed: true, Version: "1.0", Latest: "2.0",
		Source: supervisor.SourceBundled, HasUpdate: true, Updatable: true, CheckedAt: time.UnixMilli(1000)}
}
func (f *fakeBinaries) UpdateProvider(_ context.Context, id core.Provider, _ chan<- provider.DownloadProgress) (*supervisor.UpdateResult, error) {
	f.updated <- id
	return &supervisor.UpdateResult{Provider: id, Version: "2.0", Restarted: []string{"world"}}, nil
}
func (f *fakeBinaries) IsUpdating(core.Provider) bool { return false }

func worldConfig() *core.Config {
	cfg := core.DefaultConfig()
	cfg.AddTunnel(core.TunnelConfig{ID: "world", Name: "World", Provider: core.ProviderCloudflared, LocalPort: 30000})
	return cfg
}

// newLiveServer runs the full handler over a real listener and starts the
// event fan-out.
func newLiveServer(t *testing.T, sup *fakeSupervisor) (*Server, *httptest.Server) {
	t.Helper()
	s := newTestServer(t, sup, worldConfig())
	sup.mu.Lock()
	sup.bus = s.bus
	sup.mu.Unlock()
	s.listenEvents()
	ts := httptest.NewServer(s.Handler())
	t.Cleanup(ts.Close)
	t.Cleanup(func() { _ = s.Stop() })
	return s, ts
}

type sseEvent struct {
	ID    string
	Event string
	Data  map[string]any
}

// sseStream reads Server-Sent Events from one open stream.
type sseStream struct {
	t      *testing.T
	events chan sseEvent
	resp   *http.Response
}

func openSSE(t *testing.T, ts *httptest.Server, path string, headers ...string) *sseStream {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, ts.URL+path, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	for i := 0; i+1 < len(headers); i += 2 {
		req.Header.Set(headers[i], headers[i+1])
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		t.Fatalf("GET %s = %d", path, resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "text/event-stream" {
		t.Fatalf("Content-Type = %q", ct)
	}

	st := &sseStream{t: t, events: make(chan sseEvent, 256), resp: resp}
	t.Cleanup(func() { resp.Body.Close() })
	go func() {
		defer close(st.events)
		sc := bufio.NewScanner(resp.Body)
		var cur sseEvent
		var data strings.Builder
		for sc.Scan() {
			line := sc.Text()
			switch {
			case line == "":
				if cur.Event != "" {
					_ = json.Unmarshal([]byte(data.String()), &cur.Data)
					st.events <- cur
				}
				cur = sseEvent{}
				data.Reset()
			case strings.HasPrefix(line, "id: "):
				cur.ID = strings.TrimPrefix(line, "id: ")
			case strings.HasPrefix(line, "event: "):
				cur.Event = strings.TrimPrefix(line, "event: ")
			case strings.HasPrefix(line, "data: "):
				data.WriteString(strings.TrimPrefix(line, "data: "))
			}
		}
	}()
	return st
}

// next returns the next event named want, skipping others.
func (st *sseStream) next(want string) sseEvent {
	st.t.Helper()
	deadline := time.After(3 * time.Second)
	for {
		select {
		case e, ok := <-st.events:
			if !ok {
				st.t.Fatalf("stream closed before a %q event", want)
			}
			if e.Event == want {
				if e.Data["type"] != want {
					st.t.Fatalf("%s event data type = %v", want, e.Data["type"])
				}
				return e
			}
		case <-deadline:
			st.t.Fatalf("no %q event", want)
		}
	}
}

// reply is a finished response: its body is already read and closed.
type reply struct {
	StatusCode int
	Header     http.Header
}

func post(t *testing.T, ts *httptest.Server, path string) (reply, map[string]any) {
	t.Helper()
	req, _ := http.NewRequest(http.MethodPost, ts.URL+path, nil)
	req.Header.Set("Origin", ts.URL)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var body map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&body)
	return reply{StatusCode: resp.StatusCode, Header: resp.Header}, body
}

func TestStartInstallsTheProviderThenStartsAndStopStops(t *testing.T) {
	sup := newFakeSupervisor()
	sup.release = make(chan struct{})
	_, ts := newLiveServer(t, sup)
	stream := openSSE(t, ts, "/api/events")

	resp, body := post(t, ts, "/api/tunnels/world/start")
	if resp.StatusCode != http.StatusOK || body["state"] != "installing" || body["id"] != "world" || body["port"] != 30000.0 {
		t.Fatalf("start = %d %v, want 200 installing", resp.StatusCode, body)
	}

	if e := stream.next(eventTunnelState); e.Data["state"] != "installing" || e.Data["provider"] != "cloudflared" {
		t.Fatalf("first tunnel_state = %v", e.Data)
	}
	if e := stream.next(eventNotification); e.Data["notification"].(map[string]any)["title"] != "Installing" {
		t.Fatalf("notification = %v", e.Data)
	}
	if e := stream.next(eventInstall); e.Data["percent"] != 40.0 || e.Data["done"] != false {
		t.Fatalf("install = %v", e.Data)
	}

	close(sup.release)
	if e := stream.next(eventInstall); e.Data["done"] != true || e.Data["install"].(map[string]any)["step"] != "Done" {
		t.Fatalf("last install = %v", e.Data)
	}
	if e := stream.next(eventTunnelState); e.Data["state"] != "online" || e.Data["publicUrl"] != "https://world.example" {
		t.Fatalf("tunnel_state = %v, want online", e.Data)
	}
	if got := sup.startedIDs(); len(got) != 1 || got[0] != "world" {
		t.Fatalf("started = %v", got)
	}

	// Started again once installed: no download, the state comes back.
	resp, body = post(t, ts, "/api/tunnels/world/start")
	if resp.StatusCode != http.StatusOK || body["state"] != "online" || body["type"] != "tunnel_state" {
		t.Fatalf("second start = %d %v", resp.StatusCode, body)
	}

	resp, body = post(t, ts, "/api/tunnels/world/stop")
	if resp.StatusCode != http.StatusOK || body["state"] != "stopped" {
		t.Fatalf("stop = %d %v", resp.StatusCode, body)
	}
	// The second start published online again; stopped follows it.
	for e := stream.next(eventTunnelState); e.Data["state"] != "stopped"; e = stream.next(eventTunnelState) {
		if e.Data["state"] != "online" {
			t.Fatalf("tunnel_state = %v, want online then stopped", e.Data)
		}
	}
}

func TestStartUnknownTunnelIsAJSONNotFound(t *testing.T) {
	_, ts := newLiveServer(t, newFakeSupervisor())

	resp, body := post(t, ts, "/api/tunnels/nope/start")
	if resp.StatusCode != http.StatusNotFound || body["code"] != "tunnel_not_found" || body["message"] != "tunnel not found" {
		t.Fatalf("start = %d %v", resp.StatusCode, body)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Fatalf("Content-Type = %q", ct)
	}
}

func TestProviderUpdateRunsInTheBackgroundAndReportsOnce(t *testing.T) {
	sup := &fakeBinaries{fakeSupervisor: newFakeSupervisor(), updated: make(chan core.Provider, 1)}
	s := newTestServer(t, sup, worldConfig())
	sup.bus = s.bus
	s.listenEvents()
	ts := httptest.NewServer(s.Handler())
	t.Cleanup(ts.Close)
	t.Cleanup(func() { _ = s.Stop() })
	stream := openSSE(t, ts, "/api/events")

	resp, body := post(t, ts, "/api/providers/cloudflared/update")
	if resp.StatusCode != http.StatusAccepted || body["ok"] != true || body["provider"] != "cloudflared" || body["state"] != "updating" {
		t.Fatalf("update = %d %v", resp.StatusCode, body)
	}
	if id := <-sup.updated; id != core.ProviderCloudflared {
		t.Fatalf("updated %s", id)
	}

	e := stream.next(eventProviderUpdated)
	binary := e.Data["binary"].(map[string]any)
	if e.Data["ok"] != true || e.Data["version"] != "2.0" || binary["source"] != "bundled" || binary["checkedAt"] != 1000.0 {
		t.Fatalf("provider_updated = %v", e.Data)
	}
	if r := e.Data["restarted"].([]any); len(r) != 1 || r[0] != "world" {
		t.Fatalf("restarted = %v", e.Data["restarted"])
	}

	rec := call(t, s, http.MethodGet, "/api/providers/installed", "")
	wantStatus(t, rec, http.StatusOK)
	var installed struct {
		Providers []map[string]any `json:"providers"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &installed); err != nil || len(installed.Providers) != 1 {
		t.Fatalf("installed = %s (%v)", rec.Body, err)
	}
	for _, key := range []string{"id", "name", "installed", "version", "latest", "source", "path", "hasUpdate", "updatable", "updating", "releaseUrl", "checkedAt"} {
		if _, ok := installed.Providers[0][key]; !ok {
			t.Errorf("installed row is missing %q", key)
		}
	}
	if e := stream.next(eventProviderUpdateAvailable); e.Data["providers"].([]any)[0].(map[string]any)["latest"] != "2.0" {
		t.Fatalf("provider_update_available = %v", e.Data)
	}
}

func TestEventsResumeAfterLastEventID(t *testing.T) {
	s, ts := newLiveServer(t, newFakeSupervisor())
	first := openSSE(t, ts, "/api/events")

	s.bus.Publish(events.TunnelDeleted{ID: "a"})
	a := first.next(eventTunnelDeleted)
	if a.ID == "" {
		t.Fatal("event without an id")
	}
	first.resp.Body.Close()

	// Missed while disconnected.
	s.bus.Publish(events.TunnelDeleted{ID: "b"})
	s.bus.Publish(events.TunnelStats{ID: "c", Visitors: 2})
	waitFor(t, func() bool { return s.hub.lastID() >= mustInt(t, a.ID)+2 })

	again := openSSE(t, ts, "/api/events", "Last-Event-ID", a.ID)
	if e := again.next(eventTunnelDeleted); e.Data["id"] != "b" {
		t.Fatalf("replayed %v, want b first", e.Data)
	}
	// Live events keep flowing after the replay, in order.
	s.bus.Publish(events.TunnelDeleted{ID: "d"})
	if e := again.next(eventTunnelDeleted); e.Data["id"] != "d" {
		t.Fatalf("got %v, want d", e.Data)
	}
}

func TestEventsAskForAResyncWhenTheGapIsGone(t *testing.T) {
	_, ts := newLiveServer(t, newFakeSupervisor())

	// An id from before a restart, or from too long ago.
	stream := openSSE(t, ts, "/api/events", "Last-Event-ID", "12")
	if e := stream.next(eventResync); e.Data["reason"] == "" {
		t.Fatalf("resync = %v", e.Data)
	}

	garbage := openSSE(t, ts, "/api/events", "Last-Event-ID", "not-a-number")
	garbage.next(eventResync)
}

func TestEventsCarryNotificationsAndUpdateAvailability(t *testing.T) {
	s, ts := newLiveServer(t, newFakeSupervisor())
	stream := openSSE(t, ts, "/api/events")

	st := core.TunnelStatus{ID: "world", Name: "World", State: core.TunnelStateError, ErrorMessage: "boom"}
	s.bus.Publish(events.TunnelStatus{Status: st})
	if e := stream.next(eventTunnelState); e.Data["state"] != "error" || e.Data["errorMessage"] != "boom" {
		t.Fatalf("tunnel_state = %v", e.Data)
	}
	if e := stream.next(eventNotification); e.Data["notification"].(map[string]any)["toastType"] != "error" {
		t.Fatalf("notification = %v", e.Data)
	}

	s.BroadcastNewVisitor(core.TunnelStatus{Name: "World", Visitors: 1})
	stream.next(eventNotification)

	s.bus.Publish(events.UpdateAvailable{Current: "1.0.0", Latest: "1.1.0", Method: "self"})
	if e := stream.next(eventUpdateAvailable); e.Data["latest"] != "1.1.0" || e.Data["method"] != "self" {
		t.Fatalf("update_available = %v", e.Data)
	}

	s.bus.Publish(events.TunnelStats{ID: "world", Visitors: 4, ActiveSessions: 2})
	if e := stream.next(eventTunnelStats); e.Data["activeSessions"] != 2.0 {
		t.Fatalf("tunnel_stats = %v", e.Data)
	}
	if s.ClientCount() != 1 {
		t.Fatalf("ClientCount = %d, want the SSE client counted", s.ClientCount())
	}
}

func TestEventsEndWhenTheServerStops(t *testing.T) {
	s, ts := newLiveServer(t, newFakeSupervisor())
	stream := openSSE(t, ts, "/api/events")

	_ = s.Stop()
	select {
	case _, ok := <-stream.events:
		for ok {
			_, ok = <-stream.events
		}
	case <-time.After(3 * time.Second):
		t.Fatal("stream still open after Stop")
	}
}

func TestLogsAnswerOnceOrFollow(t *testing.T) {
	sup := newFakeSupervisor()
	sup.logs["world"] = []string{"one", "two"}
	s, ts := newLiveServer(t, sup)

	rec := call(t, s, http.MethodGet, "/api/tunnels/world/logs", "")
	wantStatus(t, rec, http.StatusOK)
	var snap LogSnapshot
	if err := json.Unmarshal(rec.Body.Bytes(), &snap); err != nil || snap.ID != "world" || len(snap.Lines) != 2 {
		t.Fatalf("logs = %s (%v)", rec.Body, err)
	}

	rec = call(t, s, http.MethodGet, "/api/logs/world", "")
	wantStatus(t, rec, http.StatusOK)
	if rec.Body.String() != "one\ntwo" || !strings.HasPrefix(rec.Header().Get("Content-Type"), "text/plain") {
		t.Fatalf("plain logs = %q %q", rec.Body, rec.Header().Get("Content-Type"))
	}

	stream := openSSE(t, ts, "/api/tunnels/world/logs?follow=1")
	if e := stream.next(eventLogSnapshot); len(e.Data["lines"].([]any)) != 2 {
		t.Fatalf("log_snapshot = %v", e.Data)
	}
	s.bus.Publish(events.LogLine{TunnelID: "other", Line: "not mine"})
	s.bus.Publish(events.LogLine{TunnelID: "world", Line: "three"})
	if e := stream.next(eventLog); e.Data["line"] != "three" || e.Data["id"] != "world" {
		t.Fatalf("log = %v", e.Data)
	}

	rec = call(t, s, http.MethodGet, "/api/tunnels/nope/logs?follow=1", "")
	wantStatus(t, rec, http.StatusNotFound)
}

func TestOpenAPIDescribesEveryRoute(t *testing.T) {
	s := newTestServer(t, nil, worldConfig())

	rec := call(t, s, http.MethodGet, "/openapi.json", "", "Sec-Fetch-Site", "")
	wantStatus(t, rec, http.StatusOK)

	var doc struct {
		OpenAPI    string                    `json:"openapi"`
		Paths      map[string]map[string]any `json:"paths"`
		Components struct {
			Schemas map[string]struct {
				Properties map[string]any `json:"properties"`
			} `json:"schemas"`
		} `json:"components"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &doc); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(doc.OpenAPI, "3.1") {
		t.Errorf("openapi = %q, want 3.1", doc.OpenAPI)
	}

	want := map[string][]string{
		"/api/tunnels":               {"get", "post", "delete"},
		"/api/tunnels/{id}":          {"get", "put", "delete"},
		"/api/tunnels/{id}/start":    {"post"},
		"/api/tunnels/{id}/stop":     {"post"},
		"/api/tunnels/{id}/logs":     {"get"},
		"/api/logs/{id}":             {"get"},
		"/api/events":                {"get"},
		"/api/status":                {"get"},
		"/api/settings":              {"get", "patch", "post"},
		"/api/providers":             {"get"},
		"/api/providers/installed":   {"get"},
		"/api/providers/{id}/update": {"post"},
		"/api/i18n":                  {"get"},
		"/api/i18n/current":          {"get"},
		"/api/update":                {"get", "post"},
		"/api/detect-port":           {"get"},
		"/api/pip":                   {"post"},
	}
	for path, methods := range want {
		for _, m := range methods {
			if _, ok := doc.Paths[path][m]; !ok {
				t.Errorf("openapi.json is missing %s %s", strings.ToUpper(m), path)
			}
		}
	}

	errSchema, ok := doc.Components.Schemas["Error"]
	if !ok {
		t.Fatal("no Error schema")
	}
	for _, key := range []string{"code", "message"} {
		if _, ok := errSchema.Properties[key]; !ok {
			t.Errorf("Error schema is missing %q", key)
		}
	}
	if strings.Contains(rec.Body.String(), `"$schema"`) {
		t.Error("responses would carry huma's $schema field")
	}
}

func TestErrorsAreJSONWithCodeAndMessage(t *testing.T) {
	s := newTestServer(t, nil, worldConfig())

	cases := []struct {
		method, path, body string
		status             int
		code               string
	}{
		{http.MethodGet, "/api/nope", "", http.StatusNotFound, "not_found"},
		{http.MethodDelete, "/api/status", "", http.StatusMethodNotAllowed, "method_not_allowed"},
		{http.MethodPost, "/api/tunnels", `{"name":`, http.StatusBadRequest, "bad_request"},
		{http.MethodPost, "/api/tunnels", `{"name":"x"}`, http.StatusBadRequest, "missing_fields"},
		{http.MethodPost, "/api/tunnels", `{"name":"x","provider":"bore","localPort":70000}`, http.StatusBadRequest, "invalid_port"},
		{http.MethodGet, "/api/tunnels/nope", "", http.StatusNotFound, "tunnel_not_found"},
		{http.MethodPatch, "/api/settings", `{"language":"klingon"}`, http.StatusBadRequest, "unsupported_language"},
		{http.MethodPost, "/api/update", `{}`, http.StatusBadRequest, "no_update"},
	}
	for _, c := range cases {
		t.Run(c.method+" "+c.path+" "+c.body, func(t *testing.T) {
			rec := call(t, s, c.method, c.path, c.body)
			wantStatus(t, rec, c.status)
			var body map[string]any
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("body %q is not JSON", rec.Body)
			}
			if body["code"] != c.code || body["message"] == "" || len(body) != 2 {
				t.Fatalf("body = %v, want code %q and a message", body, c.code)
			}
		})
	}
}

func TestTunnelRoutesKeepTheirShapes(t *testing.T) {
	s := newTestServer(t, newFakeSupervisor(), core.DefaultConfig())

	rec := call(t, s, http.MethodPost, "/api/tunnels", `{"name":"Table","provider":"bore","extra":"ignored"}`)
	wantStatus(t, rec, http.StatusOK)
	var created map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &created)
	for key, want := range map[string]any{"type": "tunnel_state", "name": "Table", "provider": "bore", "port": 30000.0, "state": "need_installing", "publicUrl": "", "errorMessage": "", "expiresAt": 0.0, "startedAt": 0.0} {
		if created[key] != want {
			t.Errorf("created[%q] = %v, want %v", key, created[key], want)
		}
	}
	id := created["id"].(string)

	rec = call(t, s, http.MethodPut, "/api/tunnels/"+id, `{"name":"","localPort":30001}`)
	wantStatus(t, rec, http.StatusOK)
	var updated map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &updated)
	if updated["name"] != "Table" || updated["port"] != 30001.0 {
		t.Fatalf("updated = %v, want the name kept and the port changed", updated)
	}

	wantStatus(t, call(t, s, http.MethodDelete, "/api/tunnels/"+id, ""), http.StatusOK)
	wantStatus(t, call(t, s, http.MethodGet, "/api/tunnels/"+id, ""), http.StatusNotFound)

	rec = call(t, s, http.MethodGet, "/api/detect-port", "")
	wantStatus(t, rec, http.StatusOK)
	var ports map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &ports)
	if _, ok := ports["ports"].([]any); !ok || ports["suggested"] == nil {
		t.Fatalf("detect-port = %v", ports)
	}
}

func TestTokenIsRequiredWithoutABrowserOrigin(t *testing.T) {
	s := newTestServer(t, nil, worldConfig())
	s.SetToken("secret")

	cases := []struct {
		name    string
		headers []string
		want    int
	}{
		{"no origin, no token", []string{"Sec-Fetch-Site", ""}, http.StatusUnauthorized},
		{"wrong token", []string{"Sec-Fetch-Site", "", "Authorization", "Bearer nope"}, http.StatusUnauthorized},
		{"bearer token", []string{"Sec-Fetch-Site", "", "Authorization", "Bearer secret"}, http.StatusOK},
		{"token header", []string{"Sec-Fetch-Site", "", TokenHeader, "secret"}, http.StatusOK},
		{"local browser origin", []string{"Sec-Fetch-Site", "", "Origin", "http://localhost:40500"}, http.StatusOK},
		{"same-origin page", []string{"Sec-Fetch-Site", "same-origin"}, http.StatusOK},
		{"another local site", []string{"Sec-Fetch-Site", "same-site"}, http.StatusUnauthorized},
		{"foreign origin", []string{"Origin", "https://evil.example"}, http.StatusForbidden},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rec := call(t, s, http.MethodGet, "/api/status", "", c.headers...)
			wantStatus(t, rec, c.want)
			if c.want != http.StatusOK {
				var body map[string]any
				if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil || body["code"] == nil {
					t.Fatalf("error body = %q", rec.Body)
				}
			}
		})
	}

	// The dashboard page itself and the API description need no token.
	wantStatus(t, call(t, s, http.MethodGet, "/openapi.json", "", "Sec-Fetch-Site", ""), http.StatusOK)

	// A preflight is answered before the token check.
	rec := call(t, s, http.MethodOptions, "/api/tunnels", "", "Sec-Fetch-Site", "", "Origin", "http://localhost:5173")
	wantStatus(t, rec, http.StatusOK)
	if !strings.Contains(rec.Header().Get("Access-Control-Allow-Headers"), TokenHeader) {
		t.Errorf("Allow-Headers = %q", rec.Header().Get("Access-Control-Allow-Headers"))
	}
}

func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatal("condition not met")
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func mustInt(t *testing.T, s string) int64 {
	t.Helper()
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		t.Fatal(err)
	}
	return n
}
