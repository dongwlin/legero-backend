package handler

import (
	"errors"
	"time"

	"github.com/dongwlin/legero-backend/internal/handler/response"
	"github.com/dongwlin/legero-backend/internal/logic"
	"github.com/dongwlin/legero-backend/internal/model/types"
	"github.com/dongwlin/legero-backend/internal/pkg/errs"
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
