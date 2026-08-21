package v1

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"github.com/dongwlin/legero-backend/internal/apperr"
	"github.com/dongwlin/legero-backend/internal/handler/v1/httpresp"
	"github.com/dongwlin/legero-backend/internal/handler/v1/dto"
	"github.com/dongwlin/legero-backend/internal/infra/config"
	"github.com/dongwlin/legero-backend/internal/infra/realtime"
	"github.com/dongwlin/legero-backend/internal/infra/timex"
)

const (
	defaultReadLimitBytes      int64 = 1024
	websocketCapabilitiesParam       = "capabilities"
)

// supportsRealtimeCapability parses the opt-in capabilities query parameter.
// A client may repeat the parameter or provide a comma-separated list, for
// example: /api/ws?ticket=...&capabilities=order.upsert_many. Unknown values
// are deliberately ignored so newer clients can talk to this server safely.
func supportsRealtimeCapability(query url.Values, wanted string) bool {
	for _, value := range query[websocketCapabilitiesParam] {
		for _, capability := range strings.Split(value, ",") {
			if strings.TrimSpace(capability) == wanted {
				return true
			}
		}
	}

	return false
}

// legacyOrderUpsertMessages expands a batch event into the legacy one-item
// envelope. The broker payload is already JSON, so RawMessage preserves each
// DTO without coupling the transport layer to domain types.
func legacyOrderUpsertMessages(message realtime.Message) ([]realtime.Message, error) {
	var payload struct {
		Items []json.RawMessage `json:"items"`
	}
	if err := json.Unmarshal(message.Data, &payload); err != nil {
		return nil, err
	}

	messages := make([]realtime.Message, 0, len(payload.Items))
	for _, item := range payload.Items {
		legacyMessage, err := realtime.NewMessage("order.upsert", struct {
			Item json.RawMessage `json:"item"`
		}{Item: item})
		if err != nil {
			return nil, err
		}
		messages = append(messages, legacyMessage)
	}

	return messages, nil
}

// Realtime handles realtime WebSocket HTTP endpoints.
type Realtime struct {
	broker            *realtime.Broker
	sessions          *realtime.SessionManager
	location          *time.Location
	heartbeatInterval time.Duration
	writeTimeout      time.Duration
	readTimeout       time.Duration
	now               func() time.Time
	upgrader          websocket.Upgrader
}

// NewRealtimeHandler creates a new Realtime handler.
func NewRealtimeHandler(
	broker *realtime.Broker,
	sessions *realtime.SessionManager,
	location *time.Location,
	cfg *config.Config,
	now func() time.Time,
) *Realtime {
	heartbeatInterval := cfg.RealtimeHeartbeatInterval
	if heartbeatInterval <= 0 {
		heartbeatInterval = 20 * time.Second
	}
	writeTimeout := cfg.WSWriteTimeout
	if writeTimeout <= 0 {
		writeTimeout = 10 * time.Second
	}
	readTimeout := cfg.WSReadTimeout
	if readTimeout <= 0 {
		readTimeout = heartbeatInterval * 3
	}
	if now == nil {
		now = time.Now
	}

	handler := &Realtime{
		broker:            broker,
		sessions:          sessions,
		location:          location,
		heartbeatInterval: heartbeatInterval,
		writeTimeout:      writeTimeout,
		readTimeout:       readTimeout,
		now:               now,
	}
	handler.upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return isOriginAllowed(r.Header.Get("Origin"), cfg.WSAllowedOrigins)
		},
	}

	return handler
}

// CreateSession creates a one-time-use ticket for WebSocket authentication.
func (h *Realtime) CreateSession(c *gin.Context) {
	authCtx, ok := AuthContext(c)
	if !ok {
		httpresp.AbortError(c, apperr.UnauthorizedError("missing auth context"))
		return
	}

	ticket, expiresAt, err := h.sessions.Issue(authCtx)
	if err != nil {
		httpresp.AbortError(c, err)
		return
	}

	httpresp.JSON(c, http.StatusOK, dto.CreateSessionResponse{
		Ticket:    ticket,
		ExpiresAt: timex.FormatTime(expiresAt, h.location),
	})
}

// ServeWS upgrades an HTTP connection to WebSocket and serves realtime messages.
func (h *Realtime) ServeWS(c *gin.Context) {
	ticket := strings.TrimSpace(c.Query("ticket"))
	if ticket == "" {
		httpresp.AbortError(c, apperr.ValidationError("ticket is required"))
		return
	}

	authCtx, err := h.sessions.Consume(ticket)
	if err != nil {
		httpresp.AbortError(c, err)
		return
	}

	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	messages, overflow, cancel := h.broker.Subscribe(authCtx.WorkspaceID)
	defer cancel()
	supportsOrderUpsertMany := supportsRealtimeCapability(
		c.Request.URL.Query(),
		realtime.CapabilityOrderUpsertMany,
	)

	h.configureConnection(conn)

	readyMessage, err := realtime.NewMessage("ready", realtime.ReadyPayload{
		ServerTime:          timex.FormatTime(h.now(), h.location),
		Capabilities:        realtime.SupportedCapabilities(),
		HeartbeatIntervalMs: h.heartbeatInterval.Milliseconds(),
	})
	if err != nil {
		_ = conn.Close()
		return
	}
	if err := h.writeJSON(conn, readyMessage); err != nil {
		_ = conn.Close()
		return
	}

	errCh := make(chan error, 2)
	go func() {
		errCh <- h.readLoop(conn)
	}()
	go func() {
		errCh <- h.writeLoop(conn, messages, overflow, supportsOrderUpsertMany)
	}()

	<-errCh
	_ = conn.Close()
}

func (h *Realtime) configureConnection(conn *websocket.Conn) {
	conn.SetReadLimit(defaultReadLimitBytes)
	_ = conn.SetReadDeadline(h.now().Add(h.readTimeout))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(h.now().Add(h.readTimeout))
	})
}

func (h *Realtime) readLoop(conn *websocket.Conn) error {
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			return err
		}
	}
}

func (h *Realtime) writeLoop(
	conn *websocket.Conn,
	messages <-chan realtime.Message,
	overflow <-chan struct{},
	supportsOrderUpsertMany bool,
) error {
	ticker := time.NewTicker(h.heartbeatInterval)
	defer ticker.Stop()

	for {
		if overflowSignaled(overflow) {
			return h.writeClose(conn, websocket.CloseTryAgainLater, "reconnect_required")
		}

		select {
		case <-overflow:
			return h.writeClose(conn, websocket.CloseTryAgainLater, "reconnect_required")
		case message, ok := <-messages:
			if !ok {
				return h.writeClose(conn, websocket.CloseTryAgainLater, "reconnect_required")
			}

			outgoing := []realtime.Message{message}
			if message.Type == realtime.CapabilityOrderUpsertMany && !supportsOrderUpsertMany {
				var err error
				outgoing, err = legacyOrderUpsertMessages(message)
				if err != nil {
					return err
				}
			}

			for _, outgoingMessage := range outgoing {
				if overflowSignaled(overflow) {
					return h.writeClose(conn, websocket.CloseTryAgainLater, "reconnect_required")
				}
				if err := h.writeJSON(conn, outgoingMessage); err != nil {
					return err
				}
			}
		case <-ticker.C:
			if overflowSignaled(overflow) {
				return h.writeClose(conn, websocket.CloseTryAgainLater, "reconnect_required")
			}
			// Keep protocol-level Ping/Pong (and its read-deadline refresh)
			// unchanged, and additionally push an application-level heartbeat
			// so browser/WebView clients can detect a healthy server link.
			if err := h.writePing(conn); err != nil {
				return err
			}
			if overflowSignaled(overflow) {
				return h.writeClose(conn, websocket.CloseTryAgainLater, "reconnect_required")
			}
			if err := h.writeHeartbeat(conn); err != nil {
				return err
			}
		}
	}
}

func overflowSignaled(overflow <-chan struct{}) bool {
	if overflow == nil {
		return false
	}

	select {
	case <-overflow:
		return true
	default:
		return false
	}
}

func (h *Realtime) writeJSON(conn *websocket.Conn, message realtime.Message) error {
	if err := conn.SetWriteDeadline(h.now().Add(h.writeTimeout)); err != nil {
		return err
	}
	return conn.WriteJSON(message)
}

func (h *Realtime) writePing(conn *websocket.Conn) error {
	return conn.WriteControl(
		websocket.PingMessage,
		[]byte("ping"),
		h.now().Add(h.writeTimeout),
	)
}

func (h *Realtime) writeHeartbeat(conn *websocket.Conn) error {
	message, err := realtime.NewMessage("heartbeat", realtime.HeartbeatPayload{
		ServerTime: timex.FormatTime(h.now(), h.location),
	})
	if err != nil {
		return err
	}
	return h.writeJSON(conn, message)
}

func (h *Realtime) writeClose(conn *websocket.Conn, code int, reason string) error {
	return conn.WriteControl(
		websocket.CloseMessage,
		websocket.FormatCloseMessage(code, reason),
		h.now().Add(h.writeTimeout),
	)
}

func isOriginAllowed(origin string, allowedOrigins []string) bool {
	if origin == "" || len(allowedOrigins) == 0 {
		return true
	}

	for _, allowedOrigin := range allowedOrigins {
		trimmed := strings.TrimSpace(allowedOrigin)
		if trimmed == "" {
			continue
		}
		if trimmed == "*" || strings.EqualFold(trimmed, origin) {
			return true
		}
	}

	return false
}
