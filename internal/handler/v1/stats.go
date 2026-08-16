package v1

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/dongwlin/legero-backend/internal/apperr"
	"github.com/dongwlin/legero-backend/internal/handler/httpresp"
	"github.com/dongwlin/legero-backend/internal/handler/v1/dto"
	"github.com/dongwlin/legero-backend/internal/service"
)

// StatsHandler handles statistics HTTP endpoints.
type StatsHandler struct {
	statsSvc service.Stats
	location *time.Location
}

// NewStatsHandler creates a new StatsHandler.
func NewStatsHandler(statsSvc service.Stats, location *time.Location) *StatsHandler {
	return &StatsHandler{statsSvc: statsSvc, location: location}
}

// Daily returns per-day order counts and revenue for a workspace within a date range.
func (h *StatsHandler) Daily(c *gin.Context) {
	actor, ok := actorFromGin(c)
	if !ok {
		httpresp.AbortError(c, apperr.UnauthorizedError("missing auth context"))
		return
	}

	from, err := time.ParseInLocation("2006-01-02", c.Query("from"), h.location)
	if err != nil {
		httpresp.AbortError(c, apperr.ValidationError("from must use YYYY-MM-DD"))
		return
	}
	to, err := time.ParseInLocation("2006-01-02", c.Query("to"), h.location)
	if err != nil {
		httpresp.AbortError(c, apperr.ValidationError("to must use YYYY-MM-DD"))
		return
	}

	items, err := h.statsSvc.Daily(c.Request.Context(), actor.WorkspaceID, from, to)
	if err != nil {
		httpresp.AbortError(c, err)
		return
	}

	responseItems := make([]dto.DailyItem, 0, len(items))
	for _, item := range items {
		responseItems = append(responseItems, dto.DailyItem{
			Date:            item.Date.In(h.location).Format("2006-01-02"),
			OrderCount:      item.OrderCount,
			TotalPriceCents: item.TotalPriceCents,
		})
	}

	httpresp.JSON(c, http.StatusOK, dto.DailyResponse{
		Items: responseItems,
	})
}
