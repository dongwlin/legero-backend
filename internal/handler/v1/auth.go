package v1

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/dongwlin/legero-backend/internal/apperr"
	"github.com/dongwlin/legero-backend/internal/domain"
	"github.com/dongwlin/legero-backend/internal/handler/httpresp"
	"github.com/dongwlin/legero-backend/internal/handler/v1/dto"
	"github.com/dongwlin/legero-backend/internal/infra/identity"
	"github.com/dongwlin/legero-backend/internal/infra/timex"
	"github.com/dongwlin/legero-backend/internal/service"
)

// AuthHandler handles authentication HTTP endpoints.
type AuthHandler struct {
	authSvc  service.Auth
	location *time.Location
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(authSvc service.Auth, location *time.Location) *AuthHandler {
	return &AuthHandler{authSvc: authSvc, location: location}
}

// Login authenticates a user and returns tokens plus bootstrap data.
func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.AbortError(c, apperr.ValidationError("invalid login payload"))
		return
	}

	result, err := h.authSvc.Login(c.Request.Context(), service.LoginRequest{
		Phone:    req.Phone,
		Password: req.Password,
	})
	if err != nil {
		httpresp.AbortError(c, err)
		return
	}

	httpresp.JSON(c, http.StatusOK, dto.LoginResponse{
		TokenPair: toTokenPairDTO(result.TokenPair, h.location),
		Bootstrap: toBootstrapDTO(result.Bootstrap, h.location),
	})
}

// Refresh validates a refresh token and returns a new token pair.
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req dto.RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.AbortError(c, apperr.ValidationError("invalid refresh payload"))
		return
	}

	pair, err := h.authSvc.Refresh(c.Request.Context(), req.RefreshToken)
	if err != nil {
		httpresp.AbortError(c, err)
		return
	}

	httpresp.JSON(c, http.StatusOK, toTokenPairDTO(*pair, h.location))
}

// Bootstrap returns the full bootstrap payload for an already-authenticated user.
func (h *AuthHandler) Bootstrap(c *gin.Context) {
	authCtx, ok := AuthContext(c)
	if !ok {
		httpresp.AbortError(c, apperr.UnauthorizedError("missing auth context"))
		return
	}

	data, err := h.authSvc.Bootstrap(c.Request.Context(), authCtx)
	if err != nil {
		httpresp.AbortError(c, err)
		return
	}

	httpresp.JSON(c, http.StatusOK, dto.BootstrapResponse{
		Bootstrap: toBootstrapDTO(*data, h.location),
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

// toTokenPairDTO converts a domain.TokenPair into its API representation.
func toTokenPairDTO(pair domain.TokenPair, location *time.Location) dto.TokenPair {
	return dto.TokenPair{
		AccessToken:           pair.AccessToken,
		TokenType:             "Bearer",
		AccessTokenExpiresAt:  timex.FormatTime(pair.AccessTokenExpiresAt, location),
		RefreshToken:          pair.RefreshToken,
		RefreshTokenExpiresAt: timex.FormatTime(pair.RefreshTokenExpiresAt, location),
	}
}

// toBootstrapDTO converts a domain.BootstrapData into its API representation.
func toBootstrapDTO(data domain.BootstrapData, location *time.Location) dto.Bootstrap {
	return dto.Bootstrap{
		User: dto.AuthUser{
			ID:    data.User.ID.String(),
			Phone: data.User.Phone,
			Role:  string(data.User.Role),
		},
		Workspace: dto.Workspace{
			ID:   data.Workspace.ID.String(),
			Name: data.Workspace.Name,
		},
		Permissions:  data.Permissions,
		ActiveOrders: toOrderDTOs(data.ActiveOrders, location),
		ServerTime:   timex.FormatTime(data.ServerTime, location),
	}
}
