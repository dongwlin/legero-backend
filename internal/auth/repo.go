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

type UserRepo struct {
	db bun.IDB
}

func NewUserRepo(db bun.IDB) *UserRepo {
	return &UserRepo{db: db}
}

type RefreshTokenRepo struct {
	db bun.IDB
}

func NewRefreshTokenRepo(db bun.IDB) *RefreshTokenRepo {
	return &RefreshTokenRepo{db: db}
}

func (r *UserRepo) GetByPhone(ctx context.Context, phone string) (*User, error) {
	model := new(UserModel)
	err := r.db.NewSelect().
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

func (r *UserRepo) GetByID(ctx context.Context, userID uuid.UUID) (*User, error) {
	model := new(UserModel)
	err := r.db.NewSelect().
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

func (r *RefreshTokenRepo) Insert(ctx context.Context, token RefreshToken) error {
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
	if _, err := r.db.NewInsert().Model(&model).Exec(ctx); err != nil {
		return fmt.Errorf("insert refresh token: %w", err)
	}
	return nil
}

func (r *RefreshTokenRepo) GetByHash(ctx context.Context, tokenHash string, forUpdate bool) (*RefreshToken, error) {
	model := new(RefreshTokenModel)
	query := r.db.NewSelect().
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

func (r *RefreshTokenRepo) Rotate(ctx context.Context, tokenID, replacementID uuid.UUID, rotatedAt time.Time) error {
	result, err := r.db.NewUpdate().
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
