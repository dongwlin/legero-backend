package performance

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
)

func TestQueryHookAggregatesOnlyRequestLocalIntervals(t *testing.T) {
	ctx, trace := Start(context.Background())
	otherCtx, other := Start(context.Background())
	now := time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC)
	hook := &QueryHook{now: func() time.Time { return now }}
	query := &bun.QueryEvent{StartTime: now.Add(-7 * time.Millisecond), Query: "PRIVATE SQL", Err: errors.New("PRIVATE ERROR")}
	require.Same(t, ctx, hook.BeforeQuery(ctx, query))
	hook.AfterQuery(ctx, query)
	hook.AfterQuery(ctx, &bun.QueryEvent{StartTime: now.Add(-3 * time.Millisecond)})
	hook.AfterQuery(otherCtx, &bun.QueryEvent{StartTime: now.Add(-2 * time.Millisecond)})
	hook.AfterQuery(context.Background(), query)
	require.Equal(t, Snapshot{DBTime: 10 * time.Millisecond, Queries: 2}, trace.Finish())
	require.Equal(t, Snapshot{DBTime: 2 * time.Millisecond, Queries: 1}, other.Finish())
	// Query text, arguments and errors are never copied to the trace.
	require.Equal(t, "PRIVATE SQL", query.Query)
	require.Nil(t, FromContext(context.Background()))
}

func TestTraceFreezesAtHeadersAndClampsInvalidIntervals(t *testing.T) {
	_, trace := Start(context.Background())
	trace.RecordQuery(-time.Second)
	require.Equal(t, Snapshot{Queries: 1}, trace.Finish())
	trace.RecordQuery(time.Hour)
	require.Equal(t, Snapshot{Queries: 1}, trace.Finish())
}

func TestTraceSupportsConcurrentQueries(t *testing.T) {
	_, trace := Start(context.Background())
	var group sync.WaitGroup
	for range 100 {
		group.Go(func() { trace.RecordQuery(time.Millisecond) })
	}
	group.Wait()
	require.Equal(t, Snapshot{DBTime: 100 * time.Millisecond, Queries: 100}, trace.Finish())
}

func BenchmarkQueryHookDisabled(b *testing.B) {
	hook := &QueryHook{}
	ctx := context.Background()
	event := &bun.QueryEvent{}
	b.ReportAllocs()
	for b.Loop() {
		hook.BeforeQuery(ctx, event)
		hook.AfterQuery(ctx, event)
	}
}

// This no-I/O driver measures Bun's event allocation/clock overhead as well as
// our hook. It does not estimate PostgreSQL or production HTTP overhead.
type benchmarkDriver struct{}
type benchmarkConnection struct{}

func (benchmarkDriver) Open(string) (driver.Conn, error) { return benchmarkConnection{}, nil }
func (benchmarkConnection) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("prepare unsupported")
}
func (benchmarkConnection) Close() error { return nil }
func (benchmarkConnection) Begin() (driver.Tx, error) {
	return nil, errors.New("transactions unsupported")
}
func (benchmarkConnection) ExecContext(context.Context, string, []driver.NamedValue) (driver.Result, error) {
	return driver.RowsAffected(1), nil
}

func init() { sql.Register("performance-benchmark-no-io", benchmarkDriver{}) }

func BenchmarkBunQueryInstrumentation(b *testing.B) {
	for _, mode := range []string{"no-hook", "disabled", "enabled"} {
		b.Run(mode, func(b *testing.B) {
			sqlDB, err := sql.Open("performance-benchmark-no-io", "")
			require.NoError(b, err)
			db := bun.NewDB(sqlDB, pgdialect.New())
			b.Cleanup(func() { _ = db.Close() })
			if mode != "no-hook" {
				db.AddQueryHook(&QueryHook{})
			}
			ctx := context.Background()
			if mode == "enabled" {
				ctx, _ = Start(ctx)
			}
			b.ReportAllocs()
			for b.Loop() {
				_, err := db.ExecContext(ctx, "SELECT 1")
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
