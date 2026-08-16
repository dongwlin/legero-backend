package schema

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// Workspace is the bun ORM mapping of the workspaces table.
type Workspace struct {
	bun.BaseModel `bun:"table:workspaces,alias:w"`

	ID        uuid.UUID `bun:",pk,type:uuid"`
	Name      string    `bun:"name,notnull"`
	Version   int64     `bun:"version,notnull,default:1"`
	CreatedAt time.Time `bun:"created_at,notnull"`
	UpdatedAt time.Time `bun:"updated_at,notnull"`
}

// WorkspaceMember is the bun ORM mapping of the workspace_members table.
type WorkspaceMember struct {
	bun.BaseModel `bun:"table:workspace_members,alias:wm"`

	WorkspaceID uuid.UUID `bun:"workspace_id,pk,type:uuid"`
	UserID      uuid.UUID `bun:"user_id,pk,type:uuid"`
	Role        string    `bun:"role,notnull"`
	CreatedAt   time.Time `bun:"created_at,notnull"`
}

// Access is the scan mapping for the workspace access join query result.
type Access struct {
	UserID        uuid.UUID `bun:"user_id"`
	WorkspaceID   uuid.UUID `bun:"workspace_id"`
	WorkspaceName string    `bun:"workspace_name"`
	Role          string    `bun:"role"`
	CreatedAt     time.Time `bun:"created_at"`
}
