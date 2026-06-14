package repo

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/dongwlin/legero-backend/internal/model"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// OrderRepo.InsertMany
// ---------------------------------------------------------------------------

func TestInsertMany_EmptySlice(t *testing.T) {
	ctx := context.Background()
	_, repo := newTestOrderRepo(t, ctx)

	require.NoError(t, repo.InsertMany(ctx, nil))
	require.NoError(t, repo.InsertMany(ctx, []model.Order{}))
}

func TestInsertMany_SingleOrder(t *testing.T) {
	ctx := context.Background()
	tx, repo := newTestOrderRepo(t, ctx)

	wsID := createTestWorkspace(t, ctx, tx)
	userID := createTestUser(t, ctx, tx)

	order := model.Order{
		ID:                   uuid.New(),
		WorkspaceID:          wsID,
		DisplayNo:            "T100",
		SizeCode:             model.SizeSmall,
		StapleAmountCode:     model.AdjustmentNormal,
		GreensCode:           model.AdjustmentNormal,
		ScallionCode:         model.AdjustmentNormal,
		PepperCode:           model.AdjustmentNormal,
		DiningMethodCode:     model.DiningMethodDineIn,
		SelectedMeatCodes:    []int16{model.MeatLeanPork},
		TotalPriceCents:      1000,
		StapleStepStatusCode: model.StepStatusUnrequired,
		MeatStepStatusCode:   model.StepStatusUnrequired,
		CreatedBy:            userID,
		UpdatedBy:            userID,
		CreatedAt:            time.Now(),
		UpdatedAt:            time.Now(),
	}

	require.NoError(t, repo.InsertMany(ctx, []model.Order{order}))

	got, err := repo.GetByID(ctx, wsID, order.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, order.ID, got.ID)
	require.Equal(t, "T100", got.DisplayNo)
}

func TestInsertMany_MultipleOrders(t *testing.T) {
	ctx := context.Background()
	tx, repo := newTestOrderRepo(t, ctx)

	wsID := createTestWorkspace(t, ctx, tx)
	userID := createTestUser(t, ctx, tx)

	count := 5
	orders := make([]model.Order, count)
	for i := range orders {
		orders[i] = model.Order{
			ID:                   uuid.New(),
			WorkspaceID:          wsID,
			DisplayNo:            fmt.Sprintf("T%03d", 100+i),
			SizeCode:             model.SizeSmall,
			StapleAmountCode:     model.AdjustmentNormal,
			GreensCode:           model.AdjustmentNormal,
			ScallionCode:         model.AdjustmentNormal,
			PepperCode:           model.AdjustmentNormal,
			DiningMethodCode:     model.DiningMethodDineIn,
			SelectedMeatCodes:    []int16{model.MeatLeanPork},
			TotalPriceCents:      1000,
			StapleStepStatusCode: model.StepStatusUnrequired,
			MeatStepStatusCode:   model.StepStatusUnrequired,
			CreatedBy:            userID,
			UpdatedBy:            userID,
			CreatedAt:            time.Now().Add(time.Duration(i) * time.Second),
			UpdatedAt:            time.Now().Add(time.Duration(i) * time.Second),
		}
	}

	require.NoError(t, repo.InsertMany(ctx, orders))

	for _, want := range orders {
		got, err := repo.GetByID(ctx, wsID, want.ID)
		require.NoError(t, err)
		require.NotNil(t, got)
	}
}

func TestInsertMany_InvalidFK(t *testing.T) {
	ctx := context.Background()
	_, repo := newTestOrderRepo(t, ctx)

	badWorkspaceID := uuid.New()
	order := model.Order{
		ID:                   uuid.New(),
		WorkspaceID:          badWorkspaceID,
		DisplayNo:            "T999",
		SizeCode:             model.SizeSmall,
		StapleAmountCode:     model.AdjustmentNormal,
		GreensCode:           model.AdjustmentNormal,
		ScallionCode:         model.AdjustmentNormal,
		PepperCode:           model.AdjustmentNormal,
		DiningMethodCode:     model.DiningMethodDineIn,
		TotalPriceCents:      1000,
		StapleStepStatusCode: model.StepStatusUnrequired,
		MeatStepStatusCode:   model.StepStatusUnrequired,
		CreatedBy:            badWorkspaceID,
		UpdatedBy:            badWorkspaceID,
		CreatedAt:            time.Now(),
		UpdatedAt:            time.Now(),
	}

	err := repo.InsertMany(ctx, []model.Order{order})
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// OrderRepo.GetByID
// ---------------------------------------------------------------------------

func TestGetByID_Found(t *testing.T) {
	ctx := context.Background()
	tx, repo := newTestOrderRepo(t, ctx)

	wsID := createTestWorkspace(t, ctx, tx)
	userID := createTestUser(t, ctx, tx)
	created := createTestOrder(t, ctx, tx, wsID, userID, func(o *model.Order) {
		o.DisplayNo = "T500"
		o.TotalPriceCents = 2500
	})

	got, err := repo.GetByID(ctx, wsID, created.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, created.ID, got.ID)
	require.Equal(t, "T500", got.DisplayNo)
	require.Equal(t, 2500, got.TotalPriceCents)
}

func TestGetByID_NotFound(t *testing.T) {
	ctx := context.Background()
	tx, repo := newTestOrderRepo(t, ctx)

	wsID := createTestWorkspace(t, ctx, tx)
	nonExistentID := uuid.New()

	got, err := repo.GetByID(ctx, wsID, nonExistentID)
	require.NoError(t, err)
	require.Nil(t, got)
}

func TestGetByID_WrongWorkspace(t *testing.T) {
	ctx := context.Background()
	tx, repo := newTestOrderRepo(t, ctx)

	wsID1 := createTestWorkspace(t, ctx, tx)
	wsID2 := createTestWorkspace(t, ctx, tx)
	userID := createTestUser(t, ctx, tx)

	created := createTestOrder(t, ctx, tx, wsID1, userID)

	got, err := repo.GetByID(ctx, wsID2, created.ID)
	require.NoError(t, err)
	require.Nil(t, got)
}

// ---------------------------------------------------------------------------
// OrderRepo.List
// ---------------------------------------------------------------------------

func TestList_StatusUncompleted(t *testing.T) {
	ctx := context.Background()
	tx, repo := newTestOrderRepo(t, ctx)

	wsID := createTestWorkspace(t, ctx, tx)
	userID := createTestUser(t, ctx, tx)

	// Uncompleted order (no CompletedAt).
	createTestOrder(t, ctx, tx, wsID, userID, func(o *model.Order) {
		o.DisplayNo = "U001"
		o.CreatedAt = time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)
		o.CompletedAt = nil
	})

	// Completed order.
	completedAt := time.Date(2025, 1, 1, 11, 0, 0, 0, time.UTC)
	createTestOrder(t, ctx, tx, wsID, userID, func(o *model.Order) {
		o.DisplayNo = "C001"
		o.CreatedAt = time.Date(2025, 1, 1, 9, 0, 0, 0, time.UTC)
		o.CompletedAt = &completedAt
	})

	orders, cursor, err := repo.List(ctx, wsID, model.ListOrdersQuery{
		Status: model.ListStatusUncompleted,
		Limit:  50,
	})
	require.NoError(t, err)
	require.Nil(t, cursor)
	require.Len(t, orders, 1)
	require.Equal(t, "U001", orders[0].DisplayNo)
}

func TestList_StatusCompleted(t *testing.T) {
	ctx := context.Background()
	tx, repo := newTestOrderRepo(t, ctx)

	wsID := createTestWorkspace(t, ctx, tx)
	userID := createTestUser(t, ctx, tx)

	// Uncompleted.
	createTestOrder(t, ctx, tx, wsID, userID, func(o *model.Order) {
		o.DisplayNo = "U001"
		o.CreatedAt = time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)
		o.CompletedAt = nil
	})

	// Completed.
	completedAt := time.Date(2025, 1, 1, 11, 0, 0, 0, time.UTC)
	createTestOrder(t, ctx, tx, wsID, userID, func(o *model.Order) {
		o.DisplayNo = "C001"
		o.CreatedAt = time.Date(2025, 1, 1, 9, 0, 0, 0, time.UTC)
		o.CompletedAt = &completedAt
	})

	orders, cursor, err := repo.List(ctx, wsID, model.ListOrdersQuery{
		Status: model.ListStatusCompleted,
		Limit:  50,
	})
	require.NoError(t, err)
	require.Nil(t, cursor)
	require.Len(t, orders, 1)
	require.Equal(t, "C001", orders[0].DisplayNo)
}

func TestList_StatusAll(t *testing.T) {
	ctx := context.Background()
	tx, repo := newTestOrderRepo(t, ctx)

	wsID := createTestWorkspace(t, ctx, tx)
	userID := createTestUser(t, ctx, tx)

	now := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	completedAt := now.Add(time.Hour)

	createTestOrder(t, ctx, tx, wsID, userID, func(o *model.Order) {
		o.DisplayNo = "U001"
		o.CreatedAt = now
		o.CompletedAt = nil
	})
	createTestOrder(t, ctx, tx, wsID, userID, func(o *model.Order) {
		o.DisplayNo = "C001"
		o.CreatedAt = now.Add(time.Minute)
		o.CompletedAt = &completedAt
	})
	createTestOrder(t, ctx, tx, wsID, userID, func(o *model.Order) {
		o.DisplayNo = "U002"
		o.CreatedAt = now.Add(2 * time.Minute)
		o.CompletedAt = nil
	})

	orders, cursor, err := repo.List(ctx, wsID, model.ListOrdersQuery{
		Status: model.ListStatusAll,
		Limit:  50,
	})
	require.NoError(t, err)
	require.Nil(t, cursor)
	require.Len(t, orders, 3)
	// StatusAll orders by created_at DESC, id DESC.
	require.Equal(t, "U002", orders[0].DisplayNo)
}

func TestList_Pagination(t *testing.T) {
	ctx := context.Background()
	tx, repo := newTestOrderRepo(t, ctx)

	wsID := createTestWorkspace(t, ctx, tx)
	userID := createTestUser(t, ctx, tx)

	base := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	for i := 0; i < 5; i++ {
		createTestOrder(t, ctx, tx, wsID, userID, func(o *model.Order) {
			o.DisplayNo = fmt.Sprintf("T%03d", i)
			o.CreatedAt = base.Add(time.Duration(i) * time.Minute)
			o.CompletedAt = nil
		})
	}

	// First page: limit=2, uncompleted (ordered by created_at ASC).
	page1, cursor1, err := repo.List(ctx, wsID, model.ListOrdersQuery{
		Status: model.ListStatusUncompleted,
		Limit:  2,
	})
	require.NoError(t, err)
	require.Len(t, page1, 2)
	require.NotNil(t, cursor1)
	require.Equal(t, "T000", page1[0].DisplayNo)

	// Second page using cursor.
	page2, cursor2, err := repo.List(ctx, wsID, model.ListOrdersQuery{
		Status: model.ListStatusUncompleted,
		Limit:  2,
		Cursor: *cursor1,
	})
	require.NoError(t, err)
	require.Len(t, page2, 2)
	require.NotNil(t, cursor2)
	require.Equal(t, "T002", page2[0].DisplayNo)

	// Third page: remaining orders.
	page3, cursor3, err := repo.List(ctx, wsID, model.ListOrdersQuery{
		Status: model.ListStatusUncompleted,
		Limit:  2,
		Cursor: *cursor2,
	})
	require.NoError(t, err)
	require.Len(t, page3, 1)
	require.Nil(t, cursor3)
}

func TestList_LimitClamping(t *testing.T) {
	ctx := context.Background()
	tx, repo := newTestOrderRepo(t, ctx)

	wsID := createTestWorkspace(t, ctx, tx)
	userID := createTestUser(t, ctx, tx)

	base := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	for i := 0; i < 3; i++ {
		createTestOrder(t, ctx, tx, wsID, userID, func(o *model.Order) {
			o.DisplayNo = fmt.Sprintf("L%03d", i)
			o.CreatedAt = base.Add(time.Duration(i) * time.Minute)
		})
	}

	// limit=0 should clamp to 50, no error.
	orders, _, err := repo.List(ctx, wsID, model.ListOrdersQuery{Status: model.ListStatusAll, Limit: 0})
	require.NoError(t, err)
	require.Len(t, orders, 3)

	// limit=300 should clamp to 200, no error.
	orders, _, err = repo.List(ctx, wsID, model.ListOrdersQuery{Status: model.ListStatusAll, Limit: 300})
	require.NoError(t, err)
	require.Len(t, orders, 3)

	// Negative limit should clamp to 50, no error.
	orders, _, err = repo.List(ctx, wsID, model.ListOrdersQuery{Status: model.ListStatusAll, Limit: -1})
	require.NoError(t, err)
	require.Len(t, orders, 3)
}

func TestList_InvalidStatus(t *testing.T) {
	ctx := context.Background()
	tx, repo := newTestOrderRepo(t, ctx)

	wsID := createTestWorkspace(t, ctx, tx)

	_, _, err := repo.List(ctx, wsID, model.ListOrdersQuery{Status: "bogus", Limit: 50})
	require.Error(t, err)
}

func TestList_WorkspaceIsolation(t *testing.T) {
	ctx := context.Background()
	tx, repo := newTestOrderRepo(t, ctx)

	wsID1 := createTestWorkspace(t, ctx, tx)
	wsID2 := createTestWorkspace(t, ctx, tx)
	userID := createTestUser(t, ctx, tx)

	createTestOrder(t, ctx, tx, wsID1, userID, func(o *model.Order) {
		o.DisplayNo = "WS1-001"
	})
	createTestOrder(t, ctx, tx, wsID2, userID, func(o *model.Order) {
		o.DisplayNo = "WS2-001"
	})

	orders1, _, err := repo.List(ctx, wsID1, model.ListOrdersQuery{Status: model.ListStatusAll, Limit: 50})
	require.NoError(t, err)
	require.Len(t, orders1, 1)
	require.Equal(t, "WS1-001", orders1[0].DisplayNo)

	orders2, _, err := repo.List(ctx, wsID2, model.ListOrdersQuery{Status: model.ListStatusAll, Limit: 50})
	require.NoError(t, err)
	require.Len(t, orders2, 1)
	require.Equal(t, "WS2-001", orders2[0].DisplayNo)
}

// ---------------------------------------------------------------------------
// OrderRepo.ListActive
// ---------------------------------------------------------------------------

func TestListActive_OnlyUncompleted(t *testing.T) {
	ctx := context.Background()
	tx, repo := newTestOrderRepo(t, ctx)

	wsID := createTestWorkspace(t, ctx, tx)
	userID := createTestUser(t, ctx, tx)

	base := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)
	completedAt := base.Add(time.Hour)

	// Two uncompleted, one completed.
	createTestOrder(t, ctx, tx, wsID, userID, func(o *model.Order) {
		o.DisplayNo = "A001"
		o.CreatedAt = base
		o.CompletedAt = nil
	})
	createTestOrder(t, ctx, tx, wsID, userID, func(o *model.Order) {
		o.DisplayNo = "A002"
		o.CreatedAt = base.Add(time.Minute)
		o.CompletedAt = nil
	})
	createTestOrder(t, ctx, tx, wsID, userID, func(o *model.Order) {
		o.DisplayNo = "C001"
		o.CreatedAt = base.Add(2 * time.Minute)
		o.CompletedAt = &completedAt
	})

	orders, err := repo.ListActive(ctx, wsID)
	require.NoError(t, err)
	require.Len(t, orders, 2)
	// Ordered by created_at ASC.
	require.Equal(t, "A001", orders[0].DisplayNo)
	require.Equal(t, "A002", orders[1].DisplayNo)
}

func TestListActive_EmptyWorkspace(t *testing.T) {
	ctx := context.Background()
	tx, repo := newTestOrderRepo(t, ctx)

	wsID := createTestWorkspace(t, ctx, tx)

	orders, err := repo.ListActive(ctx, wsID)
	require.NoError(t, err)
	require.NotNil(t, orders)
	require.Len(t, orders, 0)
}

// ---------------------------------------------------------------------------
// OrderRepo.Update
// ---------------------------------------------------------------------------

func TestUpdate_FieldsChanged(t *testing.T) {
	ctx := context.Background()
	tx, repo := newTestOrderRepo(t, ctx)

	wsID := createTestWorkspace(t, ctx, tx)
	userID := createTestUser(t, ctx, tx)
	created := createTestOrder(t, ctx, tx, wsID, userID)

	created.DisplayNo = "UPDATED-001"
	created.TotalPriceCents = 9999
	created.Note = "updated note"
	created.UpdatedAt = time.Now()

	require.NoError(t, repo.Update(ctx, &created))

	got, err := repo.GetByID(ctx, wsID, created.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, "UPDATED-001", got.DisplayNo)
	require.Equal(t, 9999, got.TotalPriceCents)
	require.Equal(t, "updated note", got.Note)
}

func TestUpdate_WrongWorkspace(t *testing.T) {
	ctx := context.Background()
	tx, repo := newTestOrderRepo(t, ctx)

	wsID1 := createTestWorkspace(t, ctx, tx)
	wsID2 := createTestWorkspace(t, ctx, tx)
	userID := createTestUser(t, ctx, tx)

	created := createTestOrder(t, ctx, tx, wsID1, userID)

	// Attempt update with wrong workspace.
	created.WorkspaceID = wsID2
	created.DisplayNo = "SHOULD-NOT-WORK"

	err := repo.Update(ctx, &created)
	require.Error(t, err)
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func TestUpdate_NonExistentID(t *testing.T) {
	ctx := context.Background()
	tx, repo := newTestOrderRepo(t, ctx)

	wsID := createTestWorkspace(t, ctx, tx)
	userID := createTestUser(t, ctx, tx)

	fakeOrder := &model.Order{
		ID:                   uuid.New(),
		WorkspaceID:          wsID,
		DisplayNo:            "FAKE-001",
		SizeCode:             model.SizeSmall,
		StapleAmountCode:     model.AdjustmentNormal,
		GreensCode:           model.AdjustmentNormal,
		ScallionCode:         model.AdjustmentNormal,
		PepperCode:           model.AdjustmentNormal,
		DiningMethodCode:     model.DiningMethodDineIn,
		SelectedMeatCodes:    []int16{model.MeatLeanPork},
		TotalPriceCents:      1000,
		StapleStepStatusCode: model.StepStatusUnrequired,
		MeatStepStatusCode:   model.StepStatusUnrequired,
		CreatedBy:            userID,
		UpdatedBy:            userID,
		CreatedAt:            time.Now(),
		UpdatedAt:            time.Now(),
	}

	err := repo.Update(ctx, fakeOrder)
	require.Error(t, err)
	require.ErrorIs(t, err, sql.ErrNoRows)
}

// ---------------------------------------------------------------------------
// OrderRepo.Delete
// ---------------------------------------------------------------------------

func TestDelete_Existing(t *testing.T) {
	ctx := context.Background()
	tx, repo := newTestOrderRepo(t, ctx)

	wsID := createTestWorkspace(t, ctx, tx)
	userID := createTestUser(t, ctx, tx)
	created := createTestOrder(t, ctx, tx, wsID, userID)

	deleted, err := repo.Delete(ctx, wsID, created.ID)
	require.NoError(t, err)
	require.True(t, deleted)

	got, err := repo.GetByID(ctx, wsID, created.ID)
	require.NoError(t, err)
	require.Nil(t, got)
}

func TestDelete_NonExistent(t *testing.T) {
	ctx := context.Background()
	tx, repo := newTestOrderRepo(t, ctx)

	wsID := createTestWorkspace(t, ctx, tx)

	deleted, err := repo.Delete(ctx, wsID, uuid.New())
	require.NoError(t, err)
	require.False(t, deleted)
}

func TestDelete_WrongWorkspace(t *testing.T) {
	ctx := context.Background()
	tx, repo := newTestOrderRepo(t, ctx)

	wsID1 := createTestWorkspace(t, ctx, tx)
	wsID2 := createTestWorkspace(t, ctx, tx)
	userID := createTestUser(t, ctx, tx)

	created := createTestOrder(t, ctx, tx, wsID1, userID)

	deleted, err := repo.Delete(ctx, wsID2, created.ID)
	require.NoError(t, err)
	require.False(t, deleted)

	// Verify order still exists in correct workspace.
	got, err := repo.GetByID(ctx, wsID1, created.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
}

// ---------------------------------------------------------------------------
// OrderRepo.ClearWorkspace
// ---------------------------------------------------------------------------

func TestClearWorkspace_All(t *testing.T) {
	ctx := context.Background()
	tx, repo := newTestOrderRepo(t, ctx)

	wsID := createTestWorkspace(t, ctx, tx)
	userID := createTestUser(t, ctx, tx)

	createTestOrder(t, ctx, tx, wsID, userID, func(o *model.Order) {
		o.DisplayNo = "CLR-001"
		o.CreatedAt = time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	})
	createTestOrder(t, ctx, tx, wsID, userID, func(o *model.Order) {
		o.DisplayNo = "CLR-002"
		o.CreatedAt = time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	})

	deleted, err := repo.ClearWorkspace(ctx, wsID, nil)
	require.NoError(t, err)
	require.Equal(t, 2, deleted)

	orders, _, err := repo.List(ctx, wsID, model.ListOrdersQuery{Status: model.ListStatusAll, Limit: 50})
	require.NoError(t, err)
	require.Len(t, orders, 0)
}

func TestClearWorkspace_WithCreatedBefore(t *testing.T) {
	ctx := context.Background()
	tx, repo := newTestOrderRepo(t, ctx)

	wsID := createTestWorkspace(t, ctx, tx)
	userID := createTestUser(t, ctx, tx)

	oldDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	newDate := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	cutoff := time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)

	createTestOrder(t, ctx, tx, wsID, userID, func(o *model.Order) {
		o.DisplayNo = "OLD-001"
		o.CreatedAt = oldDate
	})
	createTestOrder(t, ctx, tx, wsID, userID, func(o *model.Order) {
		o.DisplayNo = "NEW-001"
		o.CreatedAt = newDate
	})

	deleted, err := repo.ClearWorkspace(ctx, wsID, &cutoff)
	require.NoError(t, err)
	require.Equal(t, 1, deleted)

	orders, _, err := repo.List(ctx, wsID, model.ListOrdersQuery{Status: model.ListStatusAll, Limit: 50})
	require.NoError(t, err)
	require.Len(t, orders, 1)
	require.Equal(t, "NEW-001", orders[0].DisplayNo)
}

func TestClearWorkspace_Empty(t *testing.T) {
	ctx := context.Background()
	tx, repo := newTestOrderRepo(t, ctx)

	wsID := createTestWorkspace(t, ctx, tx)

	deleted, err := repo.ClearWorkspace(ctx, wsID, nil)
	require.NoError(t, err)
	require.Equal(t, 0, deleted)
}

// ---------------------------------------------------------------------------
// CounterRepo.Allocate
// ---------------------------------------------------------------------------

func TestAllocate_FirstAllocation(t *testing.T) {
	ctx := context.Background()
	tx, repo := newTestCounterRepo(t, ctx)

	wsID := createTestWorkspace(t, ctx, tx)
	bizDate := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	now := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)

	startSeq, err := repo.Allocate(ctx, wsID, bizDate, 1, now)
	require.NoError(t, err)
	require.Equal(t, 1, startSeq)
}

func TestAllocate_SubsequentAllocation(t *testing.T) {
	ctx := context.Background()
	tx, repo := newTestCounterRepo(t, ctx)

	wsID := createTestWorkspace(t, ctx, tx)
	bizDate := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	now := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)

	startSeq1, err := repo.Allocate(ctx, wsID, bizDate, 1, now)
	require.NoError(t, err)
	require.Equal(t, 1, startSeq1)

	startSeq2, err := repo.Allocate(ctx, wsID, bizDate, 1, now)
	require.NoError(t, err)
	require.Equal(t, 2, startSeq2)
}

func TestAllocate_MultipleQuantity(t *testing.T) {
	ctx := context.Background()
	tx, repo := newTestCounterRepo(t, ctx)

	wsID := createTestWorkspace(t, ctx, tx)
	bizDate := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	now := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)

	// First batch: 3 sequences.
	startSeq1, err := repo.Allocate(ctx, wsID, bizDate, 3, now)
	require.NoError(t, err)
	require.Equal(t, 1, startSeq1)

	// Second batch: 2 sequences.
	startSeq2, err := repo.Allocate(ctx, wsID, bizDate, 2, now)
	require.NoError(t, err)
	require.Equal(t, 4, startSeq2)
}

func TestAllocate_ZeroQuantity(t *testing.T) {
	ctx := context.Background()
	tx, repo := newTestCounterRepo(t, ctx)

	wsID := createTestWorkspace(t, ctx, tx)
	bizDate := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	now := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)

	_, err := repo.Allocate(ctx, wsID, bizDate, 0, now)
	require.Error(t, err)
}

func TestAllocate_NegativeQuantity(t *testing.T) {
	ctx := context.Background()
	tx, repo := newTestCounterRepo(t, ctx)

	wsID := createTestWorkspace(t, ctx, tx)
	bizDate := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	now := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)

	_, err := repo.Allocate(ctx, wsID, bizDate, -1, now)
	require.Error(t, err)
}

func TestAllocate_DifferentBizDates(t *testing.T) {
	ctx := context.Background()
	tx, repo := newTestCounterRepo(t, ctx)

	wsID := createTestWorkspace(t, ctx, tx)
	date1 := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	date2 := time.Date(2025, 6, 2, 0, 0, 0, 0, time.UTC)
	now := time.Date(2025, 6, 2, 12, 0, 0, 0, time.UTC)

	// Allocate on date1.
	start1, err := repo.Allocate(ctx, wsID, date1, 1, now)
	require.NoError(t, err)
	require.Equal(t, 1, start1)

	// Allocate on date2 — independent counter.
	start2, err := repo.Allocate(ctx, wsID, date2, 1, now)
	require.NoError(t, err)
	require.Equal(t, 1, start2)
}

// ---------------------------------------------------------------------------
// CounterRepo.ResetWorkspace
// ---------------------------------------------------------------------------

func TestResetWorkspace_All(t *testing.T) {
	ctx := context.Background()
	tx, repo := newTestCounterRepo(t, ctx)

	wsID := createTestWorkspace(t, ctx, tx)
	date1 := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	date2 := time.Date(2025, 6, 2, 0, 0, 0, 0, time.UTC)
	now := time.Date(2025, 6, 2, 12, 0, 0, 0, time.UTC)

	// Create two counters on different dates.
	_, err := repo.Allocate(ctx, wsID, date1, 1, now)
	require.NoError(t, err)
	_, err = repo.Allocate(ctx, wsID, date2, 1, now)
	require.NoError(t, err)

	require.NoError(t, repo.ResetWorkspace(ctx, wsID, nil))

	// Allocating again should start from 1 (counter was reset).
	startSeq, err := repo.Allocate(ctx, wsID, date1, 1, now)
	require.NoError(t, err)
	require.Equal(t, 1, startSeq)
}

func TestResetWorkspace_WithBizDateBefore(t *testing.T) {
	ctx := context.Background()
	tx, repo := newTestCounterRepo(t, ctx)

	wsID := createTestWorkspace(t, ctx, tx)
	date1 := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	date2 := time.Date(2025, 6, 10, 0, 0, 0, 0, time.UTC)
	cutoff := time.Date(2025, 6, 5, 0, 0, 0, 0, time.UTC)
	now := time.Date(2025, 6, 10, 12, 0, 0, 0, time.UTC)

	// Create counters on two dates.
	_, err := repo.Allocate(ctx, wsID, date1, 3, now)
	require.NoError(t, err)
	_, err = repo.Allocate(ctx, wsID, date2, 2, now)
	require.NoError(t, err)

	// Reset only counters before cutoff.
	require.NoError(t, repo.ResetWorkspace(ctx, wsID, &cutoff))

	// date1 counter was reset — next allocation starts at 1.
	startSeq1, err := repo.Allocate(ctx, wsID, date1, 1, now)
	require.NoError(t, err)
	require.Equal(t, 1, startSeq1)

	// date2 counter was NOT reset — next allocation continues from 3.
	startSeq2, err := repo.Allocate(ctx, wsID, date2, 1, now)
	require.NoError(t, err)
	require.Equal(t, 3, startSeq2)
}

func TestResetWorkspace_Empty(t *testing.T) {
	ctx := context.Background()
	tx, repo := newTestCounterRepo(t, ctx)

	wsID := createTestWorkspace(t, ctx, tx)

	require.NoError(t, repo.ResetWorkspace(ctx, wsID, nil))
}
