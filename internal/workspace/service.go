package workspace

import (
	"context"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type Service struct {
	db   *bun.DB
	repo Repository
}

func NewService(database *bun.DB, repo Repository) *Service {
	return &Service{
		db:   database,
		repo: repo,
	}
}

func (s *Service) ResolvePrimary(ctx context.Context, userID uuid.UUID) (*Access, error) {
	return s.repo.GetPrimaryAccess(ctx, s.db, userID)
}

func (s *Service) Resolve(ctx context.Context, userID, workspaceID uuid.UUID) (*Access, error) {
	return s.repo.GetAccess(ctx, s.db, userID, workspaceID)
}

func Permissions(role Role) []string {
	permissions := []string{"orders:read", "orders:write"}
	if role == RoleOwner {
		permissions = append(permissions, "orders:clear")
	}
	return permissions
}

func CanClear(role Role) bool {
	return role == RoleOwner
}
