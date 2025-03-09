package logic

import (
	"context"
	"time"

	"github.com/dongwlin/legero-backend/internal/model"
	"github.com/dongwlin/legero-backend/internal/model/types"
	"github.com/dongwlin/legero-backend/internal/pkg/errs"
	"github.com/dongwlin/legero-backend/internal/repo"
	"golang.org/x/crypto/bcrypt"
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
		UserID      uint64
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

	_, err := l.userRepo.Update(ctx, &model.User{
		ID:       params.UserID,
		Nickname: params.NewNickname,
	})

	return err
}

// UpdatePassword implements User.
func (l *UserImpl) UpdatePassword(ctx context.Context, params UserUpdatePasswordParams) error {

	user, err := l.userRepo.GetByID(ctx, params.UserID)
	if err != nil {
		return err
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(params.OldPassword))
	if err != nil {
		return errs.ErrWrongPassword
	}

	passworddHashBytes, err := bcrypt.GenerateFromPassword([]byte(params.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	passwordHash := string(passworddHashBytes)

	_, err = l.userRepo.Update(ctx, &model.User{
		ID:           params.UserID,
		PasswordHash: passwordHash,
	})

	return err
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
