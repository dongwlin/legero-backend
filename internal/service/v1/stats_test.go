package v1

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/dongwlin/legero-backend/internal/apperr"
	"github.com/dongwlin/legero-backend/internal/domain"
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

	otherReport, err := svc.Report(ctx, otherWorkspaceID, domain.ReportQuery{
		Period: domain.ReportPeriodDay,
		Date:   dayStart,
	})
	require.NoError(t, err)
	require.Equal(t, 1, otherReport.Metrics.CompletedOrderCount)
	require.Equal(t, 2200, otherReport.Metrics.RevenueCents)
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
