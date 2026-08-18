package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestAggregateReportUsesCompletedAtWindowAndCalculatesMetrics(t *testing.T) {
	location := time.FixedZone("CST", 8*60*60)
	window := NewDayReportWindow(time.Date(2026, 8, 18, 12, 0, 0, 0, location), location)
	createdAt := window.StartAt.Add(10 * time.Minute)
	completedAt := window.StartAt.Add(9 * time.Hour)
	completedAtLater := window.StartAt.Add(9*time.Hour + 20*time.Minute)
	completedAtInvalid := window.StartAt.Add(8 * time.Hour)
	unknownStaple := int16(99)
	orders := []ReportOrder{
		{
			TotalPriceCents:   1001,
			StapleTypeCode:    ptrInt16(StapleTypeRiceSheet),
			SizeCode:          SizeSmall,
			FriedEggCount:     2,
			DiningMethodCode:  DiningMethodTakeout,
			SelectedMeatCodes: []int16{MeatLeanPork},
			CreatedAt:         createdAt,
			CompletedAt:       &completedAt,
		},
		{
			TotalPriceCents:   1000,
			StapleTypeCode:    ptrInt16(StapleTypeRice),
			SizeCode:          SizeMedium,
			DiningMethodCode:  DiningMethodDineIn,
			SelectedMeatCodes: []int16{MeatLargeIntestine},
			CreatedAt:         createdAt,
			CompletedAt:       &completedAtLater,
		},
		{
			TotalPriceCents:  900,
			StapleTypeCode:   &unknownStaple,
			SizeCode:         SizeCustom,
			DiningMethodCode: DiningMethodDineIn,
			CreatedAt:        completedAt,
			CompletedAt:      &completedAtInvalid,
		},
		{
			// This order is outside the report by completed_at, even though its
			// created_at is within the business day.
			TotalPriceCents: 5000,
			SizeCode:        SizeLarge,
			CreatedAt:       createdAt,
			CompletedAt:     timePtr(window.EndAt),
		},
		{
			// A nil completed_at order must never enter any report metric.
			TotalPriceCents: 7000,
			SizeCode:        SizeLarge,
		},
	}

	report := AggregateReport(window, orders, location)
	metrics := report.Metrics
	require.Equal(t, 3, metrics.CompletedOrderCount)
	require.Equal(t, 2901, metrics.RevenueCents)
	require.Equal(t, 967, metrics.AverageOrderValueCents)
	require.Equal(t, 32400, metrics.AveragePreparationSeconds)
	require.Len(t, metrics.Peak30MinuteBuckets, 1)
	require.Equal(t, 9*60, metrics.Peak30MinuteBuckets[0].StartMinute)
	require.Equal(t, 9*60+30, metrics.Peak30MinuteBuckets[0].EndMinute)
	require.Equal(t, 2, metrics.Peak30MinuteBuckets[0].OrderCount)
	require.Equal(t, 1, metrics.Takeout.Count)
	require.Equal(t, 3, metrics.Takeout.Denominator)
	require.InDelta(t, 1.0/3.0, metrics.Takeout.Ratio, 1e-9)
	require.Equal(t, 2, metrics.StandardSize.StandardCount)
	require.Equal(t, 1, metrics.StandardSize.CustomCount)
	require.Equal(t, 1, metrics.StandardSize.Small.Count)
	require.Equal(t, 1, metrics.StandardSize.Medium.Count)
	require.InDelta(t, 0.5, metrics.StandardSize.Small.Ratio, 1e-9)
	require.Equal(t, 2, metrics.TotalFriedEggCount)
	require.Equal(t, 0, metrics.NoStapleOrderCount)
	require.Equal(t, 1, metrics.UnknownStapleOrderCount)
	require.Equal(t, 1, metrics.Customizations.LeanMeatOnly.Count)
	require.Equal(t, 2, metrics.Customizations.NoIntestine.Count)
	require.Equal(t, 2, metrics.Customizations.Union.Count)

	require.Equal(t, []StapleSale{
		{StapleTypeCode: StapleTypeRiceSheet, OrderCount: 1},
		{StapleTypeCode: StapleTypeRiceVermicelli, OrderCount: 0},
		{StapleTypeCode: StapleTypeYiNoodle, OrderCount: 0},
		{StapleTypeCode: StapleTypeRice, OrderCount: 1},
	}, metrics.StapleSales)
}

func TestAggregateReportZeroOrders(t *testing.T) {
	location := time.FixedZone("CST", 8*60*60)
	window := NewDayReportWindow(time.Date(2026, 8, 18, 0, 0, 0, 0, location), location)
	report := AggregateReport(window, nil, location)
	metrics := report.Metrics
	require.Equal(t, 0, metrics.CompletedOrderCount)
	require.Equal(t, 0, metrics.AverageOrderValueCents)
	require.Equal(t, 0, metrics.AveragePreparationSeconds)
	require.Empty(t, metrics.Peak30MinuteBuckets)
	require.Equal(t, 0, metrics.Takeout.Denominator)
	require.Equal(t, 0, metrics.StandardSize.Small.Denominator)
	require.Equal(t, 0, metrics.Customizations.Union.Denominator)
}

func TestAggregateReportPeak30MinuteBoundariesAndEarliestTie(t *testing.T) {
	location := time.FixedZone("CST", 8*60*60)
	window := NewDayReportWindow(time.Date(2026, 8, 18, 0, 0, 0, 0, location), location)

	tests := []struct {
		name       string
		completed  []time.Duration
		wantStart  string
		wantEnd    string
		wantOrders int
	}{
		{
			name:       "09:29 remains in the 09:00 bucket",
			completed:  []time.Duration{9*time.Hour + 29*time.Minute},
			wantStart:  "09:00",
			wantEnd:    "09:30",
			wantOrders: 1,
		},
		{
			name:       "09:30 starts the next bucket",
			completed:  []time.Duration{9*time.Hour + 30*time.Minute},
			wantStart:  "09:30",
			wantEnd:    "10:00",
			wantOrders: 1,
		},
		{
			name:       "ties retain the earlier bucket",
			completed:  []time.Duration{9*time.Hour + 5*time.Minute, 9*time.Hour + 35*time.Minute},
			wantStart:  "09:00",
			wantEnd:    "09:30",
			wantOrders: 1,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			orders := make([]ReportOrder, 0, len(test.completed))
			for _, offset := range test.completed {
				completedAt := window.StartAt.Add(offset)
				createdAt := completedAt.Add(-time.Minute)
				orders = append(orders, ReportOrder{
					CreatedAt:   createdAt,
					CompletedAt: &completedAt,
				})
			}

			report := AggregateReport(window, orders, location)
			require.Len(t, report.Metrics.Peak30MinuteBuckets, 1)
			peak := report.Metrics.Peak30MinuteBuckets[0]
			require.Equal(t, parseClockMinute(test.wantStart), peak.StartMinute)
			require.Equal(t, parseClockMinute(test.wantEnd), peak.EndMinute)
			require.Equal(t, test.wantOrders, peak.OrderCount)
		})
	}
}

func TestAggregateReportPeakUsesWallClockMinutesAcrossDST(t *testing.T) {
	location, err := time.LoadLocation("America/New_York")
	require.NoError(t, err)
	fallFirst, err := time.Parse(time.RFC3339, "2026-11-01T01:40:00-04:00")
	require.NoError(t, err)
	fallSecond, err := time.Parse(time.RFC3339, "2026-11-01T01:40:00-05:00")
	require.NoError(t, err)
	require.Equal(t, "01:40", fallFirst.In(location).Format("15:04"))
	require.Equal(t, "01:40", fallSecond.In(location).Format("15:04"))

	tests := []struct {
		name           string
		date           time.Time
		completedAt    []time.Time
		wantStart      int
		wantEnd        int
		wantOrderCount int
	}{
		{
			name: "spring forward 03:10",
			date: time.Date(2026, 3, 8, 12, 0, 0, 0, location),
			completedAt: []time.Time{
				time.Date(2026, 3, 8, 3, 10, 0, 0, location),
			},
			wantStart:      3 * 60,
			wantEnd:        3*60 + 30,
			wantOrderCount: 1,
		},
		{
			name: "spring forward 01:40 nominal end 02:00",
			date: time.Date(2026, 3, 8, 12, 0, 0, 0, location),
			completedAt: []time.Time{
				time.Date(2026, 3, 8, 1, 40, 0, 0, location),
			},
			wantStart:      90,
			wantEnd:        120,
			wantOrderCount: 1,
		},
		{
			name:           "fall back repeated 01:40",
			date:           time.Date(2026, 11, 1, 12, 0, 0, 0, location),
			completedAt:    []time.Time{fallFirst, fallSecond},
			wantStart:      90,
			wantEnd:        120,
			wantOrderCount: 2,
		},
		{
			name: "last bucket ends at next midnight",
			date: time.Date(2026, 3, 8, 12, 0, 0, 0, location),
			completedAt: []time.Time{
				time.Date(2026, 3, 8, 23, 40, 0, 0, location),
			},
			wantStart:      23*60 + 30,
			wantEnd:        24 * 60,
			wantOrderCount: 1,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			window := NewDayReportWindow(test.date, location)
			orders := make([]ReportOrder, 0, len(test.completedAt))
			for _, completedAt := range test.completedAt {
				completedAt := completedAt
				orders = append(orders, ReportOrder{
					CreatedAt:   completedAt.Add(-time.Minute),
					CompletedAt: &completedAt,
				})
			}

			report := AggregateReport(window, orders, location)
			require.Len(t, report.Metrics.Peak30MinuteBuckets, 1)
			peak := report.Metrics.Peak30MinuteBuckets[0]
			require.Equal(t, test.wantStart, peak.StartMinute)
			require.Equal(t, test.wantEnd, peak.EndMinute)
			require.Equal(t, test.wantOrderCount, peak.OrderCount)
		})
	}
}

func TestNewDailyReportWindowPreservesBusinessDateRange(t *testing.T) {
	location := time.FixedZone("CST", 8*60*60)
	from := time.Date(2026, 8, 18, 23, 55, 0, 0, time.UTC)
	to := time.Date(2026, 8, 20, 0, 5, 0, 0, time.UTC)
	window, err := NewDailyReportWindow(from, to, location)
	require.NoError(t, err)
	require.Equal(t, "2026-08-19", window.StartAt.Format("2006-01-02"))
	require.Equal(t, "2026-08-21", window.EndAt.Format("2006-01-02"))
}

func ptrInt16(value int16) *int16 {
	return &value
}

func timePtr(value time.Time) *time.Time {
	return &value
}

func parseClockMinute(value string) int {
	return int(value[0]-'0')*600 + int(value[1]-'0')*60 +
		int(value[3]-'0')*10 + int(value[4]-'0')
}
