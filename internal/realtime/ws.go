package realtime

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"github.com/dongwlin/legero-backend/internal/auth"
	"github.com/dongwlin/legero-backend/internal/infra/httpx"
)

const defaultReadLimitBytes int64 = 1024

type Handler struct {
	broker            *Broker
	sessions          *SessionManager
	location          *time.Location
	heartbeatInterval time.Duration
	writeTimeout      time.Duration
	readTimeout       time.Duration
	now               func() time.Time
	upgrader          websocket.Upgrader
}

func NewHandler(
	broker *Broker,
	sessions *SessionManager,
	location *time.Location,
	heartbeatInterval time.Duration,
	writeTimeout time.Duration,
	readTimeout time.Duration,
	allowedOrigins []string,
	now func() time.Time,
) *Handler {
	if heartbeatInterval <= 0 {
		heartbeatInterval = 20 * time.Second
	}
	if writeTimeout <= 0 {
		writeTimeout = 10 * time.Second
	}
	if readTimeout <= 0 {
		readTimeout = heartbeatInterval * 3
	}
	if now == nil {
		now = time.Now
	}

	handler := &Handler{
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
			return isOriginAllowed(r.Header.Get("Origin"), allowedOrigins)
		},
	}

	return handler
}

func (h *Handler) CreateSession(c *gin.Context) {
	authCtx, ok := auth.ContextFromGin(c)
	if !ok {
		httpx.AbortError(c, httpx.UnauthorizedError("missing auth context"))
		return
	}

	ticket, expiresAt, err := h.sessions.Issue(authCtx)
	if err != nil {
		httpx.AbortError(c, err)
		return
	}

	httpx.JSON(c, http.StatusOK, gin.H{
		"ticket":    ticket,
		"expiresAt": formatTime(expiresAt, h.location),
	})
}

func (h *Handler) ServeWS(c *gin.Context) {
	ticket := strings.TrimSpace(c.Query("ticket"))
	if ticket == "" {
		httpx.AbortError(c, httpx.ValidationError("ticket is required"))
		return
	}

	authCtx, err := h.sessions.Consume(ticket)
	if err != nil {
		httpx.AbortError(c, err)
		return
	}

	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	messages, cancel := h.broker.Subscribe(authCtx.WorkspaceID)
	defer cancel()

	h.configureConnection(conn)

	readyMessage, err := newMessage("ready", ReadyPayload{
		ServerTime: formatTime(h.now(), h.location),
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
		errCh <- h.writeLoop(conn, messages)
	}()

	<-errCh
	_ = conn.Close()
}

func (h *Handler) configureConnection(conn *websocket.Conn) {
	conn.SetReadLimit(defaultReadLimitBytes)
	_ = conn.SetReadDeadline(h.now().Add(h.readTimeout))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(h.now().Add(h.readTimeout))
	})
}

func (h *Handler) readLoop(conn *websocket.Conn) error {
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			return err
		}
	}
}

func (h *Handler) writeLoop(conn *websocket.Conn, messages <-chan Message) error {
	ticker := time.NewTicker(h.heartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case message, ok := <-messages:
			if !ok {
				return h.writeClose(conn, websocket.CloseTryAgainLater, "reconnect_required")
			}
			if err := h.writeJSON(conn, message); err != nil {
				return err
			}
		case <-ticker.C:
			if err := h.writePing(conn); err != nil {
				return err
			}
		}
	}
}

func (h *Handler) writeJSON(conn *websocket.Conn, message Message) error {
	if err := conn.SetWriteDeadline(h.now().Add(h.writeTimeout)); err != nil {
		return err
	}
	return conn.WriteJSON(message)
}

func (h *Handler) writePing(conn *websocket.Conn) error {
	return conn.WriteControl(
		websocket.PingMessage,
		[]byte("ping"),
		h.now().Add(h.writeTimeout),
	)
}

func (h *Handler) writeClose(conn *websocket.Conn, code int, reason string) error {
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

func formatTime(value time.Time, location *time.Location) string {
	if location == nil {
		return value.Format(time.RFC3339)
	}
	return value.In(location).Format(time.RFC3339)
}
