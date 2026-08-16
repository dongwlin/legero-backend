package v1

import (
	"context"
	"errors"
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

// TestUpdateForm_LegacyExpectedUpdatedAt verifies backward compatibility: the
// deprecated expectedUpdatedAt token still rejects stale writes when no
// expectedVersion is provided. The token must come from the DB round-trip,
// since the in-memory timestamp carries nanosecond precision that Postgres
// truncates to microseconds.
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
	legacyToken := fresh.UpdatedAt

	// A write carrying the current UpdatedAt succeeds...
	_, err = svc.UpdateForm(ctx, ownerActor(userID, wsID), created.ID, domain.UpdateOrderInput{
		Form:              validOrderForm(),
		ExpectedUpdatedAt: &legacyToken,
	})
	require.NoError(t, err)

	// ...but reusing the same token afterwards must conflict.
	_, err = svc.UpdateForm(ctx, ownerActor(userID, wsID), created.ID, domain.UpdateOrderInput{
		Form:              validOrderForm(),
		ExpectedUpdatedAt: &legacyToken,
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
