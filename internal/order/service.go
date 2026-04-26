package order

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"

	clockpkg "github.com/dongwlin/legero-backend/internal/infra/clock"
	"github.com/dongwlin/legero-backend/internal/infra/httpx"
	idspkg "github.com/dongwlin/legero-backend/internal/infra/ids"
	"github.com/dongwlin/legero-backend/internal/workspace"
)

type Service struct {
	db        *bun.DB
	repo      Repository
	counters  CounterRepository
	clock     clockpkg.Clock
	ids       idspkg.Generator
	location  *time.Location
	publisher Publisher
}

func NewService(
	database *bun.DB,
	repo Repository,
	counters CounterRepository,
	clock clockpkg.Clock,
	ids idspkg.Generator,
	location *time.Location,
	publisher Publisher,
) *Service {
	return &Service{
		db:        database,
		repo:      repo,
		counters:  counters,
		clock:     clock,
		ids:       ids,
		location:  location,
		publisher: publisher,
	}
}

func (s *Service) ListActive(ctx context.Context, workspaceID uuid.UUID) ([]Order, error) {
	return s.repo.ListActive(ctx, s.db, workspaceID)
}

func (s *Service) List(ctx context.Context, actor Actor, query ListOrdersQuery) (*ListOrdersResult, error) {
	if !query.Status.Valid() {
		return nil, httpx.ValidationError("status must be one of uncompleted, completed, all")
	}

	items, nextCursor, err := s.repo.List(ctx, s.db, actor.WorkspaceID, query)
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

	now := s.clock.Now()
	bizDate := businessDate(now, s.location)
	items := make([]Order, 0, input.Quantity)

	if err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		startSeq, err := s.counters.Allocate(ctx, tx, actor.WorkspaceID, bizDate, input.Quantity, now)
		if err != nil {
			return err
		}

		for idx := 0; idx < input.Quantity; idx++ {
			stapleStatus, meatStatus, completedAt := InitialStepStatuses(form)
			item := Order{
				ID:                   s.ids.New(),
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

		return s.repo.InsertMany(ctx, tx, items)
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

	now := s.clock.Now()
	var updated Order

	if err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		current, err := s.repo.GetByID(ctx, tx, actor.WorkspaceID, orderID)
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

		return s.repo.Update(ctx, tx, updated)
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

	now := s.clock.Now()
	var updated Order
	var changed bool

	if err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		current, err := s.repo.GetByID(ctx, tx, actor.WorkspaceID, orderID)
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
		return s.repo.Update(ctx, tx, updated)
	}); err != nil {
		return nil, wrapOrderServiceError("failed to toggle order step", err)
	}

	if changed {
		s.publishUpserts([]Order{updated})
	}
	return &updated, nil
}

func (s *Service) ToggleServed(ctx context.Context, actor Actor, orderID uuid.UUID, input ToggleServedInput) (*Order, error) {
	now := s.clock.Now()
	var updated Order

	if err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		current, err := s.repo.GetByID(ctx, tx, actor.WorkspaceID, orderID)
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
		return s.repo.Update(ctx, tx, updated)
	}); err != nil {
		return nil, wrapOrderServiceError("failed to toggle served status", err)
	}

	s.publishUpserts([]Order{updated})
	return &updated, nil
}

func (s *Service) Remove(ctx context.Context, actor Actor, orderID uuid.UUID) error {
	if err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		removed, err := s.repo.Delete(ctx, tx, actor.WorkspaceID, orderID)
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

func (s *Service) ClearWorkspace(ctx context.Context, actor Actor, confirm bool) (int, error) {
	if !workspace.CanClear(actor.Role) {
		return 0, httpx.ForbiddenError("only owner can clear workspace orders")
	}
	if !confirm {
		return 0, httpx.ValidationError("confirm must be true")
	}

	var cleared int
	if err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		count, err := s.repo.ClearWorkspace(ctx, tx, actor.WorkspaceID)
		if err != nil {
			return err
		}
		if err := s.counters.ResetWorkspace(ctx, tx, actor.WorkspaceID); err != nil {
			return err
		}
		cleared = count
		return nil
	}); err != nil {
		return 0, httpx.InternalError("failed to clear workspace orders", err)
	}

	if s.publisher != nil {
		s.publisher.Publish(actor.WorkspaceID, EventOrderCleared, ClearedEvent{ClearedCount: cleared})
	}
	return cleared, nil
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
