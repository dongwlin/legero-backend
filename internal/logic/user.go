package logic

import (
	"context"
	"errors"
	"time"

	"github.com/dongwlin/legero-backend/internal/model"
	"github.com/dongwlin/legero-backend/internal/model/types"
	"github.com/dongwlin/legero-backend/internal/repo"
)

var (
	ErrUsernameAlreadyExists    = errors.New("username already exists")
	ErrPhoneNumberAlreadyExists = errors.New("phone number already exists")
)

type User interface {
	GetUserInfo(ctx context.Context, userID uint64) (*UserInfo, error)
	UpdatePassword(ctx context.Context, params UserUpdatePasswordParams) error
	UpdatePhoneNumber(ctx context.Context, params UserUpdatePhoneNumberParams) error
	UpdateNickname(ctx context.Context, params UserUpdateNicknameParams) error
	BlockUser(ctx context.Context, userID uint64) error
}

type (
	UserUpdatePasswordParams struct {
		UserID      int64
		OldPassword string
		NewPassword string
	}

	UserUpdatePhoneNumberParams struct {
		UserID         uint64
		NewPhoneNumber string
	}

	UserUpdateNicknameParams struct {
		UserID      uint64
		NewNickname string
	}
)

type (
	UserInfo struct {
		UserID      uint64
		Nickname    string
		Username    string
		PhoneNumber string
		Status      types.Status
		BlockedAt   time.Time
		CreatedAt   time.Time
		UpdatedAt   time.Time
	}
)

type UserImpl struct {
	userRepo repo.User
}

// BlockUser implements User.
func (l *UserImpl) BlockUser(ctx context.Context, userID uint64) error {

	_, err := l.userRepo.Update(ctx, &model.User{
		ID:     userID,
		Status: types.StatusBlocked,
	})

	return err
}

// GetUserInfo implements User.
func (l *UserImpl) GetUserInfo(ctx context.Context, userID uint64) (*UserInfo, error) {

	result, err := l.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	return &UserInfo{
		UserID:      result.ID,
		Nickname:    result.Nickname,
		Username:    result.Username,
		PhoneNumber: result.PhoneNumber,
		Status:      result.Status,
		BlockedAt:   result.BlockedAt,
		CreatedAt:   result.CreatedAt,
		UpdatedAt:   result.UpdatedAt,
	}, nil

}

// UpdateNickname implements User.
func (l *UserImpl) UpdateNickname(ctx context.Context, params UserUpdateNicknameParams) error {
	panic("unimplemented")
}

// UpdatePassword implements User.
func (l *UserImpl) UpdatePassword(ctx context.Context, params UserUpdatePasswordParams) error {
	panic("unimplemented")
}

// UpdatePhoneNumber implements User.
func (l *UserImpl) UpdatePhoneNumber(ctx context.Context, params UserUpdatePhoneNumberParams) error {
	panic("unimplemented")
}

func NewUser(userRepo repo.User) User {
	return &UserImpl{
		userRepo: userRepo,
	}
}
