package repo

import (
	"context"
	"testing"
	"time"

	"github.com/dongwlin/legero-backend/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestStatsDaily_UsesCompletedAtBusinessWindowAndWorkspaceIsolation(t *testing.T) {
	ctx := context.Background()
	location, err := time.LoadLocation("Asia/Shanghai")
	require.NoError(t, err)

	tx, _ := newTestOrderRepo(t, ctx)
	statsRepo := NewStats(tx)
	userID := createTestUser(t, ctx, tx)
	workspaceID := createTestWorkspace(t, ctx, tx)
	otherWorkspaceID := createTestWorkspace(t, ctx, tx)

	start := time.Date(2026, 8, 17, 0, 0, 0, 0, location)
	to := time.Date(2026, 8, 18, 12, 0, 0, 0, location)
	exclusiveBoundary := time.Date(2026, 8, 19, 0, 0, 0, 0, location)

	// Completion time, rather than created_at, determines the business day.
	createTestOrder(t, ctx, tx, workspaceID, userID, func(order *domain.Order) {
		order.DisplayNo = "COMPLETED-AFTER-MIDNIGHT"
		order.CreatedAt = time.Date(2026, 8, 17, 23, 55, 0, 0, location)
		order.CompletedAt = timePtr(time.Date(2026, 8, 18, 0, 5, 0, 0, location))
		order.TotalPriceCents = 1200
	})
	createTestOrder(t, ctx, tx, workspaceID, userID, func(order *domain.Order) {
		order.DisplayNo = "COMPLETED-AT-START"
		order.CreatedAt = time.Date(2026, 8, 18, 1, 0, 0, 0, location)
		order.CompletedAt = timePtr(time.Date(2026, 8, 18, 0, 0, 0, 0, location))
		order.TotalPriceCents = 800
	})
	createTestOrder(t, ctx, tx, workspaceID, userID, func(order *domain.Order) {
		order.DisplayNo = "UNCOMPLETED"
		order.CreatedAt = time.Date(2026, 8, 18, 2, 0, 0, 0, location)
		order.CompletedAt = nil
		order.TotalPriceCents = 9999
	})
	createTestOrder(t, ctx, tx, workspaceID, userID, func(order *domain.Order) {
		order.DisplayNo = "COMPLETED-AT-END"
		order.CreatedAt = time.Date(2026, 8, 19, 1, 0, 0, 0, location)
		order.CompletedAt = timePtr(exclusiveBoundary)
		order.TotalPriceCents = 7777
	})
	createTestOrder(t, ctx, tx, otherWorkspaceID, userID, func(order *domain.Order) {
		order.DisplayNo = "OTHER-WORKSPACE"
		order.CompletedAt = timePtr(time.Date(2026, 8, 18, 12, 0, 0, 0, location))
		order.TotalPriceCents = 4000
	})

	rows, err := statsRepo.Daily(ctx, workspaceID, location.String(), start, to)
	require.NoError(t, err)
	require.Equal(t, []domain.DailyRow{
		{Date: time.Date(2026, 8, 18, 0, 0, 0, 0, location), OrderCount: 2, TotalPriceCents: 2000},
		{Date: time.Date(2026, 8, 17, 0, 0, 0, 0, location), OrderCount: 0, TotalPriceCents: 0},
	}, rows)

	// A same-day query remains inclusive of its only business date and does
	// not accidentally turn the date-only `from`/`to` pair into an empty range.
	sameDay, err := statsRepo.Daily(ctx, workspaceID, location.String(), start.AddDate(0, 0, 1), start.AddDate(0, 0, 1))
	require.NoError(t, err)
	require.Equal(t, []domain.DailyRow{
		{Date: time.Date(2026, 8, 18, 0, 0, 0, 0, location), OrderCount: 2, TotalPriceCents: 2000},
	}, sameDay)
}

func TestStatsDaily_IncludesEveryBusinessDateWithZeroes(t *testing.T) {
	ctx := context.Background()
	location, err := time.LoadLocation("Asia/Shanghai")
	require.NoError(t, err)

	tx, _ := newTestOrderRepo(t, ctx)
	statsRepo := NewStats(tx)
	userID := createTestUser(t, ctx, tx)
	workspaceID := createTestWorkspace(t, ctx, tx)

	createTestOrder(t, ctx, tx, workspaceID, userID, func(order *domain.Order) {
		order.DisplayNo = "ONE-DAY"
		order.CompletedAt = timePtr(time.Date(2026, 8, 17, 10, 0, 0, 0, location))
		order.TotalPriceCents = 1500
	})

	rows, err := statsRepo.Daily(
		ctx,
		workspaceID,
		location.String(),
		time.Date(2026, 8, 15, 12, 0, 0, 0, location),
		time.Date(2026, 8, 19, 12, 0, 0, 0, location),
	)
	require.NoError(t, err)
	require.Equal(t, []domain.DailyRow{
		{Date: time.Date(2026, 8, 19, 0, 0, 0, 0, location), OrderCount: 0, TotalPriceCents: 0},
		{Date: time.Date(2026, 8, 18, 0, 0, 0, 0, location), OrderCount: 0, TotalPriceCents: 0},
		{Date: time.Date(2026, 8, 17, 0, 0, 0, 0, location), OrderCount: 1, TotalPriceCents: 1500},
		{Date: time.Date(2026, 8, 16, 0, 0, 0, 0, location), OrderCount: 0, TotalPriceCents: 0},
		{Date: time.Date(2026, 8, 15, 0, 0, 0, 0, location), OrderCount: 0, TotalPriceCents: 0},
	}, rows)
}

func TestStatsCompletedOrders_UsesHalfOpenRangeAndWorkspaceIsolation(t *testing.T) {
	ctx := context.Background()
	location, err := time.LoadLocation("Asia/Shanghai")
	require.NoError(t, err)

	tx, _ := newTestOrderRepo(t, ctx)
	statsRepo := NewStats(tx)
	userID := createTestUser(t, ctx, tx)
	workspaceID := createTestWorkspace(t, ctx, tx)
	otherWorkspaceID := createTestWorkspace(t, ctx, tx)
	start := time.Date(2026, 8, 18, 0, 0, 0, 0, location)
	end := start.AddDate(0, 0, 1)

	createTestOrder(t, ctx, tx, workspaceID, userID, func(order *domain.Order) {
		order.DisplayNo = "INCLUSIVE-START"
		order.CompletedAt = timePtr(start)
		order.TotalPriceCents = 1234
	})
	createTestOrder(t, ctx, tx, workspaceID, userID, func(order *domain.Order) {
		order.DisplayNo = "EXCLUSIVE-END"
		order.CompletedAt = timePtr(end)
		order.TotalPriceCents = 2345
	})
	createTestOrder(t, ctx, tx, workspaceID, userID, func(order *domain.Order) {
		order.DisplayNo = "UNCOMPLETED"
		order.CompletedAt = nil
		order.TotalPriceCents = 3456
	})
	createTestOrder(t, ctx, tx, otherWorkspaceID, userID, func(order *domain.Order) {
		order.DisplayNo = "OTHER-WORKSPACE"
		order.CompletedAt = timePtr(start.Add(2 * time.Hour))
		order.TotalPriceCents = 4567
	})

	orders, err := statsRepo.CompletedOrders(ctx, workspaceID, start, end)
	require.NoError(t, err)
	require.Len(t, orders, 1)
	require.Equal(t, 1234, orders[0].TotalPriceCents)
	require.NotNil(t, orders[0].CompletedAt)
	require.True(t, orders[0].CompletedAt.Equal(start))
}

// TestStatsCompletedOrders_RoundTripsReportProjectionFields verifies the
// PostgreSQL -> Bun -> domain projection used by service.Report. In
// particular, selected_meat_codes exercises the smallint[] scan rather than
// only the scalar fields that are also present in the daily query.
func TestStatsCompletedOrders_RoundTripsReportProjectionFields(t *testing.T) {
	ctx := context.Background()
	location, err := time.LoadLocation("Asia/Shanghai")
	require.NoError(t, err)

	tx, _ := newTestOrderRepo(t, ctx)
	statsRepo := NewStats(tx)
	userID := createTestUser(t, ctx, tx)
	workspaceID := createTestWorkspace(t, ctx, tx)

	start := time.Date(2026, 8, 18, 0, 0, 0, 0, location)
	end := start.AddDate(0, 0, 1)
	stapleType := domain.StapleTypeYiNoodle
	createdAt := start.Add(8*time.Hour + 7*time.Minute + 3*time.Second)
	completedAt := start.Add(9*time.Hour + 29*time.Minute + 11*time.Second)
	wantMeatCodes := []int16{domain.MeatLeanPork, domain.MeatBloodCurd, domain.MeatKidney}

	createTestOrder(t, ctx, tx, workspaceID, userID, func(order *domain.Order) {
		order.DisplayNo = "REPORT-PROJECTION"
		order.StapleTypeCode = &stapleType
		order.SizeCode = domain.SizeLarge
		order.FriedEggCount = 3
		order.DiningMethodCode = domain.DiningMethodTakeout
		order.SelectedMeatCodes = wantMeatCodes
		order.CreatedAt = createdAt
		order.CompletedAt = &completedAt
		order.TotalPriceCents = 4321
	})

	orders, err := statsRepo.CompletedOrders(ctx, workspaceID, start, end)
	require.NoError(t, err)
	require.Len(t, orders, 1)

	got := orders[0]
	require.Equal(t, 4321, got.TotalPriceCents)
	require.NotNil(t, got.StapleTypeCode)
	require.Equal(t, stapleType, *got.StapleTypeCode)
	require.Equal(t, domain.SizeLarge, got.SizeCode)
	require.Equal(t, int16(3), got.FriedEggCount)
	require.Equal(t, domain.DiningMethodTakeout, got.DiningMethodCode)
	require.Equal(t, wantMeatCodes, got.SelectedMeatCodes)
	require.True(t, createdAt.Equal(got.CreatedAt))
	require.NotNil(t, got.CompletedAt)
	require.True(t, completedAt.Equal(*got.CompletedAt))
}

func TestStatsReflectsOrderMutationsImmediately(t *testing.T) {
	ctx := context.Background()
	location, err := time.LoadLocation("Asia/Shanghai")
	require.NoError(t, err)

	tx, orderRepo := newTestOrderRepo(t, ctx)
	statsRepo := NewStats(tx)
	userID := createTestUser(t, ctx, tx)
	workspaceID := createTestWorkspace(t, ctx, tx)
	completion := time.Date(2026, 8, 18, 10, 0, 0, 0, location)
	start := time.Date(2026, 8, 18, 0, 0, 0, 0, location)
	end := start.AddDate(0, 0, 1)

	order := createTestOrder(t, ctx, tx, workspaceID, userID, func(order *domain.Order) {
		order.DisplayNo = "MUTABLE"
		order.CreatedAt = time.Date(2026, 8, 18, 9, 0, 0, 0, location)
		order.CompletedAt = nil
		order.TotalPriceCents = 1600
	})
	assertStatsSummary(t, ctx, statsRepo, workspaceID, location.String(), start, end, 0, 0)

	_, err = tx.NewRaw("UPDATE orders SET completed_at = ? WHERE id = ?", completion, order.ID).Exec(ctx)
	require.NoError(t, err)
	assertStatsSummary(t, ctx, statsRepo, workspaceID, location.String(), start, end, 1, 1600)

	_, err = tx.NewRaw("UPDATE orders SET completed_at = NULL WHERE id = ?", order.ID).Exec(ctx)
	require.NoError(t, err)
	assertStatsSummary(t, ctx, statsRepo, workspaceID, location.String(), start, end, 0, 0)

	_, err = tx.NewRaw("UPDATE orders SET completed_at = ? WHERE id = ?", completion, order.ID).Exec(ctx)
	require.NoError(t, err)
	assertStatsSummary(t, ctx, statsRepo, workspaceID, location.String(), start, end, 1, 1600)

	deleted, err := orderRepo.Delete(ctx, workspaceID, order.ID)
	require.NoError(t, err)
	require.True(t, deleted)
	assertStatsSummary(t, ctx, statsRepo, workspaceID, location.String(), start, end, 0, 0)

	oldOrder := createTestOrder(t, ctx, tx, workspaceID, userID, func(order *domain.Order) {
		order.DisplayNo = "OLD"
		order.CreatedAt = start.Add(-24 * time.Hour)
		order.CompletedAt = timePtr(start.Add(-2 * time.Hour))
		order.TotalPriceCents = 1700
	})
	todayOrder := createTestOrder(t, ctx, tx, workspaceID, userID, func(order *domain.Order) {
		order.DisplayNo = "TODAY"
		order.CreatedAt = start.Add(time.Hour)
		order.CompletedAt = timePtr(start.Add(2 * time.Hour))
		order.TotalPriceCents = 2300
	})
	_ = oldOrder
	assertStatsSummary(t, ctx, statsRepo, workspaceID, location.String(), start.Add(-24*time.Hour), end, 2, 4000)

	cutoff := start
	cleared, err := orderRepo.ClearWorkspace(ctx, workspaceID, &cutoff)
	require.NoError(t, err)
	require.Equal(t, 1, cleared)
	assertStatsSummary(t, ctx, statsRepo, workspaceID, location.String(), start.Add(-24*time.Hour), end, 1, 2300)

	cleared, err = orderRepo.ClearWorkspace(ctx, workspaceID, nil)
	require.NoError(t, err)
	require.Equal(t, 1, cleared)
	assertStatsSummary(t, ctx, statsRepo, workspaceID, location.String(), start.Add(-24*time.Hour), end, 0, 0)
	_ = todayOrder
}

func assertStatsSummary(t *testing.T, ctx context.Context, statsRepo *Stats, workspaceID uuid.UUID, timezone string, start, end time.Time, wantCount, wantRevenue int) {
	t.Helper()
	orders, err := statsRepo.CompletedOrders(ctx, workspaceID, start, end)
	require.NoError(t, err)
	require.Len(t, orders, wantCount)

	rows, err := statsRepo.Daily(ctx, workspaceID, timezone, start, end.Add(-time.Nanosecond))
	require.NoError(t, err)
	var dailyCount, dailyRevenue int
	for _, row := range rows {
		dailyCount += row.OrderCount
		dailyRevenue += row.TotalPriceCents
	}
	require.Equal(t, wantCount, dailyCount)
	require.Equal(t, wantRevenue, dailyRevenue)
}

func timePtr(value time.Time) *time.Time {
	return &value
}
