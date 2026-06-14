package repo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/uptrace/bun"

	"github.com/dongwlin/legero-backend/internal/model"
)

type User struct {
	db bun.IDB
}

func NewUser(db bun.IDB) *User {
	return &User{db: db}
}

func (r *User) GetByPhone(ctx context.Context, phone string) (*model.User, error) {
	user := new(model.User)
	err := r.db.NewSelect().
		Model(user).
		Where("phone = ?", phone).
		Limit(1).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("select user by phone: %w", err)
	}
	return user, nil
}

func (r *User) GetByID(ctx context.Context, userID uuid.UUID) (*model.User, error) {
	user := new(model.User)
	err := r.db.NewSelect().
		Model(user).
		Where("id = ?", userID).
		Limit(1).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("select user by id: %w", err)
	}
	return user, nil
}

// Insert creates a new user.
func (r *User) Insert(ctx context.Context, user *model.User) error {
	if _, err := r.db.NewInsert().Model(user).Exec(ctx); err != nil {
		return fmt.Errorf("insert user: %w", err)
	}
	return nil
}
