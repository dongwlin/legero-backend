package repo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"

	"github.com/dongwlin/legero-backend/internal/domain"
	"github.com/dongwlin/legero-backend/internal/repo/schema"
)

type RefreshToken struct {
	db bun.IDB
}

func NewRefreshToken(db bun.IDB) *RefreshToken {
	return &RefreshToken{db: db}
}

func (r *RefreshToken) Insert(ctx context.Context, token *domain.RefreshToken) error {
	s := &schema.RefreshToken{
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
	if _, err := r.db.NewInsert().Model(s).Exec(ctx); err != nil {
		return fmt.Errorf("insert refresh token: %w", err)
	}
	return nil
}

func (r *RefreshToken) GetByHash(ctx context.Context, tokenHash string, forUpdate bool) (*domain.RefreshToken, error) {
	s := new(schema.RefreshToken)
	query := r.db.NewSelect().
		Model(s).
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

	return &domain.RefreshToken{
		ID:           s.ID,
		UserID:       s.UserID,
		WorkspaceID:  s.WorkspaceID,
		TokenHash:    s.TokenHash,
		ExpiresAt:    s.ExpiresAt,
		CreatedAt:    s.CreatedAt,
		RotatedAt:    s.RotatedAt,
		RevokedAt:    s.RevokedAt,
		ReplacedByID: s.ReplacedByID,
	}, nil
}

func (r *RefreshToken) Rotate(ctx context.Context, tokenID, replacementID uuid.UUID, rotatedAt time.Time) error {
	result, err := r.db.NewUpdate().
		Model((*schema.RefreshToken)(nil)).
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
