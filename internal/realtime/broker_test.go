package realtime

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
)

func TestBrokerPublishDeliversToWorkspaceSubscribers(t *testing.T) {
	broker := NewBroker()
	workspaceID := uuid.New()
	otherWorkspaceID := uuid.New()

	messages, cancel := broker.Subscribe(workspaceID)
	defer cancel()

	otherMessages, otherCancel := broker.Subscribe(otherWorkspaceID)
	defer otherCancel()

	payload := map[string]string{"id": "123"}
	broker.Publish(workspaceID, "order.deleted", payload)

	select {
	case message := <-messages:
		if message.Type != "order.deleted" {
			t.Fatalf("message.Type = %q, want %q", message.Type, "order.deleted")
		}

		var decoded map[string]string
		if err := json.Unmarshal(message.Data, &decoded); err != nil {
			t.Fatalf("Unmarshal() error = %v", err)
		}
		if decoded["id"] != "123" {
			t.Fatalf("decoded[id] = %q, want %q", decoded["id"], "123")
		}
	default:
		t.Fatal("expected message for subscribed workspace")
	}

	select {
	case message := <-otherMessages:
		t.Fatalf("unexpected message for other workspace: %+v", message)
	default:
	}
}

func TestBrokerClosesSlowSubscribers(t *testing.T) {
	broker := NewBroker()
	workspaceID := uuid.New()

	messages, _ := broker.Subscribe(workspaceID)

	for index := 0; index < subscriberBufferSize; index += 1 {
		broker.Publish(workspaceID, "order.deleted", map[string]int{"index": index})
	}
	broker.Publish(workspaceID, "order.deleted", map[string]int{"index": subscriberBufferSize})

	received := 0
	for range messages {
		received += 1
	}

	if received != subscriberBufferSize {
		t.Fatalf("received = %d, want %d", received, subscriberBufferSize)
	}
}
