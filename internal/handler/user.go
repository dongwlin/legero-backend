package handler

import (
	"errors"
	"time"

	"github.com/dongwlin/legero-backend/internal/handler/response"
	"github.com/dongwlin/legero-backend/internal/logic"
	"github.com/dongwlin/legero-backend/internal/model/types"
	"github.com/dongwlin/legero-backend/internal/pkg/errs"
	"github.com/dongwlin/legero-backend/internal/pkg/validator"
	"github.com/gofiber/fiber/v2"
)

type User struct {
	userLogic logic.User
}

func NewUser(userLogic logic.User) *User {
	return &User{
		userLogic: userLogic,
	}
}

func (h *User) RegisterRoutes(r fiber.Router) {

	user := r.Group("/users")

	user.Get("/profile", h.Profile)
	user.Put("/password", h.UpdatePassword)
	user.Put("/nickname", h.UpdateNickname)
}

type UserProfileResponseData struct {
	Nickname    string       `json:"nickname"`
	Username    string       `json:"username"`
	PhoneNumber string       `json:"phone_number"`
	Status      types.Status `json:"status"`
	BlockedAt   time.Time    `json:"blocked_at"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}

func (h *User) Profile(c *fiber.Ctx) error {

	userID, ok := c.Locals("userID").(uint64)
	if ok == false {
		resp := response.BusinessError("not authenticated")
		return c.Status(fiber.StatusForbidden).JSON(resp)
	}

	result, err := h.userLogic.GetUserInfo(c.UserContext(), userID)
	if err != nil {

		if errors.Is(err, errs.ErrUserNotFound) {
			resp := response.BusinessError("user not found")
			return c.Status(fiber.StatusNotFound).JSON(resp)
		}

		resp := response.UnexpectedError()
		return c.Status(fiber.StatusInternalServerError).JSON(resp)
	}

	data := UserProfileResponseData{
		Nickname:    result.Nickname,
		Username:    result.Username,
		PhoneNumber: result.PhoneNumber,
		Status:      result.Status,
		BlockedAt:   result.BlockedAt,
		CreatedAt:   result.CreatedAt,
		UpdatedAt:   result.UpdatedAt,
	}
	resp := response.Success(data)
	return c.Status(fiber.StatusOK).JSON(resp)
}

type UserUpdatePasswordRequest struct {
	OldPassword string `json:"old_password" validate:"required,min=8,max=64"`
	NewPassword string `json:"new_password" validate:"required,min=8,max=64"`
}

func (h *User) UpdatePassword(c *fiber.Ctx) error {

	userID, ok := c.Locals("userID").(uint64)
	if ok == false {
		resp := response.BusinessError("not authenticated")
		return c.Status(fiber.StatusForbidden).JSON(resp)
	}

	var req UserUpdatePasswordRequest
	if err := validator.ValidateBody(c, &req); err != nil {
		return err
	}

	if req.OldPassword == req.NewPassword {
		resp := response.BusinessError("new password should be different from old password")
		return c.Status(fiber.StatusBadRequest).JSON(resp)
	}

	err := h.userLogic.UpdatePassword(c.UserContext(), logic.UserUpdatePasswordParams{
		UserID:      userID,
		OldPassword: req.OldPassword,
		NewPassword: req.NewPassword,
	})

	if err != nil {

		if errors.Is(err, errs.ErrUserNotFound) {
			resp := response.BusinessError("user not found")
			return c.Status(fiber.StatusNotFound).JSON(resp)
		}

		if errors.Is(err, errs.ErrWrongPassword) {
			resp := response.BusinessError("wrong password")
			return c.Status(fiber.StatusBadRequest).JSON(resp)
		}

		resp := response.UnexpectedError()
		return c.Status(fiber.StatusInternalServerError).JSON(resp)
	}

	resp := response.Success(nil)
	return c.Status(fiber.StatusOK).JSON(resp)
}

type UserUpdateNicknameRequest struct {
	Nickname string `json:"nickname" validate:"required,max=64"`
}

func (h *User) UpdateNickname(c *fiber.Ctx) error {

	userID, ok := c.Locals("userID").(uint64)
	if ok == false {
		resp := response.BusinessError("not authenticated")
		return c.Status(fiber.StatusForbidden).JSON(resp)
	}

	var req UserUpdateNicknameRequest
	if err := c.BodyParser(&req); err != nil {
		resp := response.BusinessError("invalid params")
		return c.Status(fiber.StatusBadRequest).JSON(resp)
	}

	err := h.userLogic.UpdateNickname(c.UserContext(), logic.UserUpdateNicknameParams{
		UserID:      userID,
		NewNickname: req.Nickname,
	})

	if err != nil {

		if errors.Is(err, errs.ErrUserNotFound) {
			resp := response.BusinessError("user not found")
			return c.Status(fiber.StatusNotFound).JSON(resp)
		}

		resp := response.UnexpectedError()
		return c.Status(fiber.StatusInternalServerError).JSON(resp)
	}

	resp := response.Success(nil)
	return c.Status(fiber.StatusOK).JSON(resp)
}

type UserUpdatePhoneNumberRequest struct {
	PhoneNumber string `json:"phone_number" validate:"required,len=11"`
}

func (h *User) UpdatePhoneNumber(c *fiber.Ctx) error {

	userID, ok := c.Locals("userID").(uint64)
	if ok == false {
		resp := response.BusinessError("not authenticated")
		return c.Status(fiber.StatusForbidden).JSON(resp)
	}

	var req UserUpdatePhoneNumberRequest
	if err := validator.ValidateBody(c, &req); err != nil {
		return err
	}

	err := h.userLogic.UpdatePhoneNumber(c.UserContext(), logic.UserUpdatePhoneNumberParams{
		UserID:         userID,
		NewPhoneNumber: req.PhoneNumber,
	})

	if err != nil {

		if errors.Is(err, errs.ErrUserNotFound) {
			resp := response.BusinessError("user not found")
			return c.Status(fiber.StatusNotFound).JSON(resp)
		}

		resp := response.UnexpectedError()
		return c.Status(fiber.StatusInternalServerError).JSON(resp)
	}

	resp := response.Success(nil)
	return c.Status(fiber.StatusOK).JSON(resp)
}
