package v1

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/dongwlin/legero-backend/internal/apperr"
	"github.com/dongwlin/legero-backend/internal/handler/httpresp"
	"github.com/dongwlin/legero-backend/internal/handler/v1/dto"
	"github.com/dongwlin/legero-backend/internal/infra/identity"
	"github.com/dongwlin/legero-backend/internal/domain"
	"github.com/dongwlin/legero-backend/internal/service"
)

// OrderHandler handles order HTTP endpoints.
type OrderHandler struct {
	orderSvc service.Order
	location *time.Location
}

// NewOrderHandler creates a new OrderHandler.
func NewOrderHandler(orderSvc service.Order, location *time.Location) *OrderHandler {
	return &OrderHandler{orderSvc: orderSvc, location: location}
}

// List returns a paginated list of orders.
func (h *OrderHandler) List(c *gin.Context) {
	actor, ok := actorFromGin(c)
	if !ok {
		httpresp.AbortError(c, apperr.UnauthorizedError("missing auth context"))
		return
	}

	limit, err := parseLimit(c.Query("limit"))
	if err != nil {
		httpresp.AbortError(c, err)
		return
	}

	query := domain.ListOrdersQuery{
		Status: domain.ListStatus(c.DefaultQuery("status", string(domain.ListStatusUncompleted))),
		Limit:  limit,
		Cursor: c.Query("cursor"),
	}

	result, err := h.orderSvc.List(c.Request.Context(), actor, query)
	if err != nil {
		httpresp.AbortError(c, err)
		return
	}

	httpresp.JSON(c, http.StatusOK, dto.ListOrdersResponse{
		Items:      toOrderDTOs(result.Items, h.location),
		NextCursor: result.NextCursor,
	})
}

// Create batch-creates orders.
func (h *OrderHandler) Create(c *gin.Context) {
	actor, ok := actorFromGin(c)
	if !ok {
		httpresp.AbortError(c, apperr.UnauthorizedError("missing auth context"))
		return
	}

	var input domain.CreateOrdersInput
	if err := c.ShouldBindJSON(&input); err != nil {
		httpresp.AbortError(c, apperr.ValidationError("invalid create order payload"))
		return
	}

	items, err := h.orderSvc.CreateBatch(c.Request.Context(), actor, input)
	if err != nil {
		httpresp.AbortError(c, err)
		return
	}

	httpresp.JSON(c, http.StatusOK, dto.CreateOrdersResponse{
		Items: toOrderDTOs(items, h.location),
	})
}

// Update replaces the form data of an existing order.
func (h *OrderHandler) Update(c *gin.Context) {
	actor, ok := actorFromGin(c)
	if !ok {
		httpresp.AbortError(c, apperr.UnauthorizedError("missing auth context"))
		return
	}
	orderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpresp.AbortError(c, apperr.ValidationError("id must be a valid uuid"))
		return
	}

	var input domain.UpdateOrderInput
	if err := c.ShouldBindJSON(&input); err != nil {
		httpresp.AbortError(c, apperr.ValidationError("invalid update order payload"))
		return
	}

	item, err := h.orderSvc.UpdateForm(c.Request.Context(), actor, orderID, input)
	if err != nil {
		httpresp.AbortError(c, err)
		return
	}

	httpresp.JSON(c, http.StatusOK, dto.UpdateOrderResponse{
		Item: item.ToDTO(h.location),
	})
}

// ToggleStep toggles the completion state of a cooking step.
func (h *OrderHandler) ToggleStep(c *gin.Context) {
	actor, ok := actorFromGin(c)
	if !ok {
		httpresp.AbortError(c, apperr.UnauthorizedError("missing auth context"))
		return
	}
	orderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpresp.AbortError(c, apperr.ValidationError("id must be a valid uuid"))
		return
	}

	var input domain.ToggleStepInput
	if err := c.ShouldBindJSON(&input); err != nil {
		httpresp.AbortError(c, apperr.ValidationError("invalid toggle step payload"))
		return
	}

	item, err := h.orderSvc.ToggleStep(c.Request.Context(), actor, orderID, input)
	if err != nil {
		httpresp.AbortError(c, err)
		return
	}

	httpresp.JSON(c, http.StatusOK, dto.UpdateOrderResponse{
		Item: item.ToDTO(h.location),
	})
}

// ToggleServed toggles the served (completed) state of an order.
func (h *OrderHandler) ToggleServed(c *gin.Context) {
	actor, ok := actorFromGin(c)
	if !ok {
		httpresp.AbortError(c, apperr.UnauthorizedError("missing auth context"))
		return
	}
	orderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpresp.AbortError(c, apperr.ValidationError("id must be a valid uuid"))
		return
	}

	var input domain.ToggleServedInput
	if err := c.ShouldBindJSON(&input); err != nil {
		httpresp.AbortError(c, apperr.ValidationError("invalid toggle served payload"))
		return
	}

	item, err := h.orderSvc.ToggleServed(c.Request.Context(), actor, orderID, input)
	if err != nil {
		httpresp.AbortError(c, err)
		return
	}

	httpresp.JSON(c, http.StatusOK, dto.UpdateOrderResponse{
		Item: item.ToDTO(h.location),
	})
}

// Delete removes an order.
func (h *OrderHandler) Delete(c *gin.Context) {
	actor, ok := actorFromGin(c)
	if !ok {
		httpresp.AbortError(c, apperr.UnauthorizedError("missing auth context"))
		return
	}
	orderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpresp.AbortError(c, apperr.ValidationError("id must be a valid uuid"))
		return
	}

	if err := h.orderSvc.Remove(c.Request.Context(), actor, orderID); err != nil {
		httpresp.AbortError(c, err)
		return
	}

	httpresp.NoContent(c)
}

// Clear deletes orders from a workspace.
func (h *OrderHandler) Clear(c *gin.Context) {
	actor, ok := actorFromGin(c)
	if !ok {
		httpresp.AbortError(c, apperr.UnauthorizedError("missing auth context"))
		return
	}

	var input domain.ClearWorkspaceInput
	if err := c.ShouldBindJSON(&input); err != nil {
		httpresp.AbortError(c, apperr.ValidationError("invalid clear payload"))
		return
	}

	count, err := h.orderSvc.ClearWorkspace(c.Request.Context(), actor, input.Confirm, input.Mode)
	if err != nil {
		httpresp.AbortError(c, err)
		return
	}

	httpresp.JSON(c, http.StatusOK, dto.ClearOrdersResponse{
		ClearedCount: count,
	})
}

// actorFromGin extracts the Actor from the gin context (set by auth middleware).
func actorFromGin(c *gin.Context) (domain.Actor, bool) {
	value, ok := c.Get(identity.GinContextKey)
	if !ok {
		return domain.Actor{}, false
	}
	authCtx, ok := value.(*identity.Context)
	if !ok {
		return domain.Actor{}, false
	}
	return domain.Actor{
		UserID:      authCtx.UserID,
		WorkspaceID: authCtx.WorkspaceID,
		Role:        domain.Role(authCtx.Role),
	}, true
}

// parseLimit parses and validates the limit query parameter.
func parseLimit(value string) (int, error) {
	if value == "" {
		return 50, nil
	}
	limit, err := strconv.Atoi(value)
	if err != nil {
		return 0, apperr.ValidationError("limit must be an integer")
	}
	if limit <= 0 {
		return 0, apperr.ValidationError("limit must be greater than 0")
	}
	return limit, nil
}

// toOrderDTOs converts a slice of Order to a slice of OrderDTO.
func toOrderDTOs(items []domain.Order, location *time.Location) []domain.OrderDTO {
	dtos := make([]domain.OrderDTO, 0, len(items))
	for _, item := range items {
		dtos = append(dtos, item.ToDTO(location))
	}
	return dtos
}
