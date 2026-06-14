package repo

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"

	"github.com/dongwlin/legero-backend/internal/model"
)

// WorkspaceMember handles workspace member database operations.
type WorkspaceMember struct {
	db bun.IDB
}

// NewWorkspaceMember creates a new WorkspaceMemberRepo.
func NewWorkspaceMember(db bun.IDB) *WorkspaceMember {
	return &WorkspaceMember{db: db}
}

// Insert creates a new workspace member.
func (r *WorkspaceMember) Insert(ctx context.Context, member *model.WorkspaceMember) error {
	if _, err := r.db.NewInsert().Model(member).Exec(ctx); err != nil {
		return fmt.Errorf("insert workspace member: %w", err)
	}
	return nil
}
