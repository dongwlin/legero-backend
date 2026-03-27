package auth

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/dongwlin/legero-backend/internal/order"
	"github.com/dongwlin/legero-backend/internal/infra/httpx"
)

type Handler struct {
	service  *Service
	location *time.Location
}

type loginRequest struct {
	Phone    string `json:"phone"`
	Password string `json:"password"`
}

type refreshRequest struct {
	RefreshToken string `json:"refreshToken"`
}

type authUserDTO struct {
	ID    string `json:"id"`
	Phone string `json:"phone"`
	Role  string `json:"role"`
}

type workspaceDTO struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type bootstrapResponse struct {
	User         authUserDTO      `json:"user"`
	Workspace    workspaceDTO     `json:"workspace"`
	Permissions  []string         `json:"permissions"`
	ActiveOrders []order.OrderDTO `json:"activeOrders"`
	ServerTime   string           `json:"serverTime"`
}

func NewHandler(service *Service, location *time.Location) *Handler {
	return &Handler{
		service:  service,
		location: location,
	}
}

func (h *Handler) Login(c *gin.Context) {
	var request loginRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		httpx.AbortError(c, httpx.ValidationError("invalid login payload"))
		return
	}

	result, err := h.service.Login(c.Request.Context(), request.Phone, request.Password)
	if err != nil {
		httpx.AbortError(c, err)
		return
	}

	httpx.JSON(c, http.StatusOK, gin.H{
		"accessToken":           result.TokenPair.AccessToken,
		"tokenType":             "Bearer",
		"accessTokenExpiresAt":  formatTime(result.TokenPair.AccessTokenExpiresAt, h.location),
		"refreshToken":          result.TokenPair.RefreshToken,
		"refreshTokenExpiresAt": formatTime(result.TokenPair.RefreshTokenExpiresAt, h.location),
		"user": authUserDTO{
			ID:    result.Bootstrap.User.ID.String(),
			Phone: result.Bootstrap.User.Phone,
			Role:  string(result.Bootstrap.User.Role),
		},
		"workspace": workspaceDTO{
			ID:   result.Bootstrap.Workspace.ID.String(),
			Name: result.Bootstrap.Workspace.Name,
		},
		"permissions":  result.Bootstrap.Permissions,
		"activeOrders": toOrderDTOs(result.Bootstrap.ActiveOrders, h.location),
		"serverTime":   formatTime(result.Bootstrap.ServerTime, h.location),
	})
}

func (h *Handler) Refresh(c *gin.Context) {
	var request refreshRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		httpx.AbortError(c, httpx.ValidationError("invalid refresh payload"))
		return
	}

	pair, err := h.service.Refresh(c.Request.Context(), request.RefreshToken)
	if err != nil {
		httpx.AbortError(c, err)
		return
	}

	httpx.JSON(c, http.StatusOK, gin.H{
		"accessToken":           pair.AccessToken,
		"tokenType":             "Bearer",
		"accessTokenExpiresAt":  formatTime(pair.AccessTokenExpiresAt, h.location),
		"refreshToken":          pair.RefreshToken,
		"refreshTokenExpiresAt": formatTime(pair.RefreshTokenExpiresAt, h.location),
	})
}

func (h *Handler) Bootstrap(c *gin.Context) {
	authCtx, ok := ContextFromGin(c)
	if !ok {
		httpx.AbortError(c, httpx.UnauthorizedError("missing auth context"))
		return
	}

	data, err := h.service.Bootstrap(c.Request.Context(), authCtx)
	if err != nil {
		httpx.AbortError(c, err)
		return
	}

	httpx.JSON(c, http.StatusOK, bootstrapResponse{
		User: authUserDTO{
			ID:    data.User.ID.String(),
			Phone: data.User.Phone,
			Role:  string(data.User.Role),
		},
		Workspace: workspaceDTO{
			ID:   data.Workspace.ID.String(),
			Name: data.Workspace.Name,
		},
		Permissions:  data.Permissions,
		ActiveOrders: toOrderDTOs(data.ActiveOrders, h.location),
		ServerTime:   formatTime(data.ServerTime, h.location),
	})
}

func toOrderDTOs(items []order.Order, location *time.Location) []order.OrderDTO {
	dtos := make([]order.OrderDTO, 0, len(items))
	for _, item := range items {
		dtos = append(dtos, order.ToDTO(item, location))
	}
	return dtos
}

func formatTime(value time.Time, location *time.Location) string {
	if location == nil {
		return value.Format(time.RFC3339)
	}
	return value.In(location).Format(time.RFC3339)
}
