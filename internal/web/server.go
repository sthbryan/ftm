package web

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"net"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"foundry-tunnel/internal/clipboard"
	"foundry-tunnel/internal/config"
	"foundry-tunnel/internal/process"
)

var tmpl = template.Must(template.New("tunnels").Parse(tunnelsTemplate))

const tunnelsTemplate = `{{range .}}
<div class="connection-item {{.StatusClass}}" id="tunnel-{{.ID}}" data-id="{{.ID}}">
    <div class="drag-handle">⠿</div>
    <div class="connection-content">
        <div class="connection-main">
            <div class="connection-info">
                <div class="connection-name" onclick="editName(this, '{{.ID}}')">{{.Name}}</div>
                <div class="connection-meta">{{.ProviderName}} &mdash; Port <span onclick="editPort(this, '{{.ID}}')">{{.Port}}</span></div>
                <div class="connection-status status-{{.StatusClass}}">
                    <span class="status-dot"></span>
                    <span class="status-text">{{.StatusText}}</span>
                </div>
            </div>
            <div class="connection-actions">
                {{if .Running}}
                <button class="btn" hx-post="/api/tunnels/{{.ID}}/stop" hx-target="#tunnel-{{.ID}}" hx-swap="outerHTML">Stop</button>
                {{else}}
                <button class="btn btn-start" hx-post="/api/tunnels/{{.ID}}/start" hx-target="#tunnel-{{.ID}}" hx-swap="outerHTML">Start</button>
                {{end}}
                <button class="btn" onclick="showLogs('{{.ID}}')">Logs</button>
                <button class="btn" hx-delete="/api/tunnels/{{.ID}}" hx-target="#tunnel-{{.ID}}" hx-swap="delete" hx-confirm="Delete this connection?">Delete</button>
            </div>
        </div>
        {{if .PublicURL}}
        <div class="connection-url-row" onclick="copyUrl('{{.PublicURL}}')">
            <svg class="copy-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <rect x="9" y="9" width="13" height="13" rx="2" ry="2"></rect>
                <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path>
            </svg>
            <span class="url-text">{{.PublicURL}}</span>
            <span class="copy-hint">Click to copy</span>
        </div>
        {{end}}
    </div>
</div>
{{else}}
<div class="empty-state" id="empty-state">
    <div class="empty-state-icon">📡</div>
    <h3>No connections yet</h3>
    <p>Create your first connection to share your Foundry world with players.</p>
</div>
{{end}}`

type TunnelView struct {
	ID           string
	Name         string
	Provider     string
	ProviderName string
	Port         int
	Running      bool
	Starting     bool
	PublicURL    string
	Error        string
	StatusClass  string
	StatusText   string
}

type Server struct {
	manager    *process.Manager
	config     *config.Config
	httpServer *http.Server
	port       int
	clients    map[chan string]bool
	clientsMu  sync.RWMutex
}

func NewServer(manager *process.Manager, cfg *config.Config) *Server {
	return &Server{
		manager: manager,
		config:  cfg,
		clients: make(map[chan string]bool),
	}
}

func (s *Server) findPort() int {
	if s.config.WebPort > 0 {
		ln, err := net.Listen("tcp", fmt.Sprintf(":%d", s.config.WebPort))
		if err == nil {
			ln.Close()
			return s.config.WebPort
		}
	}
	for port := 8080; port <= 8090; port++ {
		ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err == nil {
			ln.Close()
			return port
		}
	}
	return 0
}

func (s *Server) Start() error {
	port := s.findPort()
	if port == 0 {
		return fmt.Errorf("no available port found")
	}
	s.port = port
	s.config.WebPort = port
	s.config.Save()

	mux := http.NewServeMux()

	staticFS, _ := fs.Sub(staticFiles, "static")
	mux.Handle("/", http.FileServer(http.FS(staticFS)))

	mux.HandleFunc("/api/tunnels", s.handleTunnels)
	mux.HandleFunc("/api/tunnels/", s.handleTunnelActions)
	mux.HandleFunc("/api/events", s.handleSSE)
	mux.HandleFunc("/api/copy-url", s.handleCopyURL)
	mux.HandleFunc("/api/logs/", s.handleLogs)
	mux.HandleFunc("/api/tunnels/reorder", s.handleReorder)
	mux.HandleFunc("/api/status", s.handleStatus)

	s.httpServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}

	go s.broadcastLoop()
	go s.httpServer.ListenAndServe()
	return nil
}

func (s *Server) Stop() error {
	if s.httpServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return s.httpServer.Shutdown(ctx)
	}
	return nil
}

func (s *Server) Port() int {
	return s.port
}

func (s *Server) URL() string {
	return fmt.Sprintf("http://localhost:%d", s.port)
}

func (s *Server) broadcastLoop() {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		s.clientsMu.RLock()
		if len(s.clients) == 0 {
			s.clientsMu.RUnlock()
			continue
		}
		s.clientsMu.RUnlock()

		for _, tunnel := range s.config.Tunnels {
			if status, ok := s.manager.GetStatus(tunnel.ID); ok {
				update := map[string]interface{}{
					"id":        tunnel.ID,
					"running":   status.Running,
					"starting":  status.Starting,
					"publicUrl": status.PublicURL,
					"error":     status.Error,
				}
				data, _ := json.Marshal(update)
				s.broadcast(string(data))
			}
		}
	}
}

func (s *Server) broadcast(msg string) {
	s.clientsMu.RLock()
	defer s.clientsMu.RUnlock()

	for ch := range s.clients {
		select {
		case ch <- msg:
		default:
		}
	}
}

func (s *Server) handleSSE(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	ch := make(chan string, 10)
	s.clientsMu.Lock()
	s.clients[ch] = true
	s.clientsMu.Unlock()

	defer func() {
		s.clientsMu.Lock()
		delete(s.clients, ch)
		s.clientsMu.Unlock()
		close(ch)
	}()

	ctx := r.Context()

	for {
		select {
		case msg, ok := <-ch:
			if !ok {
				return
			}
			fmt.Fprintf(w, "data: %s\n\n", msg)
			flusher.Flush()
		case <-ctx.Done():
			return
		}
	}
}

func (s *Server) handleTunnels(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.listTunnels(w, r)
	case http.MethodPost:
		s.createTunnel(w, r)
	case http.MethodPut:
		s.updateTunnel(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) listTunnels(w http.ResponseWriter, r *http.Request) {
	html, err := s.renderTunnels()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

func (s *Server) renderTunnels() (string, error) {

	sortedTunnels := make([]config.TunnelConfig, len(s.config.Tunnels))
	copy(sortedTunnels, s.config.Tunnels)
	sort.Slice(sortedTunnels, func(i, j int) bool {
		return sortedTunnels[i].Order < sortedTunnels[j].Order
	})

	var tunnels []TunnelView
	for _, t := range sortedTunnels {
		tv := TunnelView{
			ID:       t.ID,
			Name:     t.Name,
			Provider: string(t.Provider),
			Port:     t.LocalPort,
		}
		tv.ProviderName = providerName(t.Provider)

		if status, ok := s.manager.GetStatus(t.ID); ok {
			tv.Running = status.Running
			tv.Starting = status.Starting
			tv.PublicURL = status.PublicURL
			tv.Error = status.Error
		}

		tv.StatusClass = "offline"
		tv.StatusText = "Offline"
		if tv.Starting {
			tv.StatusClass = "starting"
			tv.StatusText = "Connecting..."
		} else if tv.Running {
			tv.StatusClass = "online"
			tv.StatusText = "Online"
		} else if tv.Error != "" {
			tv.StatusClass = "error"
			tv.StatusText = "Error"
		}

		tunnels = append(tunnels, tv)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, tunnels); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (s *Server) createTunnel(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	name := r.FormValue("name")
	providerStr := r.FormValue("provider")
	portStr := r.FormValue("port")

	if name == "" || providerStr == "" || portStr == "" {
		http.Error(w, "missing fields", http.StatusBadRequest)
		return
	}

	port, _ := strconv.Atoi(portStr)
	if port < 1 || port > 65535 {
		port = 30000
	}

	maxOrder := -1
	for _, t := range s.config.Tunnels {
		if t.Order > maxOrder {
			maxOrder = t.Order
		}
	}

	tunnel := config.TunnelConfig{
		ID:        fmt.Sprintf("tunnel-%d", time.Now().Unix()),
		Name:      name,
		Provider:  config.Provider(providerStr),
		LocalPort: port,
		AutoStart: false,
		Order:     maxOrder + 1,
	}

	s.config.Tunnels = append(s.config.Tunnels, tunnel)
	s.config.Save()

	s.listTunnels(w, r)
}

func (s *Server) updateTunnel(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	id := r.FormValue("id")
	name := r.FormValue("name")
	providerStr := r.FormValue("provider")
	portStr := r.FormValue("port")

	if id == "" {
		http.Error(w, "missing id", http.StatusBadRequest)
		return
	}

	for i := range s.config.Tunnels {
		if s.config.Tunnels[i].ID == id {
			if name != "" {
				s.config.Tunnels[i].Name = name
			}
			if providerStr != "" {
				s.config.Tunnels[i].Provider = config.Provider(providerStr)
			}
			if portStr != "" {
				port, _ := strconv.Atoi(portStr)
				if port >= 1 && port <= 65535 {
					s.config.Tunnels[i].LocalPort = port
				}
			}
			s.config.Save()
			s.listTunnels(w, r)
			return
		}
	}

	http.Error(w, "tunnel not found", http.StatusNotFound)
}

func (s *Server) handleTunnelActions(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/tunnels/")
	parts := strings.Split(path, "/")
	if len(parts) < 1 {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}
	id := parts[0]

	action := ""
	if len(parts) > 1 {
		action = parts[1]
	}

	switch r.Method {
	case http.MethodPost:
		switch action {
		case "start":
			s.startTunnel(w, r, id)
		case "stop":
			s.stopTunnel(w, r, id)
		default:
			http.Error(w, "unknown action", http.StatusBadRequest)
		}
	case http.MethodDelete:
		s.deleteTunnel(w, r, id)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) startTunnel(w http.ResponseWriter, r *http.Request, id string) {
	var tunnel *config.TunnelConfig
	for i := range s.config.Tunnels {
		if s.config.Tunnels[i].ID == id {
			tunnel = &s.config.Tunnels[i]
			break
		}
	}
	if tunnel == nil {
		http.Error(w, "tunnel not found", http.StatusNotFound)
		return
	}

	if needsInstall, canInstall := s.manager.CheckInstallation(tunnel.Provider); needsInstall && canInstall {
		go s.manager.InstallProvider(tunnel.Provider)
		http.Error(w, "INSTALLING PROVIDER", http.StatusAccepted)
		return
	}

	err := s.manager.Start(*tunnel, func(status config.TunnelStatus) {})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	html, _ := s.renderSingleTunnel(*tunnel)
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

func (s *Server) stopTunnel(w http.ResponseWriter, r *http.Request, id string) {
	if err := s.manager.Stop(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	for _, t := range s.config.Tunnels {
		if t.ID == id {
			html, _ := s.renderSingleTunnel(t)
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(html))
			return
		}
	}
}

func (s *Server) renderSingleTunnel(t config.TunnelConfig) (string, error) {
	tv := TunnelView{
		ID:       t.ID,
		Name:     t.Name,
		Provider: string(t.Provider),
		Port:     t.LocalPort,
	}
	tv.ProviderName = providerName(t.Provider)

	if status, ok := s.manager.GetStatus(t.ID); ok {
		tv.Running = status.Running
		tv.Starting = status.Starting
		tv.PublicURL = status.PublicURL
		tv.Error = status.Error
	}

	tv.StatusClass = "offline"
	tv.StatusText = "Offline"
	if tv.Starting {
		tv.StatusClass = "starting"
		tv.StatusText = "Connecting..."
	} else if tv.Running {
		tv.StatusClass = "online"
		tv.StatusText = "Online"
	} else if tv.Error != "" {
		tv.StatusClass = "error"
		tv.StatusText = "Error"
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, []TunnelView{tv}); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (s *Server) deleteTunnel(w http.ResponseWriter, r *http.Request, id string) {
	s.manager.Stop(id)

	for i, t := range s.config.Tunnels {
		if t.ID == id {
			s.config.Tunnels = append(s.config.Tunnels[:i], s.config.Tunnels[i+1:]...)
			break
		}
	}
	s.config.Save()
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleCopyURL(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	clipboard.Write(req.URL)
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/logs/")
	parts := strings.Split(path, "/")
	if len(parts) < 1 {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}
	id := parts[0]

	logs := s.manager.GetLogs(id)
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(strings.Join(logs, "\n")))
}

func (s *Server) handleReorder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Order []string `json:"order"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	for i, id := range req.Order {
		for j := range s.config.Tunnels {
			if s.config.Tunnels[j].ID == id {
				s.config.Tunnels[j].Order = i
				break
			}
		}
	}

	s.config.Save()
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"port":    s.port,
		"version": "0.2.0",
	})
}

func providerName(p config.Provider) string {
	names := map[config.Provider]string{
		config.ProviderCloudflared:  "Cloudflared",
		config.ProviderPlayitgg:     "Playit.gg",
		config.ProviderLocalhostRun: "localhost.run",
		config.ProviderServeo:       "Serveo",
		config.ProviderPinggy:       "Pinggy",
		config.ProviderTunnelmole:   "Tunnelmole",
	}
	if n, ok := names[p]; ok {
		return n
	}
	return strings.Title(string(p))
}

//go:embed static/*
var staticFiles embed.FS
