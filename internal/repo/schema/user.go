package schema

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// User is the bun ORM mapping of the users table.
type User struct {
	bun.BaseModel `bun:"table:users,alias:u"`

	ID           uuid.UUID `bun:",pk,type:uuid"`
	Phone        string    `bun:"phone,notnull"`
	PasswordHash string    `bun:"password_hash,notnull"`
	IsActive     bool      `bun:"is_active,notnull"`
	CreatedAt    time.Time `bun:"created_at,notnull"`
	UpdatedAt    time.Time `bun:"updated_at,notnull"`
}
