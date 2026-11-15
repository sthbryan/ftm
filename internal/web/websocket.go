package web

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		_, ok := allowedOrigin(r)
		return ok
	},
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
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

	// A failed deadline write leaves the connection unusable, and the next read
	// or write reports it; there is nothing to recover here.
	_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
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
			_ = client.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := client.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				s.removeClient(client)
				return
			}
		case <-pingTicker.C:
			_ = client.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := client.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				s.removeClient(client)
				return
			}
		}
	}
}

func (s *Server) handleClientMessage(client *wsClient, payload []byte) {
	var message struct {
		Type      string `json:"type"`
		ID        string `json:"id"`
		RequestID string `json:"requestId"`
	}
	if err := json.Unmarshal(payload, &message); err != nil {
		return
	}

	switch message.Type {
	case "logs_subscribe":
		if err := s.subscribeLogs(client, message.ID); err != nil {
			s.sendCommandNack(client, message.Type, message.ID, message.RequestID, err.Error())
			return
		}
		s.sendCommandAck(client, message.Type, message.ID, message.RequestID)
	case "logs_unsubscribe":
		if err := s.unsubscribeLogs(client, message.ID); err != nil {
			s.sendCommandNack(client, message.Type, message.ID, message.RequestID, err.Error())
			return
		}
		s.sendCommandAck(client, message.Type, message.ID, message.RequestID)
	default:
		s.sendCommandNack(client, message.Type, message.ID, message.RequestID, "unknown command")
	}
}

func (s *Server) sendCommandAck(client *wsClient, command, tunnelID, requestID string) {
	payload := map[string]interface{}{
		"type":    "ack",
		"command": command,
	}
	if tunnelID != "" {
		payload["id"] = tunnelID
	}
	if requestID != "" {
		payload["requestId"] = requestID
	}
	s.sendClientJSON(client, payload)
}

func (s *Server) sendCommandNack(client *wsClient, command, tunnelID, requestID, reason string) {
	payload := map[string]interface{}{
		"type":    "nack",
		"command": command,
		"reason":  reason,
	}
	if tunnelID != "" {
		payload["id"] = tunnelID
	}
	if requestID != "" {
		payload["requestId"] = requestID
	}
	s.sendClientJSON(client, payload)
}

func (s *Server) sendClientJSON(client *wsClient, payload map[string]interface{}) {
	data, err := json.Marshal(payload)
	if err != nil {
		return
	}
	if !client.enqueue(data) {
		s.removeClient(client)
	}
}

func (s *Server) subscribeLogs(client *wsClient, tunnelID string) error {
	if tunnelID == "" {
		return errors.New("missing tunnel id")
	}

	client.subsMu.Lock()
	if _, ok := client.logSubscriptions[tunnelID]; ok {
		client.subsMu.Unlock()
		return nil
	}

	logCh, unsubscribe := s.manager.SubscribeLogs(tunnelID)
	if logCh == nil {
		client.subsMu.Unlock()
		return errors.New("logs unavailable")
	}

	ctx, cancel := context.WithCancel(context.Background())
	client.logSubscriptions[tunnelID] = func() {
		cancel()
		unsubscribe()
	}
	client.subsMu.Unlock()

	go func() {
		defer func() { _ = s.unsubscribeLogs(client, tunnelID) }()
		for {
			select {
			case <-ctx.Done():
				return
			case line, ok := <-logCh:
				if !ok {
					return
				}
				payload, err := json.Marshal(map[string]interface{}{
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

	return nil
}

func (s *Server) unsubscribeLogs(client *wsClient, tunnelID string) error {
	if tunnelID == "" {
		return errors.New("missing tunnel id")
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

	return nil
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
