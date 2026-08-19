package v1

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/dongwlin/legero-backend/internal/domain"
	"github.com/dongwlin/legero-backend/internal/infra/identity"
	"github.com/dongwlin/legero-backend/internal/service"
)

type authHandlerServiceStub struct {
	bootstrap *domain.BootstrapData
}

func (s authHandlerServiceStub) Login(context.Context, service.LoginRequest) (*domain.LoginResult, error) {
	return nil, nil
}

func (s authHandlerServiceStub) Refresh(context.Context, string) (*domain.TokenPair, error) {
	return nil, nil
}

func (s authHandlerServiceStub) Bootstrap(context.Context, *identity.Context) (*domain.BootstrapData, error) {
	return s.bootstrap, nil
}

func (s authHandlerServiceStub) RequireAccessToken(context.Context, string) (*identity.Context, error) {
	return nil, nil
}

func TestBootstrapUsesPrivatePolicyWithoutETag(t *testing.T) {
	gin.SetMode(gin.TestMode)
	location := time.FixedZone("CST", 8*60*60)
	data := &domain.BootstrapData{
		User: domain.AuthUser{
			ID:    uuid.New(),
			Phone: "13800000001",
			Role:  domain.RoleOwner,
		},
		Workspace: domain.WorkspaceInfo{
			ID:   uuid.New(),
			Name: "Test workspace",
		},
		Permissions: []string{"orders:read"},
		ServerTime:  time.Date(2026, 8, 19, 11, 0, 1, 0, location),
	}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/bootstrap", nil)
	ctx.Request.Header.Set("If-None-Match", `"stale"`)
	ctx.Set(identity.GinContextKey, &identity.Context{})

	handler := NewAuthHandler(authHandlerServiceStub{bootstrap: data}, location)
	handler.Bootstrap(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Empty(t, recorder.Header().Get("ETag"))
	require.Equal(t, "private, no-cache", recorder.Header().Get("Cache-Control"))
	require.Equal(t, "Authorization", recorder.Header().Get("Vary"))
	require.Contains(t, recorder.Body.String(), `"serverTime":"2026-08-19T11:00:01+08:00"`)
}
