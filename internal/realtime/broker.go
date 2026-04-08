package realtime

import (
	"encoding/json"
	"sync"

	"github.com/google/uuid"
)

const subscriberBufferSize = 16

type Message struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data,omitempty"`
}

type ReadyPayload struct {
	ServerTime string `json:"serverTime"`
}

type subscriber struct {
	channel chan Message
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
	message, err := newMessage(eventType, payload)
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
			close(subscriber.channel)
			delete(subscribers, subscriber)
		}
	}

	if len(subscribers) == 0 {
		delete(b.subscribers, workspaceID)
	}
}

func (b *Broker) Subscribe(workspaceID uuid.UUID) (<-chan Message, func()) {
	subscription := &subscriber{
		channel: make(chan Message, subscriberBufferSize),
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

	return subscription.channel, cancel
}

func newMessage(eventType string, payload any) (Message, error) {
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
