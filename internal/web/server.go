package web

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/sthbryan/ftm/internal/clipboard"
	"github.com/sthbryan/ftm/internal/config"
	"github.com/sthbryan/ftm/internal/process"
	"github.com/sthbryan/ftm/internal/version"
)

type TunnelView struct {
	ID           string
	Name         string
	Provider     string
	ProviderName string
	Port         int
	State        string
	PublicURL    string
	ErrorMessage string
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
		return s.config.WebPort
	}
	for port := 40500; port <= 40550; port++ {
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

	webDist := filepath.Join("web-svelte", "dist")
	var staticFS fs.FS
	if _, err := os.Stat(webDist); err == nil {
		staticFS, _ = fs.Sub(os.DirFS(webDist), ".")
	} else {
		staticFS, _ = fs.Sub(staticFiles, "static")
	}
	fileServer := http.FileServer(http.FS(staticFS))

	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		switch {
		case r.URL.Path == "/api/tunnels" || r.URL.Path == "/api/tunnels/":
			s.handleTunnels(w, r)
		case r.URL.Path == "/api/events":
			s.handleSSE(w, r)
		case r.URL.Path == "/api/copy-url":
			s.handleCopyURL(w, r)
		case strings.HasPrefix(r.URL.Path, "/api/logs/"):
			if strings.HasSuffix(r.URL.Path, "/stream") {
				s.handleLogsStream(w, r)
			} else {
				s.handleLogs(w, r)
			}
		case r.URL.Path == "/api/tunnels/reorder":
			s.handleReorder(w, r)
		case r.URL.Path == "/api/status":
			s.handleStatus(w)
		case r.URL.Path == "/api/providers":
			s.handleProviders(w)
		case r.URL.Path == "/api/detect-port":
			s.handleDetectPort(w)
		case strings.HasPrefix(r.URL.Path, "/api/tunnels/"):
			s.handleTunnelActions(w, r)
		default:
			http.NotFound(w, r)
		}
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			return
		}

		path := r.URL.Path
		if path != "/" && !strings.Contains(path, ".") {
			r.URL.Path = "/"
		}

		fileServer.ServeHTTP(w, r)
	})

	s.httpServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}

	go s.broadcastLoop()
	go s.installProgressLoop()
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
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		s.clientsMu.RLock()
		if len(s.clients) == 0 {
			s.clientsMu.RUnlock()
			continue
		}
		s.clientsMu.RUnlock()

		for _, tunnel := range s.config.Tunnels {
			status, ok := s.manager.GetStatus(tunnel.ID)
			if !ok {
				update := map[string]interface{}{
					"id":        tunnel.ID,
					"state":     "stopped",
					"publicUrl": "",
				}
				data, _ := json.Marshal(update)
				s.broadcast(string(data))
				continue
			}

			update := map[string]interface{}{
				"id":           tunnel.ID,
				"state":        string(status.State),
				"publicUrl":    status.PublicURL,
				"errorMessage": status.ErrorMessage,
			}
			data, _ := json.Marshal(update)
			s.broadcast(string(data))
		}
	}
}

func (s *Server) installProgressLoop() {
	for progress := range s.manager.DownloadProgress {
		update := map[string]interface{}{
			"type":    "install",
			"percent": progress.Percent,
			"current": progress.Current,
			"total":   progress.Total,
			"done":    progress.Done,
		}
		data, _ := json.Marshal(update)
		s.broadcast(string(data))
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
		s.listTunnels(w)
		return
	case http.MethodPost:
		s.createTunnel(w, r)
		return
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) listTunnels(w http.ResponseWriter) {
	sortedTunnels := make([]config.TunnelConfig, len(s.config.Tunnels))
	copy(sortedTunnels, s.config.Tunnels)

	var result []map[string]interface{}
	for _, t := range sortedTunnels {
		item := map[string]interface{}{
			"id":       t.ID,
			"name":     t.Name,
			"provider": string(t.Provider),
			"port":     t.LocalPort,
			"status":   "stopped",
		}

		if status, ok := s.manager.GetStatus(t.ID); ok {
			item["publicUrl"] = status.PublicURL
			item["errorMessage"] = status.ErrorMessage
			item["state"] = string(status.State)
		}

		if needsInstall, canInstall := s.manager.CheckInstallation(t.Provider); needsInstall && canInstall {
			item["status"] = "installing"
			item["installProgress"] = 0
		}

		result = append(result, item)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (s *Server) createTunnel(w http.ResponseWriter, r *http.Request) {
	var name, providerStr string
	var port int

	contentType := r.Header.Get("Content-Type")
	if strings.Contains(contentType, "application/json") {
		var req struct {
			Name      string `json:"name"`
			Provider  string `json:"provider"`
			LocalPort int    `json:"localPort"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		name = req.Name
		providerStr = req.Provider
		port = req.LocalPort
	} else {
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		name = r.FormValue("name")
		providerStr = r.FormValue("provider")
		portStr := r.FormValue("port")
		port, _ = strconv.Atoi(portStr)
	}

	if name == "" || providerStr == "" || port < 1 {
		http.Error(w, "missing fields", http.StatusBadRequest)
		return
	}

	if port < 1 || port > 65535 {
		port = 30000
	}

	tunnel := config.TunnelConfig{
		ID:        fmt.Sprintf("tunnel-%d", time.Now().Unix()),
		Name:      name,
		Provider:  config.Provider(providerStr),
		LocalPort: port,
	}

	s.config.Tunnels = append(s.config.Tunnels, tunnel)
	s.config.Save()

	update := map[string]interface{}{
		"id":       tunnel.ID,
		"name":     tunnel.Name,
		"provider": string(tunnel.Provider),
		"port":     tunnel.LocalPort,
		"state":    "stopped",
	}
	data, _ := json.Marshal(update)
	s.broadcast(string(data))

	s.writeTunnelJSON(w, tunnel)
}

func (s *Server) updateTunnel(w http.ResponseWriter, r *http.Request, id string) {
	var name, providerStr string
	var port int

	contentType := r.Header.Get("Content-Type")
	if strings.Contains(contentType, "application/json") {
		var req struct {
			Name      string `json:"name"`
			Provider  string `json:"provider"`
			LocalPort int    `json:"localPort"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		name = req.Name
		providerStr = req.Provider
		port = req.LocalPort
	} else {
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		name = r.FormValue("name")
		providerStr = r.FormValue("provider")
		portStr := r.FormValue("port")
		port, _ = strconv.Atoi(portStr)
	}

	for i := range s.config.Tunnels {
		if s.config.Tunnels[i].ID == id {
			if name != "" {
				s.config.Tunnels[i].Name = name
			}
			if providerStr != "" {
				s.config.Tunnels[i].Provider = config.Provider(providerStr)
			}
			if port >= 1 && port <= 65535 {
				s.config.Tunnels[i].LocalPort = port
			}
			s.config.Save()

			update := map[string]interface{}{
				"id":       s.config.Tunnels[i].ID,
				"name":     s.config.Tunnels[i].Name,
				"provider": string(s.config.Tunnels[i].Provider),
				"port":     s.config.Tunnels[i].LocalPort,
			}
			if status, ok := s.manager.GetStatus(id); ok {
				update["state"] = string(status.State)
				update["publicUrl"] = status.PublicURL
			}
			data, _ := json.Marshal(update)
			s.broadcast(string(data))

			s.writeTunnelJSON(w, s.config.Tunnels[i])
			return
		}
	}

	http.Error(w, "tunnel not found", http.StatusNotFound)
}

func (s *Server) getTunnel(w http.ResponseWriter, id string) {
	for _, t := range s.config.Tunnels {
		if t.ID == id {
			s.writeTunnelJSON(w, t)
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
	case http.MethodGet:
		if action == "" {
			s.getTunnel(w, id)
		} else {
			http.Error(w, "unknown action", http.StatusBadRequest)
		}
	case http.MethodPut:
		if action == "" {
			s.updateTunnel(w, r, id)
		} else {
			http.Error(w, "unknown action", http.StatusBadRequest)
		}
	case http.MethodPost:
		switch action {
		case "start":
			s.startTunnel(w, id)
		case "stop":
			s.stopTunnel(w, id)
		default:
			http.Error(w, "unknown action", http.StatusBadRequest)
		}
	case http.MethodDelete:
		s.deleteTunnel(w, id)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) startTunnel(w http.ResponseWriter, id string) {
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
		update := map[string]interface{}{
			"id":       tunnel.ID,
			"status":   "installing",
			"provider": string(tunnel.Provider),
		}
		data, _ := json.Marshal(update)
		s.broadcast(string(data))

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":       tunnel.ID,
			"name":     tunnel.Name,
			"provider": string(tunnel.Provider),
			"port":     tunnel.LocalPort,
			"status":   "installing",
		})

		go func() {
			if err := s.manager.InstallProvider(tunnel.Provider); err != nil {
				update := map[string]interface{}{
					"id":     tunnel.ID,
					"status": "error",
					"error":  "Installation failed: " + err.Error(),
				}
				data, _ := json.Marshal(update)
				s.broadcast(string(data))
				return
			}

			if err := s.manager.Start(*tunnel, func(status config.TunnelStatus) {}); err != nil {
				update := map[string]interface{}{
					"id":     tunnel.ID,
					"status": "error",
					"error":  err.Error(),
				}
				data, _ := json.Marshal(update)
				s.broadcast(string(data))
			}
		}()

		return
	}

	err := s.manager.Start(*tunnel, func(status config.TunnelStatus) {})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.writeTunnelJSON(w, *tunnel)
}

func (s *Server) stopTunnel(w http.ResponseWriter, id string) {
	if err := s.manager.Stop(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	update := map[string]interface{}{
		"id":        id,
		"state":     "stopped",
		"publicUrl": "",
	}
	data, _ := json.Marshal(update)
	s.broadcast(string(data))

	for _, t := range s.config.Tunnels {
		if t.ID == id {
			s.writeTunnelJSON(w, t)
			return
		}
	}
}

func (s *Server) writeTunnelJSON(w http.ResponseWriter, t config.TunnelConfig) {
	status := "stopped"
	var publicURL, errorMsg string

	if tunnelStatus, ok := s.manager.GetStatus(t.ID); ok {
		publicURL = tunnelStatus.PublicURL
		errorMsg = tunnelStatus.ErrorMessage
		status = string(tunnelStatus.State)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":        t.ID,
		"name":      t.Name,
		"provider":  string(t.Provider),
		"port":      t.LocalPort,
		"status":    status,
		"publicUrl": publicURL,
		"error":     errorMsg,
	})
}

func (s *Server) deleteTunnel(w http.ResponseWriter, id string) {
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

func (s *Server) handleLogsStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/logs/")
	path = strings.TrimSuffix(path, "/stream")
	parts := strings.Split(path, "/")
	if len(parts) < 1 {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}
	id := parts[0]

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	initialLogs := s.manager.GetLogs(id)
	for _, line := range initialLogs {
		fmt.Fprintf(w, "data: %s\n\n", line)
		flusher.Flush()
	}

	logChan := s.manager.SubscribeLogs(id)
	if logChan == nil {

		fmt.Fprintf(w, "data: [tunnel stopped]\n\n")
		flusher.Flush()
		return
	}

	for line := range logChan {
		fmt.Fprintf(w, "data: %s\n\n", line)
		flusher.Flush()
	}
}

func (s *Server) handleReorder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if err := json.NewDecoder(r.Body).Decode(&r); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.config.Save()
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleStatus(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"port":    s.port,
		"version": version.Version,
	})
}

func (s *Server) handleProviders(w http.ResponseWriter) {
	providers := []map[string]string{
		{"id": "cloudflared", "name": "Cloudflared"},
		{"id": "tunnelmole", "name": "Tunnelmole"},
		{"id": "localhostrun", "name": "localhost.run"},
		{"id": "serveo", "name": "Serveo"},
		{"id": "pinggy", "name": "Pinggy"},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(providers)
}

func (s *Server) handleDetectPort(w http.ResponseWriter) {
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

//go:embed static/*
var staticFiles embed.FS
