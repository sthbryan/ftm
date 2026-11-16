package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"
)

const (
	// sseRetry is the reconnect delay the browser is told to use.
	sseRetry = 2 * time.Second
	// sseHeartbeat keeps idle streams open through proxies and lets the
	// server notice a client that went away.
	sseHeartbeat = 20 * time.Second
)

type eventsInput struct {
	LastEventID string `header:"Last-Event-ID" doc:"Id of the last event received; the stream resumes after it, or sends resync when it cannot"`
}

type logsInput struct {
	ID          string `path:"id" doc:"Tunnel id"`
	Follow      string `query:"follow" enum:",0,1,true,false" doc:"1 streams the output as Server-Sent Events instead of answering once"`
	LastEventID string `header:"Last-Event-ID" doc:"Sent by EventSource on reconnect; the stream starts again with a fresh log_snapshot"`
}

type logTextOutput struct {
	ContentType string `header:"Content-Type"`
	Body        []byte
}

// sseRequired are the fields every documented event has.
var sseRequired = []string{
	"event",
	"data",
}

func typeOf(v any) reflect.Type { return reflect.TypeOf(v) }

// sseContent documents a text/event-stream response whose events are the
// given types, one per event name.
func sseContent(api huma.API, kinds map[string]any) map[string]*huma.MediaType {
	registry := api.OpenAPI().Components.Schemas
	names := make([]string, 0, len(kinds))
	for name := range kinds {
		names = append(names, name)
	}
	sort.Strings(names)

	oneOf := make([]*huma.Schema, 0, len(kinds))
	for _, name := range names {
		data := registry.Schema(typeOf(kinds[name]), true, name)
		oneOf = append(oneOf, &huma.Schema{
			Type:  huma.TypeObject,
			Title: "Event " + name,
			Properties: map[string]*huma.Schema{
				"id":    {Type: huma.TypeInteger, Description: "Event id, for Last-Event-ID"},
				"event": {Type: huma.TypeString, Enum: []any{name}},
				"data":  data,
				"retry": {Type: huma.TypeInteger},
			},
			Required: sseRequired,
		})
	}
	return map[string]*huma.MediaType{
		"text/event-stream": {Schema: &huma.Schema{
			Description: "Server-Sent Events. Each event's data is JSON whose type field equals the event name.",
			OneOf:       oneOf,
		}},
	}
}

// sseWriter writes Server-Sent Events and flushes each one.
type sseWriter struct {
	w  io.Writer
	rc *http.ResponseController
}

func startSSE(hctx huma.Context) *sseWriter {
	_, w := humago.Unwrap(hctx)
	h := w.Header()
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-cache")
	h.Set("Connection", "keep-alive")
	h.Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)

	sw := &sseWriter{w: w, rc: http.NewResponseController(w)}
	_, _ = fmt.Fprintf(w, "retry: %d\n\n", sseRetry.Milliseconds())
	sw.flush()
	return sw
}

func (sw *sseWriter) flush() error { return sw.rc.Flush() }

func (sw *sseWriter) send(id int64, event string, data []byte) error {
	var b strings.Builder
	if id != 0 {
		fmt.Fprintf(&b, "id: %d\n", id)
	}
	fmt.Fprintf(&b, "event: %s\n", event)
	for _, line := range strings.Split(string(data), "\n") {
		fmt.Fprintf(&b, "data: %s\n", line)
	}
	b.WriteString("\n")
	if _, err := io.WriteString(sw.w, b.String()); err != nil {
		return err
	}
	return sw.flush()
}

func (sw *sseWriter) sendJSON(event string, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return sw.send(0, event, data)
}

func (sw *sseWriter) heartbeat() error {
	if _, err := io.WriteString(sw.w, ": ping\n\n"); err != nil {
		return err
	}
	return sw.flush()
}

func (s *Server) registerEvents(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "events",
		Method:      http.MethodGet,
		Path:        "/api/events",
		Summary:     "Stream every change as Server-Sent Events",
		Description: "Tunnel state, stats and deletes, install progress, notifications, ftm and " +
			"provider update availability, and provider update results. Every event has an id; " +
			"a client that reconnects with Last-Event-ID gets what it missed, or a resync event " +
			"when that is no longer kept, after which it should reload over REST. A comment line " +
			"is sent every 20 seconds while idle.",
		Tags: []string{tagEvents},
		Responses: map[string]*huma.Response{
			"200": {
				Description: "Event stream",
				Content: sseContent(api, map[string]any{
					eventTunnelState:             TunnelState{},
					eventTunnelDeleted:           tunnelDeletedMessage{},
					eventTunnelStats:             tunnelStatsMessage{},
					eventInstall:                 installMessage{},
					eventNotification:            notificationMessage{},
					eventUpdateAvailable:         updateAvailableMessage{},
					eventProviderUpdateAvailable: providerUpdateAvailableMessage{},
					eventProviderUpdated:         providerUpdatedMessage{},
					eventResync:                  resyncMessage{},
				}),
			},
		},
	}, func(ctx context.Context, in *eventsInput) (*huma.StreamResponse, error) {
		lastID, resume := parseEventID(in.LastEventID)
		return &huma.StreamResponse{Body: func(hctx huma.Context) {
			s.streamEvents(hctx, lastID, resume)
		}}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "tunnel-logs",
		Method:      http.MethodGet,
		Path:        "/api/tunnels/{id}/logs",
		Summary:     "Read or follow a tunnel's output",
		Description: "Without follow, answers once with the recent lines. With follow=1, streams " +
			"Server-Sent Events: a log_snapshot with the recent lines (again on every reconnect, so " +
			"the client replaces what it has), then one log event per new line, also across restarts " +
			"of the tunnel.",
		Tags: []string{tagEvents, tagTunnels},
		Responses: map[string]*huma.Response{
			"200": {
				Description: "Recent lines, or the stream with follow=1",
				Content: mergeContent(
					jsonContent(api, LogSnapshot{}, "LogSnapshot"),
					sseContent(api, map[string]any{eventLogSnapshot: LogSnapshot{}, eventLog: logMessage{}}),
				),
			},
		},
	}, func(ctx context.Context, in *logsInput) (*huma.StreamResponse, error) {
		if _, ok := s.svc.Tunnel(in.ID); !ok {
			return nil, errTunnelNotFound()
		}
		follow := in.Follow == "1" || in.Follow == "true"
		return &huma.StreamResponse{Body: func(hctx huma.Context) {
			if !follow {
				hctx.SetHeader("Content-Type", "application/json")
				hctx.SetStatus(http.StatusOK)
				_ = json.NewEncoder(hctx.BodyWriter()).Encode(LogSnapshot{ID: in.ID, Lines: s.svc.Logs(in.ID)})
				return
			}
			s.streamLogs(hctx, in.ID)
		}}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "tunnel-logs-text",
		Method:      http.MethodGet,
		Path:        "/api/logs/{id}",
		Summary:     "Recent output of a tunnel as plain text",
		Tags:        []string{tagTunnels},
		Responses: map[string]*huma.Response{
			"200": {
				Description: "One line per log line",
				Content:     map[string]*huma.MediaType{"text/plain": {Schema: &huma.Schema{Type: huma.TypeString}}},
			},
		},
	}, func(ctx context.Context, in *tunnelIDInput) (*logTextOutput, error) {
		return &logTextOutput{
			ContentType: "text/plain",
			Body:        []byte(strings.Join(s.svc.Logs(in.ID), "\n")),
		}, nil
	})
}

func mergeContent(parts ...map[string]*huma.MediaType) map[string]*huma.MediaType {
	out := map[string]*huma.MediaType{}
	for _, p := range parts {
		for k, v := range p {
			out[k] = v
		}
	}
	return out
}

func parseEventID(raw string) (int64, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, false
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		// Not one of ours: resume as if every event was missed.
		return -1, true
	}
	return id, true
}

func (s *Server) streamEvents(hctx huma.Context, lastID int64, resume bool) {
	client, replay, gap := s.hub.subscribe(lastID, resume)
	defer s.hub.unsubscribe(client)

	sw := startSSE(hctx)
	if gap {
		if err := sw.sendJSON(eventResync, resyncMessage{Type: eventResync, Reason: "missed events are no longer kept"}); err != nil {
			return
		}
	}
	for _, f := range replay {
		if err := sw.send(f.ID, f.Event, f.Data); err != nil {
			return
		}
	}

	ctx := hctx.Context()
	beat := time.NewTicker(sseHeartbeat)
	defer beat.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.ctx.Done():
			return
		case <-client.done:
			// Dropped for falling behind; the browser reconnects and resumes.
			return
		case f := <-client.ch:
			if err := sw.send(f.ID, f.Event, f.Data); err != nil {
				return
			}
		case <-beat.C:
			if err := sw.heartbeat(); err != nil {
				return
			}
		}
	}
}

func (s *Server) streamLogs(hctx huma.Context, tunnelID string) {
	// Subscribe before reading the snapshot so no line falls between them.
	lines, stop := s.followLogs(tunnelID)
	defer stop()

	sw := startSSE(hctx)
	if err := sw.sendJSON(eventLogSnapshot, LogSnapshot{Type: eventLogSnapshot, ID: tunnelID, Lines: s.svc.Logs(tunnelID)}); err != nil {
		return
	}

	ctx := hctx.Context()
	beat := time.NewTicker(sseHeartbeat)
	defer beat.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.ctx.Done():
			return
		case line, ok := <-lines:
			if !ok {
				return
			}
			if err := sw.sendJSON(eventLog, logMessage{Type: eventLog, ID: tunnelID, Line: line}); err != nil {
				return
			}
		case <-beat.C:
			if err := sw.heartbeat(); err != nil {
				return
			}
		}
	}
}
