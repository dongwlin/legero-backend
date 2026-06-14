package repo

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type Counter struct {
	db bun.IDB
}

func NewCounter(db bun.IDB) *Counter {
	return &Counter{db: db}
}

func (r *Counter) Allocate(ctx context.Context, workspaceID uuid.UUID, bizDate time.Time, quantity int, now time.Time) (int, error) {
	if quantity <= 0 {
		return 0, fmt.Errorf("quantity must be greater than 0")
	}

	if _, err := r.db.ExecContext(ctx, `
insert into workspace_daily_counters (workspace_id, biz_date, last_seq, updated_at)
values (?, ?, 0, ?)
on conflict (workspace_id, biz_date) do nothing
`, workspaceID, bizDate.Format("2006-01-02"), now); err != nil {
		return 0, fmt.Errorf("ensure workspace counter: %w", err)
	}

	var lastSeq int
	if err := r.db.NewRaw(`
update workspace_daily_counters
set last_seq = last_seq + ?, updated_at = ?
where workspace_id = ? and biz_date = ?
returning last_seq
`, quantity, now, workspaceID, bizDate.Format("2006-01-02")).Scan(ctx, &lastSeq); err != nil {
		return 0, fmt.Errorf("allocate workspace counter: %w", err)
	}

	return lastSeq - quantity + 1, nil
}

func (r *Counter) ResetWorkspace(ctx context.Context, workspaceID uuid.UUID, bizDateBefore *time.Time) error {
	query := r.db.NewDelete().
		Table("workspace_daily_counters").
		Where("workspace_id = ?", workspaceID)

	if bizDateBefore != nil {
		query = query.Where("biz_date < ?", bizDateBefore.Format("2006-01-02"))
	}

	if _, err := query.Exec(ctx); err != nil {
		return fmt.Errorf("reset workspace counter: %w", err)
	}

	return nil
}
