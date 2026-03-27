package workspace

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type Role string

const (
	RoleOwner Role = "owner"
	RoleStaff Role = "staff"
)

type Workspace struct {
	ID        uuid.UUID
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Access struct {
	UserID        uuid.UUID
	WorkspaceID   uuid.UUID
	WorkspaceName string
	Role          Role
	CreatedAt     time.Time
}

type WorkspaceModel struct {
	bun.BaseModel `bun:"table:workspaces,alias:w"`

	ID        uuid.UUID `bun:",pk,type:uuid"`
	Name      string    `bun:"name,notnull"`
	CreatedAt time.Time `bun:"created_at,notnull"`
	UpdatedAt time.Time `bun:"updated_at,notnull"`
}

type WorkspaceMemberModel struct {
	bun.BaseModel `bun:"table:workspace_members,alias:wm"`

	WorkspaceID uuid.UUID `bun:"workspace_id,pk,type:uuid"`
	UserID      uuid.UUID `bun:"user_id,pk,type:uuid"`
	Role        string    `bun:"role,notnull"`
	CreatedAt   time.Time `bun:"created_at,notnull"`
}
