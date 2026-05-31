package order

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/dongwlin/legero-backend/internal/infra/database"
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

func newTestOrderRepo(t *testing.T, ctx context.Context) (*bun.Tx, *OrderRepo) {
	t.Helper()

	tx, err := testDB.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("failed to begin transaction: %v", err)
	}

	t.Cleanup(func() {
		_ = tx.Rollback()
	})

	return &tx, NewOrderRepo(&tx)
}

func newTestCounterRepo(t *testing.T, ctx context.Context) (*bun.Tx, *CounterRepo) {
	t.Helper()

	tx, err := testDB.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("failed to begin transaction: %v", err)
	}

	t.Cleanup(func() {
		_ = tx.Rollback()
	})

	return &tx, NewCounterRepo(&tx)
}

func createTestUser(t *testing.T, ctx context.Context, db bun.IDB) uuid.UUID {
	t.Helper()

	userID := uuid.New()
	_, err := db.NewRaw(
		"INSERT INTO users (id, phone, password_hash) VALUES (?, ?, ?)",
		userID, fmt.Sprintf("1%s", userID.String()[:8]), "hash",
	).Exec(ctx)
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	return userID
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

func createTestOrder(t *testing.T, ctx context.Context, db bun.IDB, workspaceID, userID uuid.UUID, opts ...func(*Order)) Order {
	t.Helper()

	now := time.Now()

	order := Order{
		ID:                   uuid.New(),
		WorkspaceID:          workspaceID,
		DisplayNo:            "T001",
		SizeCode:             SizeSmall,
		StapleAmountCode:     AdjustmentNormal,
		GreensCode:           AdjustmentNormal,
		ScallionCode:         AdjustmentNormal,
		PepperCode:           AdjustmentNormal,
		DiningMethodCode:     DiningMethodDineIn,
		TotalPriceCents:      1000,
		StapleStepStatusCode: StepStatusUnrequired,
		MeatStepStatusCode:   StepStatusUnrequired,
		CreatedBy:            userID,
		UpdatedBy:            userID,
		CreatedAt:            now,
		UpdatedAt:            now,
	}

	for _, opt := range opts {
		opt(&order)
	}

	model := orderToModel(order)

	_, err := db.NewInsert().
		Model(&model).
		Exec(ctx)
	if err != nil {
		t.Fatalf("failed to create test order: %v", err)
	}

	return modelToOrder(model)
}
