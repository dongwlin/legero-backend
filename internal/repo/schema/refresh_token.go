package schema

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// RefreshToken is the bun ORM mapping of the refresh_tokens table.
type RefreshToken struct {
	bun.BaseModel `bun:"table:refresh_tokens,alias:rt"`

	ID           uuid.UUID  `bun:",pk,type:uuid"`
	UserID       uuid.UUID  `bun:"user_id,type:uuid,notnull"`
	WorkspaceID  uuid.UUID  `bun:"workspace_id,type:uuid,notnull"`
	TokenHash    string     `bun:"token_hash,notnull"`
	ExpiresAt    time.Time  `bun:"expires_at,notnull"`
	CreatedAt    time.Time  `bun:"created_at,notnull"`
	RotatedAt    *time.Time `bun:"rotated_at"`
	RevokedAt    *time.Time `bun:"revoked_at"`
	ReplacedByID *uuid.UUID `bun:"replaced_by_id,type:uuid"`
}
