package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type UserRepository interface {
	GetByPhone(ctx context.Context, db bun.IDB, phone string) (*User, error)
	GetByID(ctx context.Context, db bun.IDB, userID uuid.UUID) (*User, error)
}

type RefreshTokenRepository interface {
	Insert(ctx context.Context, db bun.IDB, token RefreshToken) error
	GetByHash(ctx context.Context, db bun.IDB, tokenHash string, forUpdate bool) (*RefreshToken, error)
	Rotate(ctx context.Context, db bun.IDB, tokenID, replacementID uuid.UUID, rotatedAt time.Time) error
}

type BunUserRepository struct{}

type BunRefreshTokenRepository struct{}

func (r *BunUserRepository) GetByPhone(ctx context.Context, db bun.IDB, phone string) (*User, error) {
	model := new(UserModel)
	err := db.NewSelect().
		Model(model).
		Where("phone = ?", phone).
		Limit(1).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("select user by phone: %w", err)
	}
	return &User{
		ID:           model.ID,
		Phone:        model.Phone,
		PasswordHash: model.PasswordHash,
		IsActive:     model.IsActive,
		CreatedAt:    model.CreatedAt,
		UpdatedAt:    model.UpdatedAt,
	}, nil
}

func (r *BunUserRepository) GetByID(ctx context.Context, db bun.IDB, userID uuid.UUID) (*User, error) {
	model := new(UserModel)
	err := db.NewSelect().
		Model(model).
		Where("id = ?", userID).
		Limit(1).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("select user by id: %w", err)
	}
	return &User{
		ID:           model.ID,
		Phone:        model.Phone,
		PasswordHash: model.PasswordHash,
		IsActive:     model.IsActive,
		CreatedAt:    model.CreatedAt,
		UpdatedAt:    model.UpdatedAt,
	}, nil
}

func (r *BunRefreshTokenRepository) Insert(ctx context.Context, db bun.IDB, token RefreshToken) error {
	model := RefreshTokenModel{
		ID:           token.ID,
		UserID:       token.UserID,
		WorkspaceID:  token.WorkspaceID,
		TokenHash:    token.TokenHash,
		ExpiresAt:    token.ExpiresAt,
		CreatedAt:    token.CreatedAt,
		RotatedAt:    token.RotatedAt,
		RevokedAt:    token.RevokedAt,
		ReplacedByID: token.ReplacedByID,
	}
	if _, err := db.NewInsert().Model(&model).Exec(ctx); err != nil {
		return fmt.Errorf("insert refresh token: %w", err)
	}
	return nil
}

func (r *BunRefreshTokenRepository) GetByHash(ctx context.Context, db bun.IDB, tokenHash string, forUpdate bool) (*RefreshToken, error) {
	model := new(RefreshTokenModel)
	query := db.NewSelect().
		Model(model).
		Where("token_hash = ?", tokenHash).
		Limit(1)
	if forUpdate {
		query = query.For("UPDATE")
	}

	if err := query.Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("select refresh token by hash: %w", err)
	}

	return &RefreshToken{
		ID:           model.ID,
		UserID:       model.UserID,
		WorkspaceID:  model.WorkspaceID,
		TokenHash:    model.TokenHash,
		ExpiresAt:    model.ExpiresAt,
		CreatedAt:    model.CreatedAt,
		RotatedAt:    model.RotatedAt,
		RevokedAt:    model.RevokedAt,
		ReplacedByID: model.ReplacedByID,
	}, nil
}

func (r *BunRefreshTokenRepository) Rotate(ctx context.Context, db bun.IDB, tokenID, replacementID uuid.UUID, rotatedAt time.Time) error {
	result, err := db.NewUpdate().
		Model((*RefreshTokenModel)(nil)).
		Set("rotated_at = ?", rotatedAt).
		Set("replaced_by_id = ?", replacementID).
		Where("id = ?", tokenID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("rotate refresh token: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}
