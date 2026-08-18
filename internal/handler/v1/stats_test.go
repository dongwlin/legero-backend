package v1

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/dongwlin/legero-backend/internal/apperr"
	"github.com/dongwlin/legero-backend/internal/domain"
	"github.com/dongwlin/legero-backend/internal/handler/v1/dto"
	"github.com/dongwlin/legero-backend/internal/infra/identity"
)

type statsServiceStub struct {
	reportFn func(context.Context, uuid.UUID, domain.ReportQuery) (*domain.Report, error)
}

func (s statsServiceStub) Daily(context.Context, uuid.UUID, time.Time, time.Time) ([]domain.DailyRow, error) {
	return nil, nil
}

func (s statsServiceStub) Report(ctx context.Context, workspaceID uuid.UUID, query domain.ReportQuery) (*domain.Report, error) {
	return s.reportFn(ctx, workspaceID, query)
}

func TestToReportDTOUsesInclusiveBusinessDatesAndStableEmptyArrays(t *testing.T) {
	location := time.FixedZone("CST", 8*60*60)
	window := domain.NewDayReportWindow(time.Date(2026, 8, 18, 13, 0, 0, 0, location), location)
	report := domain.AggregateReport(window, nil, location)

	dtoReport := toReportDTO(report, location)
	require.Equal(t, "day", dtoReport.Period)
	require.Equal(t, "2026-08-18", dtoReport.StartDate)
	require.Equal(t, "2026-08-18", dtoReport.EndDate)
	require.NotNil(t, dtoReport.Metrics.Peak30MinuteBuckets)
	require.Empty(t, dtoReport.Metrics.Peak30MinuteBuckets)
	require.Len(t, dtoReport.Metrics.StapleSales, 4)
	require.Equal(t, 0, dtoReport.Metrics.StandardSize.Small.Denominator)
}

func TestFormatPeakMinuteKeepsWallClockLabels(t *testing.T) {
	for _, test := range []struct {
		minute int
		want   string
	}{
		{minute: 0, want: "00:00"},
		{minute: 90, want: "01:30"},
		{minute: 120, want: "02:00"},
		{minute: 1440, want: "00:00"},
	} {
		t.Run(test.want, func(t *testing.T) {
			require.Equal(t, test.want, formatPeakMinute(test.minute))
		})
	}
}

func TestToReportDTOFormatsRankedPeakBucketsAsAPIJSON(t *testing.T) {
	location := time.FixedZone("CST", 8*60*60)
	window := domain.NewDayReportWindow(time.Date(2026, 8, 18, 12, 0, 0, 0, location), location)
	report := domain.Report{
		Period:  window.Period,
		Date:    window.Date,
		StartAt: window.StartAt,
		EndAt:   window.EndAt,
		Metrics: domain.ReportMetrics{
			Peak30MinuteBuckets: []domain.Peak30MinuteBucket{
				{StartMinute: 9 * 60, EndMinute: 9*60 + 30, OrderCount: 4},
				{StartMinute: 10 * 60, EndMinute: 10*60 + 30, OrderCount: 4},
				{StartMinute: 23*60 + 30, EndMinute: 24 * 60, OrderCount: 1},
			},
		},
	}

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/stats/report?period=day&date=2026-08-18", nil)
	ctx.Set(identity.GinContextKey, &identity.Context{WorkspaceID: uuid.New()})

	handler := NewStatsHandler(statsServiceStub{
		reportFn: func(context.Context, uuid.UUID, domain.ReportQuery) (*domain.Report, error) {
			return &report, nil
		},
	}, location)
	handler.Report(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	var body dto.ReportResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	require.Equal(t, []dto.Peak30Minute{
		{Start: "09:00", End: "09:30", OrderCount: 4},
		{Start: "10:00", End: "10:30", OrderCount: 4},
		{Start: "23:30", End: "00:00", OrderCount: 1},
	}, body.Metrics.Peak30MinuteBuckets)
}

func TestStatsHandlerReportReturnsUnsupportedPeriodCode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/stats/report?period=week&date=2026-08-18", nil)
	ctx.Set(identity.GinContextKey, &identity.Context{WorkspaceID: uuid.New()})

	handler := NewStatsHandler(statsServiceStub{
		reportFn: func(_ context.Context, _ uuid.UUID, _ domain.ReportQuery) (*domain.Report, error) {
			return nil, apperr.New(apperr.KindInvalidArgument, "report_period_unsupported", "the requested report period is not supported")
		},
	}, time.FixedZone("CST", 8*60*60))
	handler.Report(ctx)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	require.Equal(t, "report_period_unsupported", body.Error.Code)
}

func TestStatsHandlerReportRejectsInvalidPeriodWithoutCallingService(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/stats/report?period=quarter&date=2026-08-18", nil)
	ctx.Set(identity.GinContextKey, &identity.Context{WorkspaceID: uuid.New()})

	called := false
	handler := NewStatsHandler(statsServiceStub{
		reportFn: func(_ context.Context, _ uuid.UUID, _ domain.ReportQuery) (*domain.Report, error) {
			called = true
			return nil, nil
		},
	}, time.FixedZone("CST", 8*60*60))
	handler.Report(ctx)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.False(t, called)
}
