package v1

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/dongwlin/legero-backend/internal/apperr"
	"github.com/dongwlin/legero-backend/internal/domain"
	"github.com/dongwlin/legero-backend/internal/handler/v1/httpresp"
	"github.com/dongwlin/legero-backend/internal/handler/v1/dto"
	"github.com/dongwlin/legero-backend/internal/service"
)

// StatsHandler handles statistics HTTP endpoints.
type StatsHandler struct {
	statsSvc service.Stats
	location *time.Location
}

// NewStatsHandler creates a new StatsHandler.
func NewStatsHandler(statsSvc service.Stats, location *time.Location) *StatsHandler {
	return &StatsHandler{statsSvc: statsSvc, location: location}
}

// Daily returns per-day order counts and revenue for a workspace within a date range.
func (h *StatsHandler) Daily(c *gin.Context) {
	actor, ok := actorFromGin(c)
	if !ok {
		httpresp.AbortError(c, apperr.UnauthorizedError("missing auth context"))
		return
	}

	from, err := time.ParseInLocation("2006-01-02", c.Query("from"), h.location)
	if err != nil {
		httpresp.AbortError(c, apperr.ValidationError("from must use YYYY-MM-DD"))
		return
	}
	to, err := time.ParseInLocation("2006-01-02", c.Query("to"), h.location)
	if err != nil {
		httpresp.AbortError(c, apperr.ValidationError("to must use YYYY-MM-DD"))
		return
	}

	items, err := h.statsSvc.Daily(c.Request.Context(), actor.WorkspaceID, from, to)
	if err != nil {
		httpresp.AbortError(c, err)
		return
	}

	responseItems := make([]dto.DailyItem, 0, len(items))
	for _, item := range items {
		responseItems = append(responseItems, dto.DailyItem{
			Date:            item.Date.In(h.location).Format("2006-01-02"),
			OrderCount:      item.OrderCount,
			TotalPriceCents: item.TotalPriceCents,
		})
	}

	httpresp.JSON(c, http.StatusOK, dto.DailyResponse{
		Items: responseItems,
	})
}

// Report returns the selected period report. M1 supports day; the service
// returns a stable unsupported-period AppError for week/month.
func (h *StatsHandler) Report(c *gin.Context) {
	actor, ok := actorFromGin(c)
	if !ok {
		httpresp.AbortError(c, apperr.UnauthorizedError("missing auth context"))
		return
	}

	period := domain.ReportPeriod(c.Query("period"))
	if !period.Valid() {
		httpresp.AbortError(c, apperr.ValidationError("period must be one of day, week, month"))
		return
	}
	date, err := time.ParseInLocation("2006-01-02", c.Query("date"), h.location)
	if err != nil {
		httpresp.AbortError(c, apperr.ValidationError("date must use YYYY-MM-DD"))
		return
	}

	report, err := h.statsSvc.Report(c.Request.Context(), actor.WorkspaceID, domain.ReportQuery{
		Period: period,
		Date:   date,
	})
	if err != nil {
		httpresp.AbortError(c, err)
		return
	}

	httpresp.JSON(c, http.StatusOK, toReportDTO(*report, h.location))
}

func toReportDTO(report domain.Report, location *time.Location) dto.ReportResponse {
	standard := report.Metrics.StandardSize
	metrics := report.Metrics
	peakBuckets := make([]dto.Peak30Minute, 0, len(metrics.Peak30MinuteBuckets))
	for _, bucket := range metrics.Peak30MinuteBuckets {
		peakBuckets = append(peakBuckets, dto.Peak30Minute{
			Start:      formatPeakMinute(bucket.StartMinute),
			End:        formatPeakMinute(bucket.EndMinute),
			OrderCount: bucket.OrderCount,
		})
	}
	stapleSales := make([]dto.StapleSale, 0, len(metrics.StapleSales))
	for _, sale := range metrics.StapleSales {
		stapleSales = append(stapleSales, dto.StapleSale{
			StapleTypeCode: sale.StapleTypeCode,
			OrderCount:     sale.OrderCount,
		})
	}

	return dto.ReportResponse{
		Period:    string(report.Period),
		StartDate: report.StartAt.In(location).Format("2006-01-02"),
		EndDate:   report.EndAt.In(location).Add(-time.Nanosecond).Format("2006-01-02"),
		Metrics: dto.ReportMetrics{
			RevenueCents:              metrics.RevenueCents,
			CompletedOrderCount:       metrics.CompletedOrderCount,
			AverageOrderValueCents:    metrics.AverageOrderValueCents,
			AveragePreparationSeconds: metrics.AveragePreparationSeconds,
			Peak30MinuteBuckets:       peakBuckets,
			StapleSales:               stapleSales,
			NoStapleOrderCount:        metrics.NoStapleOrderCount,
			UnknownStapleOrderCount:   metrics.UnknownStapleOrderCount,
			StandardSize: dto.StandardSizeMetrics{
				StandardCount:        standard.StandardCount,
				CustomSizeOrderCount: standard.CustomCount,
				Small:                toRatioMetric(standard.Small),
				Medium:               toRatioMetric(standard.Medium),
				Large:                toRatioMetric(standard.Large),
			},
			TotalFriedEggCount: metrics.TotalFriedEggCount,
			Takeout:            toRatioMetric(metrics.Takeout),
			Customizations: dto.CustomizationMetrics{
				LeanMeatOnly: toRatioMetric(metrics.Customizations.LeanMeatOnly),
				NoIntestine:  toRatioMetric(metrics.Customizations.NoIntestine),
				Union:        toRatioMetric(metrics.Customizations.Union),
			},
		},
	}
}

func formatPeakMinute(minute int) string {
	const minutesPerDay = 24 * 60
	minute %= minutesPerDay
	if minute < 0 {
		minute += minutesPerDay
	}
	return fmt.Sprintf("%02d:%02d", minute/60, minute%60)
}

func toRatioMetric(metric domain.RatioMetric) dto.RatioMetric {
	return dto.RatioMetric{
		Count:       metric.Count,
		Denominator: metric.Denominator,
		Ratio:       metric.Ratio,
	}
}
