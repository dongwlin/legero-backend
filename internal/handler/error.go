package handler

import (
	"github.com/dongwlin/legero-backend/internal/handler/response"
	"github.com/dongwlin/legero-backend/internal/pkg/errs"
	"github.com/gofiber/fiber/v2"
)

func Error(c *fiber.Ctx, err error) error {

	var (
		httpCode int
		msg      string
		data     any
	)

	switch e := err.(type) {
	case *fiber.Error:

		return fiber.DefaultErrorHandler(c, e)

	case *errs.Error:

		httpCode = e.HTTPCode()
		msg = e.Message()
		data = e.Details()
		resp := response.BusinessError(msg, data)
		return c.Status(httpCode).JSON(resp)

	default:

		resp := response.UnexpectedError()
		return c.Status(fiber.StatusInternalServerError).JSON(resp)
	}
}
