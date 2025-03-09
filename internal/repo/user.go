package repo

import (
	"context"

	"github.com/dongwlin/legero-backend/internal/ent"
	"github.com/dongwlin/legero-backend/internal/ent/user"
	"github.com/dongwlin/legero-backend/internal/model"
	"github.com/dongwlin/legero-backend/internal/model/types"
	"github.com/dongwlin/legero-backend/internal/pkg/errs"
)

type User interface {
	GetByID(ctx context.Context, id uint64) (*model.User, error)
	GetByUsername(ctx context.Context, username string) (*model.User, error)
	GetByPhoneNumber(ctx context.Context, phoneNumber string) (*model.User, error)
	Create(ctx context.Context, user *model.User) (*model.User, error)
	Update(ctx context.Context, user *model.User) (*model.User, error)
	Delete(ctx context.Context, id uint64) (uint64, error)

	ExistsByUsername(ctx context.Context, username string) (bool, error)
	ExistsByPhoneNumber(ctx context.Context, phoneNumber string) (bool, error)
}

type UserImpl struct {
	db *ent.Client
}

// Create implements User.
func (r *UserImpl) Create(ctx context.Context, params *model.User) (*model.User, error) {

	result, err := r.db.User.Create().
		SetNickname(params.Nickname).
		SetUsername(params.Username).
		SetPhoneNumber(params.PhoneNumber).
		SetRole(user.Role(params.Role)).
		SetPasswordHash(params.PasswordHash).
		Save(ctx)

	if err != nil {
		return nil, err
	}

	return &model.User{
		ID:           result.ID,
		Nickname:     result.Nickname,
		Username:     result.Username,
		PhoneNumber:  result.PhoneNumber,
		Role:         types.Role(result.Role),
		PasswordHash: result.PasswordHash,
		Status:       types.Status(result.Status),
		BlockedAt:    result.BlockedAt,
		CreatedAt:    result.CreatedAt,
		UpdatedAt:    result.UpdatedAt,
	}, nil
}

// Delete implements User.
func (r *UserImpl) Delete(ctx context.Context, id uint64) (uint64, error) {

	err := r.db.User.DeleteOneID(id).
		Exec(ctx)

	if err != nil {
		return 0, err
	}

	return id, nil
}

// ExistsByPhoneNumber implements User.
func (r *UserImpl) ExistsByPhoneNumber(ctx context.Context, phoneNumber string) (bool, error) {

	return r.db.User.Query().
		Where(
			user.PhoneNumberEQ(phoneNumber),
		).
		Exist(ctx)
}

// ExistsByUsername implements User.
func (r *UserImpl) ExistsByUsername(ctx context.Context, username string) (bool, error) {

	return r.db.User.Query().
		Where(
			user.UsernameEQ(username),
		).
		Exist(ctx)
}

// GetByID implements User.
func (r *UserImpl) GetByID(ctx context.Context, id uint64) (*model.User, error) {

	result, err := r.db.User.Query().
		Where(
			user.IDEQ(id),
		).
		Only(ctx)

	if err != nil {

		if ent.IsNotFound(err) {
			return nil, errs.ErrUserNotFound
		}

		return nil, err
	}

	return &model.User{
		ID:           result.ID,
		Nickname:     result.Nickname,
		Username:     result.Username,
		PhoneNumber:  result.PhoneNumber,
		Role:         types.Role(result.Role),
		PasswordHash: result.PasswordHash,
		Status:       types.Status(result.Status),
		BlockedAt:    result.BlockedAt,
		CreatedAt:    result.CreatedAt,
		UpdatedAt:    result.UpdatedAt,
	}, nil
}

// GetByPhoneNumber implements User.
func (r *UserImpl) GetByPhoneNumber(ctx context.Context, phoneNumber string) (*model.User, error) {

	result, err := r.db.User.Query().
		Where(
			user.PhoneNumberEQ(phoneNumber),
		).
		Only(ctx)

	if err != nil {
		return nil, err
	}

	return &model.User{
		ID:           result.ID,
		Nickname:     result.Nickname,
		Username:     result.Username,
		PhoneNumber:  result.PhoneNumber,
		Role:         types.Role(result.Role),
		PasswordHash: result.PasswordHash,
		Status:       types.Status(result.Status),
		BlockedAt:    result.BlockedAt,
		CreatedAt:    result.CreatedAt,
		UpdatedAt:    result.UpdatedAt,
	}, nil
}

// GetByUsername implements User.
func (r *UserImpl) GetByUsername(ctx context.Context, username string) (*model.User, error) {

	result, err := r.db.User.Query().
		Where(
			user.UsernameEQ(username),
		).
		Only(ctx)

	if err != nil {
		return nil, err
	}

	return &model.User{
		ID:           result.ID,
		Nickname:     result.Nickname,
		Username:     result.Username,
		PhoneNumber:  result.PhoneNumber,
		Role:         types.Role(result.Role),
		PasswordHash: result.PasswordHash,
		Status:       types.Status(result.Status),
		BlockedAt:    result.BlockedAt,
		CreatedAt:    result.CreatedAt,
		UpdatedAt:    result.UpdatedAt,
	}, nil
}

// Update implements User.
func (r *UserImpl) Update(ctx context.Context, params *model.User) (*model.User, error) {

	update := r.db.User.UpdateOneID(params.ID)

	if params.Nickname != "" {
		update = update.SetNickname(params.Nickname)
	}

	if params.Username != "" {
		update = update.SetUsername(params.Username)
	}

	if params.PhoneNumber != "" {
		update = update.SetPhoneNumber(params.PhoneNumber)
	}

	if params.Role != "" {
		update = update.SetRole(user.Role(params.Role))
	}

	if params.PasswordHash != "" {
		update = update.SetPasswordHash(params.PasswordHash)
	}

	if params.Status != "" {
		update = update.SetStatus(user.Status(params.Status))
	}

	result, err := update.Save(ctx)

	if err != nil {
		return nil, err
	}

	return &model.User{
		ID:           result.ID,
		Nickname:     result.Nickname,
		Username:     result.Username,
		PhoneNumber:  result.PhoneNumber,
		Role:         types.Role(result.Role),
		PasswordHash: result.PasswordHash,
		Status:       types.Status(result.Status),
		BlockedAt:    result.BlockedAt,
		CreatedAt:    result.CreatedAt,
		UpdatedAt:    result.UpdatedAt,
	}, nil
}

func NewUser(db *ent.Client) User {
	return &UserImpl{
		db: db,
	}
}
