package httpapi

import (
	"encoding/json"
	"log"
	"sync"
	"time"
)

const (
	// replaySize is how many recent events an SSE client can resume from.
	replaySize = 512
	// sseQueue is how far an SSE client may fall behind before it is dropped.
	// It reconnects with Last-Event-ID and resumes from the replay buffer.
	sseQueue = 256
)

// frame is one encoded event.
type frame struct {
	ID    int64
	Event string
	Data  []byte
}

// hub fans every broadcast out to the SSE clients of /api/events and keeps
// the recent ones so an SSE client that reconnects resumes where it left off.
//
// Event ids start at the server's start time in microseconds and grow by one
// per event, so an id from before a restart is always older than any id after
// it and is detected as a gap instead of being mistaken for a recent event.
type hub struct {
	mu   sync.Mutex
	last int64
	ring []frame
	sse  map[*sseClient]struct{}
}

func newHub() *hub {
	return &hub{
		last: time.Now().UnixMicro(),
		sse:  make(map[*sseClient]struct{}),
	}
}

// publish encodes v and hands it to every client. It never blocks on a slow
// client: one whose queue is full is dropped.
func (h *hub) publish(event string, v any) {
	data, err := json.Marshal(v)
	if err != nil {
		log.Printf("httpapi: could not encode a %T update: %v", v, err)
		return
	}

	// Delivery happens under the lock so every client sees frames in id
	// order; each send is non-blocking.
	h.mu.Lock()
	h.last++
	f := frame{ID: h.last, Event: event, Data: data}
	h.ring = append(h.ring, f)
	if len(h.ring) > replaySize {
		h.ring = append(h.ring[:0:0], h.ring[len(h.ring)-replaySize:]...)
	}
	for c := range h.sse {
		if !c.offer(f) {
			delete(h.sse, c)
			c.close()
		}
	}
	h.mu.Unlock()
}

// subscribe registers an SSE client. When resume is set, replay holds the
// frames after lastID, or gap is set when some of them are gone.
func (h *hub) subscribe(lastID int64, resume bool) (c *sseClient, replay []frame, gap bool) {
	c = &sseClient{ch: make(chan frame, sseQueue), done: make(chan struct{})}

	h.mu.Lock()
	defer h.mu.Unlock()

	if resume {
		replay, gap = h.since(lastID)
	}
	h.sse[c] = struct{}{}
	return c, replay, gap
}

// since returns the frames after id, or gap when some of them are no longer
// kept or id is not one this hub handed out.
func (h *hub) since(id int64) (frames []frame, gap bool) {
	switch {
	case id == h.last:
		return nil, false
	case id > h.last:
		// From a server with a clock ahead of ours, or made up.
		return nil, true
	case len(h.ring) == 0 || id < h.ring[0].ID-1:
		return nil, true
	}

	start := len(h.ring)
	for i, f := range h.ring {
		if f.ID > id {
			start = i
			break
		}
	}
	return append([]frame(nil), h.ring[start:]...), false
}

func (h *hub) unsubscribe(c *sseClient) {
	h.mu.Lock()
	delete(h.sse, c)
	h.mu.Unlock()
	c.close()
}

// clients counts the open /api/events streams.
func (h *hub) clients() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.sse)
}

// sseClient is one open /api/events stream.
type sseClient struct {
	ch   chan frame
	done chan struct{}
	once sync.Once
}

func (c *sseClient) offer(f frame) bool {
	select {
	case c.ch <- f:
		return true
	default:
		return false
	}
}

func (c *sseClient) close() {
	c.once.Do(func() { close(c.done) })
}

// lastID is the id of the newest event.
func (h *hub) lastID() int64 {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.last
}
