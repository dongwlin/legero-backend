package repo

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/dongwlin/legero-backend/internal/domain"
	"github.com/dongwlin/legero-backend/internal/infra/database"
	"github.com/dongwlin/legero-backend/internal/repo/schema"
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

func createTestUser(t *testing.T, ctx context.Context, db bun.IDB, opts ...func(*domain.User)) uuid.UUID {
	t.Helper()

	user := domain.User{
		ID:           uuid.New(),
		Phone:        fmt.Sprintf("1%s", uuid.New().String()[:11]),
		PasswordHash: "$argon2id$v=19$m=65536,t=3,p=2$c2FsdHNhbHRzYWx0$hash",
		IsActive:     true,
		Version:      1,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	for _, opt := range opts {
		opt(&user)
	}

	if _, err := db.NewInsert().Model(toSchemaUser(&user)).Exec(ctx); err != nil {
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

func createTestRefreshToken(t *testing.T, ctx context.Context, db bun.IDB, userID, workspaceID uuid.UUID, opts ...func(*domain.RefreshToken)) domain.RefreshToken {
	t.Helper()

	token := domain.RefreshToken{
		ID:          uuid.New(),
		UserID:      userID,
		WorkspaceID: workspaceID,
		TokenHash:   fmt.Sprintf("hash-%s", uuid.New().String()),
		ExpiresAt:   time.Now().Add(7 * 24 * time.Hour),
		Version:     1,
		CreatedAt:   time.Now(),
	}

	for _, opt := range opts {
		opt(&token)
	}

	s := &schema.RefreshToken{
		ID:           token.ID,
		UserID:       token.UserID,
		WorkspaceID:  token.WorkspaceID,
		TokenHash:    token.TokenHash,
		ExpiresAt:    token.ExpiresAt,
		Version:      token.Version,
		CreatedAt:    token.CreatedAt,
		RotatedAt:    token.RotatedAt,
		RevokedAt:    token.RevokedAt,
		ReplacedByID: token.ReplacedByID,
	}
	if _, err := db.NewInsert().Model(s).Exec(ctx); err != nil {
		t.Fatalf("failed to create test refresh token: %v", err)
	}

	return token
}

func createTestOrder(t *testing.T, ctx context.Context, db bun.IDB, workspaceID, userID uuid.UUID, opts ...func(*domain.Order)) domain.Order {
	t.Helper()

	now := time.Now()

	order := domain.Order{
		ID:                   uuid.New(),
		WorkspaceID:          workspaceID,
		DisplayNo:            "T001",
		Version:              1,
		SizeCode:             domain.SizeSmall,
		StapleAmountCode:     domain.AdjustmentNormal,
		GreensCode:           domain.AdjustmentNormal,
		ScallionCode:         domain.AdjustmentNormal,
		PepperCode:           domain.AdjustmentNormal,
		DiningMethodCode:     domain.DiningMethodDineIn,
		SelectedMeatCodes:    []int16{domain.MeatLeanPork},
		TotalPriceCents:      1000,
		StapleStepStatusCode: domain.StepStatusUnrequired,
		MeatStepStatusCode:   domain.StepStatusUnrequired,
		CreatedBy:            userID,
		UpdatedBy:            userID,
		CreatedAt:            now,
		UpdatedAt:            now,
	}

	for _, opt := range opts {
		opt(&order)
	}

	if _, err := db.NewInsert().Model(toSchemaOrder(&order)).Exec(ctx); err != nil {
		t.Fatalf("failed to create test order: %v", err)
	}

	return order
}
