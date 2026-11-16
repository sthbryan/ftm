package proxy

import (
	"sync"
	"time"
)

type counter struct {
	mu       sync.Mutex
	window   time.Duration
	lastSeen map[string]time.Time
	known    map[string]struct{}
	sessions int
	requests int64
	now      func() time.Time
}

func newCounter(window time.Duration) *counter {
	if window <= 0 {
		window = 5 * time.Minute
	}

	return &counter{
		window:   window,
		lastSeen: make(map[string]time.Time),
		known:    make(map[string]struct{}),
		now:      time.Now,
	}
}

func (c *counter) seen(key string) (isNew bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.requests++
	c.lastSeen[key] = c.now()
	c.pruneLocked()

	if _, ok := c.known[key]; ok {
		return false
	}

	c.known[key] = struct{}{}

	return true
}

func (c *counter) openSession() {
	c.mu.Lock()
	c.sessions++
	c.mu.Unlock()
}

func (c *counter) closeSession() {
	c.mu.Lock()
	if c.sessions > 0 {
		c.sessions--
	}
	c.mu.Unlock()
}

func (c *counter) stats() Stats {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.pruneLocked()

	return Stats{
		ActiveSessions: c.sessions,
		Visitors:       len(c.lastSeen),
		Requests:       c.requests,
	}
}

func (c *counter) pruneLocked() {
	cutoff := c.now().Add(-c.window)

	for key, at := range c.lastSeen {
		if at.Before(cutoff) {
			delete(c.lastSeen, key)
		}
	}
}
