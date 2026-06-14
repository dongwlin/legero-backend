package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/dongwlin/legero-backend/internal/infra/httpx"
	"github.com/dongwlin/legero-backend/internal/infra/identity"
	"github.com/dongwlin/legero-backend/internal/model"
	"github.com/dongwlin/legero-backend/internal/service"
)

// Order handles order HTTP endpoints.
type Order struct {
	svc      *service.Order
	location *time.Location
}

// NewOrder creates a new OrderHandler.
func NewOrder(svc *service.Order, location *time.Location) *Order {
	return &Order{
		svc:      svc,
		location: location,
	}
}

// List returns a paginated list of orders.
func (h *Order) List(c *gin.Context) {
	actor, ok := actorFromGin(c)
	if !ok {
		httpx.AbortError(c, httpx.UnauthorizedError("missing auth context"))
		return
	}

	limit, err := parseLimit(c.Query("limit"))
	if err != nil {
		httpx.AbortError(c, err)
		return
	}

	query := model.ListOrdersQuery{
		Status: model.ListStatus(c.DefaultQuery("status", string(model.ListStatusUncompleted))),
		Limit:  limit,
		Cursor: c.Query("cursor"),
	}

	result, err := h.svc.List(c.Request.Context(), actor, query)
	if err != nil {
		httpx.AbortError(c, err)
		return
	}

	items := make([]model.OrderDTO, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, item.ToDTO(h.location))
	}

	httpx.JSON(c, http.StatusOK, gin.H{
		"items":      items,
		"nextCursor": result.NextCursor,
	})
}

// Create batch-creates orders.
func (h *Order) Create(c *gin.Context) {
	actor, ok := actorFromGin(c)
	if !ok {
		httpx.AbortError(c, httpx.UnauthorizedError("missing auth context"))
		return
	}

	var input model.CreateOrdersInput
	if err := c.ShouldBindJSON(&input); err != nil {
		httpx.AbortError(c, httpx.ValidationError("invalid create order payload"))
		return
	}

	items, err := h.svc.CreateBatch(c.Request.Context(), actor, input)
	if err != nil {
		httpx.AbortError(c, err)
		return
	}

	dtos := make([]model.OrderDTO, 0, len(items))
	for _, item := range items {
		dtos = append(dtos, item.ToDTO(h.location))
	}

	httpx.JSON(c, http.StatusOK, gin.H{
		"items": dtos,
	})
}

// Update replaces the form data of an existing order.
func (h *Order) Update(c *gin.Context) {
	actor, ok := actorFromGin(c)
	if !ok {
		httpx.AbortError(c, httpx.UnauthorizedError("missing auth context"))
		return
	}
	orderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpx.AbortError(c, httpx.ValidationError("id must be a valid uuid"))
		return
	}

	var input model.UpdateOrderInput
	if err := c.ShouldBindJSON(&input); err != nil {
		httpx.AbortError(c, httpx.ValidationError("invalid update order payload"))
		return
	}

	item, err := h.svc.UpdateForm(c.Request.Context(), actor, orderID, input)
	if err != nil {
		httpx.AbortError(c, err)
		return
	}

	httpx.JSON(c, http.StatusOK, gin.H{
		"item": item.ToDTO(h.location),
	})
}

// ToggleStep toggles the completion state of a cooking step.
func (h *Order) ToggleStep(c *gin.Context) {
	actor, ok := actorFromGin(c)
	if !ok {
		httpx.AbortError(c, httpx.UnauthorizedError("missing auth context"))
		return
	}
	orderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpx.AbortError(c, httpx.ValidationError("id must be a valid uuid"))
		return
	}

	var input model.ToggleStepInput
	if err := c.ShouldBindJSON(&input); err != nil {
		httpx.AbortError(c, httpx.ValidationError("invalid toggle step payload"))
		return
	}

	item, err := h.svc.ToggleStep(c.Request.Context(), actor, orderID, input)
	if err != nil {
		httpx.AbortError(c, err)
		return
	}

	httpx.JSON(c, http.StatusOK, gin.H{
		"item": item.ToDTO(h.location),
	})
}

// ToggleServed toggles the served (completed) state of an order.
func (h *Order) ToggleServed(c *gin.Context) {
	actor, ok := actorFromGin(c)
	if !ok {
		httpx.AbortError(c, httpx.UnauthorizedError("missing auth context"))
		return
	}
	orderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpx.AbortError(c, httpx.ValidationError("id must be a valid uuid"))
		return
	}

	var input model.ToggleServedInput
	if err := c.ShouldBindJSON(&input); err != nil {
		httpx.AbortError(c, httpx.ValidationError("invalid toggle served payload"))
		return
	}

	item, err := h.svc.ToggleServed(c.Request.Context(), actor, orderID, input)
	if err != nil {
		httpx.AbortError(c, err)
		return
	}

	httpx.JSON(c, http.StatusOK, gin.H{
		"item": item.ToDTO(h.location),
	})
}

// Delete removes an order.
func (h *Order) Delete(c *gin.Context) {
	actor, ok := actorFromGin(c)
	if !ok {
		httpx.AbortError(c, httpx.UnauthorizedError("missing auth context"))
		return
	}
	orderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpx.AbortError(c, httpx.ValidationError("id must be a valid uuid"))
		return
	}

	if err := h.svc.Remove(c.Request.Context(), actor, orderID); err != nil {
		httpx.AbortError(c, err)
		return
	}

	httpx.NoContent(c)
}

// Clear deletes orders from a workspace.
func (h *Order) Clear(c *gin.Context) {
	actor, ok := actorFromGin(c)
	if !ok {
		httpx.AbortError(c, httpx.UnauthorizedError("missing auth context"))
		return
	}

	var input model.ClearWorkspaceInput
	if err := c.ShouldBindJSON(&input); err != nil {
		httpx.AbortError(c, httpx.ValidationError("invalid clear payload"))
		return
	}

	count, err := h.svc.ClearWorkspace(c.Request.Context(), actor, input.Confirm, input.Mode)
	if err != nil {
		httpx.AbortError(c, err)
		return
	}

	httpx.JSON(c, http.StatusOK, gin.H{
		"clearedCount": count,
	})
}

// actorFromGin extracts the Actor from the gin context (set by auth middleware).
func actorFromGin(c *gin.Context) (model.Actor, bool) {
	value, ok := c.Get(identity.GinContextKey)
	if !ok {
		return model.Actor{}, false
	}
	authCtx, ok := value.(*identity.Context)
	if !ok {
		return model.Actor{}, false
	}
	return model.Actor{
		UserID:      authCtx.UserID,
		WorkspaceID: authCtx.WorkspaceID,
		Role:        model.Role(authCtx.Role),
	}, true
}

// parseLimit parses and validates the limit query parameter.
func parseLimit(value string) (int, error) {
	if value == "" {
		return 50, nil
	}
	limit, err := strconv.Atoi(value)
	if err != nil {
		return 0, httpx.ValidationError("limit must be an integer")
	}
	if limit <= 0 {
		return 0, httpx.ValidationError("limit must be greater than 0")
	}
	return limit, nil
}
