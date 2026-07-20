// Package broker provides an in-memory publish/subscribe hub used to fan out newly
// ingested log entries to active SSE subscribers, keyed by project ID.
package broker

import (
	"sync"

	"github.com/jmstewart1127/openlogs/internal/db"
)

// subChanBuffer is the per-subscriber buffer. If a subscriber falls this far
// behind, further events are dropped for that subscriber rather than blocking
// the ingest path. The client re-syncs from the database on its next page load.
const subChanBuffer = 16

// Broker fans out log events to subscribers grouped by project ID.
type Broker struct {
	mu   sync.RWMutex
	subs map[string]map[chan db.Log]struct{}
}

// New creates an empty Broker.
func New() *Broker {
	return &Broker{subs: make(map[string]map[chan db.Log]struct{})}
}

// Subscribe registers a new subscriber for a project and returns its channel.
// The caller MUST call Unsubscribe with the same channel when done.
func (b *Broker) Subscribe(projectID string) chan db.Log {
	ch := make(chan db.Log, subChanBuffer)
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.subs[projectID] == nil {
		b.subs[projectID] = make(map[chan db.Log]struct{})
	}
	b.subs[projectID][ch] = struct{}{}
	return ch
}

// Unsubscribe removes a subscriber and closes its channel.
func (b *Broker) Unsubscribe(projectID string, ch chan db.Log) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if set, ok := b.subs[projectID]; ok {
		if _, ok := set[ch]; ok {
			delete(set, ch)
			close(ch)
		}
		if len(set) == 0 {
			delete(b.subs, projectID)
		}
	}
}

// Publish delivers a log event to all subscribers of its project. Delivery is
// non-blocking: if a subscriber's buffer is full the event is dropped for that
// subscriber. Publishing to a project with no subscribers is a no-op.
func (b *Broker) Publish(projectID string, log db.Log) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for ch := range b.subs[projectID] {
		select {
		case ch <- log:
		default:
			// Subscriber is behind; drop rather than block ingest.
		}
	}
}
