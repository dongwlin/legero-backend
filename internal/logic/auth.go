package logic

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/dongwlin/legero-backend/internal/model"
	"github.com/dongwlin/legero-backend/internal/model/types"
	"github.com/dongwlin/legero-backend/internal/pkg/errs"
	"github.com/dongwlin/legero-backend/internal/pkg/token"
	"github.com/dongwlin/legero-backend/internal/repo"
	"golang.org/x/crypto/bcrypt"
)

type Auth interface {
	// Login user
	Login(ctx context.Context, params AuthLoginParams) (*AuthLoginResult, error)
	// Register user
	Register(ctx context.Context, params AuthRegisterParams) (uint64, error)
}

type (
	AuthLoginParams struct {
		Identifier string
		Password   string
	}

	AuthRegisterParams struct {
		Nickname    string
		Username    string
		PhoneNumber string
		Password    string
		Role        types.Role
	}
)

type (
	AuthLoginResult struct {
		UserID      uint64
		AccessToken string
		ExpireAt    time.Time
	}
)

type AuthImpl struct {
	userRepo  repo.User
	tokenRepo repo.Token
}

// Login implements Auth.
func (l *AuthImpl) Login(ctx context.Context, params AuthLoginParams) (*AuthLoginResult, error) {

	isPasswordValid := func(storedHash, providedPassword string) bool {
		return bcrypt.CompareHashAndPassword(
			[]byte(storedHash),
			[]byte(providedPassword),
		) == nil
	}

	if user, err := l.userRepo.GetByUsername(ctx, params.Identifier); err == nil {
		if isPasswordValid(user.PasswordHash, params.Password) {
			return l.createLoginResponse(ctx, user)
		}
	} else if !errors.Is(err, errs.ErrUserNotFound) {
		return nil, err
	}

	user, err := l.userRepo.GetByPhoneNumber(ctx, params.Identifier)
	if err != nil {
		return nil, err
	}

	if !isPasswordValid(user.PasswordHash, params.Password) {
		return nil, errs.ErrWrongPassword
	}

	return l.createLoginResponse(ctx, user)
}

func (l *AuthImpl) createLoginResponse(ctx context.Context, user *model.User) (*AuthLoginResult, error) {

	if user.Status == types.StatusBlocked {
		return nil, errs.ErrUserBlocked
	}

	token, err := token.Generate(32)
	if err != nil {
		return nil, fmt.Errorf("token generation failed: %w", err)
	}

	expiredAt := time.Now().Add(90 * 24 * time.Hour)

	if _, err := l.tokenRepo.Create(ctx, &model.Token{
		UserID:      user.ID,
		AccessToken: token,
		ExpiredAt:   expiredAt,
	}); err != nil {
		return nil, err
	}

	return &AuthLoginResult{
		UserID:      user.ID,
		AccessToken: token,
		ExpireAt:    expiredAt,
	}, nil
}

// Register implements Auth.
func (l *AuthImpl) Register(ctx context.Context, params AuthRegisterParams) (uint64, error) {

	if exists, err := l.userRepo.ExistsByUsername(ctx, params.Username); err == nil && exists {
		return 0, errs.ErrUsernameAlreadyExists
	} else if err != nil {
		return 0, err
	}

	if exists, err := l.userRepo.ExistsByPhoneNumber(ctx, params.PhoneNumber); err == nil && exists {
		return 0, errs.ErrPhoneNumberAlreadyExists
	} else if err != nil {
		return 0, err
	}

	passwordHashBytes, err := bcrypt.GenerateFromPassword([]byte(params.Password), bcrypt.DefaultCost)
	if err != nil {
		return 0, err
	}
	passwordHash := string(passwordHashBytes)

	user, err := l.userRepo.Create(ctx, &model.User{
		Nickname:     params.Nickname,
		Username:     params.Username,
		PhoneNumber:  params.PhoneNumber,
		PasswordHash: passwordHash,
		Role:         params.Role,
	})
	if err != nil {
		return 0, err
	}

	return user.ID, nil
}

func NewAuth(userRepo repo.User, tokenRepo repo.Token) Auth {
	return &AuthImpl{
		userRepo:  userRepo,
		tokenRepo: tokenRepo,
	}
}
