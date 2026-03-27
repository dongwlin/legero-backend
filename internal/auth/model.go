package auth

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"

	"github.com/dongwlin/legero-backend/internal/order"
	"github.com/dongwlin/legero-backend/internal/workspace"
)

type User struct {
	ID           uuid.UUID
	Phone        string
	PasswordHash string
	IsActive     bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type RefreshToken struct {
	ID           uuid.UUID
	UserID       uuid.UUID
	WorkspaceID  uuid.UUID
	TokenHash    string
	ExpiresAt    time.Time
	CreatedAt    time.Time
	RotatedAt    *time.Time
	RevokedAt    *time.Time
	ReplacedByID *uuid.UUID
}

type UserModel struct {
	bun.BaseModel `bun:"table:users,alias:u"`

	ID           uuid.UUID `bun:",pk,type:uuid"`
	Phone        string    `bun:"phone,notnull"`
	PasswordHash string    `bun:"password_hash,notnull"`
	IsActive     bool      `bun:"is_active,notnull"`
	CreatedAt    time.Time `bun:"created_at,notnull"`
	UpdatedAt    time.Time `bun:"updated_at,notnull"`
}

type RefreshTokenModel struct {
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

type TokenPair struct {
	AccessToken           string
	AccessTokenExpiresAt  time.Time
	RefreshToken          string
	RefreshTokenExpiresAt time.Time
}

type AuthUser struct {
	ID    uuid.UUID
	Phone string
	Role  workspace.Role
}

type WorkspaceInfo struct {
	ID   uuid.UUID
	Name string
}

type BootstrapData struct {
	User         AuthUser
	Workspace    WorkspaceInfo
	Permissions  []string
	ActiveOrders []order.Order
	ServerTime   time.Time
}

type LoginResult struct {
	TokenPair TokenPair
	Bootstrap BootstrapData
}

type TokenClaims struct {
	UserID      uuid.UUID
	WorkspaceID uuid.UUID
	Role        workspace.Role
	Type        string
	JTI         string
	ExpiresAt   time.Time
}

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
