package model

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// User is the unified domain + ORM model for the users table.
type User struct {
	bun.BaseModel `bun:"table:users,alias:u"`

	ID           uuid.UUID `bun:",pk,type:uuid"`
	Phone        string    `bun:"phone,notnull"`
	PasswordHash string    `bun:"password_hash,notnull"`
	IsActive     bool      `bun:"is_active,notnull"`
	CreatedAt    time.Time `bun:"created_at,notnull"`
	UpdatedAt    time.Time `bun:"updated_at,notnull"`
}

// NormalizePhone strips non-digits and removes a leading "86" country code
// if the resulting number is longer than 11 digits.
func NormalizePhone(phone string) string {
	digits := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, phone)

	if strings.HasPrefix(digits, "86") && len(digits) > 11 {
		return digits[2:]
	}

	return digits
}
