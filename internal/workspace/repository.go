package workspace

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type Repository interface {
	GetPrimaryAccess(ctx context.Context, db bun.IDB, userID uuid.UUID) (*Access, error)
	GetAccess(ctx context.Context, db bun.IDB, userID, workspaceID uuid.UUID) (*Access, error)
}

type BunRepository struct{}

type accessRow struct {
	UserID        uuid.UUID `bun:"user_id"`
	WorkspaceID   uuid.UUID `bun:"workspace_id"`
	WorkspaceName string    `bun:"workspace_name"`
	Role          string    `bun:"role"`
	CreatedAt     time.Time `bun:"created_at"`
}

func (r *BunRepository) GetPrimaryAccess(ctx context.Context, db bun.IDB, userID uuid.UUID) (*Access, error) {
	row := new(accessRow)
	err := db.NewSelect().
		TableExpr("workspace_members AS wm").
		ColumnExpr("wm.user_id").
		ColumnExpr("wm.workspace_id").
		ColumnExpr("wm.role").
		ColumnExpr("wm.created_at").
		ColumnExpr("w.name AS workspace_name").
		Join("JOIN workspaces AS w ON w.id = wm.workspace_id").
		Where("wm.user_id = ?", userID).
		OrderExpr("wm.created_at ASC").
		Limit(1).
		Scan(ctx, row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("select primary workspace access: %w", err)
	}
	return mapAccess(row), nil
}

func (r *BunRepository) GetAccess(ctx context.Context, db bun.IDB, userID, workspaceID uuid.UUID) (*Access, error) {
	row := new(accessRow)
	err := db.NewSelect().
		TableExpr("workspace_members AS wm").
		ColumnExpr("wm.user_id").
		ColumnExpr("wm.workspace_id").
		ColumnExpr("wm.role").
		ColumnExpr("wm.created_at").
		ColumnExpr("w.name AS workspace_name").
		Join("JOIN workspaces AS w ON w.id = wm.workspace_id").
		Where("wm.user_id = ?", userID).
		Where("wm.workspace_id = ?", workspaceID).
		Limit(1).
		Scan(ctx, row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("select workspace access: %w", err)
	}
	return mapAccess(row), nil
}

func mapAccess(row *accessRow) *Access {
	return &Access{
		UserID:        row.UserID,
		WorkspaceID:   row.WorkspaceID,
		WorkspaceName: row.WorkspaceName,
		Role:          Role(row.Role),
		CreatedAt:     row.CreatedAt,
	}
}
