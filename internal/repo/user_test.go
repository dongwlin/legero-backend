package repo

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/dongwlin/legero-backend/internal/model"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// UserRepo.GetByPhone
// ---------------------------------------------------------------------------

func TestUserRepo_GetByPhone_Found(t *testing.T) {
	ctx := context.Background()
	tx, repo := newTestUserRepo(t, ctx)

	userID := createTestUser(t, ctx, tx, func(u *model.User) {
		u.Phone = "13800138000"
		u.IsActive = true
	})

	got, err := repo.GetByPhone(ctx, "13800138000")
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, userID, got.ID)
	require.Equal(t, "13800138000", got.Phone)
	require.True(t, got.IsActive)
	require.NotEmpty(t, got.PasswordHash)
}

func TestUserRepo_GetByPhone_NotFound(t *testing.T) {
	ctx := context.Background()
	_, repo := newTestUserRepo(t, ctx)

	got, err := repo.GetByPhone(ctx, "00000000000")
	require.NoError(t, err)
	require.Nil(t, got)
}

// ---------------------------------------------------------------------------
// UserRepo.GetByID
// ---------------------------------------------------------------------------

func TestUserRepo_GetByID_Found(t *testing.T) {
	ctx := context.Background()
	tx, repo := newTestUserRepo(t, ctx)

	userID := createTestUser(t, ctx, tx, func(u *model.User) {
		u.Phone = "13900139000"
		u.IsActive = false
	})

	got, err := repo.GetByID(ctx, userID)
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, userID, got.ID)
	require.Equal(t, "13900139000", got.Phone)
	require.False(t, got.IsActive)
	require.NotEmpty(t, got.PasswordHash)
}

func TestUserRepo_GetByID_NotFound(t *testing.T) {
	ctx := context.Background()
	_, repo := newTestUserRepo(t, ctx)

	got, err := repo.GetByID(ctx, uuid.New())
	require.NoError(t, err)
	require.Nil(t, got)
}

// ---------------------------------------------------------------------------
// UserRepo.Insert
// ---------------------------------------------------------------------------

func TestUserRepo_Insert(t *testing.T) {
	ctx := context.Background()
	_, repo := newTestUserRepo(t, ctx)

	user := &model.User{
		ID:           uuid.New(),
		Phone:        "13800138001",
		PasswordHash: "hash",
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	err := repo.Insert(ctx, user)
	require.NoError(t, err)

	got, err := repo.GetByID(ctx, user.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, user.Phone, got.Phone)
}

// ---------------------------------------------------------------------------
// RefreshTokenRepo.Insert
// ---------------------------------------------------------------------------

func TestRefreshTokenRepo_Insert(t *testing.T) {
	ctx := context.Background()
	tx, repo := newTestRefreshTokenRepo(t, ctx)

	userID := createTestUser(t, ctx, tx)
	wsID := createTestWorkspace(t, ctx, tx)
	createTestWorkspaceMember(t, ctx, tx, userID, wsID, "owner")

	token := createTestRefreshToken(t, ctx, tx, userID, wsID)

	got, err := repo.GetByHash(ctx, token.TokenHash, false)
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, token.ID, got.ID)
}

// ---------------------------------------------------------------------------
// RefreshTokenRepo.GetByHash
// ---------------------------------------------------------------------------

func TestRefreshTokenRepo_GetByHash_Found(t *testing.T) {
	ctx := context.Background()
	tx, repo := newTestRefreshTokenRepo(t, ctx)

	userID := createTestUser(t, ctx, tx)
	wsID := createTestWorkspace(t, ctx, tx)
	createTestWorkspaceMember(t, ctx, tx, userID, wsID, "owner")

	token := createTestRefreshToken(t, ctx, tx, userID, wsID)

	got, err := repo.GetByHash(ctx, token.TokenHash, false)
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, token.ID, got.ID)
	require.Equal(t, userID, got.UserID)
	require.Equal(t, wsID, got.WorkspaceID)
	require.Equal(t, token.TokenHash, got.TokenHash)
	require.Nil(t, got.RotatedAt)
	require.Nil(t, got.RevokedAt)
	require.Nil(t, got.ReplacedByID)
}

func TestRefreshTokenRepo_GetByHash_NotFound(t *testing.T) {
	ctx := context.Background()
	_, repo := newTestRefreshTokenRepo(t, ctx)

	got, err := repo.GetByHash(ctx, "nonexistent-hash", false)
	require.NoError(t, err)
	require.Nil(t, got)
}

func TestRefreshTokenRepo_GetByHash_ForUpdate(t *testing.T) {
	ctx := context.Background()
	tx, repo := newTestRefreshTokenRepo(t, ctx)

	userID := createTestUser(t, ctx, tx)
	wsID := createTestWorkspace(t, ctx, tx)
	createTestWorkspaceMember(t, ctx, tx, userID, wsID, "owner")

	token := createTestRefreshToken(t, ctx, tx, userID, wsID)

	got, err := repo.GetByHash(ctx, token.TokenHash, true)
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, token.ID, got.ID)
}

// ---------------------------------------------------------------------------
// RefreshTokenRepo.Rotate
// ---------------------------------------------------------------------------

func TestRefreshTokenRepo_Rotate(t *testing.T) {
	ctx := context.Background()
	tx, repo := newTestRefreshTokenRepo(t, ctx)

	userID := createTestUser(t, ctx, tx)
	wsID := createTestWorkspace(t, ctx, tx)
	createTestWorkspaceMember(t, ctx, tx, userID, wsID, "owner")

	token := createTestRefreshToken(t, ctx, tx, userID, wsID)
	replacement := createTestRefreshToken(t, ctx, tx, userID, wsID)
	rotatedAt := time.Now()

	err := repo.Rotate(ctx, token.ID, replacement.ID, rotatedAt)
	require.NoError(t, err)

	got, err := repo.GetByHash(ctx, token.TokenHash, false)
	require.NoError(t, err)
	require.NotNil(t, got)
	require.NotNil(t, got.RotatedAt)
	require.NotNil(t, got.ReplacedByID)
	require.Equal(t, replacement.ID, *got.ReplacedByID)
}

func TestRefreshTokenRepo_Rotate_NotFound(t *testing.T) {
	ctx := context.Background()
	_, repo := newTestRefreshTokenRepo(t, ctx)

	err := repo.Rotate(ctx, uuid.New(), uuid.New(), time.Now())
	require.Error(t, err)
	require.ErrorIs(t, err, sql.ErrNoRows)
}
