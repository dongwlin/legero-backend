package migrations

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

var dsn string

func TestMain(m *testing.M) {
	ctx := context.Background()

	pgContainer, err := postgres.Run(ctx, "postgres:18",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("postgres"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).WithStartupTimeout(5*time.Second),
		),
	)
	if err != nil {
		log.Fatalf("failed to start container: %v", err)
	}
	defer func() {
		if err := pgContainer.Terminate(ctx); err != nil {
			log.Printf("failed to terminate postgres container: %v", err)
		}
	}()

	host, _ := pgContainer.Host(ctx)
	port, _ := pgContainer.MappedPort(ctx, "5432")
	dsn = fmt.Sprintf("postgres://postgres:postgres@%s:%s/testdb?sslmode=disable&timezone=UTC",
		host, port.Port())

	os.Exit(m.Run())
}

// migrateInstance is a fresh migrate instance over the embedded SQL files.
func migrateInstance() (*migrate.Migrate, error) {
	source, err := iofs.New(migrations, ".")
	if err != nil {
		return nil, err
	}
	return migrate.NewWithSourceInstance("iofs", source, dsn)
}

func versionColumnCount(t *testing.T, ctx context.Context) int {
	t.Helper()
	conn, err := pgx.Connect(ctx, dsn)
	require.NoError(t, err)
	defer conn.Close(ctx)

	var count int
	err = conn.QueryRow(ctx, `
		SELECT count(*)
		FROM information_schema.columns
		WHERE column_name = 'version'
		  AND table_name IN ('orders', 'users', 'workspaces', 'refresh_tokens')`).Scan(&count)
	require.NoError(t, err)
	return count
}

// TestMigrations_RoundTrip applies all migrations, asserts the version
// columns exist on every versioned table, rolls everything back with
// migrate.Down, and re-applies them, leaving the schema in the same
// migratable state.
func TestMigrations_RoundTrip(t *testing.T) {
	ctx := context.Background()

	require.NoError(t, Migrate(dsn))
	require.Equal(t, 4, versionColumnCount(t, ctx), "each versioned table has a version column")

	m, err := migrateInstance()
	require.NoError(t, err)
	require.NoError(t, m.Down())
	require.Equal(t, 0, versionColumnCount(t, ctx), "down migration removes version columns")

	require.NoError(t, Migrate(dsn))
	require.Equal(t, 4, versionColumnCount(t, ctx))
}