package order

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/dongwlin/legero-backend/internal/infra/httpx"
	"github.com/dongwlin/legero-backend/internal/infra/identity"
	"github.com/dongwlin/legero-backend/internal/workspace"
)

type Handler struct {
	service  *Service
	location *time.Location
}

func NewHandler(service *Service, location *time.Location) *Handler {
	return &Handler{
		service:  service,
		location: location,
	}
}

func (h *Handler) List(c *gin.Context) {
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

	query := ListOrdersQuery{
		Status: ListStatus(c.DefaultQuery("status", string(ListStatusUncompleted))),
		Limit:  limit,
		Cursor: c.Query("cursor"),
	}

	result, err := h.service.List(c.Request.Context(), actor, query)
	if err != nil {
		httpx.AbortError(c, err)
		return
	}

	items := make([]OrderDTO, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, ToDTO(item, h.location))
	}

	httpx.JSON(c, http.StatusOK, gin.H{
		"items":      items,
		"nextCursor": result.NextCursor,
	})
}

func (h *Handler) Create(c *gin.Context) {
	actor, ok := actorFromGin(c)
	if !ok {
		httpx.AbortError(c, httpx.UnauthorizedError("missing auth context"))
		return
	}

	var input CreateOrdersInput
	if err := c.ShouldBindJSON(&input); err != nil {
		httpx.AbortError(c, httpx.ValidationError("invalid create order payload"))
		return
	}

	items, err := h.service.CreateBatch(c.Request.Context(), actor, input)
	if err != nil {
		httpx.AbortError(c, err)
		return
	}

	dtos := make([]OrderDTO, 0, len(items))
	for _, item := range items {
		dtos = append(dtos, ToDTO(item, h.location))
	}

	httpx.JSON(c, http.StatusOK, gin.H{
		"items": dtos,
	})
}

func (h *Handler) Update(c *gin.Context) {
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

	var input UpdateOrderInput
	if err := c.ShouldBindJSON(&input); err != nil {
		httpx.AbortError(c, httpx.ValidationError("invalid update order payload"))
		return
	}

	item, err := h.service.UpdateForm(c.Request.Context(), actor, orderID, input)
	if err != nil {
		httpx.AbortError(c, err)
		return
	}

	httpx.JSON(c, http.StatusOK, gin.H{
		"item": ToDTO(*item, h.location),
	})
}

func (h *Handler) ToggleStep(c *gin.Context) {
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

	var input ToggleStepInput
	if err := c.ShouldBindJSON(&input); err != nil {
		httpx.AbortError(c, httpx.ValidationError("invalid toggle step payload"))
		return
	}

	item, err := h.service.ToggleStep(c.Request.Context(), actor, orderID, input)
	if err != nil {
		httpx.AbortError(c, err)
		return
	}

	httpx.JSON(c, http.StatusOK, gin.H{
		"item": ToDTO(*item, h.location),
	})
}

func (h *Handler) ToggleServed(c *gin.Context) {
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

	var input ToggleServedInput
	if err := c.ShouldBindJSON(&input); err != nil {
		httpx.AbortError(c, httpx.ValidationError("invalid toggle served payload"))
		return
	}

	item, err := h.service.ToggleServed(c.Request.Context(), actor, orderID, input)
	if err != nil {
		httpx.AbortError(c, err)
		return
	}

	httpx.JSON(c, http.StatusOK, gin.H{
		"item": ToDTO(*item, h.location),
	})
}

func (h *Handler) Delete(c *gin.Context) {
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

	if err := h.service.Remove(c.Request.Context(), actor, orderID); err != nil {
		httpx.AbortError(c, err)
		return
	}

	httpx.NoContent(c)
}

func (h *Handler) Clear(c *gin.Context) {
	actor, ok := actorFromGin(c)
	if !ok {
		httpx.AbortError(c, httpx.UnauthorizedError("missing auth context"))
		return
	}

	var input ClearWorkspaceInput
	if err := c.ShouldBindJSON(&input); err != nil {
		httpx.AbortError(c, httpx.ValidationError("invalid clear payload"))
		return
	}

	count, err := h.service.ClearWorkspace(c.Request.Context(), actor, input.Confirm, input.Mode)
	if err != nil {
		httpx.AbortError(c, err)
		return
	}

	httpx.JSON(c, http.StatusOK, gin.H{
		"clearedCount": count,
	})
}

func actorFromGin(c *gin.Context) (Actor, bool) {
	value, ok := c.Get(identity.GinContextKey)
	if !ok {
		return Actor{}, false
	}
	authCtx, ok := value.(*identity.Context)
	if !ok {
		return Actor{}, false
	}
	return Actor{
		UserID:      authCtx.UserID,
		WorkspaceID: authCtx.WorkspaceID,
		Role:        workspace.Role(authCtx.Role),
	}, true
}

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
