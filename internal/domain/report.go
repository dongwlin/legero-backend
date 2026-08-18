package domain

import (
	"errors"
	"math"
	"sort"
	"time"
)

// ReportPeriod identifies the calendar period used by a report.  The service
// currently implements day reports; week and month are part of the contract
// so callers can adopt the same request shape before those periods are
// enabled.
type ReportPeriod string

const (
	ReportPeriodDay   ReportPeriod = "day"
	ReportPeriodWeek  ReportPeriod = "week"
	ReportPeriodMonth ReportPeriod = "month"
)

// Valid reports whether p is one of the periods in the report contract.
func (p ReportPeriod) Valid() bool {
	switch p {
	case ReportPeriodDay, ReportPeriodWeek, ReportPeriodMonth:
		return true
	default:
		return false
	}
}

// ErrUnsupportedReportPeriod indicates a period that is known to the
// contract but is not implemented by the current release.
var ErrUnsupportedReportPeriod = errors.New("report period is not supported")

// ReportQuery carries the transport-independent report selector.
type ReportQuery struct {
	Period ReportPeriod
	Date   time.Time
}

// ReportWindow is an absolute half-open time range derived from a business
// calendar date. StartAt and EndAt are in the configured business location;
// callers should pass these values directly to SQL range predicates.
type ReportWindow struct {
	Period  ReportPeriod
	Date    time.Time
	StartAt time.Time
	EndAt   time.Time
}

// NewDayReportWindow returns the business-day window containing date.
func NewDayReportWindow(date time.Time, location *time.Location) ReportWindow {
	if location == nil {
		location = time.UTC
	}
	localDate := date.In(location)
	start := time.Date(localDate.Year(), localDate.Month(), localDate.Day(), 0, 0, 0, 0, location)
	return ReportWindow{
		Period:  ReportPeriodDay,
		Date:    start,
		StartAt: start,
		EndAt:   start.AddDate(0, 0, 1),
	}
}

// NewDailyReportWindow returns one window covering all business dates from
// through to, inclusive. It is shared by the daily endpoint and report
// aggregation so both APIs use completed_at and the same timezone boundaries.
func NewDailyReportWindow(from, to time.Time, location *time.Location) (ReportWindow, error) {
	first := NewDayReportWindow(from, location)
	last := NewDayReportWindow(to, location)
	if last.Date.Before(first.Date) {
		return ReportWindow{}, errors.New("to must be greater than or equal to from")
	}
	first.EndAt = last.EndAt
	return first, nil
}

// ReportOrder is the order projection required by the report metrics.  It is
// deliberately smaller than Order so report queries can remain easy to
// extend without coupling them to the order API model.
type ReportOrder struct {
	TotalPriceCents   int
	StapleTypeCode    *int16
	SizeCode          int16
	FriedEggCount     int16
	DiningMethodCode  int16
	SelectedMeatCodes []int16
	CreatedAt         time.Time
	CompletedAt       *time.Time
}

// Peak30MinuteBucket is the busiest half-hour bucket in the report window.
// StartMinute and EndMinute are wall-clock minutes from the business day's
// midnight. Keeping labels as minutes avoids fabricating instants for
// daylight-saving gaps/folds; the API adapter formats them as HH:mm.
type Peak30MinuteBucket struct {
	StartMinute int
	EndMinute   int
	OrderCount  int
}

// StapleSale is the completed-order count for one staple type code.
type StapleSale struct {
	StapleTypeCode int16
	OrderCount     int
}

// RatioMetric preserves both the numerator and denominator used to derive a
// ratio. This keeps the API useful to clients that need to display exact
// counts alongside the rounded ratio.
type RatioMetric struct {
	Count       int
	Denominator int
	Ratio       float64
}

// StandardSizeMetrics reports the distribution of non-custom sizes. Each
// ratio uses StandardCount as its denominator; custom orders are reported
// separately and do not dilute the standard-size ratios.
type StandardSizeMetrics struct {
	Small         RatioMetric
	Medium        RatioMetric
	Large         RatioMetric
	StandardCount int
	CustomCount   int
}

// CustomizationMetrics contains mutually overlapping user-facing order
// customisation counts. Union counts each order at most once even when it
// matches both LeanMeatOnly and NoIntestine.
type CustomizationMetrics struct {
	LeanMeatOnly RatioMetric
	NoIntestine  RatioMetric
	Union        RatioMetric
}

// ReportMetrics contains the M1 report's ten business metrics.
type ReportMetrics struct {
	RevenueCents              int
	CompletedOrderCount       int
	AverageOrderValueCents    int
	AveragePreparationSeconds int
	Peak30MinuteBuckets       []Peak30MinuteBucket
	StapleSales               []StapleSale
	NoStapleOrderCount        int
	UnknownStapleOrderCount   int
	StandardSize              StandardSizeMetrics
	TotalFriedEggCount        int
	Takeout                   RatioMetric
	Customizations            CustomizationMetrics
}

// Report is the period and metrics returned by the reporting service.
type Report struct {
	Period  ReportPeriod
	Date    time.Time
	StartAt time.Time
	EndAt   time.Time
	Metrics ReportMetrics
}

// AggregateReport computes all report metrics from completed-order rows. It
// rechecks the window in memory as a defensive invariant for alternate repo
// implementations and unit tests; the SQL repo also applies the same range
// predicate so normal production queries remain index friendly.
func AggregateReport(window ReportWindow, orders []ReportOrder, location *time.Location) Report {
	if location == nil {
		location = time.UTC
	}

	stapleCounts := map[int16]int{
		StapleTypeRiceSheet:      0,
		StapleTypeRiceVermicelli: 0,
		StapleTypeYiNoodle:       0,
		StapleTypeRice:           0,
	}
	metrics := ReportMetrics{
		Peak30MinuteBuckets: make([]Peak30MinuteBucket, 0, 1),
		StapleSales:         make([]StapleSale, 0, len(stapleCounts)),
	}

	peakCounts := make([]int, 48)
	validPreparationCount := 0
	validPreparationSeconds := 0.0
	for _, order := range orders {
		if order.CompletedAt == nil || order.CompletedAt.Before(window.StartAt) || !order.CompletedAt.Before(window.EndAt) {
			continue
		}

		metrics.CompletedOrderCount++
		metrics.RevenueCents += order.TotalPriceCents
		metrics.TotalFriedEggCount += int(order.FriedEggCount)
		if order.DiningMethodCode == DiningMethodTakeout {
			metrics.Takeout.Count++
		}
		switch order.SizeCode {
		case SizeCustom:
			metrics.StandardSize.CustomCount++
		case SizeSmall, SizeMedium, SizeLarge:
			metrics.StandardSize.StandardCount++
			switch order.SizeCode {
			case SizeSmall:
				metrics.StandardSize.Small.Count++
			case SizeMedium:
				metrics.StandardSize.Medium.Count++
			case SizeLarge:
				metrics.StandardSize.Large.Count++
			}
		}

		if order.StapleTypeCode != nil {
			if _, known := stapleCounts[*order.StapleTypeCode]; known {
				stapleCounts[*order.StapleTypeCode]++
			} else {
				metrics.UnknownStapleOrderCount++
			}
		} else {
			metrics.NoStapleOrderCount++
		}

		leanMeatOnly, noIntestine := customizationFlags(order.SelectedMeatCodes)
		if leanMeatOnly {
			metrics.Customizations.LeanMeatOnly.Count++
		}
		if noIntestine {
			metrics.Customizations.NoIntestine.Count++
		}
		if leanMeatOnly || noIntestine {
			metrics.Customizations.Union.Count++
		}

		completedAt := order.CompletedAt.In(location)
		minutes := completedAt.Hour()*60 + completedAt.Minute()
		bucket := minutes / 30
		if bucket >= 0 && bucket < len(peakCounts) {
			peakCounts[bucket]++
		}

		if !order.CreatedAt.IsZero() && !order.CompletedAt.Before(order.CreatedAt) {
			validPreparationCount++
			validPreparationSeconds += order.CompletedAt.Sub(order.CreatedAt).Seconds()
		}
	}

	if metrics.CompletedOrderCount > 0 {
		metrics.AverageOrderValueCents = int(math.Round(float64(metrics.RevenueCents) / float64(metrics.CompletedOrderCount)))
	}
	metrics.Takeout.Denominator = metrics.CompletedOrderCount
	metrics.Takeout.Ratio = ratio(metrics.Takeout.Count, metrics.Takeout.Denominator)
	metrics.StandardSize.Small.Denominator = metrics.StandardSize.StandardCount
	metrics.StandardSize.Medium.Denominator = metrics.StandardSize.StandardCount
	metrics.StandardSize.Large.Denominator = metrics.StandardSize.StandardCount
	metrics.StandardSize.Small.Ratio = ratio(metrics.StandardSize.Small.Count, metrics.StandardSize.Small.Denominator)
	metrics.StandardSize.Medium.Ratio = ratio(metrics.StandardSize.Medium.Count, metrics.StandardSize.Medium.Denominator)
	metrics.StandardSize.Large.Ratio = ratio(metrics.StandardSize.Large.Count, metrics.StandardSize.Large.Denominator)
	metrics.Customizations.LeanMeatOnly.Denominator = metrics.CompletedOrderCount
	metrics.Customizations.NoIntestine.Denominator = metrics.CompletedOrderCount
	metrics.Customizations.Union.Denominator = metrics.CompletedOrderCount
	metrics.Customizations.LeanMeatOnly.Ratio = ratio(metrics.Customizations.LeanMeatOnly.Count, metrics.CompletedOrderCount)
	metrics.Customizations.NoIntestine.Ratio = ratio(metrics.Customizations.NoIntestine.Count, metrics.CompletedOrderCount)
	metrics.Customizations.Union.Ratio = ratio(metrics.Customizations.Union.Count, metrics.CompletedOrderCount)
	if validPreparationCount > 0 {
		metrics.AveragePreparationSeconds = int(math.Round(validPreparationSeconds / float64(validPreparationCount)))
	}

	peakIndex := 0
	for idx := 1; idx < len(peakCounts); idx++ {
		// Strictly greater preserves the earliest bucket on ties.
		if peakCounts[idx] > peakCounts[peakIndex] {
			peakIndex = idx
		}
	}
	if metrics.CompletedOrderCount > 0 {
		metrics.Peak30MinuteBuckets = append(metrics.Peak30MinuteBuckets, Peak30MinuteBucket{
			StartMinute: peakIndex * 30,
			EndMinute:   (peakIndex + 1) * 30,
			OrderCount:  peakCounts[peakIndex],
		})
	}

	codes := make([]int16, 0, len(stapleCounts))
	for code := range stapleCounts {
		codes = append(codes, code)
	}
	sort.Slice(codes, func(i, j int) bool { return codes[i] < codes[j] })
	for _, code := range codes {
		metrics.StapleSales = append(metrics.StapleSales, StapleSale{
			StapleTypeCode: code,
			OrderCount:     stapleCounts[code],
		})
	}

	return Report{
		Period:  window.Period,
		Date:    window.Date,
		StartAt: window.StartAt,
		EndAt:   window.EndAt,
		Metrics: metrics,
	}
}

func ratio(count, denominator int) float64 {
	if denominator <= 0 {
		return 0
	}
	return float64(count) / float64(denominator)
}

func customizationFlags(selected []int16) (leanMeatOnly, noIntestine bool) {
	leanMeatOnly = len(selected) == 1 && selected[0] == MeatLeanPork
	for _, code := range selected {
		if code == MeatLargeIntestine || code == MeatSmallIntestine {
			return leanMeatOnly, false
		}
	}
	return leanMeatOnly, true
}
