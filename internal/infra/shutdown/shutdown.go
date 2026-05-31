package shutdown

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// Handler manages graceful shutdown with signal handling and cleanup coordination.
type Handler struct {
	ctx    context.Context // app context, carries values like logger
	cancel context.CancelFunc
	done   <-chan struct{} // closed when signal is received
}

// New creates a new shutdown handler that listens for SIGINT and SIGTERM signals.
// The provided ctx is the application context that carries values (e.g., logger).
// It returns a Handler that can provide a signal context and coordinate cleanup.
func New(ctx context.Context) *Handler {
	signalCtx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	return &Handler{
		ctx:    ctx,
		cancel: cancel,
		done:   signalCtx.Done(),
	}
}

// Done returns a channel that is closed when a shutdown signal is received.
func (h *Handler) Done() <-chan struct{} {
	return h.done
}

// Shutdown executes cleanup functions in order with a timeout.
// It creates a timeout context derived from the app context, executes
// each function sequentially, then cancels the signal listener.
// If any function returns an error, it stops and returns that error.
// If the timeout is reached, it returns a context.DeadlineExceeded error.
func (h *Handler) Shutdown(timeout time.Duration, funcs ...func(context.Context) error) error {
	// Cancel signal listener to stop accepting new work
	h.cancel()

	// Create a timeout context from the app context.
	// Cleanup functions can access app context values (e.g., logger).
	ctx, cancel := context.WithTimeout(h.ctx, timeout)
	defer cancel()

	// Execute cleanup functions in order
	for _, f := range funcs {
		if err := f(ctx); err != nil {
			return err
		}
	}

	return nil
}
