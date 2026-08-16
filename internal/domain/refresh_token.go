package domain

import (
	"time"

	"github.com/google/uuid"
)

// RefreshToken is the domain model for a stored refresh token record.
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
