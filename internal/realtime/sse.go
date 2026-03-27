package realtime

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type event struct {
	Type string
	Data []byte
}

type Hub struct {
	mu          sync.RWMutex
	subscribers map[uuid.UUID]map[chan event]struct{}
	location    *time.Location
	pingEvery   time.Duration
	now         func() time.Time
}

func NewHub(location *time.Location, pingEvery time.Duration, now func() time.Time) *Hub {
	return &Hub{
		subscribers: make(map[uuid.UUID]map[chan event]struct{}),
		location:    location,
		pingEvery:   pingEvery,
		now:         now,
	}
}

func (h *Hub) Publish(workspaceID uuid.UUID, eventType string, payload any) {
	body, err := json.Marshal(payload)
	if err != nil {
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for subscriber := range h.subscribers[workspaceID] {
		select {
		case subscriber <- event{Type: eventType, Data: body}:
		default:
		}
	}
}

func (h *Hub) Serve(c *gin.Context, workspaceID uuid.UUID) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.Status(http.StatusInternalServerError)
		return
	}

	channel, cancel := h.subscribe(workspaceID)
	defer cancel()

	ticker := time.NewTicker(h.pingEvery)
	defer ticker.Stop()

	writeEvent(c.Writer, "ping", map[string]string{
		"serverTime": formatTime(h.now(), h.location),
	})
	flusher.Flush()

	for {
		select {
		case <-c.Request.Context().Done():
			return
		case message := <-channel:
			if err := writeEvent(c.Writer, message.Type, json.RawMessage(message.Data)); err != nil {
				return
			}
			flusher.Flush()
		case <-ticker.C:
			if err := writeEvent(c.Writer, "ping", map[string]string{
				"serverTime": formatTime(h.now(), h.location),
			}); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

func (h *Hub) subscribe(workspaceID uuid.UUID) (chan event, func()) {
	channel := make(chan event, 16)

	h.mu.Lock()
	if _, ok := h.subscribers[workspaceID]; !ok {
		h.subscribers[workspaceID] = make(map[chan event]struct{})
	}
	h.subscribers[workspaceID][channel] = struct{}{}
	h.mu.Unlock()

	return channel, func() {
		h.mu.Lock()
		defer h.mu.Unlock()
		if subscribers, ok := h.subscribers[workspaceID]; ok {
			delete(subscribers, channel)
			if len(subscribers) == 0 {
				delete(h.subscribers, workspaceID)
			}
		}
		close(channel)
	}
}

func writeEvent(writer http.ResponseWriter, eventType string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	if _, err := fmt.Fprintf(writer, "event: %s\n", eventType); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(writer, "data: %s\n\n", body); err != nil {
		return err
	}
	return nil
}

func formatTime(value time.Time, location *time.Location) string {
	if location == nil {
		return value.Format(time.RFC3339)
	}
	return value.In(location).Format(time.RFC3339)
}
