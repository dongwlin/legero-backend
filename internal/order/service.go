package order

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"

	"github.com/dongwlin/legero-backend/internal/infra/httpx"
	"github.com/dongwlin/legero-backend/internal/workspace"
)

type Service struct {
	db        *bun.DB
	location  *time.Location
	publisher Publisher
}

func NewService(
	db *bun.DB,
	location *time.Location,
	publisher Publisher,
) *Service {
	return &Service{
		db:        db,
		location:  location,
		publisher: publisher,
	}
}

func (s *Service) ListActive(ctx context.Context, workspaceID uuid.UUID) ([]Order, error) {
	orderRepo := NewOrderRepo(s.db)
	return orderRepo.ListActive(ctx, workspaceID)
}

func (s *Service) List(ctx context.Context, actor Actor, query ListOrdersQuery) (*ListOrdersResult, error) {
	if !query.Status.Valid() {
		return nil, httpx.ValidationError("status must be one of uncompleted, completed, all")
	}

	orderRepo := NewOrderRepo(s.db)
	items, nextCursor, err := orderRepo.List(ctx, actor.WorkspaceID, query)
	if err != nil {
		return nil, httpx.InternalError("failed to list orders", err)
	}

	return &ListOrdersResult{
		Items:      items,
		NextCursor: nextCursor,
	}, nil
}

func (s *Service) CreateBatch(ctx context.Context, actor Actor, input CreateOrdersInput) ([]Order, error) {
	if input.Quantity <= 0 {
		return nil, httpx.ValidationError("quantity must be greater than 0")
	}

	form, err := NormalizeForm(input.Form)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	bizDate := businessDate(now, s.location)
	items := make([]Order, 0, input.Quantity)

	if err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		counterRepo := NewCounterRepo(tx)
		startSeq, err := counterRepo.Allocate(ctx, actor.WorkspaceID, bizDate, input.Quantity, now)
		if err != nil {
			return err
		}

		for idx := 0; idx < input.Quantity; idx++ {
			stapleStatus, meatStatus, completedAt := InitialStepStatuses(form)
			item := Order{
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
				SelectedMeatCodes:    cloneInt16s(form.SelectedMeatCodes),
				GreensCode:           form.GreensCode,
				ScallionCode:         form.ScallionCode,
				PepperCode:           form.PepperCode,
				DiningMethodCode:     form.DiningMethodCode,
				PackagingCode:        form.PackagingCode,
				PackagingMethodCode:  form.PackagingMethodCode,
				TotalPriceCents:      CalculateTotalPriceCents(form),
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

		orderRepo := NewOrderRepo(tx)
		return orderRepo.InsertMany(ctx, items)
	}); err != nil {
		return nil, httpx.InternalError("failed to create orders", err)
	}

	s.publishUpserts(items)
	return items, nil
}

func (s *Service) UpdateForm(ctx context.Context, actor Actor, orderID uuid.UUID, input UpdateOrderInput) (*Order, error) {
	form, err := NormalizeForm(input.Form)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	var updated Order

	if err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		orderRepo := NewOrderRepo(tx)
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

		stapleStatus, meatStatus, completedAt := InitialStepStatuses(form)
		updated = Order{
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
			SelectedMeatCodes:    cloneInt16s(form.SelectedMeatCodes),
			GreensCode:           form.GreensCode,
			ScallionCode:         form.ScallionCode,
			PepperCode:           form.PepperCode,
			DiningMethodCode:     form.DiningMethodCode,
			PackagingCode:        form.PackagingCode,
			PackagingMethodCode:  form.PackagingMethodCode,
			TotalPriceCents:      CalculateTotalPriceCents(form),
			StapleStepStatusCode: stapleStatus,
			MeatStepStatusCode:   meatStatus,
			Note:                 form.Note,
			CreatedBy:            current.CreatedBy,
			UpdatedBy:            actor.UserID,
			CreatedAt:            current.CreatedAt,
			UpdatedAt:            now,
			CompletedAt:          completedAt,
		}

		return orderRepo.Update(ctx, updated)
	}); err != nil {
		return nil, wrapOrderServiceError("failed to update order", err)
	}

	s.publishUpserts([]Order{updated})
	return &updated, nil
}

func (s *Service) ToggleStep(ctx context.Context, actor Actor, orderID uuid.UUID, input ToggleStepInput) (*Order, error) {
	if input.Step != "staple" && input.Step != "meat" {
		return nil, httpx.ValidationError("step must be one of staple or meat")
	}

	now := time.Now()
	var updated Order
	var changed bool

	if err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		orderRepo := NewOrderRepo(tx)
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

		updated = ToggleStep(*current, input.Step)
		if sameOrderProgress(*current, updated) {
			updated = *current
			return nil
		}

		changed = true
		updated.UpdatedBy = actor.UserID
		updated.UpdatedAt = now
		return orderRepo.Update(ctx, updated)
	}); err != nil {
		return nil, wrapOrderServiceError("failed to toggle order step", err)
	}

	if changed {
		s.publishUpserts([]Order{updated})
	}
	return &updated, nil
}

func (s *Service) ToggleServed(ctx context.Context, actor Actor, orderID uuid.UUID, input ToggleServedInput) (*Order, error) {
	now := time.Now()
	var updated Order

	if err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		orderRepo := NewOrderRepo(tx)
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
		if !CanServe(*current) {
			return httpx.ValidationError("order cannot be served until required steps are completed")
		}

		updated = ToggleServed(*current, now)
		updated.UpdatedBy = actor.UserID
		updated.UpdatedAt = now
		return orderRepo.Update(ctx, updated)
	}); err != nil {
		return nil, wrapOrderServiceError("failed to toggle served status", err)
	}

	s.publishUpserts([]Order{updated})
	return &updated, nil
}

func (s *Service) Remove(ctx context.Context, actor Actor, orderID uuid.UUID) error {
	if err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		orderRepo := NewOrderRepo(tx)
		removed, err := orderRepo.Delete(ctx, actor.WorkspaceID, orderID)
		if err != nil {
			return err
		}
		if !removed {
			return httpx.NotFoundError("order_not_found", "order not found")
		}
		return nil
	}); err != nil {
		return wrapOrderServiceError("failed to delete order", err)
	}

	if s.publisher != nil {
		s.publisher.Publish(actor.WorkspaceID, EventOrderDeleted, DeletedEvent{ID: orderID.String()})
	}
	return nil
}

func (s *Service) ClearWorkspace(ctx context.Context, actor Actor, confirm bool, mode ClearWorkspaceMode) (int, error) {
	if !workspace.CanClear(actor.Role) {
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
	if resolvedMode == ClearWorkspaceModeBeforeToday {
		todayStart := businessDate(time.Now(), s.location)
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

func (s *Service) clearWorkspaceInTx(
	ctx context.Context,
	db bun.IDB,
	actor Actor,
	mode ClearWorkspaceMode,
	clearBefore *time.Time,
) (int, error) {
	orderRepo := NewOrderRepo(db)
	count, err := orderRepo.ClearWorkspace(ctx, actor.WorkspaceID, clearBefore)
	if err != nil {
		return 0, err
	}
	counterRepo := NewCounterRepo(db)
	if err := counterRepo.ResetWorkspace(ctx, actor.WorkspaceID, clearBefore); err != nil {
		return 0, err
	}

	if s.publisher != nil {
		s.publisher.Publish(actor.WorkspaceID, EventOrderCleared, ClearedEvent{
			ClearedCount: count,
			Mode:         mode,
		})
	}

	return count, nil
}

func buildDisplayNo(bizDate time.Time, seq int) string {
	return fmt.Sprintf("%s%04d", bizDate.Format("20060102"), seq)
}

func businessDate(now time.Time, location *time.Location) time.Time {
	if location != nil {
		now = now.In(location)
	}
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
}

func checkExpectedUpdatedAt(current Order, expected *time.Time) error {
	if expected == nil {
		return nil
	}
	if !current.UpdatedAt.Equal(*expected) {
		return httpx.ConflictError("order_conflict", "order has been modified by another request")
	}
	return nil
}

func sameOrderProgress(before, after Order) bool {
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

func (s *Service) publishUpserts(items []Order) {
	if s.publisher == nil {
		return
	}
	for _, item := range items {
		s.publisher.Publish(item.WorkspaceID, EventOrderUpsert, UpsertEvent{
			Item: ToDTO(item, s.location),
		})
	}
}

func wrapOrderServiceError(message string, err error) error {
	var appErr *httpx.AppError
	if ok := errorAs(err, &appErr); ok {
		return err
	}
	return httpx.InternalError(message, err)
}

func errorAs(err error, target any) bool {
	return errors.As(err, target)
}
