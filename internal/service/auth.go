package service

import (
	"context"

	"github.com/dongwlin/legero-backend/internal/domain"
	"github.com/dongwlin/legero-backend/internal/infra/identity"
	"github.com/google/uuid"
)

// ActiveOrderLoader abstracts the order list-active dependency used by Auth.
type ActiveOrderLoader interface {
	ListActive(ctx context.Context, workspaceID uuid.UUID) ([]domain.Order, error)
}

// WorkspaceAccessLoader abstracts the workspace-access dependency used by Auth.
// It lets Auth resolve a user's role inside a workspace without depending on a
// concrete repository, so tests can simulate roles the database schema does not
// permit (e.g. a future role) without mutating the shared schema.
type WorkspaceAccessLoader interface {
	GetPrimaryAccess(ctx context.Context, userID uuid.UUID) (*domain.Access, error)
	GetAccess(ctx context.Context, userID, workspaceID uuid.UUID) (*domain.Access, error)
}

// LoginRequest carries the credentials for authentication.
type LoginRequest struct {
	Phone    string
	Password string
}

// Auth handles authentication: login, token refresh, and bootstrap.
type Auth interface {
	Login(ctx context.Context, req LoginRequest) (*domain.LoginResult, error)
	Refresh(ctx context.Context, rawRefreshToken string) (*domain.TokenPair, error)
	Bootstrap(ctx context.Context, authCtx *identity.Context) (*domain.BootstrapData, error)
	RequireAccessToken(ctx context.Context, rawToken string) (*identity.Context, error)
}
