package events

import (
	"sync"
	"sync/atomic"
)

// DefaultQueue is how many undelivered events a subscriber may fall behind by
// before the oldest ones are dropped.
const DefaultQueue = 1024

// Publisher is the side of the bus that code making changes needs.
type Publisher interface {
	Publish(Event)
}

// Bus fans every published event out to every subscriber.
type Bus struct {
	mu     sync.RWMutex
	subs   map[*subscriber]struct{}
	closed bool
}

// New returns an empty bus.
func New() *Bus {
	return &Bus{subs: make(map[*subscriber]struct{})}
}

// Option changes one subscription.
type Option func(*subscriber)

// WithQueue sets how far the subscriber may fall behind before the oldest
// undelivered events are dropped.
func WithQueue(n int) Option {
	return func(s *subscriber) {
		if n > 0 {
			s.max = n
		}
	}
}

// Only delivers just the events of the given kinds.
func Only(kinds ...Kind) Option {
	return func(s *subscriber) {
		set := make(map[Kind]bool, len(kinds))
		for _, k := range kinds {
			set[k] = true
		}
		s.accept = func(e Event) bool { return set[e.Kind()] }
	}
}

// Subscribe returns a channel that receives every event published from now on,
// in order, and a cancel that closes it. Cancel is safe to call more than once.
func (b *Bus) Subscribe(opts ...Option) (<-chan Event, func()) {
	s := &subscriber{
		max:  DefaultQueue,
		wake: make(chan struct{}, 1),
		done: make(chan struct{}),
		out:  make(chan Event),
	}
	for _, opt := range opts {
		opt(s)
	}

	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		close(s.out)
		return s.out, func() {}
	}
	b.subs[s] = struct{}{}
	b.mu.Unlock()

	go s.pump()

	cancel := func() {
		b.mu.Lock()
		delete(b.subs, s)
		b.mu.Unlock()
		s.stop()
	}
	return s.out, cancel
}

// Publish hands e to every subscriber and returns at once. It never waits for
// a subscriber to read.
func (b *Bus) Publish(e Event) {
	if b == nil || e == nil {
		return
	}

	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.closed {
		return
	}
	for s := range b.subs {
		s.enqueue(e)
	}
}

// Close cancels every subscription. Later publishes are dropped and later
// subscriptions get a closed channel.
func (b *Bus) Close() {
	b.mu.Lock()
	b.closed = true
	subs := b.subs
	b.subs = make(map[*subscriber]struct{})
	b.mu.Unlock()

	for s := range subs {
		s.stop()
	}
}

// Dropped reports how many events were dropped across every subscriber that
// fell too far behind. It is for tests and diagnostics.
func (b *Bus) Dropped() uint64 {
	b.mu.RLock()
	defer b.mu.RUnlock()
	var n uint64
	for s := range b.subs {
		n += s.dropped.Load()
	}
	return n
}

type subscriber struct {
	accept func(Event) bool
	max    int

	mu    sync.Mutex
	queue []Event

	dropped atomic.Uint64
	wake    chan struct{}
	done    chan struct{}
	once    sync.Once
	out     chan Event
}

func (s *subscriber) enqueue(e Event) {
	if s.accept != nil && !s.accept(e) {
		return
	}

	s.mu.Lock()
	if len(s.queue) >= s.max {
		s.queue[0] = nil
		s.queue = s.queue[1:]
		s.dropped.Add(1)
	}
	s.queue = append(s.queue, e)
	s.mu.Unlock()

	select {
	case s.wake <- struct{}{}:
	default:
	}
}

// pump moves queued events to the subscriber's channel, one at a time, until
// the subscription is cancelled.
func (s *subscriber) pump() {
	defer close(s.out)

	for {
		s.mu.Lock()
		if len(s.queue) == 0 {
			s.queue = nil
			s.mu.Unlock()
			select {
			case <-s.wake:
				continue
			case <-s.done:
				return
			}
		}
		e := s.queue[0]
		s.queue[0] = nil
		s.queue = s.queue[1:]
		s.mu.Unlock()

		select {
		case s.out <- e:
		case <-s.done:
			return
		}
	}
}

func (s *subscriber) stop() {
	s.once.Do(func() { close(s.done) })
}
