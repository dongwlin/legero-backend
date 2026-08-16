package service

import (
	"context"

	"github.com/dongwlin/legero-backend/internal/domain"
	"github.com/google/uuid"
)

// Order handles order CRUD, step toggling, and workspace clearing.
type Order interface {
	ListActive(ctx context.Context, workspaceID uuid.UUID) ([]domain.Order, error)
	List(ctx context.Context, actor domain.Actor, query domain.ListOrdersQuery) (*domain.ListOrdersResult, error)
	CreateBatch(ctx context.Context, actor domain.Actor, input domain.CreateOrdersInput) ([]domain.Order, error)
	UpdateForm(ctx context.Context, actor domain.Actor, orderID uuid.UUID, input domain.UpdateOrderInput) (*domain.Order, error)
	ToggleStep(ctx context.Context, actor domain.Actor, orderID uuid.UUID, input domain.ToggleStepInput) (*domain.Order, error)
	ToggleServed(ctx context.Context, actor domain.Actor, orderID uuid.UUID, input domain.ToggleServedInput) (*domain.Order, error)
	Remove(ctx context.Context, actor domain.Actor, orderID uuid.UUID) error
	ClearWorkspace(ctx context.Context, actor domain.Actor, confirm bool, mode domain.ClearWorkspaceMode) (int, error)
}
