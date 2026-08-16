package repo

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"

	"github.com/dongwlin/legero-backend/internal/domain"
	"github.com/dongwlin/legero-backend/internal/repo/schema"
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
func (r *WorkspaceMember) Insert(ctx context.Context, member *domain.WorkspaceMember) error {
	s := &schema.WorkspaceMember{
		WorkspaceID: member.WorkspaceID,
		UserID:      member.UserID,
		Role:        member.Role,
		CreatedAt:   member.CreatedAt,
	}
	if _, err := r.db.NewInsert().Model(s).Exec(ctx); err != nil {
		return fmt.Errorf("insert workspace member: %w", err)
	}
	return nil
}
