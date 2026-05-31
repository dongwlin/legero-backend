package main

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/dongwlin/legero-backend/internal/auth"
	"github.com/dongwlin/legero-backend/internal/infra/config"
	"github.com/dongwlin/legero-backend/internal/infra/database"
	"github.com/dongwlin/legero-backend/internal/workspace"
)

// create-user command flags
const (
	flagPhone       = "phone"
	flagPassword    = "password"
	flagWorkspace   = "workspace"
	flagWorkspaceID = "workspace-id"
	flagRole        = "role"
)

var createUserCmd = &cobra.Command{
	Use:   "create-user",
	Short: "Create a user with workspace membership",
	Long:  "Create a new user with phone and password, optionally creating a new workspace or attaching to an existing one.",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runCreateUser(cmd)
	},
}

func init() {
	createUserCmd.Flags().String(flagPhone, "", "login phone number")
	createUserCmd.Flags().String(flagPassword, "", "login password")
	createUserCmd.Flags().String(flagWorkspace, "", "workspace name to create when workspace-id is omitted")
	createUserCmd.Flags().String(flagWorkspaceID, "", "existing workspace id to attach the user to")
	createUserCmd.Flags().String(flagRole, string(workspace.RoleOwner), "membership role: owner or staff")

	// Mark required flags
	_ = createUserCmd.MarkFlagRequired(flagPhone)
	_ = createUserCmd.MarkFlagRequired(flagPassword)

	rootCmd.AddCommand(createUserCmd)
}

func runCreateUser(cmd *cobra.Command) error {
	// Get flag values
	phone, _ := cmd.Flags().GetString(flagPhone)
	password, _ := cmd.Flags().GetString(flagPassword)
	workspaceName, _ := cmd.Flags().GetString(flagWorkspace)
	workspaceIDText, _ := cmd.Flags().GetString(flagWorkspaceID)
	roleText, _ := cmd.Flags().GetString(flagRole)

	// Load config
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Parse workspace ID if provided
	var workspaceID *uuid.UUID
	if workspaceIDText != "" {
		parsed, err := uuid.Parse(workspaceIDText)
		if err != nil {
			return fmt.Errorf("parse --workspace-id: %w", err)
		}
		workspaceID = &parsed
	}

	// Create context
	ctx := context.Background()

	// Connect to database
	db, err := database.New(ctx, database.Options{DSN: cfg.DatabaseURL})
	if err != nil {
		return err
	}
	defer func() {
		_ = db.Close()
	}()

	// Create user service
	service := auth.NewBootstrapUserService(
		db,
		auth.NewPasswordHasher(cfg.Argon2),
	)

	// Create user
	result, err := service.CreateUser(ctx, auth.CreateUserInput{
		Phone:       phone,
		Password:    password,
		WorkspaceID: workspaceID,
		Workspace:   workspaceName,
		Role:        workspace.Role(roleText),
	})
	if err != nil {
		return err
	}

	// Print result
	fmt.Printf("user_id=%s\n", result.UserID)
	fmt.Printf("phone=%s\n", result.Phone)
	fmt.Printf("workspace_id=%s\n", result.WorkspaceID)
	fmt.Printf("workspace=%s\n", result.WorkspaceName)
	fmt.Printf("role=%s\n", result.Role)
	fmt.Printf("created_workspace=%t\n", result.CreatedWorkspace)

	return nil
}
