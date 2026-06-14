package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"aidanwoods.dev/go-paseto"
	"github.com/google/uuid"
	"github.com/uptrace/bun"

	"github.com/dongwlin/legero-backend/internal/infra/crypto"
	"github.com/dongwlin/legero-backend/internal/infra/httpx"
	"github.com/dongwlin/legero-backend/internal/infra/identity"
	"github.com/dongwlin/legero-backend/internal/model"
	"github.com/dongwlin/legero-backend/internal/repo"
)

// ActiveOrderLoader abstracts the order list-active dependency.
type ActiveOrderLoader interface {
	ListActive(ctx context.Context, workspaceID uuid.UUID) ([]model.Order, error)
}

// Auth handles authentication: login, token refresh, and bootstrap.
type Auth struct {
	db         *bun.DB
	orders     ActiveOrderLoader
	hasher     *crypto.PasswordHasher
	location   *time.Location
	accessTTL  time.Duration
	refreshTTL time.Duration
	key        paseto.V4SymmetricKey
}

// NewAuth creates a new AuthService.
func NewAuth(
	db *bun.DB,
	orders ActiveOrderLoader,
	hasher *crypto.PasswordHasher,
	location *time.Location,
	accessTTL time.Duration,
	refreshTTL time.Duration,
	keyBytes []byte,
) (*Auth, error) {
	key, err := paseto.V4SymmetricKeyFromBytes(keyBytes)
	if err != nil {
		return nil, fmt.Errorf("create paseto symmetric key: %w", err)
	}

	return &Auth{
		db:         db,
		orders:     orders,
		hasher:     hasher,
		location:   location,
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
		key:        key,
	}, nil
}

// Login authenticates a user by phone and password, issues tokens, and returns bootstrap data.
func (s *Auth) Login(ctx context.Context, phone, password string) (*model.LoginResult, error) {
	normalizedPhone := model.NormalizePhone(phone)
	if normalizedPhone == "" || strings.TrimSpace(password) == "" {
		return nil, httpx.ValidationError("phone and password are required")
	}

	userRepo := repo.NewUser(s.db)
	user, err := userRepo.GetByPhone(ctx, normalizedPhone)
	if err != nil {
		return nil, httpx.InternalError("failed to load user", err)
	}
	if user == nil || !user.IsActive {
		return nil, httpx.NewError(401, "invalid_credentials", "invalid phone or password")
	}

	matched, err := s.hasher.Compare(password, user.PasswordHash)
	if err != nil {
		return nil, httpx.InternalError("failed to verify password", err)
	}
	if !matched {
		return nil, httpx.NewError(401, "invalid_credentials", "invalid phone or password")
	}

	wsRepo := repo.NewWorkspace(s.db)
	access, err := wsRepo.GetPrimaryAccess(ctx, user.ID)
	if err != nil {
		return nil, httpx.InternalError("failed to resolve workspace access", err)
	}
	if access == nil {
		return nil, httpx.NotFoundError("workspace_not_found", "workspace not found")
	}

	activeOrders, err := s.orders.ListActive(ctx, access.WorkspaceID)
	if err != nil {
		return nil, httpx.InternalError("failed to load active orders", err)
	}

	now := time.Now()
	tokenPair, refreshRecord, err := s.issueTokenPair(now, user.ID, access)
	if err != nil {
		return nil, httpx.InternalError("failed to issue token pair", err)
	}

	if err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		txRefreshRepo := repo.NewRefreshToken(tx)
		return txRefreshRepo.Insert(ctx, &refreshRecord)
	}); err != nil {
		return nil, httpx.InternalError("failed to persist refresh token", err)
	}

	return &model.LoginResult{
		TokenPair: tokenPair,
		Bootstrap: model.BootstrapData{
			User: model.AuthUser{
				ID:    user.ID,
				Phone: user.Phone,
				Role:  access.Role,
			},
			Workspace: model.WorkspaceInfo{
				ID:   access.WorkspaceID,
				Name: access.WorkspaceName,
			},
			Permissions:  access.Role.Permissions(),
			ActiveOrders: activeOrders,
			ServerTime:   now,
		},
	}, nil
}

// Refresh validates a refresh token, rotates it, and issues a new token pair.
func (s *Auth) Refresh(ctx context.Context, rawRefreshToken string) (*model.TokenPair, error) {
	claims, err := s.parseToken(rawRefreshToken, "refresh")
	if err != nil {
		return nil, err
	}

	now := time.Now()
	var pair model.TokenPair

	if err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		txRefreshRepo := repo.NewRefreshToken(tx)
		txWsRepo := repo.NewWorkspace(tx)

		stored, err := txRefreshRepo.GetByHash(ctx, crypto.HashToken(rawRefreshToken), true)
		if err != nil {
			return err
		}
		if stored == nil {
			return httpx.NewError(401, "refresh_token_reused", "refresh token is invalid")
		}
		if stored.RevokedAt != nil || stored.RotatedAt != nil || stored.ReplacedByID != nil {
			return httpx.NewError(401, "refresh_token_reused", "refresh token has already been used")
		}
		if now.After(stored.ExpiresAt) {
			return httpx.NewError(401, "refresh_token_expired", "refresh token has expired")
		}

		access, err := txWsRepo.GetAccess(ctx, claims.UserID, claims.WorkspaceID)
		if err != nil {
			return err
		}
		if access == nil {
			return httpx.NotFoundError("workspace_not_found", "workspace not found")
		}

		var replacementRecord model.RefreshToken
		pair, replacementRecord, err = s.issueTokenPair(now, claims.UserID, access)
		if err != nil {
			return err
		}

		if err := txRefreshRepo.Insert(ctx, &replacementRecord); err != nil {
			return err
		}
		if err := txRefreshRepo.Rotate(ctx, stored.ID, replacementRecord.ID, now); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, wrapError("failed to refresh tokens", err)
	}

	return &pair, nil
}

// Bootstrap returns the full bootstrap payload for an already-authenticated user.
func (s *Auth) Bootstrap(ctx context.Context, authCtx *identity.Context) (*model.BootstrapData, error) {
	userRepo := repo.NewUser(s.db)
	user, err := userRepo.GetByID(ctx, authCtx.UserID)
	if err != nil {
		return nil, httpx.InternalError("failed to load user", err)
	}
	if user == nil || !user.IsActive {
		return nil, httpx.UnauthorizedError("user is inactive")
	}

	wsRepo := repo.NewWorkspace(s.db)
	access, err := wsRepo.GetAccess(ctx, authCtx.UserID, authCtx.WorkspaceID)
	if err != nil {
		return nil, httpx.InternalError("failed to resolve workspace access", err)
	}
	if access == nil {
		return nil, httpx.NotFoundError("workspace_not_found", "workspace not found")
	}

	activeOrders, err := s.orders.ListActive(ctx, access.WorkspaceID)
	if err != nil {
		return nil, httpx.InternalError("failed to load active orders", err)
	}

	return &model.BootstrapData{
		User: model.AuthUser{
			ID:    user.ID,
			Phone: user.Phone,
			Role:  access.Role,
		},
		Workspace: model.WorkspaceInfo{
			ID:   access.WorkspaceID,
			Name: access.WorkspaceName,
		},
		Permissions:  access.Role.Permissions(),
		ActiveOrders: activeOrders,
		ServerTime:   time.Now(),
	}, nil
}

// RequireAccessToken parses and validates an access token, returning the identity context.
func (s *Auth) RequireAccessToken(_ context.Context, rawToken string) (*identity.Context, error) {
	claims, err := s.parseToken(rawToken, "access")
	if err != nil {
		return nil, err
	}

	return &identity.Context{
		UserID:      claims.UserID,
		WorkspaceID: claims.WorkspaceID,
		Role:        string(claims.Role),
	}, nil
}

// issueTokenPair creates a new access/refresh token pair and the refresh record for persistence.
func (s *Auth) issueTokenPair(now time.Time, userID uuid.UUID, access *model.Access) (model.TokenPair, model.RefreshToken, error) {
	accessExpiresAt := now.Add(s.accessTTL)
	refreshExpiresAt := now.Add(s.refreshTTL)
	refreshID := uuid.New()

	accessToken, err := s.encryptToken(model.TokenClaims{
		UserID:      userID,
		WorkspaceID: access.WorkspaceID,
		Role:        access.Role,
		Type:        "access",
		JTI:         uuid.New().String(),
		ExpiresAt:   accessExpiresAt,
	}, now)
	if err != nil {
		return model.TokenPair{}, model.RefreshToken{}, err
	}

	refreshToken, err := s.encryptToken(model.TokenClaims{
		UserID:      userID,
		WorkspaceID: access.WorkspaceID,
		Role:        access.Role,
		Type:        "refresh",
		JTI:         refreshID.String(),
		ExpiresAt:   refreshExpiresAt,
	}, now)
	if err != nil {
		return model.TokenPair{}, model.RefreshToken{}, err
	}

	return model.TokenPair{
			AccessToken:           accessToken,
			AccessTokenExpiresAt:  accessExpiresAt,
			RefreshToken:          refreshToken,
			RefreshTokenExpiresAt: refreshExpiresAt,
		}, model.RefreshToken{
			ID:          refreshID,
			UserID:      userID,
			WorkspaceID: access.WorkspaceID,
			TokenHash:   crypto.HashToken(refreshToken),
			ExpiresAt:   refreshExpiresAt,
			CreatedAt:   now,
		}, nil
}

// encryptToken creates a PASETO v4 symmetric token with the given claims.
func (s *Auth) encryptToken(claims model.TokenClaims, now time.Time) (string, error) {
	token := paseto.NewToken()
	token.SetIssuedAt(now)
	token.SetNotBefore(now)
	token.SetExpiration(claims.ExpiresAt)
	token.SetSubject(claims.UserID.String())
	token.SetJti(claims.JTI)
	token.SetString("wid", claims.WorkspaceID.String())
	token.SetString("role", string(claims.Role))
	token.SetString("typ", claims.Type)

	return token.V4Encrypt(s.key, nil), nil
}

// parseToken validates a PASETO v4 symmetric token and extracts claims.
func (s *Auth) parseToken(rawToken, expectedType string) (*model.TokenClaims, error) {
	parser := paseto.NewParser()
	parsed, err := parser.ParseV4Local(s.key, rawToken, nil)
	if err != nil {
		lowered := strings.ToLower(err.Error())
		if strings.Contains(lowered, "exp") || strings.Contains(lowered, "expired") {
			if expectedType == "refresh" {
				return nil, httpx.NewError(401, "refresh_token_expired", "refresh token has expired")
			}
			return nil, httpx.NewError(401, "token_expired", "access token has expired")
		}
		return nil, httpx.UnauthorizedError("invalid token")
	}

	subject, err := parsed.GetSubject()
	if err != nil {
		return nil, httpx.UnauthorizedError("invalid token subject")
	}
	jti, err := parsed.GetJti()
	if err != nil {
		return nil, httpx.UnauthorizedError("invalid token identifier")
	}
	wid, err := parsed.GetString("wid")
	if err != nil {
		return nil, httpx.UnauthorizedError("invalid token")
	}
	roleText, err := parsed.GetString("role")
	if err != nil {
		return nil, httpx.UnauthorizedError("invalid token")
	}
	tokenType, err := parsed.GetString("typ")
	if err != nil {
		return nil, httpx.UnauthorizedError("invalid token")
	}
	if tokenType != expectedType {
		return nil, httpx.UnauthorizedError("invalid token type")
	}

	userID, err := uuid.Parse(subject)
	if err != nil {
		return nil, httpx.UnauthorizedError("invalid token subject")
	}
	workspaceID, err := uuid.Parse(wid)
	if err != nil {
		return nil, httpx.UnauthorizedError("invalid token workspace")
	}
	expiresAt, err := parsed.GetExpiration()
	if err != nil {
		return nil, httpx.UnauthorizedError("invalid token expiration")
	}

	return &model.TokenClaims{
		UserID:      userID,
		WorkspaceID: workspaceID,
		Role:        model.Role(roleText),
		Type:        tokenType,
		JTI:         jti,
		ExpiresAt:   expiresAt,
	}, nil
}

