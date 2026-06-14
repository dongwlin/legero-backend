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

type Workspace struct {
	db bun.IDB
}

func NewWorkspace(db bun.IDB) *Workspace {
	return &Workspace{db: db}
}

func (r *Workspace) GetPrimaryAccess(ctx context.Context, userID uuid.UUID) (*model.Access, error) {
	access := new(model.Access)
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
		Scan(ctx, access)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("select primary workspace access: %w", err)
	}
	return access, nil
}

func (r *Workspace) GetAccess(ctx context.Context, userID, workspaceID uuid.UUID) (*model.Access, error) {
	access := new(model.Access)
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
		Scan(ctx, access)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("select workspace access: %w", err)
	}
	return access, nil
}

// Insert creates a new workspace.
func (r *Workspace) Insert(ctx context.Context, workspace *model.Workspace) error {
	if _, err := r.db.NewInsert().Model(workspace).Exec(ctx); err != nil {
		return fmt.Errorf("insert workspace: %w", err)
	}
	return nil
}

// GetByID returns a workspace by ID.
func (r *Workspace) GetByID(ctx context.Context, workspaceID uuid.UUID) (*model.Workspace, error) {
	workspace := new(model.Workspace)
	err := r.db.NewSelect().
		Model(workspace).
		Where("id = ?", workspaceID).
		Limit(1).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("select workspace by id: %w", err)
	}
	return workspace, nil
}
