package order

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"

	"github.com/dongwlin/legero-backend/internal/infra/httpx"
	"github.com/dongwlin/legero-backend/internal/workspace"
)

func TestServiceClearWorkspaceRejectsInvalidMode(t *testing.T) {
	service := &Service{}
	actor := Actor{
		WorkspaceID: uuid.New(),
		Role:        workspace.RoleOwner,
	}

	_, err := service.ClearWorkspace(context.Background(), actor, true, ClearWorkspaceMode("unknown"))
	assertAppError(t, err, 400, "validation_failed")
}

func TestServiceClearWorkspaceRejectsUnconfirmedRequest(t *testing.T) {
	service := &Service{}
	actor := Actor{
		WorkspaceID: uuid.New(),
		Role:        workspace.RoleOwner,
	}

	_, err := service.ClearWorkspace(context.Background(), actor, false, ClearWorkspaceModeAll)
	assertAppError(t, err, 400, "validation_failed")
}

func TestServiceClearWorkspaceRejectsStaffRole(t *testing.T) {
	service := &Service{}
	actor := Actor{
		WorkspaceID: uuid.New(),
		Role:        workspace.RoleStaff,
	}

	_, err := service.ClearWorkspace(context.Background(), actor, true, ClearWorkspaceModeAll)
	assertAppError(t, err, 403, "forbidden")
}

func TestServiceClearWorkspaceInTxPassesBeforeTodayCutoffAndPublishesEvent(t *testing.T) {
	actor := Actor{
		WorkspaceID: uuid.New(),
		Role:        workspace.RoleOwner,
	}
	expectedCutoff := time.Date(2026, time.April, 26, 0, 0, 0, 0, time.FixedZone("UTC+8", 8*60*60))
	repo := &clearWorkspaceRepoSpy{count: 4}
	counters := &clearWorkspaceCounterSpy{}
	publisher := &publisherSpy{}
	service := &Service{
		repo:      repo,
		counters:  counters,
		publisher: publisher,
	}

	count, err := service.clearWorkspaceInTx(
		context.Background(),
		nil,
		actor,
		ClearWorkspaceModeBeforeToday,
		&expectedCutoff,
	)
	if err != nil {
		t.Fatalf("clearWorkspaceInTx() error = %v", err)
	}
	if count != 4 {
		t.Fatalf("clearWorkspaceInTx() count = %d, want 4", count)
	}
	if !repo.called {
		t.Fatal("expected repository clear to be called")
	}
	if repo.workspaceID != actor.WorkspaceID {
		t.Fatalf("repository workspaceID = %s, want %s", repo.workspaceID, actor.WorkspaceID)
	}
	if repo.createdBefore == nil || !repo.createdBefore.Equal(expectedCutoff) {
		t.Fatalf("repository createdBefore = %v, want %v", repo.createdBefore, expectedCutoff)
	}
	if counters.bizDateBefore == nil || !counters.bizDateBefore.Equal(expectedCutoff) {
		t.Fatalf("counter reset bizDateBefore = %v, want %v", counters.bizDateBefore, expectedCutoff)
	}
	if len(publisher.events) != 1 {
		t.Fatalf("publisher events = %d, want 1", len(publisher.events))
	}
	event, ok := publisher.events[0].payload.(ClearedEvent)
	if !ok {
		t.Fatalf("publisher payload type = %T, want ClearedEvent", publisher.events[0].payload)
	}
	if event.ClearedCount != 4 {
		t.Fatalf("publisher clearedCount = %d, want 4", event.ClearedCount)
	}
	if event.Mode != ClearWorkspaceModeBeforeToday {
		t.Fatalf("publisher mode = %q, want %q", event.Mode, ClearWorkspaceModeBeforeToday)
	}
}

func TestServiceClearWorkspaceInTxSkipsPublishWhenCounterResetFails(t *testing.T) {
	actor := Actor{
		WorkspaceID: uuid.New(),
		Role:        workspace.RoleOwner,
	}
	repo := &clearWorkspaceRepoSpy{count: 2}
	counterErr := errors.New("counter reset failed")
	counters := &clearWorkspaceCounterSpy{err: counterErr}
	publisher := &publisherSpy{}
	service := &Service{
		repo:      repo,
		counters:  counters,
		publisher: publisher,
	}

	count, err := service.clearWorkspaceInTx(
		context.Background(),
		nil,
		actor,
		ClearWorkspaceModeAll,
		nil,
	)
	if !errors.Is(err, counterErr) {
		t.Fatalf("clearWorkspaceInTx() error = %v, want %v", err, counterErr)
	}
	if count != 0 {
		t.Fatalf("clearWorkspaceInTx() count = %d, want 0 on failure", count)
	}
	if !repo.called {
		t.Fatal("expected repository clear to be called")
	}
	if len(publisher.events) != 0 {
		t.Fatalf("publisher events = %d, want 0", len(publisher.events))
	}
}

type clearWorkspaceRepoSpy struct {
	called        bool
	workspaceID   uuid.UUID
	createdBefore *time.Time
	count         int
	err           error
}

func (r *clearWorkspaceRepoSpy) List(context.Context, bun.IDB, uuid.UUID, ListOrdersQuery) ([]Order, *string, error) {
	panic("unexpected List call")
}

func (r *clearWorkspaceRepoSpy) ListActive(context.Context, bun.IDB, uuid.UUID) ([]Order, error) {
	panic("unexpected ListActive call")
}

func (r *clearWorkspaceRepoSpy) GetByID(context.Context, bun.IDB, uuid.UUID, uuid.UUID) (*Order, error) {
	panic("unexpected GetByID call")
}

func (r *clearWorkspaceRepoSpy) InsertMany(context.Context, bun.IDB, []Order) error {
	panic("unexpected InsertMany call")
}

func (r *clearWorkspaceRepoSpy) Update(context.Context, bun.IDB, Order) error {
	panic("unexpected Update call")
}

func (r *clearWorkspaceRepoSpy) Delete(context.Context, bun.IDB, uuid.UUID, uuid.UUID) (bool, error) {
	panic("unexpected Delete call")
}

func (r *clearWorkspaceRepoSpy) ClearWorkspace(_ context.Context, _ bun.IDB, workspaceID uuid.UUID, createdBefore *time.Time) (int, error) {
	r.called = true
	r.workspaceID = workspaceID
	r.createdBefore = createdBefore
	if r.err != nil {
		return 0, r.err
	}
	return r.count, nil
}

type clearWorkspaceCounterSpy struct {
	bizDateBefore *time.Time
	err           error
}

func (r *clearWorkspaceCounterSpy) Allocate(context.Context, bun.IDB, uuid.UUID, time.Time, int, time.Time) (int, error) {
	panic("unexpected Allocate call")
}

func (r *clearWorkspaceCounterSpy) ResetWorkspace(_ context.Context, _ bun.IDB, _ uuid.UUID, bizDateBefore *time.Time) error {
	r.bizDateBefore = bizDateBefore
	return r.err
}

type publisherEvent struct {
	workspaceID uuid.UUID
	eventType   string
	payload     any
}

type publisherSpy struct {
	events []publisherEvent
}

func (p *publisherSpy) Publish(workspaceID uuid.UUID, eventType string, payload any) {
	p.events = append(p.events, publisherEvent{
		workspaceID: workspaceID,
		eventType:   eventType,
		payload:     payload,
	})
}

func assertAppError(t *testing.T, err error, status int, code string) {
	t.Helper()

	var appErr *httpx.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected *httpx.AppError, got %v", err)
	}
	if appErr.Status != status {
		t.Fatalf("appErr.Status = %d, want %d", appErr.Status, status)
	}
	if appErr.Code != code {
		t.Fatalf("appErr.Code = %q, want %q", appErr.Code, code)
	}
}
