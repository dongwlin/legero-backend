package domain

import (
	"time"

	"github.com/google/uuid"
)

// Role represents a user's role within a workspace.
type Role string

const (
	RoleOwner Role = "owner"
	RoleStaff Role = "staff"
)

// Valid reports whether r is one of the supported roles.
func (r Role) Valid() bool {
	return r == RoleOwner || r == RoleStaff
}

// Workspace is the domain model for a restaurant workspace.
type Workspace struct {
	ID        uuid.UUID
	Name      string
	Version   int64
	CreatedAt time.Time
	UpdatedAt time.Time
}

// WorkspaceMember is the domain model for a user's membership in a workspace.
type WorkspaceMember struct {
	WorkspaceID uuid.UUID
	UserID      uuid.UUID
	Role        string
	CreatedAt   time.Time
}

// Access is a value object representing a user's access to a workspace.
type Access struct {
	UserID        uuid.UUID
	WorkspaceID   uuid.UUID
	WorkspaceName string
	Role          Role
	CreatedAt     time.Time
}

// Permissions returns the list of permission strings for the given role.
// Unknown or invalid roles fail closed and receive no permissions.
func (r Role) Permissions() []string {
	switch r {
	case RoleOwner:
		return []string{"orders:read", "orders:write", "orders:clear"}
	case RoleStaff:
		return []string{"orders:read", "orders:write"}
	default:
		return nil
	}
}

// CanClear reports whether the given role has permission to clear orders.
func (r Role) CanClear() bool {
	return r == RoleOwner
}
