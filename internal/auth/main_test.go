package auth

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/dongwlin/legero-backend/internal/infra/database"
	"github.com/dongwlin/legero-backend/internal/workspace"
	"github.com/dongwlin/legero-backend/migrations"
	"github.com/google/uuid"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/uptrace/bun"
)

var testDB *bun.DB

func TestMain(m *testing.M) {
	ctx := context.Background()

	name := "testdb"
	username := "postgres"
	password := "postgres"

	pgContainer, err := postgres.Run(ctx, "postgres:18",
		postgres.WithDatabase(name),
		postgres.WithUsername(username),
		postgres.WithPassword(password),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).WithStartupTimeout(5*time.Second),
		),
	)
	if err != nil {
		log.Fatalf("failed to start container: %v", err)
	}
	defer func() {
		if err = pgContainer.Terminate(ctx); err != nil {
			log.Printf("failed to terminate postgres container: %v", err)
		}
	}()

	host, _ := pgContainer.Host(ctx)
	port, _ := pgContainer.MappedPort(ctx, "5432")
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable&timezone=UTC",
		username,
		password,
		host,
		port.Port(),
		name,
	)

	if err := migrations.Migrate(dsn); err != nil {
		log.Printf("failed to run migrations: %v", err)
		return
	}

	testDB, err = database.New(ctx, database.Options{
		DSN: dsn,
	})
	if err != nil {
		log.Printf("failed to create database connection: %v", err)
		return
	}

	code := m.Run()

	testDB.Close()

	os.Exit(code)
}

func newTestUserRepo(t *testing.T, ctx context.Context) (*bun.Tx, *UserRepo) {
	t.Helper()

	tx, err := testDB.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("failed to begin transaction: %v", err)
	}

	t.Cleanup(func() {
		_ = tx.Rollback()
	})

	return &tx, NewUserRepo(&tx)
}

func newTestRefreshTokenRepo(t *testing.T, ctx context.Context) (*bun.Tx, *RefreshTokenRepo) {
	t.Helper()

	tx, err := testDB.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("failed to begin transaction: %v", err)
	}

	t.Cleanup(func() {
		_ = tx.Rollback()
	})

	return &tx, NewRefreshTokenRepo(&tx)
}

func newTestWorkspaceRepo(t *testing.T, ctx context.Context) (*bun.Tx, *workspace.WorkspaceRepo) {
	t.Helper()

	tx, err := testDB.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("failed to begin transaction: %v", err)
	}

	t.Cleanup(func() {
		_ = tx.Rollback()
	})

	return &tx, workspace.NewWorkspaceRepo(&tx)
}

func createTestUser(t *testing.T, ctx context.Context, db bun.IDB, opts ...func(*User)) uuid.UUID {
	t.Helper()

	user := User{
		ID:           uuid.New(),
		Phone:        fmt.Sprintf("1%s", uuid.New().String()[:11]),
		PasswordHash: MustHashForTests("password123"),
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	for _, opt := range opts {
		opt(&user)
	}

	model := UserModel{
		ID:           user.ID,
		Phone:        user.Phone,
		PasswordHash: user.PasswordHash,
		IsActive:     user.IsActive,
		CreatedAt:    user.CreatedAt,
		UpdatedAt:    user.UpdatedAt,
	}

	_, err := db.NewInsert().Model(&model).Exec(ctx)
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	return user.ID
}

func createTestWorkspace(t *testing.T, ctx context.Context, db bun.IDB) uuid.UUID {
	t.Helper()

	workspaceID := uuid.New()
	_, err := db.NewRaw(
		"INSERT INTO workspaces (id, name) VALUES (?, ?)",
		workspaceID, "test-workspace",
	).Exec(ctx)
	if err != nil {
		t.Fatalf("failed to create test workspace: %v", err)
	}

	return workspaceID
}

func createTestWorkspaceMember(t *testing.T, ctx context.Context, db bun.IDB, userID, workspaceID uuid.UUID, role string) {
	t.Helper()

	_, err := db.NewRaw(
		"INSERT INTO workspace_members (workspace_id, user_id, role, created_at) VALUES (?, ?, ?, ?)",
		workspaceID, userID, role, time.Now(),
	).Exec(ctx)
	if err != nil {
		t.Fatalf("failed to create test workspace member: %v", err)
	}
}

func createTestRefreshToken(t *testing.T, ctx context.Context, db bun.IDB, userID, workspaceID uuid.UUID, opts ...func(*RefreshToken)) RefreshToken {
	t.Helper()

	token := RefreshToken{
		ID:          uuid.New(),
		UserID:      userID,
		WorkspaceID: workspaceID,
		TokenHash:   fmt.Sprintf("hash-%s", uuid.New().String()),
		ExpiresAt:   time.Now().Add(7 * 24 * time.Hour),
		CreatedAt:   time.Now(),
	}

	for _, opt := range opts {
		opt(&token)
	}

	model := RefreshTokenModel{
		ID:           token.ID,
		UserID:       token.UserID,
		WorkspaceID:  token.WorkspaceID,
		TokenHash:    token.TokenHash,
		ExpiresAt:    token.ExpiresAt,
		CreatedAt:    token.CreatedAt,
		RotatedAt:    token.RotatedAt,
		RevokedAt:    token.RevokedAt,
		ReplacedByID: token.ReplacedByID,
	}

	_, err := db.NewInsert().Model(&model).Exec(ctx)
	if err != nil {
		t.Fatalf("failed to create test refresh token: %v", err)
	}

	return token
}
