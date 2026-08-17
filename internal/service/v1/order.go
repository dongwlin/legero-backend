package v1

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"

	"github.com/dongwlin/legero-backend/internal/apperr"
	"github.com/dongwlin/legero-backend/internal/domain"
	"github.com/dongwlin/legero-backend/internal/repo"
	"github.com/dongwlin/legero-backend/internal/service"
)

// maxOrderUpsertBatchSize bounds the number of order DTOs in one realtime
// frame. A large create request is split into several batch events so the
// protocol remains efficient without allowing an unbounded payload.
const maxOrderUpsertBatchSize = 64

// order implements service.Order.
type order struct {
	db        *bun.DB
	location  *time.Location
	publisher domain.Publisher
}

// NewOrder creates a new Order service.
func NewOrder(
	db *bun.DB,
	location *time.Location,
	publisher domain.Publisher,
) service.Order {
	return &order{
		db:        db,
		location:  location,
		publisher: publisher,
	}
}

// ListActive returns all uncompleted orders for a workspace.
func (s *order) ListActive(ctx context.Context, workspaceID uuid.UUID) ([]domain.Order, error) {
	orderRepo := repo.NewOrder(s.db)
	return orderRepo.ListActive(ctx, workspaceID)
}

// List returns a paginated list of orders for a workspace.
func (s *order) List(ctx context.Context, actor domain.Actor, query domain.ListOrdersQuery) (*domain.ListOrdersResult, error) {
	if !query.Status.Valid() {
		return nil, apperr.ValidationError("status must be one of uncompleted, completed, all")
	}

	orderRepo := repo.NewOrder(s.db)
	items, nextCursor, err := orderRepo.List(ctx, actor.WorkspaceID, query)
	if err != nil {
		return nil, apperr.InternalError("failed to list orders", err)
	}

	return &domain.ListOrdersResult{
		Items:      items,
		NextCursor: nextCursor,
	}, nil
}

// CreateBatch creates multiple orders in a single transaction, allocating display numbers atomically.
func (s *order) CreateBatch(ctx context.Context, actor domain.Actor, input domain.CreateOrdersInput) ([]domain.Order, error) {
	if input.Quantity <= 0 {
		return nil, apperr.ValidationError("quantity must be greater than 0")
	}

	form, err := input.Form.Normalize()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	bizDate := orderBusinessDate(now, s.location)
	items := make([]domain.Order, 0, input.Quantity)

	if err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		counterRepo := repo.NewCounter(tx)
		startSeq, err := counterRepo.Allocate(ctx, actor.WorkspaceID, bizDate, input.Quantity, now)
		if err != nil {
			return err
		}

		for idx := 0; idx < input.Quantity; idx++ {
			stapleStatus, meatStatus, completedAt := form.InitialStepStatuses()
			item := domain.Order{
				ID:                   uuid.New(),
				WorkspaceID:          actor.WorkspaceID,
				DisplayNo:            buildDisplayNo(bizDate, startSeq+idx),
				Version:              1,
				StapleTypeCode:       form.StapleTypeCode,
				SizeCode:             form.SizeCode,
				CustomSizePriceCents: form.CustomSizePriceCents,
				StapleAmountCode:     form.StapleAmountCode,
				ExtraStapleUnits:     form.ExtraStapleUnits,
				FriedEggCount:        form.FriedEggCount,
				TofuSkewerCount:      form.TofuSkewerCount,
				SelectedMeatCodes:    domain.CloneInt16s(form.SelectedMeatCodes),
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
		return nil, apperr.InternalError("failed to create orders", err)
	}

	s.publishUpserts(items)
	return items, nil
}

// UpdateForm replaces the form data of an existing order.
func (s *order) UpdateForm(ctx context.Context, actor domain.Actor, orderID uuid.UUID, input domain.UpdateOrderInput) (*domain.Order, error) {
	form, err := input.Form.Normalize()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	var updated domain.Order

	if err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		orderRepo := repo.NewOrder(tx)
		current, err := orderRepo.GetByID(ctx, actor.WorkspaceID, orderID)
		if err != nil {
			return err
		}
		if current == nil {
			return apperr.NotFoundError("order_not_found", "order not found")
		}
		if input.ExpectedVersion != nil {
			if err := checkExpectedVersion(*current, input.ExpectedVersion); err != nil {
				return err
			}
		} else if err := checkExpectedUpdatedAt(*current, input.ExpectedUpdatedAt); err != nil {
			return err
		}

		stapleStatus, meatStatus, completedAt := form.InitialStepStatuses()
		updated = domain.Order{
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
			SelectedMeatCodes:    domain.CloneInt16s(form.SelectedMeatCodes),
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

		return orderRepo.Update(ctx, &updated, input.ExpectedVersion)
	}); err != nil {
		return nil, wrapOrderMutationError("failed to update order", err)
	}

	s.publishUpserts([]domain.Order{updated})
	return &updated, nil
}

// ToggleStep toggles the completion state of a cooking step ("staple" or "meat").
func (s *order) ToggleStep(ctx context.Context, actor domain.Actor, orderID uuid.UUID, input domain.ToggleStepInput) (*domain.Order, error) {
	if input.Step != "staple" && input.Step != "meat" {
		return nil, apperr.ValidationError("step must be one of staple or meat")
	}

	now := time.Now()
	var updated domain.Order
	var changed bool

	if err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		orderRepo := repo.NewOrder(tx)
		current, err := orderRepo.GetByID(ctx, actor.WorkspaceID, orderID)
		if err != nil {
			return err
		}
		if current == nil {
			return apperr.NotFoundError("order_not_found", "order not found")
		}
		if input.ExpectedVersion != nil {
			if err := checkExpectedVersion(*current, input.ExpectedVersion); err != nil {
				return err
			}
		} else if err := checkExpectedUpdatedAt(*current, input.ExpectedUpdatedAt); err != nil {
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
		return orderRepo.Update(ctx, &updated, input.ExpectedVersion)
	}); err != nil {
		return nil, wrapOrderMutationError("failed to toggle order step", err)
	}

	if changed {
		s.publishUpserts([]domain.Order{updated})
	}
	return &updated, nil
}

// ToggleServed toggles the served (completed) state of an order.
func (s *order) ToggleServed(ctx context.Context, actor domain.Actor, orderID uuid.UUID, input domain.ToggleServedInput) (*domain.Order, error) {
	now := time.Now()
	var updated domain.Order

	if err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		orderRepo := repo.NewOrder(tx)
		current, err := orderRepo.GetByID(ctx, actor.WorkspaceID, orderID)
		if err != nil {
			return err
		}
		if current == nil {
			return apperr.NotFoundError("order_not_found", "order not found")
		}
		if input.ExpectedVersion != nil {
			if err := checkExpectedVersion(*current, input.ExpectedVersion); err != nil {
				return err
			}
		} else if err := checkExpectedUpdatedAt(*current, input.ExpectedUpdatedAt); err != nil {
			return err
		}
		if !current.CanServe() {
			return apperr.ValidationError("order cannot be served until required steps are completed")
		}

		updated = current.ToggleServed(now)
		updated.UpdatedBy = actor.UserID
		updated.UpdatedAt = now
		return orderRepo.Update(ctx, &updated, input.ExpectedVersion)
	}); err != nil {
		return nil, wrapOrderMutationError("failed to toggle served status", err)
	}

	s.publishUpserts([]domain.Order{updated})
	return &updated, nil
}

// Remove deletes an order.
func (s *order) Remove(ctx context.Context, actor domain.Actor, orderID uuid.UUID) error {
	if err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		orderRepo := repo.NewOrder(tx)
		removed, err := orderRepo.Delete(ctx, actor.WorkspaceID, orderID)
		if err != nil {
			return err
		}
		if !removed {
			return apperr.NotFoundError("order_not_found", "order not found")
		}
		return nil
	}); err != nil {
		return wrapError("failed to delete order", err)
	}

	if s.publisher != nil {
		s.publisher.Publish(actor.WorkspaceID, domain.EventOrderDeleted, domain.DeletedEvent{ID: orderID.String()})
	}
	return nil
}

// ClearWorkspace deletes orders from a workspace, optionally filtering by date.
func (s *order) ClearWorkspace(ctx context.Context, actor domain.Actor, confirm bool, mode domain.ClearWorkspaceMode) (int, error) {
	if !actor.Role.CanClear() {
		return 0, apperr.ForbiddenError("only owner can clear workspace orders")
	}
	if !confirm {
		return 0, apperr.ValidationError("confirm must be true")
	}
	if !mode.Valid() {
		return 0, apperr.ValidationError("mode must be one of all, before_today")
	}

	resolvedMode := mode.Normalize()
	var clearBefore *time.Time
	if resolvedMode == domain.ClearWorkspaceModeBeforeToday {
		todayStart := orderBusinessDate(time.Now(), s.location)
		clearBefore = &todayStart
	}

	var cleared int
	if err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		count, err := s.clearWorkspaceInTx(ctx, tx, actor, clearBefore)
		if err != nil {
			return err
		}
		cleared = count
		return nil
	}); err != nil {
		return 0, apperr.InternalError("failed to clear workspace orders", err)
	}

	// The delete committed: only now is the cleared state durable and
	// observable to other transactions (e.g. the follow-up snapshot a client
	// starts as soon as it receives this event). Publishing inside the
	// transaction would let such a snapshot race the commit and read the
	// pre-clear rows, breaking the client-side full-clear epoch barrier (the
	// snapshot would be treated as guaranteed post-clear and reaffirm ids
	// the not-yet-committed clear deleted). This matches the
	// commit-then-publish ordering of Create, Update and Remove.
	if s.publisher != nil {
		// The before_today cutoff is the server's authoritative business day:
		// carry it (YYYY-MM-DD in the workspace timezone) so clients pin
		// their barrier to the date the clear actually used — a delayed,
		// cross-midnight event or a skewed client clock must never re-derive
		// a different boundary. Empty for 'all' clears.
		clearDateKey := ""
		if clearBefore != nil {
			clearDateKey = clearBefore.Format("2006-01-02")
		}

		s.publisher.Publish(actor.WorkspaceID, domain.EventOrderCleared, domain.ClearedEvent{
			ClearedCount: cleared,
			Mode:         resolvedMode,
			ClearDateKey: clearDateKey,
		})
	}

	return cleared, nil
}

// clearWorkspaceInTx performs the workspace clear within an existing
// transaction. It only mutates the database — the caller publishes the
// realtime event after the transaction commits, so a client that reacts to
// the event can never observe the pre-clear state (see ClearWorkspace).
func (s *order) clearWorkspaceInTx(
	ctx context.Context,
	db bun.IDB,
	actor domain.Actor,
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

// orderConflictError reports an optimistic-concurrency conflict on an order.
func orderConflictError() *apperr.AppError {
	return apperr.ConflictError("order_conflict", "order has been modified by another request")
}

// checkExpectedVersion enforces optimistic concurrency on order updates using
// the monotonic version. A provided expected version must equal the current
// version, otherwise the order was modified concurrently.
func checkExpectedVersion(current domain.Order, expected *int64) error {
	if expected == nil {
		return nil
	}
	if current.Version != *expected {
		return orderConflictError()
	}
	return nil
}

// checkExpectedUpdatedAt enforces optimistic concurrency on order updates
// using the deprecated expectedUpdatedAt token. The token is compared at the
// precision the API actually exposes: OrderDTO.UpdatedAt is formatted with
// time.RFC3339, which has no fractional seconds, so both sides are truncated
// to the second before comparing. Sub-second concurrent writes are handled by
// the monotonic version (expectedVersion) instead of this fallback.
func checkExpectedUpdatedAt(current domain.Order, expected *time.Time) error {
	if expected == nil {
		return nil
	}
	if !current.UpdatedAt.Truncate(time.Second).Equal(expected.Truncate(time.Second)) {
		return orderConflictError()
	}
	return nil
}

// sameOrderProgress reports whether the step status and completion state are unchanged.
func sameOrderProgress(before, after domain.Order) bool {
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

// publishUpserts publishes one legacy single-order event for one item and
// bounded batch events for multiple items. CreateBatch uses this helper after
// its transaction commits, so subscribers never observe an uncommitted order.
func (s *order) publishUpserts(items []domain.Order) {
	if s.publisher == nil || len(items) == 0 {
		return
	}

	legacySingleEvent := len(items) == 1
	for start := 0; start < len(items); {
		end := start + 1
		workspaceID := items[start].WorkspaceID
		for end < len(items) && end-start < maxOrderUpsertBatchSize && items[end].WorkspaceID == workspaceID {
			end++
		}

		if legacySingleEvent {
			item := items[start]
			s.publisher.Publish(workspaceID, domain.EventOrderUpsert, domain.UpsertEvent{
				Item: item.ToDTO(s.location),
			})
		} else {
			dtos := make([]domain.OrderDTO, 0, end-start)
			for _, item := range items[start:end] {
				dtos = append(dtos, item.ToDTO(s.location))
			}
			s.publisher.Publish(workspaceID, domain.EventOrderUpsertMany, domain.UpsertManyEvent{
				Items: dtos,
			})
		}

		start = end
	}
}

// wrapError passes through AppError instances and wraps everything else as InternalError.
func wrapError(message string, err error) error {
	var appErr *apperr.AppError
	if errors.As(err, &appErr) {
		return err
	}
	return apperr.InternalError(message, err)
}

// wrapOrderMutationError translates order-concurrency sentinels into conflict
// errors before falling back to wrapError.
func wrapOrderMutationError(message string, err error) error {
	if errors.Is(err, domain.ErrOrderConflict) {
		return orderConflictError()
	}
	return wrapError(message, err)
}
