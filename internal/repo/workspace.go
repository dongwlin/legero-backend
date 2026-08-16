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

type Workspace struct {
	db bun.IDB
}

func NewWorkspace(db bun.IDB) *Workspace {
	return &Workspace{db: db}
}

func (r *Workspace) GetPrimaryAccess(ctx context.Context, userID uuid.UUID) (*domain.Access, error) {
	s := new(schema.Access)
	err := r.db.NewSelect().
		TableExpr("workspace_members AS wm").
		ColumnExpr("wm.user_id AS user_id").
		ColumnExpr("wm.workspace_id AS workspace_id").
		ColumnExpr("wm.role AS role").
		ColumnExpr("wm.created_at AS created_at").
		ColumnExpr("w.name AS workspace_name").
		Join("JOIN workspaces AS w ON w.id = wm.workspace_id").
		Where("wm.user_id = ?", userID).
		OrderExpr("wm.created_at ASC").
		Limit(1).
		Scan(ctx, s)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("select primary workspace access: %w", err)
	}
	return toDomainAccess(s), nil
}

func (r *Workspace) GetAccess(ctx context.Context, userID, workspaceID uuid.UUID) (*domain.Access, error) {
	s := new(schema.Access)
	err := r.db.NewSelect().
		TableExpr("workspace_members AS wm").
		ColumnExpr("wm.user_id AS user_id").
		ColumnExpr("wm.workspace_id AS workspace_id").
		ColumnExpr("wm.role AS role").
		ColumnExpr("wm.created_at AS created_at").
		ColumnExpr("w.name AS workspace_name").
		Join("JOIN workspaces AS w ON w.id = wm.workspace_id").
		Where("wm.user_id = ?", userID).
		Where("wm.workspace_id = ?", workspaceID).
		Limit(1).
		Scan(ctx, s)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("select workspace access: %w", err)
	}
	return toDomainAccess(s), nil
}

// Insert creates a new workspace.
func (r *Workspace) Insert(ctx context.Context, workspace *domain.Workspace) error {
	s := &schema.Workspace{
		ID:        workspace.ID,
		Name:      workspace.Name,
		Version:   workspace.Version,
		CreatedAt: workspace.CreatedAt,
		UpdatedAt: workspace.UpdatedAt,
	}
	if _, err := r.db.NewInsert().Model(s).Exec(ctx); err != nil {
		return fmt.Errorf("insert workspace: %w", err)
	}
	return nil
}

// GetByID returns a workspace by ID.
func (r *Workspace) GetByID(ctx context.Context, workspaceID uuid.UUID) (*domain.Workspace, error) {
	s := new(schema.Workspace)
	err := r.db.NewSelect().
		Model(s).
		Where("id = ?", workspaceID).
		Limit(1).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("select workspace by id: %w", err)
	}
	return &domain.Workspace{
		ID:        s.ID,
		Name:      s.Name,
		Version:   s.Version,
		CreatedAt: s.CreatedAt,
		UpdatedAt: s.UpdatedAt,
	}, nil
}

func toDomainAccess(s *schema.Access) *domain.Access {
	return &domain.Access{
		UserID:        s.UserID,
		WorkspaceID:   s.WorkspaceID,
		WorkspaceName: s.WorkspaceName,
		Role:          domain.Role(s.Role),
		CreatedAt:     s.CreatedAt,
	}
}
