package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	paseto "aidanwoods.dev/go-paseto"
	"github.com/google/uuid"
	"github.com/uptrace/bun"

	"github.com/dongwlin/legero-backend/internal/order"
	clockpkg "github.com/dongwlin/legero-backend/internal/infra/clock"
	"github.com/dongwlin/legero-backend/internal/infra/httpx"
	"github.com/dongwlin/legero-backend/internal/infra/identity"
	idspkg "github.com/dongwlin/legero-backend/internal/infra/ids"
	"github.com/dongwlin/legero-backend/internal/workspace"
)

type ActiveOrderLoader interface {
	ListActive(ctx context.Context, workspaceID uuid.UUID) ([]order.Order, error)
}

type Service struct {
	db            *bun.DB
	users         UserRepository
	refreshTokens RefreshTokenRepository
	workspaces    workspace.Repository
	orders        ActiveOrderLoader
	hasher        *PasswordHasher
	clock         clockpkg.Clock
	ids           idspkg.Generator
	location      *time.Location
	accessTTL     time.Duration
	refreshTTL    time.Duration
	key           paseto.V4SymmetricKey
}

func NewService(
	database *bun.DB,
	users UserRepository,
	refreshTokens RefreshTokenRepository,
	workspaces workspace.Repository,
	orders ActiveOrderLoader,
	hasher *PasswordHasher,
	clock clockpkg.Clock,
	ids idspkg.Generator,
	location *time.Location,
	accessTTL time.Duration,
	refreshTTL time.Duration,
	keyBytes []byte,
) (*Service, error) {
	key, err := paseto.V4SymmetricKeyFromBytes(keyBytes)
	if err != nil {
		return nil, fmt.Errorf("create paseto symmetric key: %w", err)
	}

	return &Service{
		db:            database,
		users:         users,
		refreshTokens: refreshTokens,
		workspaces:    workspaces,
		orders:        orders,
		hasher:        hasher,
		clock:         clock,
		ids:           ids,
		location:      location,
		accessTTL:     accessTTL,
		refreshTTL:    refreshTTL,
		key:           key,
	}, nil
}

func (s *Service) Login(ctx context.Context, phone, password string) (*LoginResult, error) {
	normalizedPhone := NormalizePhone(phone)
	if normalizedPhone == "" || strings.TrimSpace(password) == "" {
		return nil, httpx.ValidationError("phone and password are required")
	}

	user, err := s.users.GetByPhone(ctx, s.db, normalizedPhone)
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

	access, err := s.workspaces.GetPrimaryAccess(ctx, s.db, user.ID)
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

	now := s.clock.Now()
	tokenPair, refreshRecord, err := s.issueTokenPair(now, user.ID, access)
	if err != nil {
		return nil, httpx.InternalError("failed to issue token pair", err)
	}

	if err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		return s.refreshTokens.Insert(ctx, tx, refreshRecord)
	}); err != nil {
		return nil, httpx.InternalError("failed to persist refresh token", err)
	}

	return &LoginResult{
		TokenPair: tokenPair,
		Bootstrap: BootstrapData{
			User: AuthUser{
				ID:    user.ID,
				Phone: user.Phone,
				Role:  access.Role,
			},
			Workspace: WorkspaceInfo{
				ID:   access.WorkspaceID,
				Name: access.WorkspaceName,
			},
			Permissions:  workspace.Permissions(access.Role),
			ActiveOrders: activeOrders,
			ServerTime:   now,
		},
	}, nil
}

func (s *Service) Refresh(ctx context.Context, rawRefreshToken string) (*TokenPair, error) {
	claims, err := s.parseToken(rawRefreshToken, "refresh")
	if err != nil {
		return nil, err
	}

	now := s.clock.Now()
	var pair TokenPair

	if err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		stored, err := s.refreshTokens.GetByHash(ctx, tx, hashToken(rawRefreshToken), true)
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

		access, err := s.workspaces.GetAccess(ctx, tx, claims.UserID, claims.WorkspaceID)
		if err != nil {
			return err
		}
		if access == nil {
			return httpx.NotFoundError("workspace_not_found", "workspace not found")
		}

		var replacementRecord RefreshToken
		pair, replacementRecord, err = s.issueTokenPair(now, claims.UserID, access)
		if err != nil {
			return err
		}

		if err := s.refreshTokens.Insert(ctx, tx, replacementRecord); err != nil {
			return err
		}
		if err := s.refreshTokens.Rotate(ctx, tx, stored.ID, replacementRecord.ID, now); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, wrapAuthError("failed to refresh tokens", err)
	}

	return &pair, nil
}

func (s *Service) Bootstrap(ctx context.Context, authCtx *identity.Context) (*BootstrapData, error) {
	user, err := s.users.GetByID(ctx, s.db, authCtx.UserID)
	if err != nil {
		return nil, httpx.InternalError("failed to load user", err)
	}
	if user == nil || !user.IsActive {
		return nil, httpx.UnauthorizedError("user is inactive")
	}

	access, err := s.workspaces.GetAccess(ctx, s.db, authCtx.UserID, authCtx.WorkspaceID)
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

	return &BootstrapData{
		User: AuthUser{
			ID:    user.ID,
			Phone: user.Phone,
			Role:  access.Role,
		},
		Workspace: WorkspaceInfo{
			ID:   access.WorkspaceID,
			Name: access.WorkspaceName,
		},
		Permissions:  workspace.Permissions(access.Role),
		ActiveOrders: activeOrders,
		ServerTime:   s.clock.Now(),
	}, nil
}

func (s *Service) RequireAccessToken(_ context.Context, rawToken string) (*identity.Context, error) {
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

func (s *Service) issueTokenPair(now time.Time, userID uuid.UUID, access *workspace.Access) (TokenPair, RefreshToken, error) {
	accessExpiresAt := now.Add(s.accessTTL)
	refreshExpiresAt := now.Add(s.refreshTTL)
	refreshID := s.ids.New()

	accessToken, err := s.encryptToken(TokenClaims{
		UserID:      userID,
		WorkspaceID: access.WorkspaceID,
		Role:        access.Role,
		Type:        "access",
		JTI:         s.ids.New().String(),
		ExpiresAt:   accessExpiresAt,
	}, now)
	if err != nil {
		return TokenPair{}, RefreshToken{}, err
	}

	refreshToken, err := s.encryptToken(TokenClaims{
		UserID:      userID,
		WorkspaceID: access.WorkspaceID,
		Role:        access.Role,
		Type:        "refresh",
		JTI:         refreshID.String(),
		ExpiresAt:   refreshExpiresAt,
	}, now)
	if err != nil {
		return TokenPair{}, RefreshToken{}, err
	}

	return TokenPair{
			AccessToken:           accessToken,
			AccessTokenExpiresAt:  accessExpiresAt,
			RefreshToken:          refreshToken,
			RefreshTokenExpiresAt: refreshExpiresAt,
		}, RefreshToken{
			ID:          refreshID,
			UserID:      userID,
			WorkspaceID: access.WorkspaceID,
			TokenHash:   hashToken(refreshToken),
			ExpiresAt:   refreshExpiresAt,
			CreatedAt:   now,
		}, nil
}

func (s *Service) encryptToken(claims TokenClaims, now time.Time) (string, error) {
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

func (s *Service) parseToken(rawToken, expectedType string) (*TokenClaims, error) {
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

	return &TokenClaims{
		UserID:      userID,
		WorkspaceID: workspaceID,
		Role:        workspace.Role(roleText),
		Type:        tokenType,
		JTI:         jti,
		ExpiresAt:   expiresAt,
	}, nil
}

func hashToken(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func wrapAuthError(message string, err error) error {
	var appErr *httpx.AppError
	if errors.As(err, &appErr) {
		return err
	}
	return httpx.InternalError(message, err)
}
