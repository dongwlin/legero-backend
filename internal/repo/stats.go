package repo

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"

	"github.com/dongwlin/legero-backend/internal/domain"
)

// Stats contains the read-only persistence operations shared by daily and
// period reports.
type Stats struct {
	db bun.IDB
}

func NewStats(db bun.IDB) *Stats {
	return &Stats{db: db}
}

type dailyRowModel struct {
	Date            time.Time `bun:"biz_date"`
	OrderCount      int       `bun:"order_count"`
	TotalPriceCents int       `bun:"total_price_cents"`
}

// Daily returns one row per business date in [from, to], including dates with
// no completed orders. The public method keeps the existing date-oriented
// repository contract; DailyWindow is the shared-window implementation used
// by the service.
func (r *Stats) Daily(ctx context.Context, workspaceID uuid.UUID, timezone string, from, to time.Time) ([]domain.DailyRow, error) {
	location, err := time.LoadLocation(timezone)
	if err != nil {
		return nil, fmt.Errorf("load stats timezone: %w", err)
	}
	window, err := domain.NewDailyReportWindow(from, to, location)
	if err != nil {
		return nil, err
	}
	return r.DailyWindow(ctx, workspaceID, timezone, window)
}

// DailyWindow uses completed_at as the only order-date source. The range
// predicate is deliberately a direct half-open comparison so the
// (workspace_id, completed_at) index remains usable; the timezone conversion
// is limited to the grouping expression and never appears in WHERE.
func (r *Stats) DailyWindow(ctx context.Context, workspaceID uuid.UUID, timezone string, window domain.ReportWindow) ([]domain.DailyRow, error) {
	location, err := time.LoadLocation(timezone)
	if err != nil {
		return nil, fmt.Errorf("load stats timezone: %w", err)
	}
	lastDate := window.EndAt.In(location).Add(-time.Nanosecond)
	var models []dailyRowModel
	if err := r.db.NewRaw(`
with days as (
  select generate_series(?::date, ?::date, interval '1 day')::date as biz_date
),
daily_orders as (
  select
    (completed_at at time zone ?)::date as biz_date,
    count(*)::integer as order_count,
    coalesce(sum(total_price_cents), 0)::integer as total_price_cents
  from orders
  where workspace_id = ?
    and completed_at is not null
    and completed_at >= ?
    and completed_at < ?
  group by 1
)
select
  days.biz_date,
  coalesce(daily_orders.order_count, 0)::integer as order_count,
  coalesce(daily_orders.total_price_cents, 0)::integer as total_price_cents
from days
left join daily_orders on daily_orders.biz_date = days.biz_date
order by days.biz_date desc
`,
		window.StartAt.In(location).Format("2006-01-02"),
		lastDate.Format("2006-01-02"),
		timezone,
		workspaceID,
		window.StartAt,
		window.EndAt,
	).Scan(ctx, &models); err != nil {
		return nil, fmt.Errorf("query daily stats: %w", err)
	}

	rows := make([]domain.DailyRow, 0, len(models))
	for _, model := range models {
		date, err := time.ParseInLocation("2006-01-02", model.Date.Format("2006-01-02"), location)
		if err != nil {
			return nil, fmt.Errorf("parse daily stats date: %w", err)
		}
		rows = append(rows, domain.DailyRow{
			Date:            date,
			OrderCount:      model.OrderCount,
			TotalPriceCents: model.TotalPriceCents,
		})
	}
	return rows, nil
}

type reportOrderModel struct {
	TotalPriceCents   int        `bun:"total_price_cents"`
	StapleTypeCode    *int16     `bun:"staple_type_code"`
	SizeCode          int16      `bun:"size_code"`
	FriedEggCount     int16      `bun:"fried_egg_count"`
	DiningMethodCode  int16      `bun:"dining_method_code"`
	SelectedMeatCodes []int16    `bun:"selected_meat_codes,array,type:smallint[]"`
	CreatedAt         time.Time  `bun:"created_at"`
	CompletedAt       *time.Time `bun:"completed_at"`
}

// CompletedOrders returns the minimal order projection needed to calculate a
// report. completed_at is constrained by a direct [start,end) range, making
// this query safe for all report periods and friendly to the workspace/date
// index. The caller still rechecks the range when aggregating.
func (r *Stats) CompletedOrders(ctx context.Context, workspaceID uuid.UUID, startAt, endAt time.Time) ([]domain.ReportOrder, error) {
	var models []reportOrderModel
	if err := r.db.NewRaw(`
select
  total_price_cents,
  staple_type_code,
  size_code,
  fried_egg_count,
  dining_method_code,
  selected_meat_codes,
  created_at,
  completed_at
from orders
where workspace_id = ?
  and completed_at is not null
  and completed_at >= ?
  and completed_at < ?
`, workspaceID, startAt, endAt).Scan(ctx, &models); err != nil {
		return nil, fmt.Errorf("query completed report orders: %w", err)
	}

	orders := make([]domain.ReportOrder, len(models))
	for index := range models {
		orders[index] = domain.ReportOrder{
			TotalPriceCents:   models[index].TotalPriceCents,
			StapleTypeCode:    models[index].StapleTypeCode,
			SizeCode:          models[index].SizeCode,
			FriedEggCount:     models[index].FriedEggCount,
			DiningMethodCode:  models[index].DiningMethodCode,
			SelectedMeatCodes: append([]int16(nil), models[index].SelectedMeatCodes...),
			CreatedAt:         models[index].CreatedAt,
			CompletedAt:       models[index].CompletedAt,
		}
	}
	return orders, nil
}
