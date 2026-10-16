// Package performance collects bounded, opt-in request timing without storing
// request payloads, SQL, query arguments, credentials, or business identifiers.
package performance

import (
	"context"
	"sync"
	"time"

	"github.com/uptrace/bun"
)

type contextKey struct{}

// Trace aggregates completed Bun query intervals for one request. Concurrent
// queries are summed, so DB time need not be less than request wall time.
type Trace struct {
	mu      sync.Mutex
	dbTime  time.Duration
	queries uint64
	closed  bool
}

type Snapshot struct {
	DBTime  time.Duration
	Queries uint64
}

// Start adds a fresh request-local trace. Ordinary requests do not call Start.
func Start(ctx context.Context) (context.Context, *Trace) {
	trace := new(Trace)
	return context.WithValue(ctx, contextKey{}, trace), trace
}

func FromContext(ctx context.Context) *Trace {
	trace, _ := ctx.Value(contextKey{}).(*Trace)
	return trace
}

// RecordQuery records only duration and count, including failed queries.
func (t *Trace) RecordQuery(elapsed time.Duration) {
	if elapsed < 0 {
		elapsed = 0
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.closed {
		t.dbTime += elapsed
		t.queries++
	}
}

// Finish freezes the snapshot at the response-header boundary. Work completing
// after headers have been sent cannot be represented in this header.
func (t *Trace) Finish() Snapshot {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.closed = true
	return Snapshot{DBTime: t.dbTime, Queries: t.queries}
}

type QueryHook struct {
	now func() time.Time
}

var _ bun.QueryHook = (*QueryHook)(nil)

func (h *QueryHook) BeforeQuery(ctx context.Context, _ *bun.QueryEvent) context.Context {
	return ctx
}

func (h *QueryHook) AfterQuery(ctx context.Context, event *bun.QueryEvent) {
	trace := FromContext(ctx)
	if trace == nil {
		return
	}
	now := time.Now
	if h.now != nil {
		now = h.now
	}
	trace.RecordQuery(now().Sub(event.StartTime))
}
