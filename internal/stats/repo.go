package stats

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type Repository interface {
	Daily(ctx context.Context, db bun.IDB, workspaceID uuid.UUID, timezone string, from, to time.Time) ([]DailyRow, error)
}

type BunRepository struct{}

type dailyRowModel struct {
	Date            time.Time `bun:"biz_date"`
	OrderCount      int       `bun:"order_count"`
	TotalPriceCents int       `bun:"total_price_cents"`
}

func (r *BunRepository) Daily(ctx context.Context, db bun.IDB, workspaceID uuid.UUID, timezone string, from, to time.Time) ([]DailyRow, error) {
	var models []dailyRowModel
	if err := db.NewRaw(`
with days as (
  select generate_series(?::date, ?::date, interval '1 day')::date as biz_date
),
daily_orders as (
  select
    (created_at at time zone ?)::date as biz_date,
    count(*)::integer as order_count,
    coalesce(sum(total_price_cents), 0)::integer as total_price_cents
  from orders
  where workspace_id = ?
    and (created_at at time zone ?)::date between ?::date and ?::date
  group by biz_date
)
select
  days.biz_date,
  coalesce(daily_orders.order_count, 0)::integer as order_count,
  coalesce(daily_orders.total_price_cents, 0)::integer as total_price_cents
from days
left join daily_orders on daily_orders.biz_date = days.biz_date
order by days.biz_date desc
`, from.Format("2006-01-02"), to.Format("2006-01-02"), timezone, workspaceID, timezone, from.Format("2006-01-02"), to.Format("2006-01-02")).Scan(ctx, &models); err != nil {
		return nil, fmt.Errorf("query daily stats: %w", err)
	}

	rows := make([]DailyRow, 0, len(models))
	for _, model := range models {
		rows = append(rows, DailyRow{
			Date:            model.Date,
			OrderCount:      model.OrderCount,
			TotalPriceCents: model.TotalPriceCents,
		})
	}
	return rows, nil
}
