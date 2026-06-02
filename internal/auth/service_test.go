package auth

import (
	"context"
	"crypto/rand"
	"errors"
	"testing"
	"time"

	"github.com/dongwlin/legero-backend/internal/infra/config"
	"github.com/dongwlin/legero-backend/internal/infra/httpx"
	"github.com/dongwlin/legero-backend/internal/infra/identity"
	"github.com/dongwlin/legero-backend/internal/order"
	"github.com/dongwlin/legero-backend/internal/workspace"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
)

// mockOrderLoader implements ActiveOrderLoader for tests.
type mockOrderLoader struct{}

func (m *mockOrderLoader) ListActive(_ context.Context, _ uuid.UUID) ([]order.Order, error) {
	return []order.Order{}, nil
}

// newTestService creates a Service wired to testDB with test-friendly settings.
func newTestService(t *testing.T, db *bun.DB) *Service {
	t.Helper()

	keyBytes := make([]byte, 32)
	_, err := rand.Read(keyBytes)
	require.NoError(t, err)

	hasher := NewPasswordHasher(config.Argon2Config{
		MemoryKiB:   8 * 1024,
		Iterations:  1,
		Parallelism: 1,
		SaltLength:  16,
		KeyLength:   32,
	})

	svc, err := NewService(
		db,
		&mockOrderLoader{},
		hasher,
		time.UTC,
		15*time.Minute,
		7*24*time.Hour,
		keyBytes,
	)
	require.NoError(t, err)
	return svc
}

// ---------------------------------------------------------------------------
// Service.Login
// ---------------------------------------------------------------------------

func TestLogin_Success(t *testing.T) {
	ctx := context.Background()
	svc := newTestService(t, testDB)

	userID := createTestUser(t, ctx, testDB, func(u *User) {
		u.Phone = "13800001001"
		u.IsActive = true
	})
	wsID := createTestWorkspace(t, ctx, testDB)
	createTestWorkspaceMember(t, ctx, testDB, userID, wsID, "owner")

	result, err := svc.Login(ctx, "13800001001", "password123")
	require.NoError(t, err)
	require.NotNil(t, result)

	// Token pair.
	require.NotEmpty(t, result.TokenPair.AccessToken)
	require.NotEmpty(t, result.TokenPair.RefreshToken)
	require.True(t, result.TokenPair.AccessTokenExpiresAt.After(time.Now()))
	require.True(t, result.TokenPair.RefreshTokenExpiresAt.After(time.Now()))

	// Bootstrap data.
	require.Equal(t, userID, result.Bootstrap.User.ID)
	require.Equal(t, "13800001001", result.Bootstrap.User.Phone)
	require.Equal(t, workspace.RoleOwner, result.Bootstrap.User.Role)
	require.Equal(t, wsID, result.Bootstrap.Workspace.ID)
	require.Equal(t, "test-workspace", result.Bootstrap.Workspace.Name)
	require.Contains(t, result.Bootstrap.Permissions, "orders:read")
	require.Contains(t, result.Bootstrap.Permissions, "orders:write")
	require.Contains(t, result.Bootstrap.Permissions, "orders:clear")
}

func TestLogin_InvalidPhone(t *testing.T) {
	ctx := context.Background()
	svc := newTestService(t, testDB)

	_, err := svc.Login(ctx, "00000000000", "password123")
	require.Error(t, err)

	var appErr *httpx.AppError
	require.True(t, errors.As(err, &appErr))
	require.Equal(t, 401, appErr.Status)
}

func TestLogin_WrongPassword(t *testing.T) {
	ctx := context.Background()
	svc := newTestService(t, testDB)

	createTestUser(t, ctx, testDB, func(u *User) {
		u.Phone = "13800001002"
		u.IsActive = true
	})

	_, err := svc.Login(ctx, "13800001002", "wrongpassword")
	require.Error(t, err)

	var appErr *httpx.AppError
	require.True(t, errors.As(err, &appErr))
	require.Equal(t, 401, appErr.Status)
}

func TestLogin_InactiveUser(t *testing.T) {
	ctx := context.Background()
	svc := newTestService(t, testDB)

	createTestUser(t, ctx, testDB, func(u *User) {
		u.Phone = "13800001003"
		u.IsActive = false
	})

	_, err := svc.Login(ctx, "13800001003", "password123")
	require.Error(t, err)

	var appErr *httpx.AppError
	require.True(t, errors.As(err, &appErr))
	require.Equal(t, 401, appErr.Status)
}

func TestLogin_NoWorkspace(t *testing.T) {
	ctx := context.Background()
	svc := newTestService(t, testDB)

	createTestUser(t, ctx, testDB, func(u *User) {
		u.Phone = "13800001004"
		u.IsActive = true
	})
	// No workspace member created.

	_, err := svc.Login(ctx, "13800001004", "password123")
	require.Error(t, err)

	var appErr *httpx.AppError
	require.True(t, errors.As(err, &appErr))
	require.Equal(t, 404, appErr.Status)
}

// ---------------------------------------------------------------------------
// Service.Refresh
// ---------------------------------------------------------------------------

func TestRefresh_Success(t *testing.T) {
	ctx := context.Background()
	svc := newTestService(t, testDB)

	userID := createTestUser(t, ctx, testDB, func(u *User) {
		u.Phone = "13800002001"
		u.IsActive = true
	})
	wsID := createTestWorkspace(t, ctx, testDB)
	createTestWorkspaceMember(t, ctx, testDB, userID, wsID, "owner")

	loginResult, err := svc.Login(ctx, "13800002001", "password123")
	require.NoError(t, err)

	pair, err := svc.Refresh(ctx, loginResult.TokenPair.RefreshToken)
	require.NoError(t, err)
	require.NotNil(t, pair)
	require.NotEmpty(t, pair.AccessToken)
	require.NotEmpty(t, pair.RefreshToken)
	require.NotEqual(t, loginResult.TokenPair.AccessToken, pair.AccessToken)
	require.NotEqual(t, loginResult.TokenPair.RefreshToken, pair.RefreshToken)
}

func TestRefresh_ExpiredToken(t *testing.T) {
	ctx := context.Background()

	// Create a service with a very short refresh TTL so the token is already expired.
	keyBytes := make([]byte, 32)
	_, err := rand.Read(keyBytes)
	require.NoError(t, err)

	hasher := NewPasswordHasher(config.Argon2Config{
		MemoryKiB:   8 * 1024,
		Iterations:  1,
		Parallelism: 1,
		SaltLength:  16,
		KeyLength:   32,
	})

	svc, err := NewService(
		testDB,
		&mockOrderLoader{},
		hasher,
		time.UTC,
		15*time.Minute,
		1*time.Millisecond, // very short refresh TTL
		keyBytes,
	)
	require.NoError(t, err)

	userID := createTestUser(t, ctx, testDB, func(u *User) {
		u.Phone = "13800002002"
		u.IsActive = true
	})
	wsID := createTestWorkspace(t, ctx, testDB)
	createTestWorkspaceMember(t, ctx, testDB, userID, wsID, "owner")

	loginResult, err := svc.Login(ctx, "13800002002", "password123")
	require.NoError(t, err)

	// Wait for the token to expire.
	time.Sleep(10 * time.Millisecond)

	_, err = svc.Refresh(ctx, loginResult.TokenPair.RefreshToken)
	require.Error(t, err)

	var appErr *httpx.AppError
	require.True(t, errors.As(err, &appErr))
	require.Equal(t, 401, appErr.Status)
}

func TestRefresh_RevokedToken(t *testing.T) {
	ctx := context.Background()
	svc := newTestService(t, testDB)

	userID := createTestUser(t, ctx, testDB, func(u *User) {
		u.Phone = "13800002003"
		u.IsActive = true
	})
	wsID := createTestWorkspace(t, ctx, testDB)
	createTestWorkspaceMember(t, ctx, testDB, userID, wsID, "owner")

	loginResult, err := svc.Login(ctx, "13800002003", "password123")
	require.NoError(t, err)

	// Revoke the refresh token by setting revoked_at.
	tokenHash := hashToken(loginResult.TokenPair.RefreshToken)
	now := time.Now()
	_, err = testDB.NewUpdate().
		Model((*RefreshTokenModel)(nil)).
		Set("revoked_at = ?", now).
		Where("token_hash = ?", tokenHash).
		Exec(ctx)
	require.NoError(t, err)

	_, err = svc.Refresh(ctx, loginResult.TokenPair.RefreshToken)
	require.Error(t, err)

	var appErr *httpx.AppError
	require.True(t, errors.As(err, &appErr))
	require.Equal(t, 401, appErr.Status)
}

// ---------------------------------------------------------------------------
// Service.Bootstrap
// ---------------------------------------------------------------------------

func TestBootstrap_Success(t *testing.T) {
	ctx := context.Background()
	svc := newTestService(t, testDB)

	userID := createTestUser(t, ctx, testDB, func(u *User) {
		u.Phone = "13800003001"
		u.IsActive = true
	})
	wsID := createTestWorkspace(t, ctx, testDB)
	createTestWorkspaceMember(t, ctx, testDB, userID, wsID, "staff")

	authCtx := &identity.Context{
		UserID:      userID,
		WorkspaceID: wsID,
		Role:        string(workspace.RoleStaff),
	}

	data, err := svc.Bootstrap(ctx, authCtx)
	require.NoError(t, err)
	require.NotNil(t, data)

	require.Equal(t, userID, data.User.ID)
	require.Equal(t, "13800003001", data.User.Phone)
	require.Equal(t, workspace.RoleStaff, data.User.Role)
	require.Equal(t, wsID, data.Workspace.ID)
	require.Equal(t, "test-workspace", data.Workspace.Name)
	require.Contains(t, data.Permissions, "orders:read")
	require.Contains(t, data.Permissions, "orders:write")
	require.NotContains(t, data.Permissions, "orders:clear")
	require.NotNil(t, data.ActiveOrders)
	require.Len(t, data.ActiveOrders, 0)
}

func TestBootstrap_InactiveUser(t *testing.T) {
	ctx := context.Background()
	svc := newTestService(t, testDB)

	userID := createTestUser(t, ctx, testDB, func(u *User) {
		u.Phone = "13800003002"
		u.IsActive = false
	})
	wsID := createTestWorkspace(t, ctx, testDB)
	createTestWorkspaceMember(t, ctx, testDB, userID, wsID, "owner")

	authCtx := &identity.Context{
		UserID:      userID,
		WorkspaceID: wsID,
		Role:        string(workspace.RoleOwner),
	}

	_, err := svc.Bootstrap(ctx, authCtx)
	require.Error(t, err)

	var appErr *httpx.AppError
	require.True(t, errors.As(err, &appErr))
	require.Equal(t, 401, appErr.Status)
}

func TestBootstrap_NoWorkspace(t *testing.T) {
	ctx := context.Background()
	svc := newTestService(t, testDB)

	userID := createTestUser(t, ctx, testDB, func(u *User) {
		u.Phone = "13800003003"
		u.IsActive = true
	})

	authCtx := &identity.Context{
		UserID:      userID,
		WorkspaceID: uuid.New(),
		Role:        string(workspace.RoleOwner),
	}

	_, err := svc.Bootstrap(ctx, authCtx)
	require.Error(t, err)

	var appErr *httpx.AppError
	require.True(t, errors.As(err, &appErr))
	require.Equal(t, 404, appErr.Status)
}
