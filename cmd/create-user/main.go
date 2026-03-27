package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/dongwlin/legero-backend/internal/auth"
	"github.com/dongwlin/legero-backend/internal/infra/cli"
	"github.com/dongwlin/legero-backend/internal/infra/clock"
	"github.com/dongwlin/legero-backend/internal/infra/config"
	dbpkg "github.com/dongwlin/legero-backend/internal/infra/db"
	"github.com/dongwlin/legero-backend/internal/infra/ids"
	"github.com/dongwlin/legero-backend/internal/workspace"
)

func main() {
	cli.Run(run)
}

func run(ctx context.Context, cfg *config.Config, _ zerolog.Logger, args []string) error {
	flags := flag.NewFlagSet("create-user", flag.ContinueOnError)
	flags.SetOutput(os.Stdout)

	phone := flags.String("phone", "", "login phone number")
	password := flags.String("password", "", "login password")
	workspaceName := flags.String("workspace", "", "workspace name to create when workspace-id is omitted")
	workspaceIDText := flags.String("workspace-id", "", "existing workspace id to attach the user to")
	roleText := flags.String("role", string(workspace.RoleOwner), "membership role: owner or staff")

	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return fmt.Errorf("parse create-user flags: %w", err)
	}

	var workspaceID *uuid.UUID
	if *workspaceIDText != "" {
		parsed, err := uuid.Parse(*workspaceIDText)
		if err != nil {
			return fmt.Errorf("parse --workspace-id: %w", err)
		}
		workspaceID = &parsed
	}

	database, err := dbpkg.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer func() {
		_ = database.Close()
	}()

	service := auth.NewBootstrapUserService(
		database,
		auth.NewPasswordHasher(cfg.Argon2),
		clock.SystemClock{},
		ids.UUIDGenerator{},
	)

	result, err := service.CreateUser(ctx, auth.CreateUserInput{
		Phone:       *phone,
		Password:    *password,
		WorkspaceID: workspaceID,
		Workspace:   *workspaceName,
		Role:        workspace.Role(*roleText),
	})
	if err != nil {
		return err
	}

	fmt.Printf("user_id=%s\n", result.UserID)
	fmt.Printf("phone=%s\n", result.Phone)
	fmt.Printf("workspace_id=%s\n", result.WorkspaceID)
	fmt.Printf("workspace=%s\n", result.WorkspaceName)
	fmt.Printf("role=%s\n", result.Role)
	fmt.Printf("created_workspace=%t\n", result.CreatedWorkspace)

	return nil
}
