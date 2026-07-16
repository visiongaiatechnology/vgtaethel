package intelligence

import (
	"encoding/json"
	"sync"
	"time"
)

// EventBusMessage wraps typed events sent over the event bus
type EventBusMessage struct {
	Type      string    `json:"type"`
	SubjectID string    `json:"subject_id,omitempty"`
	Payload   any       `json:"payload,omitempty"`
	At        time.Time `json:"at"`
}

// EventBus manages asynchronous publish/subscribe channels
type EventBus struct {
	mu          sync.Mutex
	nextID      uint64
	subscribers map[uint64]chan EventBusMessage
}

func NewEventBus() *EventBus {
	return &EventBus{
		subscribers: make(map[uint64]chan EventBusMessage),
	}
}

func (b *EventBus) Subscribe() (uint64, <-chan EventBusMessage) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.nextID++
	ch := make(chan EventBusMessage, 32)
	b.subscribers[b.nextID] = ch
	return b.nextID, ch
}

func (b *EventBus) Unsubscribe(id uint64) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if ch, ok := b.subscribers[id]; ok {
		delete(b.subscribers, id)
		close(ch)
	}
}

func (b *EventBus) Publish(msg EventBusMessage) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, ch := range b.subscribers {
		select {
		case ch <- msg:
		default:
			// Non-blocking write: if subscriber is slow, drop event to prevent locking main process
		}
	}
}

func (b *EventBus) PublishEvent(eventType, subjectID string, payload any) {
	b.Publish(EventBusMessage{
		Type:      eventType,
		SubjectID: subjectID,
		Payload:   payload,
		At:        time.Now().UTC(),
	})
}

func (b *EventBus) Encode(msg EventBusMessage) ([]byte, error) {
	return json.Marshal(msg)
}
