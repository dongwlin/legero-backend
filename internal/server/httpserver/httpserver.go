package httpserver

import (
	"context"
	"fmt"

	"github.com/bytedance/sonic"
	"github.com/dongwlin/legero-backend/internal/config"
	"github.com/dongwlin/legero-backend/internal/handler"
	"github.com/dongwlin/legero-backend/internal/middleware"
	"github.com/dongwlin/legero-backend/internal/repo"
	"github.com/gofiber/fiber/v2"
)

type HttpServer struct {
	config *config.Config

	app *fiber.App

	tokenRepo repo.Token

	authHandler *handler.Auth
	userHandler *handler.User
}

func NewHttpServer(
	conf *config.Config,
	tokenRepo repo.Token,
	authHandler *handler.Auth,
	userHandler *handler.User,
) *HttpServer {

	app := fiber.New(fiber.Config{
		AppName: "Legero Backend",

		JSONEncoder: sonic.Marshal,
		JSONDecoder: sonic.Unmarshal,

		ErrorHandler: handler.Error,
	})

	srv := &HttpServer{
		config: conf,

		app: app,

		tokenRepo:   tokenRepo,
		authHandler: authHandler,
		userHandler: userHandler,
	}

	srv.SetupRoutes()

	return srv
}

func (s *HttpServer) SetupRoutes() {

	r := s.app

	r.Get("/ping", func(c *fiber.Ctx) error {
		return c.SendString("pong")
	})

	s.authHandler.RegisterRoutes(r)

	authMiddleware := middleware.Auth(s.tokenRepo)
	auth := r.Use(authMiddleware)
	s.userHandler.RegisterRoutes(auth)
}

func (s *HttpServer) Run() error {
	addr := fmt.Sprintf(":%d", s.config.HttpServer.Port)
	return s.app.Listen(addr)
}

func (s *HttpServer) ShutdownWithContext(ctx context.Context) error {
	if s.app == nil {
		return nil
	}
	return s.app.ShutdownWithContext(ctx)
}
