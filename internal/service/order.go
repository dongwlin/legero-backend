package service

import (
	"context"

	"github.com/dongwlin/legero-backend/internal/model"
	"github.com/google/uuid"
)

// Order handles order CRUD, step toggling, and workspace clearing.
type Order interface {
	ListActive(ctx context.Context, workspaceID uuid.UUID) ([]model.Order, error)
	List(ctx context.Context, actor model.Actor, query model.ListOrdersQuery) (*model.ListOrdersResult, error)
	CreateBatch(ctx context.Context, actor model.Actor, input model.CreateOrdersInput) ([]model.Order, error)
	UpdateForm(ctx context.Context, actor model.Actor, orderID uuid.UUID, input model.UpdateOrderInput) (*model.Order, error)
	ToggleStep(ctx context.Context, actor model.Actor, orderID uuid.UUID, input model.ToggleStepInput) (*model.Order, error)
	ToggleServed(ctx context.Context, actor model.Actor, orderID uuid.UUID, input model.ToggleServedInput) (*model.Order, error)
	Remove(ctx context.Context, actor model.Actor, orderID uuid.UUID) error
	ClearWorkspace(ctx context.Context, actor model.Actor, confirm bool, mode model.ClearWorkspaceMode) (int, error)
}
