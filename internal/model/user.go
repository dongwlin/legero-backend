package model

import (
	"time"

	"github.com/dongwlin/legero-backend/internal/model/types"
)

type User struct {
	ID           uint64
	Nickname     string
	Username     string
	Role         types.Role
	PasswordHash string
	PhoneNumber  string
	Status       types.Status
	BlockedAt    time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
