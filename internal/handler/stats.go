package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/dongwlin/legero-backend/internal/infra/httpx"
	"github.com/dongwlin/legero-backend/internal/service"
)

// Stats handles statistics HTTP endpoints.
type Stats struct {
	svc      *service.Stats
	location *time.Location
}

// NewStats creates a new StatsHandler.
func NewStats(svc *service.Stats, location *time.Location) *Stats {
	return &Stats{
		svc:      svc,
		location: location,
	}
}

// Daily returns per-day order counts and revenue for a workspace within a date range.
func (h *Stats) Daily(c *gin.Context, workspaceID uuid.UUID) {
	from, err := time.ParseInLocation("2006-01-02", c.Query("from"), h.location)
	if err != nil {
		httpx.AbortError(c, httpx.ValidationError("from must use YYYY-MM-DD"))
		return
	}
	to, err := time.ParseInLocation("2006-01-02", c.Query("to"), h.location)
	if err != nil {
		httpx.AbortError(c, httpx.ValidationError("to must use YYYY-MM-DD"))
		return
	}

	items, err := h.svc.Daily(c.Request.Context(), workspaceID, from, to)
	if err != nil {
		httpx.AbortError(c, err)
		return
	}

	type itemDTO struct {
		Date            string `json:"date"`
		OrderCount      int    `json:"orderCount"`
		TotalPriceCents int    `json:"totalPriceCents"`
	}

	responseItems := make([]itemDTO, 0, len(items))
	for _, item := range items {
		responseItems = append(responseItems, itemDTO{
			Date:            item.Date.In(h.location).Format("2006-01-02"),
			OrderCount:      item.OrderCount,
			TotalPriceCents: item.TotalPriceCents,
		})
	}

	httpx.JSON(c, http.StatusOK, gin.H{
		"items": responseItems,
	})
}
