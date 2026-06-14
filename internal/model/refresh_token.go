package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// RefreshToken is the unified domain + ORM model for the refresh_tokens table.
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

// TokenPair holds a newly issued access + refresh token pair.
type TokenPair struct {
	AccessToken           string
	AccessTokenExpiresAt  time.Time
	RefreshToken          string
	RefreshTokenExpiresAt time.Time
}

// AuthUser is the minimal user representation returned in auth responses.
type AuthUser struct {
	ID    uuid.UUID
	Phone string
	Role  Role
}

// WorkspaceInfo is the minimal workspace representation returned in auth responses.
type WorkspaceInfo struct {
	ID   uuid.UUID
	Name string
}

// BootstrapData is the full bootstrap payload returned after login.
type BootstrapData struct {
	User         AuthUser
	Workspace    WorkspaceInfo
	Permissions  []string
	ActiveOrders []Order
	ServerTime   time.Time
}

// LoginResult combines a token pair with the bootstrap data.
type LoginResult struct {
	TokenPair TokenPair
	Bootstrap BootstrapData
}

// TokenClaims represents the decoded claims of a PASETO token.
type TokenClaims struct {
	UserID      uuid.UUID
	WorkspaceID uuid.UUID
	Role        Role
	Type        string
	JTI         string
	ExpiresAt   time.Time
}
