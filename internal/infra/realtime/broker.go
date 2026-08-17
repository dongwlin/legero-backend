package realtime

import (
	"encoding/json"
	"sync"

	"github.com/google/uuid"
)

// subscriberBufferSize absorbs short event bursts without treating an
// otherwise healthy subscriber as slow. Once it is genuinely full, Publish
// still closes that subscriber rather than dropping an event silently.
const subscriberBufferSize = 128

type Message struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data,omitempty"`
}

// CapabilityOrderUpsertMany identifies the compact batch order-upsert event.
// Clients opt into receiving this event through the WebSocket capabilities
// query parameter; clients that do not opt in receive legacy order.upsert
// messages instead.
const CapabilityOrderUpsertMany = "order.upsert_many"

type ReadyPayload struct {
	ServerTime   string   `json:"serverTime"`
	Capabilities []string `json:"capabilities,omitempty"`
}

// SupportedCapabilities returns the capabilities this server can publish on a
// realtime connection. Return a fresh slice so callers cannot mutate the
// server's advertised protocol contract.
func SupportedCapabilities() []string {
	return []string{CapabilityOrderUpsertMany}
}

// HeartbeatPayload is the lightweight application-level heartbeat pushed on
// the write loop so clients can refresh lastServerActivityAt and detect
// half-open connections without reading protocol-level control frames. It
// carries the same serverTime shape as ReadyPayload.
type HeartbeatPayload = ReadyPayload

type subscriber struct {
	channel  chan Message
	overflow chan struct{}
}

type Broker struct {
	mu          sync.Mutex
	subscribers map[uuid.UUID]map[*subscriber]struct{}
}

func NewBroker() *Broker {
	return &Broker{
		subscribers: make(map[uuid.UUID]map[*subscriber]struct{}),
	}
}

func (b *Broker) Publish(workspaceID uuid.UUID, eventType string, payload any) {
	message, err := NewMessage(eventType, payload)
	if err != nil {
		return
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	subscribers := b.subscribers[workspaceID]
	for subscriber := range subscribers {
		select {
		case subscriber.channel <- message:
		default:
			// The subscription is already behind, so queued messages no longer
			// form a complete stream. Signal overflow before closing the message
			// channel so writeLoop can stop even if a stale message is buffered.
			close(subscriber.overflow)
			// Discard the stale queue as well. This bounds the time and memory
			// spent on a subscription that already needs to reconnect.
			discardQueuedMessages(subscriber.channel)
			close(subscriber.channel)
			delete(subscribers, subscriber)
		}
	}

	if len(subscribers) == 0 {
		delete(b.subscribers, workspaceID)
	}
}

func discardQueuedMessages(messages <-chan Message) {
	for {
		select {
		case <-messages:
		default:
			return
		}
	}
}

func (b *Broker) Subscribe(workspaceID uuid.UUID) (<-chan Message, <-chan struct{}, func()) {
	subscription := &subscriber{
		channel:  make(chan Message, subscriberBufferSize),
		overflow: make(chan struct{}),
	}

	b.mu.Lock()
	if _, ok := b.subscribers[workspaceID]; !ok {
		b.subscribers[workspaceID] = make(map[*subscriber]struct{})
	}
	b.subscribers[workspaceID][subscription] = struct{}{}
	b.mu.Unlock()

	var once sync.Once
	cancel := func() {
		once.Do(func() {
			b.mu.Lock()
			defer b.mu.Unlock()

			if subscribers, ok := b.subscribers[workspaceID]; ok {
				if _, exists := subscribers[subscription]; exists {
					delete(subscribers, subscription)
					close(subscription.channel)
				}
				if len(subscribers) == 0 {
					delete(b.subscribers, workspaceID)
				}
			}
		})
	}

	return subscription.channel, subscription.overflow, cancel
}

func NewMessage(eventType string, payload any) (Message, error) {
	if payload == nil {
		return Message{Type: eventType}, nil
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return Message{}, err
	}

	return Message{
		Type: eventType,
		Data: json.RawMessage(body),
	}, nil
}
