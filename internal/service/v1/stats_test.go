package v1

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/dongwlin/legero-backend/internal/apperr"
	"github.com/dongwlin/legero-backend/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestStatsService_DailyAndReportShareCompletedRangeAndWorkspace(t *testing.T) {
	ctx := context.Background()
	location, err := time.LoadLocation("Asia/Shanghai")
	require.NoError(t, err)

	svc := NewStats(testDB, location.String())
	userID := createTestUser(t, ctx, testDB)
	workspaceID := createTestWorkspace(t, ctx, testDB)
	otherWorkspaceID := createTestWorkspace(t, ctx, testDB)

	dayStart := time.Date(2026, 8, 18, 0, 0, 0, 0, location)
	createTestOrder(t, ctx, testDB, workspaceID, userID, func(order *domain.Order) {
		order.DisplayNo = "COMPLETES-AFTER-MIDNIGHT"
		order.CreatedAt = time.Date(2026, 8, 17, 23, 55, 0, 0, location)
		order.CompletedAt = statsTimePtr(dayStart.Add(5 * time.Minute))
		order.TotalPriceCents = 1100
	})
	createTestOrder(t, ctx, testDB, workspaceID, userID, func(order *domain.Order) {
		order.DisplayNo = "UNCOMPLETED"
		order.CreatedAt = dayStart.Add(time.Hour)
		order.CompletedAt = nil
		order.TotalPriceCents = 9900
	})
	createTestOrder(t, ctx, testDB, otherWorkspaceID, userID, func(order *domain.Order) {
		order.DisplayNo = "OTHER-WORKSPACE"
		order.CompletedAt = statsTimePtr(dayStart.Add(2 * time.Hour))
		order.TotalPriceCents = 2200
	})

	daily, err := svc.Daily(
		ctx,
		workspaceID,
		time.Date(2026, 8, 17, 12, 0, 0, 0, location),
		time.Date(2026, 8, 18, 12, 0, 0, 0, location),
	)
	require.NoError(t, err)
	require.Equal(t, []domain.DailyRow{
		{Date: dayStart, OrderCount: 1, TotalPriceCents: 1100},
		{Date: dayStart.AddDate(0, 0, -1), OrderCount: 0, TotalPriceCents: 0},
	}, daily)

	report, err := svc.Report(ctx, workspaceID, domain.ReportQuery{
		Period: domain.ReportPeriodDay,
		Date:   dayStart.Add(13 * time.Hour),
	})
	require.NoError(t, err)
	require.Equal(t, domain.ReportPeriodDay, report.Period)
	require.Equal(t, dayStart, report.StartAt)
	require.Equal(t, dayStart.AddDate(0, 0, 1), report.EndAt)
	require.Equal(t, daily[0].OrderCount, report.Metrics.CompletedOrderCount)
	require.Equal(t, daily[0].TotalPriceCents, report.Metrics.RevenueCents)
	require.Equal(t, 1100, report.Metrics.AverageOrderValueCents)
	require.Equal(t, []domain.Peak30MinuteBucket{
		{
			StartMinute: 23*60 + 30,
			EndMinute:   24 * 60,
			OrderCount:  1,
		},
	}, report.Metrics.Peak30MinuteBuckets)

	otherReport, err := svc.Report(ctx, otherWorkspaceID, domain.ReportQuery{
		Period: domain.ReportPeriodDay,
		Date:   dayStart,
	})
	require.NoError(t, err)
	require.Equal(t, 1, otherReport.Metrics.CompletedOrderCount)
	require.Equal(t, 2200, otherReport.Metrics.RevenueCents)
}

// TestStatsService_ReportCoversAllM1Metrics exercises the complete
// PostgreSQL -> repo projection -> AggregateReport path with deliberately
// different orders. Keeping this at the service boundary catches projection
// regressions that a domain-only test cannot see (notably smallint[] meat
// codes and the scalar report fields).
func TestStatsService_ReportCoversAllM1Metrics(t *testing.T) {
	ctx := context.Background()
	location, err := time.LoadLocation("Asia/Shanghai")
	require.NoError(t, err)

	svc := NewStats(testDB, location.String())
	userID := createTestUser(t, ctx, testDB)
	workspaceID := createTestWorkspace(t, ctx, testDB)
	otherWorkspaceID := createTestWorkspace(t, ctx, testDB)
	start := time.Date(2026, 8, 18, 0, 0, 0, 0, location)
	createdAt := start.Add(8 * time.Hour)
	unknownStaple := int16(99)

	createTestOrder(t, ctx, testDB, workspaceID, userID, func(order *domain.Order) {
		order.DisplayNo = "REPORT-SMALL-TAKEOUT"
		order.StapleTypeCode = statsInt16Ptr(domain.StapleTypeRiceSheet)
		order.SizeCode = domain.SizeSmall
		order.FriedEggCount = 2
		order.DiningMethodCode = domain.DiningMethodTakeout
		order.SelectedMeatCodes = []int16{domain.MeatLeanPork}
		order.CreatedAt = createdAt
		order.CompletedAt = statsTimePtr(start.Add(9*time.Hour + 29*time.Minute))
		order.TotalPriceCents = 1001
	})
	createTestOrder(t, ctx, testDB, workspaceID, userID, func(order *domain.Order) {
		order.DisplayNo = "REPORT-MEDIUM-DINE-IN"
		order.StapleTypeCode = statsInt16Ptr(domain.StapleTypeRiceVermicelli)
		order.SizeCode = domain.SizeMedium
		order.FriedEggCount = 1
		order.DiningMethodCode = domain.DiningMethodDineIn
		order.SelectedMeatCodes = []int16{domain.MeatLeanPork, domain.MeatLiver}
		order.CreatedAt = createdAt
		order.CompletedAt = statsTimePtr(start.Add(9*time.Hour + 30*time.Minute))
		order.TotalPriceCents = 1000
	})
	createTestOrder(t, ctx, testDB, workspaceID, userID, func(order *domain.Order) {
		order.DisplayNo = "REPORT-LARGE-INTST-TAKEOUT"
		order.StapleTypeCode = nil
		order.SizeCode = domain.SizeLarge
		order.FriedEggCount = 0
		order.DiningMethodCode = domain.DiningMethodTakeout
		order.SelectedMeatCodes = []int16{domain.MeatLargeIntestine}
		order.CreatedAt = createdAt
		order.CompletedAt = statsTimePtr(start.Add(9*time.Hour + 45*time.Minute))
		order.TotalPriceCents = 900
	})
	createTestOrder(t, ctx, testDB, workspaceID, userID, func(order *domain.Order) {
		order.DisplayNo = "REPORT-CUSTOM-UNKNOWN"
		order.StapleTypeCode = &unknownStaple
		order.SizeCode = domain.SizeCustom
		order.FriedEggCount = 3
		order.DiningMethodCode = domain.DiningMethodDineIn
		order.SelectedMeatCodes = []int16{}
		order.CreatedAt = createdAt
		order.CompletedAt = statsTimePtr(start.Add(10 * time.Hour))
		order.TotalPriceCents = 1100
	})

	// A same-day order in another workspace must not affect any metric.
	createTestOrder(t, ctx, testDB, otherWorkspaceID, userID, func(order *domain.Order) {
		order.DisplayNo = "REPORT-OTHER-WORKSPACE"
		order.CompletedAt = statsTimePtr(start.Add(9 * time.Hour))
		order.TotalPriceCents = 99999
	})

	report, err := svc.Report(ctx, workspaceID, domain.ReportQuery{
		Period: domain.ReportPeriodDay,
		Date:   start.Add(12 * time.Hour),
	})
	require.NoError(t, err)
	require.Equal(t, domain.ReportPeriodDay, report.Period)
	require.Equal(t, start, report.StartAt)
	require.Equal(t, start.AddDate(0, 0, 1), report.EndAt)

	metrics := report.Metrics
	require.Equal(t, 4, metrics.CompletedOrderCount)
	require.Equal(t, 4001, metrics.RevenueCents)
	require.Equal(t, 1000, metrics.AverageOrderValueCents)
	require.Equal(t, 6060, metrics.AveragePreparationSeconds)
	require.Equal(t, 6, metrics.TotalFriedEggCount)

	require.Equal(t, []domain.Peak30MinuteBucket{
		{
			StartMinute: 8 * 60,
			EndMinute:   8*60 + 30,
			OrderCount:  4,
		},
	}, metrics.Peak30MinuteBuckets)
	require.Equal(t, []domain.StapleSale{
		{StapleTypeCode: domain.StapleTypeRiceSheet, OrderCount: 1},
		{StapleTypeCode: domain.StapleTypeRiceVermicelli, OrderCount: 1},
		{StapleTypeCode: domain.StapleTypeYiNoodle, OrderCount: 0},
		{StapleTypeCode: domain.StapleTypeRice, OrderCount: 0},
	}, metrics.StapleSales)
	require.Equal(t, 1, metrics.NoStapleOrderCount)
	require.Equal(t, 1, metrics.UnknownStapleOrderCount)

	require.Equal(t, 3, metrics.StandardSize.StandardCount)
	require.Equal(t, 1, metrics.StandardSize.CustomCount)
	require.Equal(t, 1, metrics.StandardSize.Small.Count)
	require.Equal(t, 1, metrics.StandardSize.Medium.Count)
	require.Equal(t, 1, metrics.StandardSize.Large.Count)
	require.Equal(t, 3, metrics.StandardSize.Small.Denominator)
	require.Equal(t, 3, metrics.StandardSize.Medium.Denominator)
	require.Equal(t, 3, metrics.StandardSize.Large.Denominator)
	require.InDelta(t, 1.0/3.0, metrics.StandardSize.Small.Ratio, 1e-9)
	require.InDelta(t, 1.0/3.0, metrics.StandardSize.Medium.Ratio, 1e-9)
	require.InDelta(t, 1.0/3.0, metrics.StandardSize.Large.Ratio, 1e-9)

	require.Equal(t, 2, metrics.Takeout.Count)
	require.Equal(t, 4, metrics.Takeout.Denominator)
	require.InDelta(t, 0.5, metrics.Takeout.Ratio, 1e-9)
	require.Equal(t, 1, metrics.Customizations.LeanMeatOnly.Count)
	require.Equal(t, 3, metrics.Customizations.NoIntestine.Count)
	require.Equal(t, 3, metrics.Customizations.Union.Count)
	require.Equal(t, 4, metrics.Customizations.LeanMeatOnly.Denominator)
	require.Equal(t, 4, metrics.Customizations.NoIntestine.Denominator)
	require.Equal(t, 4, metrics.Customizations.Union.Denominator)
	require.InDelta(t, 0.25, metrics.Customizations.LeanMeatOnly.Ratio, 1e-9)
	require.InDelta(t, 0.75, metrics.Customizations.NoIntestine.Ratio, 1e-9)
	require.InDelta(t, 0.75, metrics.Customizations.Union.Ratio, 1e-9)
}

// TestStatsService_ReportTracksOrderMutations verifies that the report reads
// current completion state after each order mutation, including the clear
// modes that delete rows based on created_at.
func TestStatsService_ReportTracksOrderMutations(t *testing.T) {
	ctx := context.Background()
	location := time.UTC
	now := time.Now().In(location)
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, location)

	statsSvc := NewStats(testDB, location.String())
	orderSvc := NewOrder(testDB, location, nil)
	userID := createTestUser(t, ctx, testDB)
	workspaceID := createTestWorkspace(t, ctx, testDB)
	actor := ownerActor(userID, workspaceID)

	order := createTestOrder(t, ctx, testDB, workspaceID, userID, func(order *domain.Order) {
		order.DisplayNo = "REPORT-MUTABLE"
		order.CreatedAt = dayStart.Add(-time.Hour)
		order.CompletedAt = nil
		order.MeatStepStatusCode = domain.StepStatusCompleted
		order.TotalPriceCents = 1600
	})
	assertServiceReportSummary(t, ctx, statsSvc, workspaceID, dayStart, 0, 0)

	_, err := orderSvc.ToggleServed(ctx, actor, order.ID, domain.ToggleServedInput{})
	require.NoError(t, err)
	assertServiceReportSummary(t, ctx, statsSvc, workspaceID, dayStart, 1, 1600)

	_, err = orderSvc.ToggleServed(ctx, actor, order.ID, domain.ToggleServedInput{})
	require.NoError(t, err)
	assertServiceReportSummary(t, ctx, statsSvc, workspaceID, dayStart, 0, 0)

	_, err = orderSvc.ToggleServed(ctx, actor, order.ID, domain.ToggleServedInput{})
	require.NoError(t, err)
	assertServiceReportSummary(t, ctx, statsSvc, workspaceID, dayStart, 1, 1600)

	require.NoError(t, orderSvc.Remove(ctx, actor, order.ID))
	assertServiceReportSummary(t, ctx, statsSvc, workspaceID, dayStart, 0, 0)

	// before_today removes an old-created row even when it was completed in
	// today's report window, so this directly proves the report changes after
	// the service-level clear mutation.
	beforeToday := createTestOrder(t, ctx, testDB, workspaceID, userID, func(order *domain.Order) {
		order.DisplayNo = "REPORT-BEFORE-TODAY"
		order.CreatedAt = dayStart.Add(-time.Hour)
		order.CompletedAt = statsTimePtr(dayStart.Add(time.Hour))
		order.TotalPriceCents = 1700
	})
	_ = beforeToday
	assertServiceReportSummary(t, ctx, statsSvc, workspaceID, dayStart, 1, 1700)

	cleared, err := orderSvc.ClearWorkspace(ctx, actor, true, domain.ClearWorkspaceModeBeforeToday)
	require.NoError(t, err)
	require.Equal(t, 1, cleared)
	assertServiceReportSummary(t, ctx, statsSvc, workspaceID, dayStart, 0, 0)

	createTestOrder(t, ctx, testDB, workspaceID, userID, func(order *domain.Order) {
		order.DisplayNo = "REPORT-CLEAR-ALL"
		order.CreatedAt = dayStart.Add(time.Hour)
		order.CompletedAt = statsTimePtr(dayStart.Add(2 * time.Hour))
		order.TotalPriceCents = 1800
	})
	assertServiceReportSummary(t, ctx, statsSvc, workspaceID, dayStart, 1, 1800)

	cleared, err = orderSvc.ClearWorkspace(ctx, actor, true, domain.ClearWorkspaceModeAll)
	require.NoError(t, err)
	require.Equal(t, 1, cleared)
	assertServiceReportSummary(t, ctx, statsSvc, workspaceID, dayStart, 0, 0)
}

func assertServiceReportSummary(t *testing.T, ctx context.Context, statsSvc interface {
	Report(context.Context, uuid.UUID, domain.ReportQuery) (*domain.Report, error)
}, workspaceID uuid.UUID, dayStart time.Time, wantCount, wantRevenue int) {
	t.Helper()
	report, err := statsSvc.Report(ctx, workspaceID, domain.ReportQuery{
		Period: domain.ReportPeriodDay,
		Date:   dayStart,
	})
	require.NoError(t, err)
	require.Equal(t, wantCount, report.Metrics.CompletedOrderCount)
	require.Equal(t, wantRevenue, report.Metrics.RevenueCents)
}

func TestStatsService_ReportZeroOrdersHasStableZeroMetrics(t *testing.T) {
	ctx := context.Background()
	location, err := time.LoadLocation("Asia/Shanghai")
	require.NoError(t, err)

	svc := NewStats(testDB, location.String())
	workspaceID := createTestWorkspace(t, ctx, testDB)
	date := time.Date(2026, 8, 19, 12, 0, 0, 0, location)

	report, err := svc.Report(ctx, workspaceID, domain.ReportQuery{
		Period: domain.ReportPeriodDay,
		Date:   date,
	})
	require.NoError(t, err)
	metrics := report.Metrics
	require.Equal(t, 0, metrics.CompletedOrderCount)
	require.Equal(t, 0, metrics.RevenueCents)
	require.Equal(t, 0, metrics.AverageOrderValueCents)
	require.Equal(t, 0, metrics.AveragePreparationSeconds)
	require.Equal(t, 0, metrics.TotalFriedEggCount)
	require.Empty(t, metrics.Peak30MinuteBuckets)
	require.Len(t, metrics.StapleSales, 4)
	for _, sale := range metrics.StapleSales {
		require.Equal(t, 0, sale.OrderCount)
	}
	require.Equal(t, 0, metrics.NoStapleOrderCount)
	require.Equal(t, 0, metrics.UnknownStapleOrderCount)
	require.Equal(t, 0, metrics.StandardSize.StandardCount)
	require.Equal(t, 0, metrics.StandardSize.CustomCount)
	require.Zero(t, metrics.Takeout)
	require.Zero(t, metrics.Customizations.LeanMeatOnly)
	require.Zero(t, metrics.Customizations.NoIntestine)
	require.Zero(t, metrics.Customizations.Union)
	for _, ratio := range []float64{
		metrics.Takeout.Ratio,
		metrics.StandardSize.Small.Ratio,
		metrics.StandardSize.Medium.Ratio,
		metrics.StandardSize.Large.Ratio,
		metrics.Customizations.LeanMeatOnly.Ratio,
		metrics.Customizations.NoIntestine.Ratio,
		metrics.Customizations.Union.Ratio,
	} {
		require.False(t, math.IsNaN(ratio))
		require.Equal(t, float64(0), ratio)
	}
}

func statsTimePtr(value time.Time) *time.Time {
	return &value
}

func statsInt16Ptr(value int16) *int16 {
	return &value
}

func TestStatsService_ReportKeepsWeekAndMonthUnsupported(t *testing.T) {
	ctx := context.Background()
	location, err := time.LoadLocation("Asia/Shanghai")
	require.NoError(t, err)

	svc := NewStats(testDB, location.String())
	workspaceID := createTestWorkspace(t, ctx, testDB)
	date := time.Date(2026, 8, 18, 0, 0, 0, 0, location)

	for _, period := range []domain.ReportPeriod{domain.ReportPeriodWeek, domain.ReportPeriodMonth} {
		period := period
		t.Run(string(period), func(t *testing.T) {
			_, err := svc.Report(ctx, workspaceID, domain.ReportQuery{
				Period: period,
				Date:   date,
			})
			var appErr *apperr.AppError
			require.ErrorAs(t, err, &appErr)
			require.Equal(t, apperr.KindInvalidArgument, appErr.Kind)
			require.Equal(t, "report_period_unsupported", appErr.Code)
		})
	}
}
