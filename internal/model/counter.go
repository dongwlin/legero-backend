package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// WorkspaceDailyCounter tracks a per-workspace daily sequence number.
type WorkspaceDailyCounter struct {
	bun.BaseModel `bun:"table:workspace_daily_counters,alias:wdc"`

	WorkspaceID uuid.UUID `bun:"workspace_id,pk,type:uuid"`
	BizDate     string    `bun:"biz_date,pk,notnull"`
	LastSeq     int       `bun:"last_seq,notnull"`
	UpdatedAt   time.Time `bun:"updated_at,notnull"`
}
