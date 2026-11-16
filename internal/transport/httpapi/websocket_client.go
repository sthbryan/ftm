package httpapi

import (
	"context"
	"sync"

	"github.com/gorilla/websocket"
)

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
