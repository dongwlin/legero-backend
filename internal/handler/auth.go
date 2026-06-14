package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/dongwlin/legero-backend/internal/infra/httpx"
	"github.com/dongwlin/legero-backend/internal/infra/identity"
	"github.com/dongwlin/legero-backend/internal/infra/timex"
	"github.com/dongwlin/legero-backend/internal/model"
	"github.com/dongwlin/legero-backend/internal/service"
)

// Auth handles authentication HTTP endpoints.
type Auth struct {
	svc      *service.Auth
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
	ActiveOrders []model.OrderDTO `json:"activeOrders"`
	ServerTime   string           `json:"serverTime"`
}

// NewAuth creates a new AuthHandler.
func NewAuth(svc *service.Auth, location *time.Location) *Auth {
	return &Auth{
		svc:      svc,
		location: location,
	}
}

// Login authenticates a user and returns tokens plus bootstrap data.
func (h *Auth) Login(c *gin.Context) {
	var request loginRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		httpx.AbortError(c, httpx.ValidationError("invalid login payload"))
		return
	}

	result, err := h.svc.Login(c.Request.Context(), request.Phone, request.Password)
	if err != nil {
		httpx.AbortError(c, err)
		return
	}

	httpx.JSON(c, http.StatusOK, gin.H{
		"accessToken":           result.TokenPair.AccessToken,
		"tokenType":             "Bearer",
		"accessTokenExpiresAt":  timex.FormatTime(result.TokenPair.AccessTokenExpiresAt, h.location),
		"refreshToken":          result.TokenPair.RefreshToken,
		"refreshTokenExpiresAt": timex.FormatTime(result.TokenPair.RefreshTokenExpiresAt, h.location),
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
		"serverTime":   timex.FormatTime(result.Bootstrap.ServerTime, h.location),
	})
}

// Refresh validates a refresh token and returns a new token pair.
func (h *Auth) Refresh(c *gin.Context) {
	var request refreshRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		httpx.AbortError(c, httpx.ValidationError("invalid refresh payload"))
		return
	}

	pair, err := h.svc.Refresh(c.Request.Context(), request.RefreshToken)
	if err != nil {
		httpx.AbortError(c, err)
		return
	}

	httpx.JSON(c, http.StatusOK, gin.H{
		"accessToken":           pair.AccessToken,
		"tokenType":             "Bearer",
		"accessTokenExpiresAt":  timex.FormatTime(pair.AccessTokenExpiresAt, h.location),
		"refreshToken":          pair.RefreshToken,
		"refreshTokenExpiresAt": timex.FormatTime(pair.RefreshTokenExpiresAt, h.location),
	})
}

// Bootstrap returns the full bootstrap payload for an already-authenticated user.
func (h *Auth) Bootstrap(c *gin.Context) {
	authCtx, ok := AuthContext(c)
	if !ok {
		httpx.AbortError(c, httpx.UnauthorizedError("missing auth context"))
		return
	}

	data, err := h.svc.Bootstrap(c.Request.Context(), authCtx)
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
		ServerTime:   timex.FormatTime(data.ServerTime, h.location),
	})
}

// AuthContext extracts the identity.Context stored by the auth middleware.
func AuthContext(c *gin.Context) (*identity.Context, bool) {
	value, ok := c.Get(identity.GinContextKey)
	if !ok {
		return nil, false
	}
	authCtx, ok := value.(*identity.Context)
	return authCtx, ok
}
