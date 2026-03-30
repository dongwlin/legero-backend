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
select
  (created_at at time zone ?)::date as biz_date,
  count(*)::integer as order_count,
  coalesce(sum(total_price_cents), 0)::integer as total_price_cents
from orders
where workspace_id = ?
  and (created_at at time zone ?)::date between ? and ?
group by biz_date
order by biz_date desc
`, timezone, workspaceID, timezone, from.Format("2006-01-02"), to.Format("2006-01-02")).Scan(ctx, &models); err != nil {
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
