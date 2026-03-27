package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/uptrace/bun"

	clockpkg "github.com/dongwlin/legero-backend/internal/infra/clock"
	idspkg "github.com/dongwlin/legero-backend/internal/infra/ids"
	"github.com/dongwlin/legero-backend/internal/workspace"
)

type CreateUserInput struct {
	Phone       string
	Password    string
	WorkspaceID *uuid.UUID
	Workspace   string
	Role        workspace.Role
}

type CreateUserResult struct {
	UserID           uuid.UUID
	Phone            string
	WorkspaceID      uuid.UUID
	WorkspaceName    string
	Role             workspace.Role
	CreatedWorkspace bool
}

type BootstrapUserService struct {
	db     *bun.DB
	hasher *PasswordHasher
	clock  clockpkg.Clock
	ids    idspkg.Generator
}

func NewBootstrapUserService(database *bun.DB, hasher *PasswordHasher, clock clockpkg.Clock, ids idspkg.Generator) *BootstrapUserService {
	return &BootstrapUserService{
		db:     database,
		hasher: hasher,
		clock:  clock,
		ids:    ids,
	}
}

func (s *BootstrapUserService) CreateUser(ctx context.Context, input CreateUserInput) (*CreateUserResult, error) {
	normalizedPhone := NormalizePhone(input.Phone)
	if normalizedPhone == "" {
		return nil, fmt.Errorf("phone is required")
	}
	if strings.TrimSpace(input.Password) == "" {
		return nil, fmt.Errorf("password is required")
	}
	if input.Role == "" {
		input.Role = workspace.RoleOwner
	}
	if input.Role != workspace.RoleOwner && input.Role != workspace.RoleStaff {
		return nil, fmt.Errorf("role must be owner or staff")
	}
	if input.WorkspaceID == nil {
		input.Workspace = strings.TrimSpace(input.Workspace)
		if input.Workspace == "" {
			input.Workspace = "Legero"
		}
	}

	existingUser := new(UserModel)
	err := s.db.NewSelect().
		Model(existingUser).
		Where("phone = ?", normalizedPhone).
		Limit(1).
		Scan(ctx)
	if err == nil {
		return nil, fmt.Errorf("user with phone %s already exists", normalizedPhone)
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("check existing user: %w", err)
	}

	passwordHash, err := s.hasher.Hash(input.Password)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	now := s.clock.Now()
	result := &CreateUserResult{
		UserID: s.ids.New(),
		Phone:  normalizedPhone,
		Role:   input.Role,
	}

	if err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if input.WorkspaceID == nil {
			result.WorkspaceID = s.ids.New()
			result.WorkspaceName = input.Workspace
			result.CreatedWorkspace = true

			workspaceModel := workspace.WorkspaceModel{
				ID:        result.WorkspaceID,
				Name:      result.WorkspaceName,
				CreatedAt: now,
				UpdatedAt: now,
			}
			if _, err := tx.NewInsert().Model(&workspaceModel).Exec(ctx); err != nil {
				return fmt.Errorf("insert workspace: %w", err)
			}
		} else {
			workspaceModel := new(workspace.WorkspaceModel)
			if err := tx.NewSelect().
				Model(workspaceModel).
				Where("id = ?", *input.WorkspaceID).
				Limit(1).
				Scan(ctx); err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return fmt.Errorf("workspace %s not found", input.WorkspaceID.String())
				}
				return fmt.Errorf("load workspace: %w", err)
			}

			result.WorkspaceID = workspaceModel.ID
			result.WorkspaceName = workspaceModel.Name
		}

		userModel := UserModel{
			ID:           result.UserID,
			Phone:        result.Phone,
			PasswordHash: passwordHash,
			IsActive:     true,
			CreatedAt:    now,
			UpdatedAt:    now,
		}
		if _, err := tx.NewInsert().Model(&userModel).Exec(ctx); err != nil {
			return fmt.Errorf("insert user: %w", err)
		}

		memberModel := workspace.WorkspaceMemberModel{
			WorkspaceID: result.WorkspaceID,
			UserID:      result.UserID,
			Role:        string(result.Role),
			CreatedAt:   now,
		}
		if _, err := tx.NewInsert().Model(&memberModel).Exec(ctx); err != nil {
			return fmt.Errorf("insert workspace member: %w", err)
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return result, nil
}
