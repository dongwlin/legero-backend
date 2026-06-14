package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"

	"github.com/dongwlin/legero-backend/internal/infra/crypto"
	"github.com/dongwlin/legero-backend/internal/model"
	"github.com/dongwlin/legero-backend/internal/repo"
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
type User struct {
	db     *bun.DB
	hasher *crypto.PasswordHasher
}

// NewUser creates a new BootstrapUserService.
func NewUser(database *bun.DB, hasher *crypto.PasswordHasher) *User {
	return &User{
		db:     database,
		hasher: hasher,
	}
}

// CreateUser creates a new user and optionally a new workspace, all within a single transaction.
func (s *User) CreateUser(ctx context.Context, input CreateUserInput) (*CreateUserResult, error) {
	normalizedPhone := model.NormalizePhone(input.Phone)
	if normalizedPhone == "" {
		return nil, fmt.Errorf("phone is required")
	}
	if strings.TrimSpace(input.Password) == "" {
		return nil, fmt.Errorf("password is required")
	}
	if input.Role == "" {
		input.Role = model.RoleOwner
	}
	if input.Role != model.RoleOwner && input.Role != model.RoleStaff {
		return nil, fmt.Errorf("role must be owner or staff")
	}
	if input.WorkspaceID == nil {
		input.Workspace = strings.TrimSpace(input.Workspace)
		if input.Workspace == "" {
			input.Workspace = "Legero"
		}
	}

	// Check if user already exists using repo
	userRepo := repo.NewUser(s.db)
	existingUser, err := userRepo.GetByPhone(ctx, normalizedPhone)
	if err != nil {
		return nil, fmt.Errorf("check existing user: %w", err)
	}
	if existingUser != nil {
		return nil, fmt.Errorf("user with phone %s already exists", normalizedPhone)
	}

	passwordHash, err := s.hasher.Hash(input.Password)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	now := time.Now()
	result := &CreateUserResult{
		UserID: uuid.New(),
		Phone:  normalizedPhone,
		Role:   input.Role,
	}

	if err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		wsRepo := repo.NewWorkspace(tx)
		txUserRepo := repo.NewUser(tx)
		memberRepo := repo.NewWorkspaceMember(tx)

		if input.WorkspaceID == nil {
			result.WorkspaceID = uuid.New()
			result.WorkspaceName = input.Workspace
			result.CreatedWorkspace = true

			workspaceModel := &model.Workspace{
				ID:        result.WorkspaceID,
				Name:      result.WorkspaceName,
				CreatedAt: now,
				UpdatedAt: now,
			}
			if err := wsRepo.Insert(ctx, workspaceModel); err != nil {
				return fmt.Errorf("insert workspace: %w", err)
			}
		} else {
			// Load existing workspace
			workspaceModel, err := wsRepo.GetByID(ctx, *input.WorkspaceID)
			if err != nil {
				return fmt.Errorf("load workspace: %w", err)
			}
			if workspaceModel == nil {
				return fmt.Errorf("workspace %s not found", input.WorkspaceID.String())
			}

			result.WorkspaceID = workspaceModel.ID
			result.WorkspaceName = workspaceModel.Name
		}

		userModel := &model.User{
			ID:           result.UserID,
			Phone:        result.Phone,
			PasswordHash: passwordHash,
			IsActive:     true,
			CreatedAt:    now,
			UpdatedAt:    now,
		}
		if err := txUserRepo.Insert(ctx, userModel); err != nil {
			return fmt.Errorf("insert user: %w", err)
		}

		memberModel := &model.WorkspaceMember{
			WorkspaceID: result.WorkspaceID,
			UserID:      result.UserID,
			Role:        string(result.Role),
			CreatedAt:   now,
		}
		if err := memberRepo.Insert(ctx, memberModel); err != nil {
			return fmt.Errorf("insert workspace member: %w", err)
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return result, nil
}
