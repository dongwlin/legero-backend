package model

import "time"

type Token struct {
	AccessToken string    `json:"access_token"`
	UserID      uint64    `json:"user_id"`
	ExpiredAt   time.Time `json:"expired_at"`
	CreatedAt   time.Time `json:"created_at"`
}
