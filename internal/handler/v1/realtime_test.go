package v1

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"

	"github.com/dongwlin/legero-backend/internal/infra/identity"
	"github.com/dongwlin/legero-backend/internal/infra/realtime"
)

func newTestRealtime(t *testing.T, heartbeatInterval time.Duration) *Realtime {
	t.Helper()
	return &Realtime{
		heartbeatInterval: heartbeatInterval,
		writeTimeout:      5 * time.Second,
		readTimeout:       time.Minute,
		now:               time.Now,
		location:          time.FixedZone("UTC+8", 8*3600),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(*http.Request) bool {
				return true
			},
		},
	}
}

// connectWS dials a running httptest server as a websocket client. The
// caller owns the returned connection and must close it before the server.
func connectWS(t *testing.T, serverURL string) *websocket.Conn {
	t.Helper()
	wsURL := "ws" + strings.TrimPrefix(serverURL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	return conn
}

func TestWriteLoopSendsHeartbeatAndPing(t *testing.T) {
	h := newTestRealtime(t, 20*time.Millisecond)
	messages := make(chan realtime.Message)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := h.upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		_ = h.writeLoop(conn, messages)
	}))
	defer func() {
		close(messages)
		server.Close()
	}()

	conn := connectWS(t, server.URL)
	defer conn.Close()

	// Protocol pings arrive as control frames, so count them in a custom
	// ping handler; application heartbeats arrive as JSON data messages.
	// Each tick writes the ping before the heartbeat, so by the time a
	// heartbeat message is read its ping handler has already run.
	var pings atomic.Int32
	conn.SetPingHandler(func(string) error {
		pings.Add(1)
		return nil
	})

	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))

	heartbeats := 0
	var serverTime time.Time
	for heartbeats < 2 {
		_, payload, readErr := conn.ReadMessage()
		if readErr != nil {
			break
		}

		var msg realtime.Message
		if err := json.Unmarshal(payload, &msg); err != nil {
			t.Fatalf("unmarshal message %q: %v", payload, err)
		}
		if msg.Type != "heartbeat" {
			t.Errorf("got message type %q, want %q", msg.Type, "heartbeat")
			continue
		}
		heartbeats++

		var data struct {
			ServerTime string `json:"serverTime"`
		}
		if err := json.Unmarshal(msg.Data, &data); err != nil {
			t.Fatalf("unmarshal heartbeat data %q: %v", msg.Data, err)
		}
		parsed, err := time.Parse(time.RFC3339, data.ServerTime)
		if err != nil {
			t.Fatalf("heartbeat serverTime %q is not RFC3339: %v", data.ServerTime, err)
		}
		serverTime = parsed
	}

	require.GreaterOrEqual(t, heartbeats, 2, "expected repeated application heartbeats within the read window")
	require.GreaterOrEqual(t, int(pings.Load()), 1, "protocol Ping must still be sent alongside the heartbeat")
	require.Equal(t, "+08:00", serverTime.Format("-07:00"), "serverTime should be formatted in the configured location")
}

func TestWriteLoopForwardsBusinessMessages(t *testing.T) {
	// A long heartbeat interval so ticks do not race the business message.
	h := newTestRealtime(t, time.Hour)
	messages := make(chan realtime.Message)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := h.upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		_ = h.writeLoop(conn, messages)
	}))
	defer func() {
		close(messages)
		server.Close()
	}()

	conn := connectWS(t, server.URL)
	defer conn.Close()

	messages <- realtime.Message{Type: "order.upsert", Data: json.RawMessage(`{"id":"a1b2c3"}`)}

	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	for {
		_, payload, err := conn.ReadMessage()
		if err != nil {
			t.Fatalf("timed out waiting for business message: %v", err)
		}

		var msg realtime.Message
		if err := json.Unmarshal(payload, &msg); err != nil {
			t.Fatalf("unmarshal message %q: %v", payload, err)
		}
		if msg.Type != "order.upsert" {
			continue
		}

		var data map[string]string
		if err := json.Unmarshal(msg.Data, &data); err != nil {
			t.Fatalf("unmarshal data %q: %v", msg.Data, err)
		}
		require.Equal(t, "a1b2c3", data["id"])
		return
	}
}

func TestWriteLoopForwardsBatchedBusinessMessages(t *testing.T) {
	h := newTestRealtime(t, time.Hour)
	messages := make(chan realtime.Message)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := h.upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		_ = h.writeLoop(conn, messages)
	}))
	defer func() {
		close(messages)
		server.Close()
	}()

	conn := connectWS(t, server.URL)
	defer conn.Close()

	messages <- realtime.Message{
		Type: "order.upsert_many",
		Data: json.RawMessage(`{"items":[{"id":"first"},{"id":"second"}]}`),
	}

	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	_, payload, err := conn.ReadMessage()
	require.NoError(t, err)

	var message realtime.Message
	require.NoError(t, json.Unmarshal(payload, &message))
	require.Equal(t, "order.upsert_many", message.Type)
	var data struct {
		Items []struct {
			ID string `json:"id"`
		} `json:"items"`
	}
	require.NoError(t, json.Unmarshal(message.Data, &data))
	require.Equal(t, []string{"first", "second"}, []string{data.Items[0].ID, data.Items[1].ID})
}

func TestWriteLoopClosesWithReconnectRequired(t *testing.T) {
	h := newTestRealtime(t, time.Hour)
	messages := make(chan realtime.Message)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := h.upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		close(messages)
		_ = h.writeLoop(conn, messages)
	}))
	defer server.Close()

	conn := connectWS(t, server.URL)
	defer conn.Close()

	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	_, _, err := conn.ReadMessage()
	require.Error(t, err)
	require.True(t, websocket.IsCloseError(err, websocket.CloseTryAgainLater), "got %v", err)
	closeErr, ok := err.(*websocket.CloseError)
	require.True(t, ok, "got %T: %v", err, err)
	require.Equal(t, "reconnect_required", closeErr.Text)
}

func TestServeWSDeliversHeartbeatToClient(t *testing.T) {
	h := newTestRealtime(t, 30*time.Millisecond)
	h.broker = realtime.NewBroker()
	h.sessions = realtime.NewSessionManager(time.Minute, time.Now)

	ticket, _, err := h.sessions.Issue(&identity.Context{
		UserID:      uuid.New(),
		WorkspaceID: uuid.New(),
		Role:        "owner",
		Phone:       "13800000001",
	})
	require.NoError(t, err)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/api/ws", h.ServeWS)

	server := httptest.NewServer(router)
	defer server.Close()

	conn := connectWS(t, server.URL+"/api/ws?ticket="+ticket)
	defer conn.Close() // must close before server.Close so readLoop unwinds

	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))

	_, payload, err := conn.ReadMessage()
	require.NoError(t, err)
	var ready realtime.Message
	require.NoError(t, json.Unmarshal(payload, &ready))
	require.Equal(t, "ready", ready.Type)

	_, payload, err = conn.ReadMessage()
	require.NoError(t, err)
	var heartbeat realtime.Message
	require.NoError(t, json.Unmarshal(payload, &heartbeat))
	require.Equal(t, "heartbeat", heartbeat.Type)

	var data struct {
		ServerTime string `json:"serverTime"`
	}
	require.NoError(t, json.Unmarshal(heartbeat.Data, &data))
	_, err = time.Parse(time.RFC3339, data.ServerTime)
	require.NoError(t, err, "heartbeat serverTime %q should be RFC3339", data.ServerTime)
}
