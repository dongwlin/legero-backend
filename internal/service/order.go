package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"

	"github.com/dongwlin/legero-backend/internal/infra/httpx"
	"github.com/dongwlin/legero-backend/internal/model"
	"github.com/dongwlin/legero-backend/internal/repo"
)

// Order handles order CRUD, step toggling, and workspace clearing.
type Order struct {
	db        *bun.DB
	location  *time.Location
	publisher model.Publisher
}

// NewOrder creates a new OrderService.
func NewOrder(
	db *bun.DB,
	location *time.Location,
	publisher model.Publisher,
) *Order {
	return &Order{
		db:        db,
		location:  location,
		publisher: publisher,
	}
}

// ListActive returns all uncompleted orders for a workspace.
func (s *Order) ListActive(ctx context.Context, workspaceID uuid.UUID) ([]model.Order, error) {
	orderRepo := repo.NewOrder(s.db)
	return orderRepo.ListActive(ctx, workspaceID)
}

// List returns a paginated list of orders for a workspace.
func (s *Order) List(ctx context.Context, actor model.Actor, query model.ListOrdersQuery) (*model.ListOrdersResult, error) {
	if !query.Status.Valid() {
		return nil, httpx.ValidationError("status must be one of uncompleted, completed, all")
	}

	orderRepo := repo.NewOrder(s.db)
	items, nextCursor, err := orderRepo.List(ctx, actor.WorkspaceID, query)
	if err != nil {
		return nil, httpx.InternalError("failed to list orders", err)
	}

	return &model.ListOrdersResult{
		Items:      items,
		NextCursor: nextCursor,
	}, nil
}

// CreateBatch creates multiple orders in a single transaction, allocating display numbers atomically.
func (s *Order) CreateBatch(ctx context.Context, actor model.Actor, input model.CreateOrdersInput) ([]model.Order, error) {
	if input.Quantity <= 0 {
		return nil, httpx.ValidationError("quantity must be greater than 0")
	}

	form, err := input.Form.Normalize()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	bizDate := orderBusinessDate(now, s.location)
	items := make([]model.Order, 0, input.Quantity)

	if err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		counterRepo := repo.NewCounter(tx)
		startSeq, err := counterRepo.Allocate(ctx, actor.WorkspaceID, bizDate, input.Quantity, now)
		if err != nil {
			return err
		}

		for idx := 0; idx < input.Quantity; idx++ {
			stapleStatus, meatStatus, completedAt := form.InitialStepStatuses()
			item := model.Order{
				ID:                   uuid.New(),
				WorkspaceID:          actor.WorkspaceID,
				DisplayNo:            buildDisplayNo(bizDate, startSeq+idx),
				StapleTypeCode:       form.StapleTypeCode,
				SizeCode:             form.SizeCode,
				CustomSizePriceCents: form.CustomSizePriceCents,
				StapleAmountCode:     form.StapleAmountCode,
				ExtraStapleUnits:     form.ExtraStapleUnits,
				FriedEggCount:        form.FriedEggCount,
				TofuSkewerCount:      form.TofuSkewerCount,
				SelectedMeatCodes:    model.CloneInt16s(form.SelectedMeatCodes),
				GreensCode:           form.GreensCode,
				ScallionCode:         form.ScallionCode,
				PepperCode:           form.PepperCode,
				DiningMethodCode:     form.DiningMethodCode,
				PackagingCode:        form.PackagingCode,
				PackagingMethodCode:  form.PackagingMethodCode,
				TotalPriceCents:      form.CalculateTotalPriceCents(),
				StapleStepStatusCode: stapleStatus,
				MeatStepStatusCode:   meatStatus,
				Note:                 form.Note,
				CreatedBy:            actor.UserID,
				UpdatedBy:            actor.UserID,
				CreatedAt:            now,
				UpdatedAt:            now,
				CompletedAt:          completedAt,
			}
			items = append(items, item)
		}

		orderRepo := repo.NewOrder(tx)
		return orderRepo.InsertMany(ctx, items)
	}); err != nil {
		return nil, httpx.InternalError("failed to create orders", err)
	}

	s.publishUpserts(items)
	return items, nil
}

// UpdateForm replaces the form data of an existing order.
func (s *Order) UpdateForm(ctx context.Context, actor model.Actor, orderID uuid.UUID, input model.UpdateOrderInput) (*model.Order, error) {
	form, err := input.Form.Normalize()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	var updated model.Order

	if err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		orderRepo := repo.NewOrder(tx)
		current, err := orderRepo.GetByID(ctx, actor.WorkspaceID, orderID)
		if err != nil {
			return err
		}
		if current == nil {
			return httpx.NotFoundError("order_not_found", "order not found")
		}
		if err := checkExpectedUpdatedAt(*current, input.ExpectedUpdatedAt); err != nil {
			return err
		}

		stapleStatus, meatStatus, completedAt := form.InitialStepStatuses()
		updated = model.Order{
			ID:                   current.ID,
			WorkspaceID:          current.WorkspaceID,
			DisplayNo:            current.DisplayNo,
			StapleTypeCode:       form.StapleTypeCode,
			SizeCode:             form.SizeCode,
			CustomSizePriceCents: form.CustomSizePriceCents,
			StapleAmountCode:     form.StapleAmountCode,
			ExtraStapleUnits:     form.ExtraStapleUnits,
			FriedEggCount:        form.FriedEggCount,
			TofuSkewerCount:      form.TofuSkewerCount,
			SelectedMeatCodes:    model.CloneInt16s(form.SelectedMeatCodes),
			GreensCode:           form.GreensCode,
			ScallionCode:         form.ScallionCode,
			PepperCode:           form.PepperCode,
			DiningMethodCode:     form.DiningMethodCode,
			PackagingCode:        form.PackagingCode,
			PackagingMethodCode:  form.PackagingMethodCode,
			TotalPriceCents:      form.CalculateTotalPriceCents(),
			StapleStepStatusCode: stapleStatus,
			MeatStepStatusCode:   meatStatus,
			Note:                 form.Note,
			CreatedBy:            current.CreatedBy,
			UpdatedBy:            actor.UserID,
			CreatedAt:            current.CreatedAt,
			UpdatedAt:            now,
			CompletedAt:          completedAt,
		}

		return orderRepo.Update(ctx, &updated)
	}); err != nil {
		return nil, wrapError("failed to update order", err)
	}

	s.publishUpserts([]model.Order{updated})
	return &updated, nil
}

// ToggleStep toggles the completion state of a cooking step ("staple" or "meat").
func (s *Order) ToggleStep(ctx context.Context, actor model.Actor, orderID uuid.UUID, input model.ToggleStepInput) (*model.Order, error) {
	if input.Step != "staple" && input.Step != "meat" {
		return nil, httpx.ValidationError("step must be one of staple or meat")
	}

	now := time.Now()
	var updated model.Order
	var changed bool

	if err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		orderRepo := repo.NewOrder(tx)
		current, err := orderRepo.GetByID(ctx, actor.WorkspaceID, orderID)
		if err != nil {
			return err
		}
		if current == nil {
			return httpx.NotFoundError("order_not_found", "order not found")
		}
		if err := checkExpectedUpdatedAt(*current, input.ExpectedUpdatedAt); err != nil {
			return err
		}

		updated = current.ToggleStep(input.Step)
		if sameOrderProgress(*current, updated) {
			updated = *current
			return nil
		}

		changed = true
		updated.UpdatedBy = actor.UserID
		updated.UpdatedAt = now
		return orderRepo.Update(ctx, &updated)
	}); err != nil {
		return nil, wrapError("failed to toggle order step", err)
	}

	if changed {
		s.publishUpserts([]model.Order{updated})
	}
	return &updated, nil
}

// ToggleServed toggles the served (completed) state of an order.
func (s *Order) ToggleServed(ctx context.Context, actor model.Actor, orderID uuid.UUID, input model.ToggleServedInput) (*model.Order, error) {
	now := time.Now()
	var updated model.Order

	if err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		orderRepo := repo.NewOrder(tx)
		current, err := orderRepo.GetByID(ctx, actor.WorkspaceID, orderID)
		if err != nil {
			return err
		}
		if current == nil {
			return httpx.NotFoundError("order_not_found", "order not found")
		}
		if err := checkExpectedUpdatedAt(*current, input.ExpectedUpdatedAt); err != nil {
			return err
		}
		if !current.CanServe() {
			return httpx.ValidationError("order cannot be served until required steps are completed")
		}

		updated = current.ToggleServed(now)
		updated.UpdatedBy = actor.UserID
		updated.UpdatedAt = now
		return orderRepo.Update(ctx, &updated)
	}); err != nil {
		return nil, wrapError("failed to toggle served status", err)
	}

	s.publishUpserts([]model.Order{updated})
	return &updated, nil
}

// Remove deletes an order.
func (s *Order) Remove(ctx context.Context, actor model.Actor, orderID uuid.UUID) error {
	if err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		orderRepo := repo.NewOrder(tx)
		removed, err := orderRepo.Delete(ctx, actor.WorkspaceID, orderID)
		if err != nil {
			return err
		}
		if !removed {
			return httpx.NotFoundError("order_not_found", "order not found")
		}
		return nil
	}); err != nil {
		return wrapError("failed to delete order", err)
	}

	if s.publisher != nil {
		s.publisher.Publish(actor.WorkspaceID, model.EventOrderDeleted, model.DeletedEvent{ID: orderID.String()})
	}
	return nil
}

// ClearWorkspace deletes orders from a workspace, optionally filtering by date.
func (s *Order) ClearWorkspace(ctx context.Context, actor model.Actor, confirm bool, mode model.ClearWorkspaceMode) (int, error) {
	if !actor.Role.CanClear() {
		return 0, httpx.ForbiddenError("only owner can clear workspace orders")
	}
	if !confirm {
		return 0, httpx.ValidationError("confirm must be true")
	}
	if !mode.Valid() {
		return 0, httpx.ValidationError("mode must be one of all, before_today")
	}

	resolvedMode := mode.Normalize()
	var clearBefore *time.Time
	if resolvedMode == model.ClearWorkspaceModeBeforeToday {
		todayStart := orderBusinessDate(time.Now(), s.location)
		clearBefore = &todayStart
	}

	var cleared int
	if err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		count, err := s.clearWorkspaceInTx(ctx, tx, actor, resolvedMode, clearBefore)
		if err != nil {
			return err
		}
		cleared = count
		return nil
	}); err != nil {
		return 0, httpx.InternalError("failed to clear workspace orders", err)
	}

	return cleared, nil
}

// clearWorkspaceInTx performs the workspace clear within an existing transaction.
func (s *Order) clearWorkspaceInTx(
	ctx context.Context,
	db bun.IDB,
	actor model.Actor,
	mode model.ClearWorkspaceMode,
	clearBefore *time.Time,
) (int, error) {
	orderRepo := repo.NewOrder(db)
	count, err := orderRepo.ClearWorkspace(ctx, actor.WorkspaceID, clearBefore)
	if err != nil {
		return 0, err
	}
	counterRepo := repo.NewCounter(db)
	if err := counterRepo.ResetWorkspace(ctx, actor.WorkspaceID, clearBefore); err != nil {
		return 0, err
	}

	if s.publisher != nil {
		s.publisher.Publish(actor.WorkspaceID, model.EventOrderCleared, model.ClearedEvent{
			ClearedCount: count,
			Mode:         mode,
		})
	}

	return count, nil
}

// buildDisplayNo creates a display number from a business date and sequence number.
func buildDisplayNo(bizDate time.Time, seq int) string {
	return fmt.Sprintf("%s%04d", bizDate.Format("20060102"), seq)
}

// orderBusinessDate returns the start of the business day for the given time.
func orderBusinessDate(now time.Time, location *time.Location) time.Time {
	if location != nil {
		now = now.In(location)
	}
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
}

// checkExpectedUpdatedAt enforces optimistic concurrency on order updates.
func checkExpectedUpdatedAt(current model.Order, expected *time.Time) error {
	if expected == nil {
		return nil
	}
	if !current.UpdatedAt.Equal(*expected) {
		return httpx.ConflictError("order_conflict", "order has been modified by another request")
	}
	return nil
}

// sameOrderProgress reports whether the step status and completion state are unchanged.
func sameOrderProgress(before, after model.Order) bool {
	if before.StapleStepStatusCode != after.StapleStepStatusCode {
		return false
	}
	if before.MeatStepStatusCode != after.MeatStepStatusCode {
		return false
	}
	if before.CompletedAt == nil && after.CompletedAt == nil {
		return true
	}
	if before.CompletedAt != nil && after.CompletedAt != nil {
		return before.CompletedAt.Equal(*after.CompletedAt)
	}
	return false
}

// publishUpserts publishes upsert events for each order item.
func (s *Order) publishUpserts(items []model.Order) {
	if s.publisher == nil {
		return
	}
	for _, item := range items {
		s.publisher.Publish(item.WorkspaceID, model.EventOrderUpsert, model.UpsertEvent{
			Item: item.ToDTO(s.location),
		})
	}
}

// wrapError passes through AppError instances and wraps everything else as InternalError.
func wrapError(message string, err error) error {
	var appErr *httpx.AppError
	if errors.As(err, &appErr) {
		return err
	}
	return httpx.InternalError(message, err)
}

