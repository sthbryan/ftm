// Package client talks to a running ftm (the desktop app, `ftm serve` or the
// TUI) over its HTTP API, so a second process can act on the tunnels that
// process owns instead of starting a manager of its own.
package client

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/sthbryan/ftm/internal/platform/instance"
)

// tokenHeader is the header the server reads the runtime token from.
const tokenHeader = "X-FTM-Token"

// Client is a typed client for one running instance.
type Client struct {
	base  string
	token string
	http  *http.Client
}

// New returns a client for the instance at baseURL that presents token.
func New(baseURL, token string) *Client {
	return &Client{base: strings.TrimRight(baseURL, "/"), token: token, http: &http.Client{}}
}

// Find returns a client for the instance that owns runtimeDir, or nil when no
// ftm is running there.
func Find(runtimeDir string) (*Client, error) {
	info, err := instance.Find(runtimeDir)
	if err != nil || info == nil {
		return nil, err
	}
	return New(info.URL(), info.Token), nil
}

// URL is the instance's base URL, which is also its web dashboard.
func (c *Client) URL() string { return c.base }

// Error is an error response from the instance.
type Error struct {
	Status  int
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *Error) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return fmt.Sprintf("ftm answered %d", e.Status)
}

// IsNotFound reports whether err is a 404 from the instance.
func IsNotFound(err error) bool {
	var apiErr *Error
	return errors.As(err, &apiErr) && apiErr.Status == http.StatusNotFound
}

// Tunnel is a saved connection with its live state.
type Tunnel struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Provider       string `json:"provider"`
	Port           int    `json:"port"`
	State          string `json:"state"`
	PublicURL      string `json:"publicUrl"`
	ErrorMessage   string `json:"errorMessage"`
	ExpiresAt      int64  `json:"expiresAt"`
	StartedAt      int64  `json:"startedAt"`
	Visitors       int    `json:"visitors"`
	ActiveSessions int    `json:"activeSessions"`
}

// TunnelInput creates or edits a connection. On an edit, empty fields keep
// the saved value.
type TunnelInput struct {
	Name      string `json:"name,omitempty"`
	Provider  string `json:"provider,omitempty"`
	LocalPort int    `json:"localPort,omitempty"`
}

// ProviderBinary is the installed copy of one provider.
type ProviderBinary struct {
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
	Error      string `json:"error,omitempty"`
}

// Settings are the preferences the instance exposes.
type Settings struct {
	NotificationsEnabled string `json:"notifications_enabled"`
	NotificationSound    bool   `json:"notification_sound"`
	Language             string `json:"language"`
	LanguageResolved     string `json:"language_resolved"`
	Onboarded            bool   `json:"onboarded"`
	AutostartSupported   bool   `json:"autostart_supported"`
	AutostartEnabled     bool   `json:"autostart_enabled"`
}

// SettingsPatch changes the preferences that are set.
type SettingsPatch struct {
	NotificationsEnabled *string `json:"notifications_enabled,omitempty"`
	NotificationSound    *bool   `json:"notification_sound,omitempty"`
	Language             *string `json:"language,omitempty"`
	Autostart            *bool   `json:"autostart_enabled,omitempty"`
}

// Tunnels lists the saved connections in their saved order.
func (c *Client) Tunnels(ctx context.Context) ([]Tunnel, error) {
	var out []Tunnel
	return out, c.do(ctx, http.MethodGet, "/api/tunnels", nil, &out)
}

// Tunnel returns one connection.
func (c *Client) Tunnel(ctx context.Context, id string) (Tunnel, error) {
	var out Tunnel
	return out, c.do(ctx, http.MethodGet, "/api/tunnels/"+url.PathEscape(id), nil, &out)
}

// CreateTunnel saves a new connection.
func (c *Client) CreateTunnel(ctx context.Context, in TunnelInput) (Tunnel, error) {
	var out Tunnel
	return out, c.do(ctx, http.MethodPost, "/api/tunnels", in, &out)
}

// UpdateTunnel edits a saved connection.
func (c *Client) UpdateTunnel(ctx context.Context, id string, in TunnelInput) (Tunnel, error) {
	var out Tunnel
	return out, c.do(ctx, http.MethodPut, "/api/tunnels/"+url.PathEscape(id), in, &out)
}

// DeleteTunnel stops and removes a saved connection.
func (c *Client) DeleteTunnel(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodDelete, "/api/tunnels/"+url.PathEscape(id), nil, nil)
}

// ClearTunnels stops and removes every saved connection.
func (c *Client) ClearTunnels(ctx context.Context) error {
	return c.do(ctx, http.MethodDelete, "/api/tunnels", nil, nil)
}

// StartTunnel starts a connection. Later states arrive on Events.
func (c *Client) StartTunnel(ctx context.Context, id string) (Tunnel, error) {
	var out Tunnel
	return out, c.do(ctx, http.MethodPost, "/api/tunnels/"+url.PathEscape(id)+"/start", nil, &out)
}

// StopTunnel stops a connection.
func (c *Client) StopTunnel(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodPost, "/api/tunnels/"+url.PathEscape(id)+"/stop", nil, nil)
}

// Logs returns a tunnel's recent output.
func (c *Client) Logs(ctx context.Context, id string) ([]string, error) {
	var out struct {
		Lines []string `json:"lines"`
	}
	err := c.do(ctx, http.MethodGet, "/api/tunnels/"+url.PathEscape(id)+"/logs", nil, &out)
	return out.Lines, err
}

// Providers lists the provider binaries. refresh skips the cache of the
// latest releases.
func (c *Client) Providers(ctx context.Context, refresh bool) ([]ProviderBinary, error) {
	path := "/api/providers/installed"
	if refresh {
		path += "?refresh=1"
	}
	var out struct {
		Providers []ProviderBinary `json:"providers"`
	}
	err := c.do(ctx, http.MethodGet, path, nil, &out)
	return out.Providers, err
}

// UpdateProvider starts an update or install of a provider. Progress and the
// result arrive on Events.
func (c *Client) UpdateProvider(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodPost, "/api/providers/"+url.PathEscape(id)+"/update", nil, nil)
}

// UpdateInfo is what the instance's last ftm update check found.
type UpdateInfo struct {
	Current    string `json:"current"`
	Latest     string `json:"latest"`
	Tag        string `json:"tag"`
	AssetName  string `json:"assetName"`
	ReleaseURL string `json:"releaseUrl"`
	HasUpdate  bool   `json:"hasUpdate"`
	Method     string `json:"method"`
}

// Update reads the instance's last ftm update check.
func (c *Client) Update(ctx context.Context) (UpdateInfo, error) {
	var out UpdateInfo
	return out, c.do(ctx, http.MethodGet, "/api/update", nil, &out)
}

// ApplyUpdate installs the update the instance found. The instance then
// shuts down and starts the new version, which ends every open stream.
func (c *Client) ApplyUpdate(ctx context.Context) error {
	return c.do(ctx, http.MethodPost, "/api/update", nil, nil)
}

// Settings reads the preferences.
func (c *Client) Settings(ctx context.Context) (Settings, error) {
	var out Settings
	return out, c.do(ctx, http.MethodGet, "/api/settings", nil, &out)
}

// UpdateSettings saves the preferences that are set.
func (c *Client) UpdateSettings(ctx context.Context, patch SettingsPatch) (Settings, error) {
	var out Settings
	return out, c.do(ctx, http.MethodPatch, "/api/settings", patch, &out)
}

func (c *Client) newRequest(ctx context.Context, method, path string, body any) (*http.Request, error) {
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.base+path, reader)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.token != "" {
		req.Header.Set(tokenHeader, c.token)
	}
	return req, nil
}

func (c *Client) do(ctx context.Context, method, path string, body, out any) error {
	req, err := c.newRequest(ctx, method, path, body)
	if err != nil {
		return err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return decodeError(resp)
	}
	if out == nil {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if len(bytes.TrimSpace(data)) == 0 {
		return nil
	}
	return json.Unmarshal(data, out)
}

func decodeError(resp *http.Response) error {
	apiErr := &Error{Status: resp.StatusCode}
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	_ = json.Unmarshal(data, apiErr)
	return apiErr
}

// Event is one Server-Sent Event: its name and its JSON data.
type Event struct {
	Name string
	Data json.RawMessage
}

// Decode unmarshals the event's data into v.
func (e Event) Decode(v any) error { return json.Unmarshal(e.Data, v) }

// EventOpen is the name of the first event a stream delivers, once the
// instance has subscribed it: nothing published after it is missed.
const EventOpen = "open"

// Events streams every change the instance publishes until ctx ends or fn
// returns false. It returns nil when fn stopped it.
func (c *Client) Events(ctx context.Context, fn func(Event) bool) error {
	return c.stream(ctx, "/api/events", fn)
}

// FollowLogs streams a tunnel's output: a log_snapshot event with the recent
// lines, then one log event per new line, until ctx ends or fn returns false.
func (c *Client) FollowLogs(ctx context.Context, id string, fn func(Event) bool) error {
	return c.stream(ctx, "/api/tunnels/"+url.PathEscape(id)+"/logs?follow=1", fn)
}

func (c *Client) stream(ctx context.Context, path string, fn func(Event) bool) error {
	req, err := c.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "text/event-stream")
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return decodeError(resp)
	}

	if !fn(Event{Name: EventOpen}) {
		return nil
	}
	err = readEvents(resp.Body, fn)
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return err
}

// errStopped ends readEvents when the callback asks to stop.
var errStopped = errors.New("stopped")

func readEvents(r io.Reader, fn func(Event) bool) error {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64<<10), 4<<20)

	var name string
	var data []string
	dispatch := func() error {
		defer func() { name, data = "", nil }()
		if len(data) == 0 {
			return nil
		}
		if name == "" {
			name = "message"
		}
		if !fn(Event{Name: name, Data: json.RawMessage(strings.Join(data, "\n"))}) {
			return errStopped
		}
		return nil
	}

	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case line == "":
			if err := dispatch(); err != nil {
				return nil
			}
		case strings.HasPrefix(line, ":"):
		case strings.HasPrefix(line, "event:"):
			name = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
		case strings.HasPrefix(line, "data:"):
			data = append(data, strings.TrimPrefix(strings.TrimPrefix(line, "data:"), " "))
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return io.ErrUnexpectedEOF
}
