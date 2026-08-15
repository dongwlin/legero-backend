package app

import (
	"context"

	"github.com/dongwlin/legero-backend/internal/service"
)

// UserCreator is the wired bootstrap for the create-user CLI command.
type UserCreator struct {
	svc service.User
}

// NewUserCreator creates a UserCreator.
func NewUserCreator(svc service.User) *UserCreator {
	return &UserCreator{svc: svc}
}

// Create creates a user and returns the result.
func (c *UserCreator) Create(ctx context.Context, input service.CreateUserInput) (*service.CreateUserResult, error) {
	return c.svc.CreateUser(ctx, input)
}
