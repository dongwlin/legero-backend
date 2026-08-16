package v1

import (
	"context"
	"fmt"
	"strings"
	"time"

	"aidanwoods.dev/go-paseto"
	"github.com/google/uuid"
	"github.com/uptrace/bun"

	"github.com/dongwlin/legero-backend/internal/apperr"
	"github.com/dongwlin/legero-backend/internal/infra/crypto"
	"github.com/dongwlin/legero-backend/internal/infra/identity"
	"github.com/dongwlin/legero-backend/internal/domain"
	"github.com/dongwlin/legero-backend/internal/repo"
	"github.com/dongwlin/legero-backend/internal/service"
)

// auth implements service.Auth.
type auth struct {
	db         *bun.DB
	orders     service.ActiveOrderLoader
	hasher     *crypto.PasswordHasher
	location   *time.Location
	accessTTL  time.Duration
	refreshTTL time.Duration
	key        paseto.V4SymmetricKey
}

// NewAuth creates a new Auth service.
func NewAuth(
	db *bun.DB,
	orders service.ActiveOrderLoader,
	hasher *crypto.PasswordHasher,
	location *time.Location,
	accessTTL time.Duration,
	refreshTTL time.Duration,
	keyBytes []byte,
) (service.Auth, error) {
	key, err := paseto.V4SymmetricKeyFromBytes(keyBytes)
	if err != nil {
		return nil, fmt.Errorf("create paseto symmetric key: %w", err)
	}

	return &auth{
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
func (s *auth) Login(ctx context.Context, req service.LoginRequest) (*domain.LoginResult, error) {
	normalizedPhone := domain.NormalizePhone(req.Phone)
	if normalizedPhone == "" || strings.TrimSpace(req.Password) == "" {
		return nil, apperr.ValidationError("phone and password are required")
	}

	userRepo := repo.NewUser(s.db)
	user, err := userRepo.GetByPhone(ctx, normalizedPhone)
	if err != nil {
		return nil, apperr.InternalError("failed to load user", err)
	}
	if user == nil || !user.IsActive {
		return nil, apperr.New(apperr.KindUnauthenticated, "invalid_credentials", "invalid phone or password")
	}

	matched, err := s.hasher.Compare(req.Password, user.PasswordHash)
	if err != nil {
		return nil, apperr.InternalError("failed to verify password", err)
	}
	if !matched {
		return nil, apperr.New(apperr.KindUnauthenticated, "invalid_credentials", "invalid phone or password")
	}

	wsRepo := repo.NewWorkspace(s.db)
	access, err := wsRepo.GetPrimaryAccess(ctx, user.ID)
	if err != nil {
		return nil, apperr.InternalError("failed to resolve workspace access", err)
	}
	if access == nil {
		return nil, apperr.NotFoundError("workspace_not_found", "workspace not found")
	}

	activeOrders, err := s.orders.ListActive(ctx, access.WorkspaceID)
	if err != nil {
		return nil, apperr.InternalError("failed to load active orders", err)
	}

	now := time.Now()
	tokenPair, refreshRecord, err := s.issueTokenPair(now, user.ID, access)
	if err != nil {
		return nil, apperr.InternalError("failed to issue token pair", err)
	}

	if err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		txRefreshRepo := repo.NewRefreshToken(tx)
		return txRefreshRepo.Insert(ctx, &refreshRecord)
	}); err != nil {
		return nil, apperr.InternalError("failed to persist refresh token", err)
	}

	return &domain.LoginResult{
		TokenPair: tokenPair,
		Bootstrap: domain.BootstrapData{
			User: domain.AuthUser{
				ID:    user.ID,
				Phone: user.Phone,
				Role:  access.Role,
			},
			Workspace: domain.WorkspaceInfo{
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
func (s *auth) Refresh(ctx context.Context, rawRefreshToken string) (*domain.TokenPair, error) {
	claims, err := s.parseToken(rawRefreshToken, "refresh")
	if err != nil {
		return nil, err
	}

	now := time.Now()
	var pair domain.TokenPair

	if err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		txRefreshRepo := repo.NewRefreshToken(tx)
		txWsRepo := repo.NewWorkspace(tx)

		stored, err := txRefreshRepo.GetByHash(ctx, crypto.HashToken(rawRefreshToken), true)
		if err != nil {
			return err
		}
		if stored == nil {
			return apperr.New(apperr.KindUnauthenticated, "refresh_token_reused", "refresh token is invalid")
		}
		if stored.RevokedAt != nil || stored.RotatedAt != nil || stored.ReplacedByID != nil {
			return apperr.New(apperr.KindUnauthenticated, "refresh_token_reused", "refresh token has already been used")
		}
		if now.After(stored.ExpiresAt) {
			return apperr.New(apperr.KindUnauthenticated, "refresh_token_expired", "refresh token has expired")
		}

		access, err := txWsRepo.GetAccess(ctx, claims.UserID, claims.WorkspaceID)
		if err != nil {
			return err
		}
		if access == nil {
			return apperr.NotFoundError("workspace_not_found", "workspace not found")
		}

		var replacementRecord domain.RefreshToken
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
func (s *auth) Bootstrap(ctx context.Context, authCtx *identity.Context) (*domain.BootstrapData, error) {
	userRepo := repo.NewUser(s.db)
	user, err := userRepo.GetByID(ctx, authCtx.UserID)
	if err != nil {
		return nil, apperr.InternalError("failed to load user", err)
	}
	if user == nil || !user.IsActive {
		return nil, apperr.UnauthorizedError("user is inactive")
	}

	wsRepo := repo.NewWorkspace(s.db)
	access, err := wsRepo.GetAccess(ctx, authCtx.UserID, authCtx.WorkspaceID)
	if err != nil {
		return nil, apperr.InternalError("failed to resolve workspace access", err)
	}
	if access == nil {
		return nil, apperr.NotFoundError("workspace_not_found", "workspace not found")
	}

	activeOrders, err := s.orders.ListActive(ctx, access.WorkspaceID)
	if err != nil {
		return nil, apperr.InternalError("failed to load active orders", err)
	}

	return &domain.BootstrapData{
		User: domain.AuthUser{
			ID:    user.ID,
			Phone: user.Phone,
			Role:  access.Role,
		},
		Workspace: domain.WorkspaceInfo{
			ID:   access.WorkspaceID,
			Name: access.WorkspaceName,
		},
		Permissions:  access.Role.Permissions(),
		ActiveOrders: activeOrders,
		ServerTime:   time.Now(),
	}, nil
}

// RequireAccessToken parses and validates an access token, returning the identity context.
func (s *auth) RequireAccessToken(_ context.Context, rawToken string) (*identity.Context, error) {
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
func (s *auth) issueTokenPair(now time.Time, userID uuid.UUID, access *domain.Access) (domain.TokenPair, domain.RefreshToken, error) {
	accessExpiresAt := now.Add(s.accessTTL)
	refreshExpiresAt := now.Add(s.refreshTTL)
	refreshID := uuid.New()

	accessToken, err := s.encryptToken(domain.TokenClaims{
		UserID:      userID,
		WorkspaceID: access.WorkspaceID,
		Role:        access.Role,
		Type:        "access",
		JTI:         uuid.New().String(),
		ExpiresAt:   accessExpiresAt,
	}, now)
	if err != nil {
		return domain.TokenPair{}, domain.RefreshToken{}, err
	}

	refreshToken, err := s.encryptToken(domain.TokenClaims{
		UserID:      userID,
		WorkspaceID: access.WorkspaceID,
		Role:        access.Role,
		Type:        "refresh",
		JTI:         refreshID.String(),
		ExpiresAt:   refreshExpiresAt,
	}, now)
	if err != nil {
		return domain.TokenPair{}, domain.RefreshToken{}, err
	}

	return domain.TokenPair{
			AccessToken:           accessToken,
			AccessTokenExpiresAt:  accessExpiresAt,
			RefreshToken:          refreshToken,
			RefreshTokenExpiresAt: refreshExpiresAt,
		}, domain.RefreshToken{
			ID:          refreshID,
			UserID:      userID,
			WorkspaceID: access.WorkspaceID,
			TokenHash:   crypto.HashToken(refreshToken),
			ExpiresAt:   refreshExpiresAt,
			CreatedAt:   now,
		}, nil
}

// encryptToken creates a PASETO v4 symmetric token with the given claims.
func (s *auth) encryptToken(claims domain.TokenClaims, now time.Time) (string, error) {
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
func (s *auth) parseToken(rawToken, expectedType string) (*domain.TokenClaims, error) {
	parser := paseto.NewParser()
	parsed, err := parser.ParseV4Local(s.key, rawToken, nil)
	if err != nil {
		lowered := strings.ToLower(err.Error())
		if strings.Contains(lowered, "exp") || strings.Contains(lowered, "expired") {
			if expectedType == "refresh" {
				return nil, apperr.New(apperr.KindUnauthenticated, "refresh_token_expired", "refresh token has expired")
			}
			return nil, apperr.New(apperr.KindUnauthenticated, "token_expired", "access token has expired")
		}
		return nil, apperr.UnauthorizedError("invalid token")
	}

	subject, err := parsed.GetSubject()
	if err != nil {
		return nil, apperr.UnauthorizedError("invalid token subject")
	}
	jti, err := parsed.GetJti()
	if err != nil {
		return nil, apperr.UnauthorizedError("invalid token identifier")
	}
	wid, err := parsed.GetString("wid")
	if err != nil {
		return nil, apperr.UnauthorizedError("invalid token")
	}
	roleText, err := parsed.GetString("role")
	if err != nil {
		return nil, apperr.UnauthorizedError("invalid token")
	}
	tokenType, err := parsed.GetString("typ")
	if err != nil {
		return nil, apperr.UnauthorizedError("invalid token")
	}
	if tokenType != expectedType {
		return nil, apperr.UnauthorizedError("invalid token type")
	}

	userID, err := uuid.Parse(subject)
	if err != nil {
		return nil, apperr.UnauthorizedError("invalid token subject")
	}
	workspaceID, err := uuid.Parse(wid)
	if err != nil {
		return nil, apperr.UnauthorizedError("invalid token workspace")
	}
	expiresAt, err := parsed.GetExpiration()
	if err != nil {
		return nil, apperr.UnauthorizedError("invalid token expiration")
	}

	return &domain.TokenClaims{
		UserID:      userID,
		WorkspaceID: workspaceID,
		Role:        domain.Role(roleText),
		Type:        tokenType,
		JTI:         jti,
		ExpiresAt:   expiresAt,
	}, nil
}
