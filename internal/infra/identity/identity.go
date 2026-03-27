package identity

import "github.com/google/uuid"

type Context struct {
	UserID      uuid.UUID
	WorkspaceID uuid.UUID
	Role        string
	Phone       string
}

const GinContextKey = "authContext"
