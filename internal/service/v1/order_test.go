package v1

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/dongwlin/legero-backend/internal/apperr"
	"github.com/dongwlin/legero-backend/internal/domain"
	"github.com/dongwlin/legero-backend/internal/repo"
	"github.com/dongwlin/legero-backend/internal/service"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
)

// createTestOrder inserts an order with Version = 1 directly through the repo.
func createTestOrder(t *testing.T, ctx context.Context, db bun.IDB, workspaceID, userID uuid.UUID, opts ...func(*domain.Order)) domain.Order {
	t.Helper()

	now := time.Now()
	order := domain.Order{
		ID:                   uuid.New(),
		WorkspaceID:          workspaceID,
		DisplayNo:            "SVC-001",
		Version:              1,
		SizeCode:             domain.SizeSmall,
		StapleAmountCode:     domain.AdjustmentNormal,
		GreensCode:           domain.AdjustmentNormal,
		ScallionCode:         domain.AdjustmentNormal,
		PepperCode:           domain.AdjustmentNormal,
		DiningMethodCode:     domain.DiningMethodDineIn,
		SelectedMeatCodes:    []int16{domain.MeatLeanPork},
		TotalPriceCents:      1000,
		StapleStepStatusCode: domain.StepStatusUnrequired,
		MeatStepStatusCode:   domain.StepStatusUnrequired,
		CreatedBy:            userID,
		UpdatedBy:            userID,
		CreatedAt:            now,
		UpdatedAt:            now,
	}

	for _, opt := range opts {
		opt(&order)
	}

	require.NoError(t, repo.NewOrder(db).InsertMany(ctx, []domain.Order{order}))
	return order
}

// validOrderForm returns a form that passes Normalize and needs a meat step.
func validOrderForm() domain.OrderFormInput {
	return domain.OrderFormInput{
		SizeCode:          domain.SizeSmall,
		StapleAmountCode:  domain.AdjustmentNormal,
		GreensCode:        domain.AdjustmentNormal,
		ScallionCode:      domain.AdjustmentNormal,
		PepperCode:        domain.AdjustmentNormal,
		DiningMethodCode:  domain.DiningMethodDineIn,
		SelectedMeatCodes: []int16{domain.MeatLeanPork},
	}
}

func newTestOrderService(t *testing.T) service.Order {
	t.Helper()
	return NewOrder(testDB, time.UTC, nil)
}

func ownerActor(userID, workspaceID uuid.UUID) domain.Actor {
	return domain.Actor{
		UserID:      userID,
		WorkspaceID: workspaceID,
		Role:        domain.RoleOwner,
	}
}

// ---------------------------------------------------------------------------
// OrderService.UpdateForm
// ---------------------------------------------------------------------------

// TestUpdateForm_AdvancesVersion verifies that successive updates advance the
// monotonic version, and that the version is still distinguishable when the
// mutations land within the same second (no sleeps needed).
func TestUpdateForm_AdvancesVersion(t *testing.T) {
	ctx := context.Background()
	svc := newTestOrderService(t)

	userID := createTestUser(t, ctx, testDB)
	wsID := createTestWorkspace(t, ctx, testDB)
	createTestWorkspaceMember(t, ctx, testDB, userID, wsID, "owner")
	created := createTestOrder(t, ctx, testDB, wsID, userID)
	require.Equal(t, int64(1), created.Version)

	actor := ownerActor(userID, wsID)

	first, err := svc.UpdateForm(ctx, actor, created.ID, domain.UpdateOrderInput{
		Form: validOrderForm(),
	})
	require.NoError(t, err)
	require.Equal(t, int64(2), first.Version)

	second, err := svc.UpdateForm(ctx, actor, created.ID, domain.UpdateOrderInput{
		Form: validOrderForm(),
	})
	require.NoError(t, err)
	require.Equal(t, int64(3), second.Version)
	require.NotEqual(t, first.Version, second.Version)
}

// TestUpdateForm_ExpectedVersionConflict verifies that a stale expectedVersion
// is rejected with a 409-style conflict error and the order is left untouched.
func TestUpdateForm_ExpectedVersionConflict(t *testing.T) {
	ctx := context.Background()
	svc := newTestOrderService(t)

	userID := createTestUser(t, ctx, testDB)
	wsID := createTestWorkspace(t, ctx, testDB)
	createTestWorkspaceMember(t, ctx, testDB, userID, wsID, "owner")
	created := createTestOrder(t, ctx, testDB, wsID, userID)

	// Advance the order once so the stored version is 2.
	actor := ownerActor(userID, wsID)
	updated, err := svc.UpdateForm(ctx, actor, created.ID, domain.UpdateOrderInput{Form: validOrderForm()})
	require.NoError(t, err)
	require.Equal(t, int64(2), updated.Version)

	// A client still holding version 1 must be rejected.
	stale := int64(1)
	_, err = svc.UpdateForm(ctx, actor, created.ID, domain.UpdateOrderInput{
		Form:            validOrderForm(),
		ExpectedVersion: &stale,
	})
	require.Error(t, err)
	var appErr *apperr.AppError
	require.True(t, errors.As(err, &appErr))
	require.Equal(t, apperr.KindConflict, appErr.Kind)
	require.Equal(t, "order_conflict", appErr.Code)

	// The stored version must not have been advanced by the rejected write.
	got, err := repo.NewOrder(testDB).GetByID(ctx, wsID, created.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, int64(2), got.Version)
}

// TestUpdateForm_ExpectedVersionSuccess verifies that a matching expectedVersion
// allows the mutation and advances the version.
func TestUpdateForm_ExpectedVersionSuccess(t *testing.T) {
	ctx := context.Background()
	svc := newTestOrderService(t)

	userID := createTestUser(t, ctx, testDB)
	wsID := createTestWorkspace(t, ctx, testDB)
	createTestWorkspaceMember(t, ctx, testDB, userID, wsID, "owner")
	created := createTestOrder(t, ctx, testDB, wsID, userID)

	expected := int64(1)
	updated, err := svc.UpdateForm(ctx, ownerActor(userID, wsID), created.ID, domain.UpdateOrderInput{
		Form:            validOrderForm(),
		ExpectedVersion: &expected,
	})
	require.NoError(t, err)
	require.Equal(t, int64(2), updated.Version)
}

// TestUpdateForm_LegacyExpectedUpdatedAt verifies backward compatibility of
// the deprecated expectedUpdatedAt token: a token built exactly the way a
// legacy client produces one — parsed back from OrderDTO.UpdatedAt, which is
// RFC3339 without fractional seconds — still passes the optimistic-concurrency
// check, while a stale token is still rejected with a conflict.
func TestUpdateForm_LegacyExpectedUpdatedAt(t *testing.T) {
	ctx := context.Background()
	svc := newTestOrderService(t)

	userID := createTestUser(t, ctx, testDB)
	wsID := createTestWorkspace(t, ctx, testDB)
	createTestWorkspaceMember(t, ctx, testDB, userID, wsID, "owner")
	created := createTestOrder(t, ctx, testDB, wsID, userID)

	fresh, err := repo.NewOrder(testDB).GetByID(ctx, wsID, created.ID)
	require.NoError(t, err)
	require.NotNil(t, fresh)

	// A real client never sees the DB timestamp directly: it only sees the
	// DTO, whose UpdatedAt is RFC3339 at second precision. Replay that exact
	// HTTP round trip instead of taking the DB model's UpdatedAt verbatim.
	token, err := time.Parse(time.RFC3339, fresh.ToDTO(time.UTC).UpdatedAt)
	require.NoError(t, err)

	// A write carrying the current DTO token succeeds...
	_, err = svc.UpdateForm(ctx, ownerActor(userID, wsID), created.ID, domain.UpdateOrderInput{
		Form:              validOrderForm(),
		ExpectedUpdatedAt: &token,
	})
	require.NoError(t, err)

	// ...but a stale token must conflict. Reusing the just-consumed token
	// cannot assert staleness deterministically: updates landing in the same
	// second share the truncated timestamp, so build a token that is provably
	// older than the current UpdatedAt.
	stale := token.Add(-2 * time.Second)
	_, err = svc.UpdateForm(ctx, ownerActor(userID, wsID), created.ID, domain.UpdateOrderInput{
		Form:              validOrderForm(),
		ExpectedUpdatedAt: &stale,
	})
	require.Error(t, err)
	var appErr *apperr.AppError
	require.True(t, errors.As(err, &appErr))
	require.Equal(t, apperr.KindConflict, appErr.Kind)
}

// ---------------------------------------------------------------------------
// OrderService.CreateBatch
// ---------------------------------------------------------------------------

// TestCreateBatch_StartsVersionAtOne verifies that newly created orders expose
// version 1.
func TestCreateBatch_StartsVersionAtOne(t *testing.T) {
	ctx := context.Background()
	svc := newTestOrderService(t)

	userID := createTestUser(t, ctx, testDB)
	wsID := createTestWorkspace(t, ctx, testDB)
	createTestWorkspaceMember(t, ctx, testDB, userID, wsID, "owner")

	items, err := svc.CreateBatch(ctx, ownerActor(userID, wsID), domain.CreateOrdersInput{
		Quantity: 2,
		Form:     validOrderForm(),
	})
	require.NoError(t, err)
	require.Len(t, items, 2)
	for _, item := range items {
		require.Equal(t, int64(1), item.Version)
	}
}

// ---------------------------------------------------------------------------
// OrderService.ToggleStep
// ---------------------------------------------------------------------------

// TestToggleStep_AdvancesVersion verifies that state-changing step toggles
// advance the version.
func TestToggleStep_AdvancesVersion(t *testing.T) {
	ctx := context.Background()
	svc := newTestOrderService(t)

	userID := createTestUser(t, ctx, testDB)
	wsID := createTestWorkspace(t, ctx, testDB)
	createTestWorkspaceMember(t, ctx, testDB, userID, wsID, "owner")

	stapleType := domain.StapleTypeRiceSheet
	created := createTestOrder(t, ctx, testDB, wsID, userID, func(o *domain.Order) {
		o.StapleTypeCode = &stapleType
		o.StapleStepStatusCode = domain.StepStatusNotStarted
		o.MeatStepStatusCode = domain.StepStatusNotStarted
	})

	updated, err := svc.ToggleStep(ctx, ownerActor(userID, wsID), created.ID, domain.ToggleStepInput{
		Step: "staple",
	})
	require.NoError(t, err)
	require.Equal(t, int64(2), updated.Version)
	require.Equal(t, domain.StepStatusCompleted, updated.StapleStepStatusCode)

	// A second toggle, even in the same second, produces a new version.
	reverted, err := svc.ToggleStep(ctx, ownerActor(userID, wsID), created.ID, domain.ToggleStepInput{
		Step: "staple",
	})
	require.NoError(t, err)
	require.Equal(t, int64(3), reverted.Version)
	require.Equal(t, domain.StepStatusNotStarted, reverted.StapleStepStatusCode)
}

// TestToggleStep_NoopDoesNotAdvance verifies that a toggle that changes
// nothing does not bump the version.
func TestToggleStep_NoopDoesNotAdvance(t *testing.T) {
	ctx := context.Background()
	svc := newTestOrderService(t)

	userID := createTestUser(t, ctx, testDB)
	wsID := createTestWorkspace(t, ctx, testDB)
	createTestWorkspaceMember(t, ctx, testDB, userID, wsID, "owner")

	// No staple step required: toggling "staple" is a no-op.
	created := createTestOrder(t, ctx, testDB, wsID, userID)

	updated, err := svc.ToggleStep(ctx, ownerActor(userID, wsID), created.ID, domain.ToggleStepInput{
		Step: "staple",
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), updated.Version)

	got, err := repo.NewOrder(testDB).GetByID(ctx, wsID, created.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, int64(1), got.Version)
}

// ---------------------------------------------------------------------------
// OrderService.ToggleServed
// ---------------------------------------------------------------------------

// TestToggleServed_AdvancesVersion verifies that toggling served state
// advances the version on each state change.
func TestToggleServed_AdvancesVersion(t *testing.T) {
	ctx := context.Background()
	svc := newTestOrderService(t)

	userID := createTestUser(t, ctx, testDB)
	wsID := createTestWorkspace(t, ctx, testDB)
	createTestWorkspaceMember(t, ctx, testDB, userID, wsID, "owner")

	// Only a meat step is required; complete it so the order can be served.
	created := createTestOrder(t, ctx, testDB, wsID, userID, func(o *domain.Order) {
		o.MeatStepStatusCode = domain.StepStatusCompleted
	})

	served, err := svc.ToggleServed(ctx, ownerActor(userID, wsID), created.ID, domain.ToggleServedInput{})
	require.NoError(t, err)
	require.Equal(t, int64(2), served.Version)
	require.NotNil(t, served.CompletedAt)

	unserved, err := svc.ToggleServed(ctx, ownerActor(userID, wsID), created.ID, domain.ToggleServedInput{})
	require.NoError(t, err)
	require.Equal(t, int64(3), unserved.Version)
	require.Nil(t, unserved.CompletedAt)
}

// ---------------------------------------------------------------------------
// OrderDTO exposure
// ---------------------------------------------------------------------------

// TestToDTO_ExposesVersion verifies that order DTOs (list/snapshot, mutation
// responses, and realtime upserts all use OrderDTO) carry the version.
func TestToDTO_ExposesVersion(t *testing.T) {
	order := domain.Order{ID: uuid.New(), Version: 17}
	dto := order.ToDTO(time.UTC)
	require.Equal(t, int64(17), dto.Version)
	require.Equal(t, order.ID.String(), dto.ID)
}

// ---------------------------------------------------------------------------
// ClearWorkspace commit-then-publish causality
// ---------------------------------------------------------------------------

// recordingPublisher implements domain.Publisher, collecting the cleared
// events it published. onPublish, when set, runs synchronously inside
// Publish: it stands in for the moment a replica client would receive the
// event and react (typically by starting a follow-up snapshot), so
// assertions made there check what that client can observe.
type recordingPublisher struct {
	mu        sync.Mutex
	events    []domain.ClearedEvent
	onPublish func(wsID uuid.UUID, event domain.ClearedEvent)
}

func (p *recordingPublisher) Publish(workspaceID uuid.UUID, eventType string, payload any) {
	if eventType != domain.EventOrderCleared {
		return
	}

	event, ok := payload.(domain.ClearedEvent)
	if !ok {
		return
	}

	p.mu.Lock()
	p.events = append(p.events, event)
	onPublish := p.onPublish
	p.mu.Unlock()

	if onPublish != nil {
		onPublish(workspaceID, event)
	}
}

// TestClearWorkspace_OrderClearedPublishedOnlyAfterCommit is the causality
// guarantee the client's full-clear epoch barrier relies on: once
// order.cleared is observable, the delete must already be committed and
// visible to other connections. Publishing inside the transaction would let
// the follow-up snapshot a client starts on receiving the event read the
// pre-clear rows and misjudge the epoch.
func TestClearWorkspace_OrderClearedPublishedOnlyAfterCommit(t *testing.T) {
	ctx := context.Background()
	userID := createTestUser(t, ctx, testDB)
	wsID := createTestWorkspace(t, ctx, testDB)
	createTestWorkspaceMember(t, ctx, testDB, userID, wsID, "owner")

	createTestOrder(t, ctx, testDB, wsID, userID, func(o *domain.Order) {
		o.DisplayNo = "CLR-A"
		o.CreatedAt = time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	})
	createTestOrder(t, ctx, testDB, wsID, userID, func(o *domain.Order) {
		o.DisplayNo = "CLR-B"
		o.CreatedAt = time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	})

	publisher := &recordingPublisher{
		onPublish: func(publishWSID uuid.UUID, _ domain.ClearedEvent) {
			require.Equal(t, wsID, publishWSID)
			// A different connection (the pool's next free one — the clear
			// transaction holds one) queries the workspace exactly like a
			// client snapshot would. Both orders must already be gone: if the
			// event had been published before the transaction committed, this
			// query would still see them (uncommitted DELETEs are invisible
			// under read committed).
			orders, _, err := repo.NewOrder(testDB).List(ctx, wsID, domain.ListOrdersQuery{
				Status: domain.ListStatusAll,
				Limit:  50,
			})
			require.NoError(t, err)
			require.Empty(t, orders, "order.cleared must only be observable after the clear transaction committed")
		},
	}

	svc := NewOrder(testDB, time.UTC, publisher)
	cleared, err := svc.ClearWorkspace(ctx, ownerActor(userID, wsID), true, domain.ClearWorkspaceModeAll)
	require.NoError(t, err)
	require.Equal(t, 2, cleared)

	publisher.mu.Lock()
	defer publisher.mu.Unlock()
	require.Len(t, publisher.events, 1)
	require.Equal(t, domain.ClearedEvent{
		ClearedCount: 2,
		Mode:         domain.ClearWorkspaceModeAll,
	}, publisher.events[0])
}

// TestClearWorkspace_BeforeToday_PublishesAuthorityCutoffAfterCommit covers
// the same commit-then-publish guarantee for before_today clears and checks
// the authoritative ClearDateKey is carried: the client pins its barrier to
// that server-computed business-day key.
func TestClearWorkspace_BeforeToday_PublishesAuthorityCutoffAfterCommit(t *testing.T) {
	ctx := context.Background()
	userID := createTestUser(t, ctx, testDB)
	wsID := createTestWorkspace(t, ctx, testDB)
	createTestWorkspaceMember(t, ctx, testDB, userID, wsID, "owner")
	now := time.Now()
	today := orderBusinessDate(now, time.UTC)

	createTestOrder(t, ctx, testDB, wsID, userID, func(o *domain.Order) {
		o.DisplayNo = "OLD"
		o.CreatedAt = today.Add(-time.Second)
	})
	createTestOrder(t, ctx, testDB, wsID, userID, func(o *domain.Order) {
		o.DisplayNo = "TODAY"
		o.CreatedAt = now
	})

	publisher := &recordingPublisher{
		onPublish: func(wsID uuid.UUID, event domain.ClearedEvent) {
			require.Equal(t, domain.ClearWorkspaceModeBeforeToday, event.Mode)
			require.Equal(t, today.Format("2006-01-02"), event.ClearDateKey)

			// At event-observability time the old order is gone but today's
			// order (which the server kept) is still visible to a snapshot.
			orders, _, err := repo.NewOrder(testDB).List(ctx, wsID, domain.ListOrdersQuery{
				Status: domain.ListStatusAll,
				Limit:  50,
			})
			require.NoError(t, err)
			require.Len(t, orders, 1)
			require.Equal(t, "TODAY", orders[0].DisplayNo)
		},
	}

	svc := NewOrder(testDB, time.UTC, publisher)
	cleared, err := svc.ClearWorkspace(ctx, ownerActor(userID, wsID), true, domain.ClearWorkspaceModeBeforeToday)
	require.NoError(t, err)
	require.Equal(t, 1, cleared)

	publisher.mu.Lock()
	defer publisher.mu.Unlock()
	require.Len(t, publisher.events, 1)
	require.Equal(t, domain.ClearWorkspaceModeBeforeToday, publisher.events[0].Mode)
	require.Equal(t, today.Format("2006-01-02"), publisher.events[0].ClearDateKey)
}
