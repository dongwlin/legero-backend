package handler

import "github.com/gofiber/fiber/v2"

type RouteRegistrar interface {
	RegisterRoutes(router fiber.Router)
}
