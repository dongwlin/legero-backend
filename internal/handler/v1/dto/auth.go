package dto

import "github.com/dongwlin/legero-backend/internal/model"

// LoginRequest carries the login credentials.
type LoginRequest struct {
	Phone    string `json:"phone"`
	Password string `json:"password"`
}

// RefreshRequest carries the refresh token for rotation.
type RefreshRequest struct {
	RefreshToken string `json:"refreshToken"`
}

// AuthUser is the minimal user representation in auth responses.
type AuthUser struct {
	ID    string `json:"id"`
	Phone string `json:"phone"`
	Role  string `json:"role"`
}

// Workspace is the minimal workspace representation in auth responses.
type Workspace struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// TokenPair is the issued access/refresh token pair.
type TokenPair struct {
	AccessToken           string `json:"accessToken"`
	TokenType             string `json:"tokenType"`
	AccessTokenExpiresAt  string `json:"accessTokenExpiresAt"`
	RefreshToken          string `json:"refreshToken"`
	RefreshTokenExpiresAt string `json:"refreshTokenExpiresAt"`
}

// Bootstrap is the full bootstrap payload returned after login.
type Bootstrap struct {
	User         AuthUser         `json:"user"`
	Workspace    Workspace        `json:"workspace"`
	Permissions  []string         `json:"permissions"`
	ActiveOrders []model.OrderDTO `json:"activeOrders"`
	ServerTime   string           `json:"serverTime"`
}

// LoginResponse flattens the token pair and bootstrap payload into one response.
type LoginResponse struct {
	TokenPair
	Bootstrap
}

// BootstrapResponse is the bootstrap payload for an already-authenticated user.
type BootstrapResponse struct {
	Bootstrap
}
