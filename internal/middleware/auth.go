package middleware

import (
	"strings"

	"github.com/dongwlin/legero-backend/internal/handler/response"
	"github.com/dongwlin/legero-backend/internal/repo"
	"github.com/gofiber/fiber/v2"
)

func Auth(tokenRepo repo.Token) func(c *fiber.Ctx) error {

	return func(c *fiber.Ctx) error {

		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(response.BusinessError("authorization header is required"))
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(response.BusinessError("invalid token format"))
		}

		token, err := tokenRepo.GetByToken(c.UserContext(), tokenString)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(response.BusinessError("invalid or expired token"))
		}

		c.Locals("userID", token.UserID)
		return c.Next()
	}
}
