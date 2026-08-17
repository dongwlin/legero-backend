package v1

import (
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"

	"github.com/dongwlin/legero-backend/internal/domain"
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

type singleConnListener struct {
	connections chan net.Conn
	closed      chan struct{}
	closeOnce   sync.Once
}

func newSingleConnListener(conn net.Conn) *singleConnListener {
	connections := make(chan net.Conn, 1)
	connections <- conn
	return &singleConnListener{
		connections: connections,
		closed:      make(chan struct{}),
	}
}

func (l *singleConnListener) Accept() (net.Conn, error) {
	select {
	case conn := <-l.connections:
		return conn, nil
	case <-l.closed:
		return nil, net.ErrClosed
	}
}

func (l *singleConnListener) Close() error {
	l.closeOnce.Do(func() {
		close(l.closed)
	})
	return nil
}

func (l *singleConnListener) Addr() net.Addr {
	return singleConnAddr{}
}

type singleConnAddr struct{}

func (singleConnAddr) Network() string { return "pipe" }
func (singleConnAddr) String() string  { return "pipe" }

func issueTestRealtimeTicket(t *testing.T, h *Realtime, workspaceID uuid.UUID) string {
	t.Helper()
	ticket, _, err := h.sessions.Issue(&identity.Context{
		UserID:      uuid.New(),
		WorkspaceID: workspaceID,
		Role:        "owner",
		Phone:       "13800000001",
	})
	require.NoError(t, err)
	return ticket
}

func TestSupportsRealtimeCapability(t *testing.T) {
	tests := []struct {
		name  string
		query url.Values
		want  bool
	}{
		{
			name: "repeated query parameters",
			query: url.Values{
				websocketCapabilitiesParam: {"future.event", realtime.CapabilityOrderUpsertMany},
			},
			want: true,
		},
		{
			name: "comma separated values with whitespace",
			query: url.Values{
				websocketCapabilitiesParam: {" future.event,  order.upsert_many  , another.event "},
			},
			want: true,
		},
		{
			name: "unknown capability",
			query: url.Values{
				websocketCapabilitiesParam: {"future.event"},
			},
			want: false,
		},
		{
			name: "empty capability",
			query: url.Values{
				websocketCapabilitiesParam: {"  , "},
			},
			want: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.want, supportsRealtimeCapability(test.query, realtime.CapabilityOrderUpsertMany))
		})
	}
}

func TestOrderUpsertManyProtocolIdentifierInvariant(t *testing.T) {
	require.Equal(t, domain.EventOrderUpsertMany, realtime.CapabilityOrderUpsertMany)
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
		_ = h.writeLoop(conn, messages, nil, false)
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
		_ = h.writeLoop(conn, messages, nil, false)
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
		_ = h.writeLoop(conn, messages, nil, true)
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

func TestWriteLoopExpandsBatchedBusinessMessagesForLegacyClient(t *testing.T) {
	h := newTestRealtime(t, time.Hour)
	messages := make(chan realtime.Message)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := h.upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		_ = h.writeLoop(conn, messages, nil, false)
	}))
	defer func() {
		close(messages)
		server.Close()
	}()

	conn := connectWS(t, server.URL)
	defer conn.Close()

	messages <- realtime.Message{
		Type: realtime.CapabilityOrderUpsertMany,
		Data: json.RawMessage(`{"items":[{"id":"first"},{"id":"second"}]}`),
	}

	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	for _, wantID := range []string{"first", "second"} {
		_, payload, err := conn.ReadMessage()
		require.NoError(t, err)

		var message realtime.Message
		require.NoError(t, json.Unmarshal(payload, &message))
		require.Equal(t, "order.upsert", message.Type)

		var data struct {
			Item struct {
				ID string `json:"id"`
			} `json:"item"`
		}
		require.NoError(t, json.Unmarshal(message.Data, &data))
		require.Equal(t, wantID, data.Item.ID)
	}
}

func TestWriteLoopStopsLegacyBatchAfterBrokerOverflow(t *testing.T) {
	h := newTestRealtime(t, time.Hour)
	broker := realtime.NewBroker()
	workspaceID := uuid.New()
	messages, overflow, cancel := broker.Subscribe(workspaceID)
	defer cancel()

	serverConn, clientConn := net.Pipe()
	listener := newSingleConnListener(serverConn)
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := h.upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		_ = h.writeLoop(conn, messages, overflow, false)
	})}
	serveDone := make(chan error, 1)
	go func() {
		serveDone <- server.Serve(listener)
	}()

	dialer := websocket.Dialer{
		NetDial: func(string, string) (net.Conn, error) {
			return clientConn, nil
		},
	}
	conn, _, err := dialer.Dial("ws://pipe/", nil)
	require.NoError(t, err)
	defer func() {
		_ = conn.Close()
		_ = listener.Close()
		_ = server.Close()
		<-serveDone
	}()

	const batchSize = 4
	items := make([]map[string]string, batchSize)
	for index := range items {
		items[index] = map[string]string{"id": string(rune('a' + index))}
	}
	broker.Publish(workspaceID, realtime.CapabilityOrderUpsertMany, map[string]any{
		"items": items,
	})

	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	_, payload, err := conn.ReadMessage()
	require.NoError(t, err)
	var first realtime.Message
	require.NoError(t, json.Unmarshal(payload, &first))
	require.Equal(t, "order.upsert", first.Type)

	// The first legacy event has been written. Overflow the broker while the
	// write loop is expanding the remaining items; the current socket write may
	// finish, but no later legacy events should be drained.
	for index := 0; index <= cap(messages); index++ {
		broker.Publish(workspaceID, "order.deleted", map[string]int{"index": index})
	}

	received := 1
	for {
		_, payload, err = conn.ReadMessage()
		if err != nil {
			require.True(t, websocket.IsCloseError(err, websocket.CloseTryAgainLater), "got %v", err)
			closeErr, ok := err.(*websocket.CloseError)
			require.True(t, ok, "got %T: %v", err, err)
			require.Equal(t, "reconnect_required", closeErr.Text)
			break
		}

		var message realtime.Message
		require.NoError(t, json.Unmarshal(payload, &message))
		require.Equal(t, "order.upsert", message.Type)
		received++
	}
	require.Less(t, received, batchSize, "overflow must stop legacy batch expansion")
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
		_ = h.writeLoop(conn, messages, nil, false)
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

func TestBrokerOverflowClosesClientWithReconnectRequired(t *testing.T) {
	h := newTestRealtime(t, time.Hour)
	broker := realtime.NewBroker()
	workspaceID := uuid.New()
	messages, overflow, cancel := broker.Subscribe(workspaceID)
	defer cancel()

	// Fill the broker queue and publish one additional event. The overflow
	// path must discard the queued messages before closing the subscription so
	// writeLoop sends the reconnect signal as soon as it starts.
	for index := 0; index <= cap(messages); index++ {
		broker.Publish(workspaceID, "order.deleted", map[string]int{"index": index})
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := h.upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		_ = h.writeLoop(conn, messages, overflow, false)
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
	var readyData realtime.ReadyPayload
	require.NoError(t, json.Unmarshal(ready.Data, &readyData))
	require.Contains(t, readyData.Capabilities, realtime.CapabilityHeartbeat)
	require.Contains(t, readyData.Capabilities, realtime.CapabilityOrderUpsertMany)
	require.Equal(t, int64(30), readyData.HeartbeatIntervalMs)

	_, payload, err = conn.ReadMessage()
	require.NoError(t, err)
	var heartbeat realtime.Message
	require.NoError(t, json.Unmarshal(payload, &heartbeat))
	require.Equal(t, "heartbeat", heartbeat.Type)

	var data realtime.HeartbeatPayload
	require.NoError(t, json.Unmarshal(heartbeat.Data, &data))
	_, err = time.Parse(time.RFC3339, data.ServerTime)
	require.NoError(t, err, "heartbeat serverTime %q should be RFC3339", data.ServerTime)
	var heartbeatFields map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(heartbeat.Data, &heartbeatFields))
	require.Len(t, heartbeatFields, 1, "application heartbeat must retain its serverTime-only payload")
	require.NotContains(t, heartbeatFields, "heartbeatIntervalMs")
}

func TestServeWSNegotiatesOrderUpsertMany(t *testing.T) {
	tests := []struct {
		name         string
		capabilities string
		wantTypes    []string
	}{
		{
			name:      "legacy client falls back to single upserts",
			wantTypes: []string{"order.upsert", "order.upsert"},
		},
		{
			name:         "capable client receives batch",
			capabilities: realtime.CapabilityOrderUpsertMany,
			wantTypes:    []string{"order.upsert_many"},
		},
		{
			name:         "unknown capability is ignored",
			capabilities: "future.event",
			wantTypes:    []string{"order.upsert", "order.upsert"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			h := newTestRealtime(t, time.Hour)
			h.broker = realtime.NewBroker()
			h.sessions = realtime.NewSessionManager(time.Minute, time.Now)

			workspaceID := uuid.New()
			gin.SetMode(gin.TestMode)
			router := gin.New()
			router.GET("/api/ws", h.ServeWS)
			server := httptest.NewServer(router)
			defer server.Close()

			wsURL := server.URL + "/api/ws?ticket=" + issueTestRealtimeTicket(t, h, workspaceID)
			if test.capabilities != "" {
				wsURL += "&capabilities=" + url.QueryEscape(test.capabilities)
			}

			conn := connectWS(t, wsURL)
			defer conn.Close()
			_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))

			_, payload, err := conn.ReadMessage()
			require.NoError(t, err)
			var ready realtime.Message
			require.NoError(t, json.Unmarshal(payload, &ready))
			require.Equal(t, "ready", ready.Type)
			var readyData realtime.ReadyPayload
			require.NoError(t, json.Unmarshal(ready.Data, &readyData))
			require.Contains(t, readyData.Capabilities, realtime.CapabilityHeartbeat)
			require.Contains(t, readyData.Capabilities, realtime.CapabilityOrderUpsertMany)
			require.Equal(t, int64(time.Hour/time.Millisecond), readyData.HeartbeatIntervalMs)

			h.broker.Publish(workspaceID, realtime.CapabilityOrderUpsertMany, map[string]any{
				"items": []map[string]string{{"id": "first"}, {"id": "second"}},
			})

			for index, wantType := range test.wantTypes {
				_, payload, err = conn.ReadMessage()
				require.NoError(t, err)
				var message realtime.Message
				require.NoError(t, json.Unmarshal(payload, &message))
				require.Equal(t, wantType, message.Type)

				if wantType == "order.upsert" {
					var data struct {
						Item struct {
							ID string `json:"id"`
						} `json:"item"`
					}
					require.NoError(t, json.Unmarshal(message.Data, &data))
					require.Equal(t, []string{"first", "second"}[index], data.Item.ID)
				}
			}
		})
	}
}
