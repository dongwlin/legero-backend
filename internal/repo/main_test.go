package repo

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/dongwlin/legero-backend/internal/infra/database"
	"github.com/dongwlin/legero-backend/internal/model"
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

func newTestUserRepo(t *testing.T, ctx context.Context) (*bun.Tx, *User) {
	t.Helper()

	tx, err := testDB.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("failed to begin transaction: %v", err)
	}

	t.Cleanup(func() {
		_ = tx.Rollback()
	})

	return &tx, NewUser(&tx)
}

func newTestRefreshTokenRepo(t *testing.T, ctx context.Context) (*bun.Tx, *RefreshToken) {
	t.Helper()

	tx, err := testDB.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("failed to begin transaction: %v", err)
	}

	t.Cleanup(func() {
		_ = tx.Rollback()
	})

	return &tx, NewRefreshToken(&tx)
}

func newTestOrderRepo(t *testing.T, ctx context.Context) (*bun.Tx, *Order) {
	t.Helper()

	tx, err := testDB.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("failed to begin transaction: %v", err)
	}

	t.Cleanup(func() {
		_ = tx.Rollback()
	})

	return &tx, NewOrder(&tx)
}

func newTestCounterRepo(t *testing.T, ctx context.Context) (*bun.Tx, *Counter) {
	t.Helper()

	tx, err := testDB.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("failed to begin transaction: %v", err)
	}

	t.Cleanup(func() {
		_ = tx.Rollback()
	})

	return &tx, NewCounter(&tx)
}

func createTestUser(t *testing.T, ctx context.Context, db bun.IDB, opts ...func(*model.User)) uuid.UUID {
	t.Helper()

	user := model.User{
		ID:           uuid.New(),
		Phone:        fmt.Sprintf("1%s", uuid.New().String()[:11]),
		PasswordHash: "$argon2id$v=19$m=65536,t=3,p=2$c2FsdHNhbHRzYWx0$hash",
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	for _, opt := range opts {
		opt(&user)
	}

	_, err := db.NewInsert().Model(&user).Exec(ctx)
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

func createTestRefreshToken(t *testing.T, ctx context.Context, db bun.IDB, userID, workspaceID uuid.UUID, opts ...func(*model.RefreshToken)) model.RefreshToken {
	t.Helper()

	token := model.RefreshToken{
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

	_, err := db.NewInsert().Model(&token).Exec(ctx)
	if err != nil {
		t.Fatalf("failed to create test refresh token: %v", err)
	}

	return token
}

func createTestOrder(t *testing.T, ctx context.Context, db bun.IDB, workspaceID, userID uuid.UUID, opts ...func(*model.Order)) model.Order {
	t.Helper()

	now := time.Now()

	order := model.Order{
		ID:                   uuid.New(),
		WorkspaceID:          workspaceID,
		DisplayNo:            "T001",
		SizeCode:             model.SizeSmall,
		StapleAmountCode:     model.AdjustmentNormal,
		GreensCode:           model.AdjustmentNormal,
		ScallionCode:         model.AdjustmentNormal,
		PepperCode:           model.AdjustmentNormal,
		DiningMethodCode:     model.DiningMethodDineIn,
		SelectedMeatCodes:    []int16{model.MeatLeanPork},
		TotalPriceCents:      1000,
		StapleStepStatusCode: model.StepStatusUnrequired,
		MeatStepStatusCode:   model.StepStatusUnrequired,
		CreatedBy:            userID,
		UpdatedBy:            userID,
		CreatedAt:            now,
		UpdatedAt:            now,
	}

	for _, opt := range opts {
		opt(&order)
	}

	_, err := db.NewInsert().Model(&order).Exec(ctx)
	if err != nil {
		t.Fatalf("failed to create test order: %v", err)
	}

	return order
}
