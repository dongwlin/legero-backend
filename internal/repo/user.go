package repo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/uptrace/bun"

	"github.com/dongwlin/legero-backend/internal/domain"
	"github.com/dongwlin/legero-backend/internal/repo/schema"
)

type User struct {
	db bun.IDB
}

func NewUser(db bun.IDB) *User {
	return &User{db: db}
}

func (r *User) GetByPhone(ctx context.Context, phone string) (*domain.User, error) {
	s := new(schema.User)
	err := r.db.NewSelect().
		Model(s).
		Where("phone = ?", phone).
		Limit(1).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("select user by phone: %w", err)
	}
	return toDomainUser(s), nil
}

func (r *User) GetByID(ctx context.Context, userID uuid.UUID) (*domain.User, error) {
	s := new(schema.User)
	err := r.db.NewSelect().
		Model(s).
		Where("id = ?", userID).
		Limit(1).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("select user by id: %w", err)
	}
	return toDomainUser(s), nil
}

// Insert creates a new user.
func (r *User) Insert(ctx context.Context, user *domain.User) error {
	s := toSchemaUser(user)
	if _, err := r.db.NewInsert().Model(s).Exec(ctx); err != nil {
		return fmt.Errorf("insert user: %w", err)
	}
	return nil
}

func toSchemaUser(u *domain.User) *schema.User {
	return &schema.User{
		ID:           u.ID,
		Phone:        u.Phone,
		PasswordHash: u.PasswordHash,
		IsActive:     u.IsActive,
		Version:      u.Version,
		CreatedAt:    u.CreatedAt,
		UpdatedAt:    u.UpdatedAt,
	}
}

func toDomainUser(s *schema.User) *domain.User {
	return &domain.User{
		ID:           s.ID,
		Phone:        s.Phone,
		PasswordHash: s.PasswordHash,
		IsActive:     s.IsActive,
		Version:      s.Version,
		CreatedAt:    s.CreatedAt,
		UpdatedAt:    s.UpdatedAt,
	}
}
