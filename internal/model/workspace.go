package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// Role represents a user's role within a workspace.
type Role string

const (
	RoleOwner Role = "owner"
	RoleStaff Role = "staff"
)

// Workspace is the unified domain + ORM model for the workspaces table.
type Workspace struct {
	bun.BaseModel `bun:"table:workspaces,alias:w"`

	ID        uuid.UUID `bun:",pk,type:uuid"`
	Name      string    `bun:"name,notnull"`
	CreatedAt time.Time `bun:"created_at,notnull"`
	UpdatedAt time.Time `bun:"updated_at,notnull"`
}

// WorkspaceMember is the unified domain + ORM model for the workspace_members table.
type WorkspaceMember struct {
	bun.BaseModel `bun:"table:workspace_members,alias:wm"`

	WorkspaceID uuid.UUID `bun:"workspace_id,pk,type:uuid"`
	UserID      uuid.UUID `bun:"user_id,pk,type:uuid"`
	Role        string    `bun:"role,notnull"`
	CreatedAt   time.Time `bun:"created_at,notnull"`
}

// Access is a value object representing a user's access to a workspace.
// It is not a table model; column tags are used for scanning raw SQL join results.
type Access struct {
	UserID        uuid.UUID `bun:"user_id"`
	WorkspaceID   uuid.UUID `bun:"workspace_id"`
	WorkspaceName string    `bun:"workspace_name"`
	Role          Role      `bun:"role"`
	CreatedAt     time.Time `bun:"created_at"`
}

// Permissions returns the list of permission strings for the given role.
func (r Role) Permissions() []string {
	permissions := []string{"orders:read", "orders:write"}
	if r == RoleOwner {
		permissions = append(permissions, "orders:clear")
	}
	return permissions
}

// CanClear reports whether the given role has permission to clear orders.
func (r Role) CanClear() bool {
	return r == RoleOwner
}
