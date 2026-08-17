package realtime

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
)

func TestSupportedCapabilitiesAdvertisesOrderUpsertMany(t *testing.T) {
	capabilities := SupportedCapabilities()
	if len(capabilities) != 1 || capabilities[0] != CapabilityOrderUpsertMany {
		t.Fatalf("SupportedCapabilities() = %v, want [%q]", capabilities, CapabilityOrderUpsertMany)
	}

	capabilities[0] = "mutated"
	if fresh := SupportedCapabilities(); fresh[0] != CapabilityOrderUpsertMany {
		t.Fatalf("SupportedCapabilities() returned mutable shared state: %v", fresh)
	}
}

func TestBrokerPublishDeliversToWorkspaceSubscribers(t *testing.T) {
	broker := NewBroker()
	workspaceID := uuid.New()
	otherWorkspaceID := uuid.New()

	messages, _, cancel := broker.Subscribe(workspaceID)
	defer cancel()

	otherMessages, _, otherCancel := broker.Subscribe(otherWorkspaceID)
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

func TestBrokerAbsorbsNormalBurstWithoutClosingSubscriber(t *testing.T) {
	broker := NewBroker()
	workspaceID := uuid.New()
	messages, _, cancel := broker.Subscribe(workspaceID)
	defer cancel()

	const burstSize = 64
	for index := 0; index < burstSize; index++ {
		broker.Publish(workspaceID, "order.deleted", map[string]int{"index": index})
	}

	for index := 0; index < burstSize; index++ {
		message, ok := <-messages
		if !ok {
			t.Fatalf("subscriber closed while draining normal burst at message %d", index)
		}
		if message.Type != "order.deleted" {
			t.Fatalf("message.Type = %q, want %q", message.Type, "order.deleted")
		}
	}

	// A subscriber that absorbed the burst must still be registered after its
	// queue is drained. The marker publish would be discarded if the channel
	// had been closed on the burst.
	broker.Publish(workspaceID, "order.deleted", map[string]int{"index": burstSize})
	select {
	case message, ok := <-messages:
		if !ok {
			t.Fatal("subscriber closed after draining a normal burst")
		}
		if message.Type != "order.deleted" {
			t.Fatalf("marker message.Type = %q, want %q", message.Type, "order.deleted")
		}
	default:
		t.Fatal("expected marker message after normal burst")
	}
}

func TestBrokerClosesSlowSubscribersAfterBufferOverflow(t *testing.T) {
	broker := NewBroker()
	workspaceID := uuid.New()

	messages, overflow, cancel := broker.Subscribe(workspaceID)
	defer cancel()

	for index := 0; index < subscriberBufferSize; index += 1 {
		broker.Publish(workspaceID, "order.deleted", map[string]int{"index": index})
	}
	broker.Publish(workspaceID, "order.deleted", map[string]int{"index": subscriberBufferSize})

	select {
	case _, ok := <-messages:
		if ok {
			t.Fatal("received a queued message after overflow; expected the queue to be discarded")
		}
	default:
		t.Fatal("subscriber channel was not closed after overflow")
	}
	select {
	case <-overflow:
	default:
		t.Fatal("overflow signal was not closed after overflow")
	}

	// Overflow removes the subscriber from the broker. Cancelling an already
	// closed subscription and publishing again must remain safe.
	cancel()
	broker.Publish(workspaceID, "order.deleted", map[string]int{"index": 0})
}

func TestNewMessageMarshalsOrderUpsertManyPayload(t *testing.T) {
	payload := struct {
		Items []map[string]string `json:"items"`
	}{
		Items: []map[string]string{
			{"id": "first"},
			{"id": "second"},
		},
	}

	message, err := NewMessage("order.upsert_many", payload)
	if err != nil {
		t.Fatalf("NewMessage() error = %v", err)
	}
	if message.Type != "order.upsert_many" {
		t.Fatalf("message.Type = %q, want %q", message.Type, "order.upsert_many")
	}

	var decoded struct {
		Items []map[string]string `json:"items"`
	}
	if err := json.Unmarshal(message.Data, &decoded); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if len(decoded.Items) != len(payload.Items) {
		t.Fatalf("decoded items = %d, want %d", len(decoded.Items), len(payload.Items))
	}
	if decoded.Items[1]["id"] != "second" {
		t.Fatalf("decoded second item = %q, want %q", decoded.Items[1]["id"], "second")
	}
}
