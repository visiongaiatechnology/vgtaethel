package intelligence

// STATUS: PLATIN
// Typed, local-only event bus. Subscribers are bounded and never block mutations.

import (
	"encoding/json"
	"sync"
	"time"
)

type IntelligenceBusEvent struct {
	Type      string              `json:"type"`
	SubjectID string              `json:"subject_id,omitempty"`
	Revision  uint64              `json:"revision"`
	At        time.Time           `json:"at"`
	Command   *GlobalWatchCommand `json:"command,omitempty"`
	Alert     *IntelligenceAlert  `json:"alert,omitempty"`
}
type GlobalWatchCommand struct {
	Action    string  `json:"action"`
	Latitude  float64 `json:"latitude,omitempty"`
	Longitude float64 `json:"longitude,omitempty"`
	Zoom      float64 `json:"zoom,omitempty"`
	Label     string  `json:"label,omitempty"`
	Layer     string  `json:"layer,omitempty"`
	Enable    *bool   `json:"enable,omitempty"`
	// Extended control surface (navigate UI, region chips, time window, report reader)
	View   string  `json:"view,omitempty"`
	Region string  `json:"region,omitempty"`
	Hours  float64 `json:"hours,omitempty"`
	Body   string  `json:"body,omitempty"`
}
type IntelligenceBus struct {
	mu          sync.Mutex
	next        uint64
	subscribers map[uint64]chan IntelligenceBusEvent
}

func NewIntelligenceBus() *IntelligenceBus {
	return &IntelligenceBus{subscribers: make(map[uint64]chan IntelligenceBusEvent)}
}
func (b *IntelligenceBus) Subscribe() (uint64, <-chan IntelligenceBusEvent) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.next++
	ch := make(chan IntelligenceBusEvent, 8)
	b.subscribers[b.next] = ch
	return b.next, ch
}
func (b *IntelligenceBus) Unsubscribe(id uint64) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if ch, ok := b.subscribers[id]; ok {
		delete(b.subscribers, id)
		close(ch)
	}
}
func (b *IntelligenceBus) Publish(event IntelligenceBusEvent) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, ch := range b.subscribers {
		select {
		case ch <- event:
		default:
		}
	}
}
func (b *IntelligenceBus) PublishGlobalWatchCommand(command GlobalWatchCommand) {
	b.Publish(IntelligenceBusEvent{Type: "global_watch.command", At: time.Now().UTC(), Command: &command})
}

func (b *IntelligenceBus) PublishGlobalWatchAlert(alert IntelligenceAlert, revision uint64) {
	b.Publish(IntelligenceBusEvent{Type: "global_watch.alert", SubjectID: alert.EventID, Revision: revision, At: time.Now().UTC(), Alert: &alert})
}
func (b *IntelligenceBus) Encode(event IntelligenceBusEvent) ([]byte, error) {
	return json.Marshal(event)
}
