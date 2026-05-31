package shutdown

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	ctx := context.Background()
	handler := New(ctx)
	if handler == nil {
		t.Fatal("New() returned nil")
	}
	if handler.ctx == nil {
		t.Fatal("handler.ctx is nil")
	}
	if handler.cancel == nil {
		t.Fatal("handler.cancel is nil")
	}
	if handler.done == nil {
		t.Fatal("handler.done is nil")
	}
}

func TestDone(t *testing.T) {
	ctx := context.Background()
	handler := New(ctx)
	done := handler.Done()
	if done == nil {
		t.Fatal("Done() returned nil")
	}
	// Channel should not be closed initially
	select {
	case <-done:
		t.Fatal("Done channel is already closed")
	default:
		// Good, channel is not closed
	}
}

func TestShutdownWithNoFunctions(t *testing.T) {
	ctx := context.Background()
	handler := New(ctx)
	err := handler.Shutdown(5 * time.Second)
	if err != nil {
		t.Fatalf("Shutdown() with no functions returned error: %v", err)
	}
	// Done channel should be closed after shutdown
	select {
	case <-handler.Done():
		// Good, channel is closed
	default:
		t.Fatal("Done channel is not closed after shutdown")
	}
}

func TestShutdownWithFunctions(t *testing.T) {
	ctx := context.Background()
	handler := New(ctx)
	executed := false
	cleanup := func(ctx context.Context) error {
		executed = true
		return nil
	}
	err := handler.Shutdown(5*time.Second, cleanup)
	if err != nil {
		t.Fatalf("Shutdown() with function returned error: %v", err)
	}
	if !executed {
		t.Fatal("Cleanup function was not executed")
	}
}

func TestShutdownWithMultipleFunctions(t *testing.T) {
	ctx := context.Background()
	handler := New(ctx)
	order := []int{}
	cleanup1 := func(ctx context.Context) error {
		order = append(order, 1)
		return nil
	}
	cleanup2 := func(ctx context.Context) error {
		order = append(order, 2)
		return nil
	}
	err := handler.Shutdown(5*time.Second, cleanup1, cleanup2)
	if err != nil {
		t.Fatalf("Shutdown() with multiple functions returned error: %v", err)
	}
	if len(order) != 2 || order[0] != 1 || order[1] != 2 {
		t.Fatalf("Functions executed in wrong order: %v", order)
	}
}

func TestShutdownWithFunctionError(t *testing.T) {
	ctx := context.Background()
	handler := New(ctx)
	expectedErr := errors.New("cleanup failed")
	cleanup := func(ctx context.Context) error {
		return expectedErr
	}
	err := handler.Shutdown(5*time.Second, cleanup)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("Shutdown() returned wrong error: got %v, want %v", err, expectedErr)
	}
}

func TestShutdownStopsOnFirstError(t *testing.T) {
	ctx := context.Background()
	handler := New(ctx)
	executed := false
	cleanup1 := func(ctx context.Context) error {
		return errors.New("first fails")
	}
	cleanup2 := func(ctx context.Context) error {
		executed = true
		return nil
	}
	_ = handler.Shutdown(5*time.Second, cleanup1, cleanup2)
	if executed {
		t.Fatal("Second function should not have been executed after first failed")
	}
}

func TestShutdownWithTimeout(t *testing.T) {
	ctx := context.Background()
	handler := New(ctx)
	slow := func(ctx context.Context) error {
		select {
		case <-time.After(2 * time.Second):
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	err := handler.Shutdown(100*time.Millisecond, slow)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Shutdown() should have timed out, got: %v", err)
	}
}

// contextKey is an unexported type for context keys to avoid collisions.
type contextKey struct{}

func TestShutdownPreservesContextValues(t *testing.T) {
	// Create a context with values
	ctx := context.WithValue(context.Background(), contextKey{}, "test-value")
	handler := New(ctx)

	var capturedValue any
	cleanup := func(ctx context.Context) error {
		capturedValue = ctx.Value(contextKey{})
		return nil
	}

	err := handler.Shutdown(5*time.Second, cleanup)
	if err != nil {
		t.Fatalf("Shutdown() returned error: %v", err)
	}
	if capturedValue != "test-value" {
		t.Fatalf("Cleanup function lost context value: got %v, want %v", capturedValue, "test-value")
	}
}

func TestShutdownWithContextCancellation(t *testing.T) {
	// Create a handler with a pre-cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	handler := New(ctx)

	// Shutdown should still work even if parent context is cancelled
	executed := false
	cleanup := func(ctx context.Context) error {
		executed = true
		return nil
	}

	err := handler.Shutdown(5*time.Second, cleanup)
	if err != nil {
		t.Fatalf("Shutdown() returned error: %v", err)
	}
	if !executed {
		t.Fatal("Cleanup function was not executed")
	}
}
