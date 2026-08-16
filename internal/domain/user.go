package domain

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

// User is the domain model for a registered user.
type User struct {
	ID           uuid.UUID
	Phone        string
	PasswordHash string
	IsActive     bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
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
