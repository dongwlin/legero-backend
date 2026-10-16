package v2

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/dongwlin/legero-backend/internal/apperr"
	"github.com/dongwlin/legero-backend/internal/domain"
	"github.com/dongwlin/legero-backend/internal/handler/httpresp"
	"github.com/dongwlin/legero-backend/internal/handler/v2/dto"
	"github.com/dongwlin/legero-backend/internal/infra/identity"
	"github.com/dongwlin/legero-backend/internal/infra/timex"
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

	httpresp.JSON(c, http.StatusOK, httpresp.Success(dto.ListOrdersResponseData{
		Items:      toOrderDTOs(result.Items, h.location),
		NextCursor: result.NextCursor,
	}))
}

// Create batch-creates orders.
func (h *OrderHandler) Create(c *gin.Context) {
	actor, ok := actorFromGin(c)
	if !ok {
		httpresp.AbortError(c, apperr.UnauthorizedError("missing auth context"))
		return
	}

	var req dto.CreateOrdersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.AbortError(c, apperr.ValidationError("invalid create order payload"))
		return
	}

	items, err := h.orderSvc.CreateBatch(c.Request.Context(), actor, domain.CreateOrdersInput{
		Quantity: req.Quantity,
		Form:     toOrderFormInput(req.Form),
	})
	if err != nil {
		httpresp.AbortError(c, err)
		return
	}

	httpresp.JSON(c, http.StatusOK, httpresp.Success(dto.CreateOrdersResponseData{
		Items: toOrderDTOs(items, h.location),
	}))
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

	var req dto.UpdateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.AbortError(c, apperr.ValidationError("invalid update order payload"))
		return
	}

	item, err := h.orderSvc.UpdateForm(c.Request.Context(), actor, orderID, domain.UpdateOrderInput{
		Form:              toOrderFormInput(req.Form),
		ExpectedVersion:   req.ExpectedVersion,
		ExpectedUpdatedAt: req.ExpectedUpdatedAt,
	})
	if err != nil {
		httpresp.AbortError(c, err)
		return
	}

	httpresp.JSON(c, http.StatusOK, httpresp.Success(dto.UpdateOrderResponseData{
		Item: toOrderDTO(*item, h.location),
	}))
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

	var req dto.ToggleStepRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.AbortError(c, apperr.ValidationError("invalid toggle step payload"))
		return
	}

	item, err := h.orderSvc.ToggleStep(c.Request.Context(), actor, orderID, domain.ToggleStepInput{
		Step:              req.Step,
		ExpectedVersion:   req.ExpectedVersion,
		ExpectedUpdatedAt: req.ExpectedUpdatedAt,
	})
	if err != nil {
		httpresp.AbortError(c, err)
		return
	}

	httpresp.JSON(c, http.StatusOK, httpresp.Success(dto.UpdateOrderResponseData{
		Item: toOrderDTO(*item, h.location),
	}))
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

	var req dto.ToggleServedRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.AbortError(c, apperr.ValidationError("invalid toggle served payload"))
		return
	}

	item, err := h.orderSvc.ToggleServed(c.Request.Context(), actor, orderID, domain.ToggleServedInput{
		ExpectedVersion:   req.ExpectedVersion,
		ExpectedUpdatedAt: req.ExpectedUpdatedAt,
	})
	if err != nil {
		httpresp.AbortError(c, err)
		return
	}

	httpresp.JSON(c, http.StatusOK, httpresp.Success(dto.UpdateOrderResponseData{
		Item: toOrderDTO(*item, h.location),
	}))
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

	httpresp.JSON(c, http.StatusOK, httpresp.Success(nil))
}

// Clear deletes orders from a workspace.
func (h *OrderHandler) Clear(c *gin.Context) {
	actor, ok := actorFromGin(c)
	if !ok {
		httpresp.AbortError(c, apperr.UnauthorizedError("missing auth context"))
		return
	}

	var req dto.ClearOrdersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.AbortError(c, apperr.ValidationError("invalid clear payload"))
		return
	}

	count, err := h.orderSvc.ClearWorkspace(c.Request.Context(), actor, req.Confirm, domain.ClearWorkspaceMode(req.Mode))
	if err != nil {
		httpresp.AbortError(c, err)
		return
	}

	httpresp.JSON(c, http.StatusOK, httpresp.Success(dto.ClearOrdersResponseData{
		ClearedCount: count,
	}))
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

// toOrderDTOs converts orders to the independently versioned v2 representation.
func toOrderDTOs(items []domain.Order, location *time.Location) []dto.Order {
	dtos := make([]dto.Order, 0, len(items))
	for _, item := range items {
		dtos = append(dtos, toOrderDTO(item, location))
	}
	return dtos
}

func toOrderDTO(order domain.Order, location *time.Location) dto.Order {
	meatCodes := make([]int16, len(order.SelectedMeatCodes))
	copy(meatCodes, order.SelectedMeatCodes)
	item := dto.Order{
		ID:                   order.ID.String(),
		Version:              order.Version,
		DisplayNo:            order.DisplayNo,
		StapleTypeCode:       order.StapleTypeCode,
		SizeCode:             order.SizeCode,
		CustomSizePriceCents: order.CustomSizePriceCents,
		StapleAmountCode:     order.StapleAmountCode,
		ExtraStapleUnits:     order.ExtraStapleUnits,
		FriedEggCount:        order.FriedEggCount,
		TofuSkewerCount:      order.TofuSkewerCount,
		SelectedMeatCodes:    meatCodes,
		GreensCode:           order.GreensCode,
		ScallionCode:         order.ScallionCode,
		PepperCode:           order.PepperCode,
		DiningMethodCode:     order.DiningMethodCode,
		PackagingCode:        order.PackagingCode,
		PackagingMethodCode:  order.PackagingMethodCode,
		TotalPriceCents:      order.TotalPriceCents,
		StapleStepStatusCode: order.StapleStepStatusCode,
		MeatStepStatusCode:   order.MeatStepStatusCode,
		Note:                 order.Note,
		CreatedAt:            timex.FormatTime(order.CreatedAt, location),
		UpdatedAt:            timex.FormatTime(order.UpdatedAt, location),
	}
	if order.CompletedAt != nil {
		value := timex.FormatTime(*order.CompletedAt, location)
		item.CompletedAt = &value
	}
	return item
}

func toOrderFormInput(form dto.OrderForm) domain.OrderFormInput {
	return domain.OrderFormInput{
		StapleTypeCode:       form.StapleTypeCode,
		SizeCode:             form.SizeCode,
		CustomSizePriceCents: form.CustomSizePriceCents,
		StapleAmountCode:     form.StapleAmountCode,
		ExtraStapleUnits:     form.ExtraStapleUnits,
		FriedEggCount:        form.FriedEggCount,
		TofuSkewerCount:      form.TofuSkewerCount,
		SelectedMeatCodes:    form.SelectedMeatCodes,
		GreensCode:           form.GreensCode,
		ScallionCode:         form.ScallionCode,
		PepperCode:           form.PepperCode,
		DiningMethodCode:     form.DiningMethodCode,
		PackagingCode:        form.PackagingCode,
		PackagingMethodCode:  form.PackagingMethodCode,
		Note:                 form.Note,
	}
}
