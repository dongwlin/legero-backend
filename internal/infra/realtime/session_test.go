package realtime

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/dongwlin/legero-backend/internal/infra/httpx"
	"github.com/dongwlin/legero-backend/internal/infra/identity"
)

func TestSessionManagerIssueAndConsumeSingleUse(t *testing.T) {
	now := time.Date(2026, time.April, 6, 10, 0, 0, 0, time.UTC)
	manager := NewSessionManager(30*time.Second, func() time.Time {
		return now
	})

	expected := &identity.Context{
		UserID:      uuid.New(),
		WorkspaceID: uuid.New(),
		Role:        "staff",
	}

	ticket, expiresAt, err := manager.Issue(expected)
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}
	if ticket == "" {
		t.Fatal("Issue() returned empty ticket")
	}
	if !expiresAt.Equal(now.Add(30 * time.Second)) {
		t.Fatalf("expiresAt = %v, want %v", expiresAt, now.Add(30*time.Second))
	}

	authCtx, err := manager.Consume(ticket)
	if err != nil {
		t.Fatalf("Consume() error = %v", err)
	}
	if authCtx.UserID != expected.UserID {
		t.Fatalf("authCtx.UserID = %v, want %v", authCtx.UserID, expected.UserID)
	}
	if authCtx.WorkspaceID != expected.WorkspaceID {
		t.Fatalf("authCtx.WorkspaceID = %v, want %v", authCtx.WorkspaceID, expected.WorkspaceID)
	}
	if authCtx.Role != expected.Role {
		t.Fatalf("authCtx.Role = %q, want %q", authCtx.Role, expected.Role)
	}

	if _, err := manager.Consume(ticket); err == nil {
		t.Fatal("second Consume() error = nil, want error")
	} else {
		var appErr *httpx.AppError
		if !errors.As(err, &appErr) {
			t.Fatalf("second Consume() error type = %T, want *httpx.AppError", err)
		}
		if appErr.Code != "realtime_session_invalid" {
			t.Fatalf("appErr.Code = %q, want %q", appErr.Code, "realtime_session_invalid")
		}
	}
}

func TestSessionManagerRejectsExpiredTickets(t *testing.T) {
	current := time.Date(2026, time.April, 6, 10, 0, 0, 0, time.UTC)
	manager := NewSessionManager(30*time.Second, func() time.Time {
		return current
	})

	ticket, _, err := manager.Issue(&identity.Context{
		UserID:      uuid.New(),
		WorkspaceID: uuid.New(),
		Role:        "owner",
	})
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}

	current = current.Add(31 * time.Second)

	_, err = manager.Consume(ticket)
	if err == nil {
		t.Fatal("Consume() error = nil, want error")
	}

	var appErr *httpx.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("Consume() error type = %T, want *httpx.AppError", err)
	}
	if appErr.Code != "realtime_session_expired" {
		t.Fatalf("appErr.Code = %q, want %q", appErr.Code, "realtime_session_expired")
	}
}
