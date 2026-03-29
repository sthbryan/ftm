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
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sthbryan/ftm/internal/config"
	"github.com/sthbryan/ftm/internal/process"
)

//go:embed static/*
var staticFiles embed.FS

func CheckOrigin(r *http.Request) bool {
	return true
}

var upgrader = websocket.Upgrader{
	CheckOrigin:     CheckOrigin,
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type wsClient struct {
	conn             *websocket.Conn
	send             chan []byte
	done             chan struct{}
	closeOnce        sync.Once
	subsMu           sync.Mutex
	logSubscriptions map[string]context.CancelFunc
}

func newWSClient(conn *websocket.Conn) *wsClient {
	return &wsClient{
		conn:             conn,
		send:             make(chan []byte, 256),
		done:             make(chan struct{}),
		logSubscriptions: make(map[string]context.CancelFunc),
	}
}

func (c *wsClient) close() {
	c.closeOnce.Do(func() {
		close(c.done)
	})
}

func (c *wsClient) enqueue(msg []byte) bool {
	select {
	case <-c.done:
		return false
	default:
	}
	select {
	case c.send <- msg:
		return true
	default:
		return false
	}
}

type Server struct {
	manager       *process.Manager
	config        *config.Config
	httpServer    *http.Server
	port          int
	clients       map[*wsClient]struct{}
	clientsMu     sync.RWMutex
	handlers      *Handlers
	StatusChannel chan config.TunnelStatus
}

func NewServer(manager *process.Manager, cfg *config.Config) *Server {
	s := &Server{
		manager:       manager,
		config:        cfg,
		clients:       make(map[*wsClient]struct{}),
		StatusChannel: make(chan config.TunnelStatus, 10),
	}
	s.handlers = NewHandlers(manager, cfg, s)
	return s
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

	mux := s.setupRoutes()

	s.httpServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}

	go s.installProgressLoop()
	go s.statusUpdateLoop()
	go s.httpServer.ListenAndServe()
	return nil
}

func (s *Server) setupRoutes() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/", s.handlers.Route)

	mux.HandleFunc("/ws/events", s.handleWebSocket)

	webDist := filepath.Join("web-svelte", "dist")
	var staticFS fs.FS
	if _, err := os.Stat(webDist); err == nil {
		staticFS, _ = fs.Sub(os.DirFS(webDist), ".")
	} else {
		staticFS, _ = fs.Sub(staticFiles, "static")
	}
	fileServer := http.FileServer(http.FS(staticFS))

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") || strings.HasPrefix(r.URL.Path, "/ws/") {
			return
		}
		path := r.URL.Path
		if path != "/" && !strings.Contains(path, ".") {
			r.URL.Path = "/"
		}
		fileServer.ServeHTTP(w, r)
	})

	return mux
}

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	client := newWSClient(conn)
	s.clientsMu.Lock()
	s.clients[client] = struct{}{}
	s.clientsMu.Unlock()

	defer s.removeClient(client)

	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	go s.writePump(client)

	for {
		_, payload, err := conn.ReadMessage()
		if err != nil {
			return
		}
		s.handleClientMessage(client, payload)
	}
}

func (s *Server) writePump(client *wsClient) {
	pingTicker := time.NewTicker(30 * time.Second)
	defer pingTicker.Stop()

	for {
		select {
		case <-client.done:
			return
		case msg := <-client.send:
			client.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := client.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				s.removeClient(client)
				return
			}
		case <-pingTicker.C:
			client.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := client.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				s.removeClient(client)
				return
			}
		}
	}
}

func (s *Server) handleClientMessage(client *wsClient, payload []byte) {
	var message struct {
		Type string `json:"type"`
		ID   string `json:"id"`
	}
	if err := json.Unmarshal(payload, &message); err != nil {
		return
	}

	switch message.Type {
	case "logs_subscribe":
		s.subscribeLogs(client, message.ID)
	case "logs_unsubscribe":
		s.unsubscribeLogs(client, message.ID)
	}
}

func (s *Server) subscribeLogs(client *wsClient, tunnelID string) {
	if tunnelID == "" {
		return
	}

	client.subsMu.Lock()
	if _, ok := client.logSubscriptions[tunnelID]; ok {
		client.subsMu.Unlock()
		return
	}

	logCh, unsubscribe := s.manager.SubscribeLogs(tunnelID)
	if logCh == nil {
		client.subsMu.Unlock()
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	client.logSubscriptions[tunnelID] = func() {
		cancel()
		unsubscribe()
	}
	client.subsMu.Unlock()

	go func() {
		defer s.unsubscribeLogs(client, tunnelID)
		for {
			select {
			case <-ctx.Done():
				return
			case line, ok := <-logCh:
				if !ok {
					return
				}
				payload, err := MarshalJSON(map[string]interface{}{
					"type": "log",
					"id":   tunnelID,
					"line": line,
				})
				if err != nil {
					continue
				}
				if !client.enqueue(payload) {
					s.removeClient(client)
					return
				}
			}
		}
	}()
}

func (s *Server) unsubscribeLogs(client *wsClient, tunnelID string) {
	if tunnelID == "" {
		return
	}

	client.subsMu.Lock()
	cancel, ok := client.logSubscriptions[tunnelID]
	if ok {
		delete(client.logSubscriptions, tunnelID)
	}
	client.subsMu.Unlock()

	if ok {
		cancel()
	}
}

func (s *Server) removeClient(client *wsClient) {
	client.close()

	s.clientsMu.Lock()
	delete(s.clients, client)
	s.clientsMu.Unlock()

	client.subsMu.Lock()
	cancels := make([]context.CancelFunc, 0, len(client.logSubscriptions))
	for tunnelID, cancel := range client.logSubscriptions {
		cancels = append(cancels, cancel)
		delete(client.logSubscriptions, tunnelID)
	}
	client.subsMu.Unlock()

	for _, cancel := range cancels {
		cancel()
	}

	client.conn.Close()
}

func (s *Server) Stop() error {
	if s.httpServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*5e9)
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

func (s *Server) installProgressLoop() {
	for progress := range s.manager.DownloadProgress {
		update := map[string]interface{}{
			"type":    "install",
			"percent": progress.Percent,
			"current": progress.Current,
			"total":   progress.Total,
			"done":    progress.Done,
		}
		data, _ := MarshalJSON(update)
		s.broadcast(string(data))
	}
}

func (s *Server) statusUpdateLoop() {
	for status := range s.StatusChannel {
		update := map[string]interface{}{
			"id":           status.ID,
			"state":        string(status.State),
			"publicUrl":    status.PublicURL,
			"errorMessage": status.ErrorMessage,
		}
		data, _ := MarshalJSON(update)
		s.broadcast(string(data))
	}
}

func (s *Server) broadcast(msg string) {
	payload := []byte(msg)

	s.clientsMu.RLock()
	clients := make([]*wsClient, 0, len(s.clients))
	for client := range s.clients {
		clients = append(clients, client)
	}
	s.clientsMu.RUnlock()

	for _, client := range clients {
		if !client.enqueue(payload) {
			s.removeClient(client)
		}
	}
}

func (s *Server) BroadcastTunnelUpdate(t config.TunnelConfig) {
	state := "stopped"
	var publicURL, errorMessage string

	if status, ok := s.manager.GetStatus(t.ID); ok {
		state = string(status.State)
		publicURL = status.PublicURL
		errorMessage = status.ErrorMessage
	}

	update := map[string]interface{}{
		"id":           t.ID,
		"name":         t.Name,
		"provider":     string(t.Provider),
		"port":         t.LocalPort,
		"state":        state,
		"publicUrl":    publicURL,
		"errorMessage": errorMessage,
	}
	data, _ := MarshalJSON(update)
	s.broadcast(string(data))
}

func (s *Server) getTunnel(id string) *config.TunnelConfig {
	for i := range s.config.Tunnels {
		if s.config.Tunnels[i].ID == id {
			return &s.config.Tunnels[i]
		}
	}
	return nil
}

func (s *Server) updateConfig() {
	s.config.Save()
}
