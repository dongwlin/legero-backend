package service

import (
	"context"

	"github.com/dongwlin/legero-backend/internal/model"
	"github.com/google/uuid"
)

// CreateUserInput carries the parameters for creating a new user during bootstrap.
type CreateUserInput struct {
	Phone       string
	Password    string
	WorkspaceID *uuid.UUID
	Workspace   string
	Role        model.Role
}

// CreateUserResult is the result of a successful user creation.
type CreateUserResult struct {
	UserID           uuid.UUID
	Phone            string
	WorkspaceID      uuid.UUID
	WorkspaceName    string
	Role             model.Role
	CreatedWorkspace bool
}

// User handles one-time user creation (seeding) for the system.
type User interface {
	CreateUser(ctx context.Context, input CreateUserInput) (*CreateUserResult, error)
}
