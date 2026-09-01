package v2

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/dongwlin/legero-backend/internal/apperr"
	"github.com/dongwlin/legero-backend/internal/handler/httpresp"
	"github.com/dongwlin/legero-backend/internal/handler/v2/dto"
	"github.com/dongwlin/legero-backend/internal/infra/realtime"
	"github.com/dongwlin/legero-backend/internal/infra/timex"
)

// RealtimeHandler issues one-time WebSocket session tickets for v2 clients.
// The WebSocket upgrade itself remains a single shared endpoint at
// GET /api/ws: its contract is the realtime message protocol, not the v2
// HTTP JSON envelope, so it is not duplicated per API version.
type RealtimeHandler struct {
	sessions *realtime.SessionManager
	location *time.Location
}

// NewRealtimeHandler creates a new RealtimeHandler.
func NewRealtimeHandler(sessions *realtime.SessionManager, location *time.Location) *RealtimeHandler {
	return &RealtimeHandler{sessions: sessions, location: location}
}

// CreateSession creates a one-time-use ticket for WebSocket authentication.
func (h *RealtimeHandler) CreateSession(c *gin.Context) {
	authCtx, ok := authContext(c)
	if !ok {
		httpresp.AbortError(c, apperr.UnauthorizedError("missing auth context"))
		return
	}

	ticket, expiresAt, err := h.sessions.Issue(authCtx)
	if err != nil {
		httpresp.AbortError(c, err)
		return
	}

	httpresp.JSON(c, http.StatusOK, httpresp.Success(dto.CreateSessionResponseData{
		Ticket:    ticket,
		ExpiresAt: timex.FormatTime(expiresAt, h.location),
	}))
}
