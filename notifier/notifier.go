package notifier

import (
	"encoding/json"
	"sync"

	"github.com/Emmanuel326/tenderwatchafrica/store"
)

// Event is what gets pushed to connected clients
type Event struct {
	Type   string       `json:"type"` // "new" | "updated"
	Tender store.Tender `json:"tender"`
}

// Notifier broadcasts tender events to subscribed SSE clients
type Notifier struct {
	mu          sync.RWMutex
	subscribers map[chan []byte]struct{}
}

// New creates a new Notifier
func New() *Notifier {
	return &Notifier{
		subscribers: make(map[chan []byte]struct{}),
	}
}

// Subscribe registers a new SSE client channel.
// The caller is responsible for calling Unsubscribe when done.
func (n *Notifier) Subscribe() chan []byte {
	ch := make(chan []byte, 32)
	n.mu.Lock()
	n.subscribers[ch] = struct{}{}
	n.mu.Unlock()
	return ch
}

// Unsubscribe removes and closes a client channel.
func (n *Notifier) Unsubscribe(ch chan []byte) {
	n.mu.Lock()
	delete(n.subscribers, ch)
	n.mu.Unlock()
	close(ch)
}

// Broadcast sends tender events to all connected subscribers.
// It drops events for slow clients rather than blocking the pipeline.
func (n *Notifier) Broadcast(events []store.TenderEvent) {
	if len(events) == 0 {
		return
	}

	n.mu.RLock()
	defer n.mu.RUnlock()

	for _, e := range events {
		payload, err := json.Marshal(Event{
			Type:   e.Type,
			Tender: e.Tender,
		})
		if err != nil {
			continue
		}

		for ch := range n.subscribers {
			select {
			case ch <- payload:
			default:
				// slow client, drop rather than block
			}
		}
	}
}
