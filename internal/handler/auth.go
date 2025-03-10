package handler

import (
	"errors"
	"time"

	"github.com/dongwlin/legero-backend/internal/handler/response"
	"github.com/dongwlin/legero-backend/internal/logic"
	"github.com/dongwlin/legero-backend/internal/pkg/errs"
	"github.com/dongwlin/legero-backend/internal/pkg/validator"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
)

type Auth struct {
	authLogic logic.Auth
}

func NewAuth(authLogic logic.Auth) *Auth {
	return &Auth{
		authLogic: authLogic,
	}
}

func (h *Auth) RegisterRoutes(r fiber.Router) {

	auth := r.Group("/auth")

	auth.Post("/login", h.Login)
	// auth.Post("/register", h.Register)
}

type AuthLoginRequest struct {
	Identifier string `json:"identifier" validate:"required,max=64"`
	Password   string `json:"password"  validate:"required,max=64"`
}

type AuthLoginResposeData struct {
	AccessToken string    `json:"access_token"`
	ExpiredAt   time.Time `json:"expired_at"`
}

func (h *Auth) Login(c *fiber.Ctx) error {

	var req AuthLoginRequest
	if err := validator.ValidateBody(c, &req); err != nil {
		return err
	}

	result, err := h.authLogic.Login(c.UserContext(), logic.AuthLoginParams{
		Identifier: req.Identifier,
		Password:   req.Password,
	})
	if err != nil {

		if errors.Is(err, errs.ErrUserNotFound) || errors.Is(err, errs.ErrWrongPassword) {
			resp := response.BusinessError("invalid identifier or password")
			return c.Status(fiber.StatusBadRequest).JSON(resp)
		}

		log.Error().
			Err(err).
			Msg("unexpected error")
		resp := response.UnexpectedError()
		return c.Status(fiber.StatusInternalServerError).JSON(resp)
	}

	resp := response.Success(AuthLoginResposeData{
		AccessToken: result.AccessToken,
		ExpiredAt:   result.ExpireAt,
	})
	return c.Status(fiber.StatusOK).JSON(resp)
}

// type AuthRegisterRequest struct {
// 	Nickname    string `json:"nickname"`
// 	Username    string `json:"username"`
// 	PhoneNumber string `json:"phone_number"`
// 	Password    string `json:"password"`
// }

// func (h *Auth) Register(c *fiber.Ctx) error {

// 	var req AuthRegisterRequest
// 	if err := c.BodyParser(&req); err != nil {
// 		resp := response.BusinessError("invalid params")
// 		return c.Status(fiber.StatusBadRequest).JSON(resp)
// 	}

// 	_, err := h.authLogic.Register(c.UserContext(), logic.AuthRegisterParams{
// 		Nickname:    req.Nickname,
// 		Username:    req.Username,
// 		PhoneNumber: req.PhoneNumber,
// 		Password:    req.Password,
// 		Role:        types.RoleWaiter,
// 	})
// 	if err != nil {

// 		if errors.Is(err, errs.ErrUsernameAlreadyExists) {
// 			resp := response.BusinessError("username already exists")
// 			return c.Status(fiber.StatusBadRequest).JSON(resp)
// 		}

// 		if errors.Is(err, errs.ErrPhoneNumberAlreadyExists) {
// 			resp := response.BusinessError("phone number already exists")
// 			return c.Status(fiber.StatusBadRequest).JSON(resp)
// 		}

// 		resp := response.UnexpectedError()
// 		return c.Status(fiber.StatusInternalServerError).JSON(resp)
// 	}

// 	resp := response.Success(nil)
// 	return c.Status(fiber.StatusOK).JSON(resp)
// }
