package realtime

import (
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/dongwlin/legero-backend/internal/infra/httpx"
	"github.com/dongwlin/legero-backend/internal/infra/identity"
)

type sessionRecord struct {
	AuthContext identity.Context
	ExpiresAt   time.Time
}

type SessionManager struct {
	mu       sync.Mutex
	sessions map[string]sessionRecord
	ttl      time.Duration
	now      func() time.Time
}

func NewSessionManager(ttl time.Duration, now func() time.Time) *SessionManager {
	if ttl <= 0 {
		ttl = 30 * time.Second
	}
	if now == nil {
		now = time.Now
	}

	return &SessionManager{
		sessions: make(map[string]sessionRecord),
		ttl:      ttl,
		now:      now,
	}
}

func (m *SessionManager) Issue(authCtx *identity.Context) (string, time.Time, error) {
	if authCtx == nil {
		return "", time.Time{}, httpx.UnauthorizedError("missing auth context")
	}

	now := m.now()
	expiresAt := now.Add(m.ttl)
	ticket := uuid.NewString()

	m.mu.Lock()
	defer m.mu.Unlock()

	m.pruneLocked(now, "")
	m.sessions[ticket] = sessionRecord{
		AuthContext: *authCtx,
		ExpiresAt:   expiresAt,
	}

	return ticket, expiresAt, nil
}

func (m *SessionManager) Consume(ticket string) (*identity.Context, error) {
	trimmed := strings.TrimSpace(ticket)
	if trimmed == "" {
		return nil, httpx.NewError(401, "realtime_session_invalid", "realtime session is invalid")
	}

	now := m.now()

	m.mu.Lock()
	defer m.mu.Unlock()

	m.pruneLocked(now, trimmed)

	record, ok := m.sessions[trimmed]
	if !ok {
		return nil, httpx.NewError(401, "realtime_session_invalid", "realtime session is invalid")
	}

	delete(m.sessions, trimmed)

	if !now.Before(record.ExpiresAt) {
		return nil, httpx.NewError(401, "realtime_session_expired", "realtime session has expired")
	}

	authCtx := record.AuthContext
	return &authCtx, nil
}

func (m *SessionManager) pruneLocked(now time.Time, skipTicket string) {
	for ticket, record := range m.sessions {
		if ticket == skipTicket {
			continue
		}
		if !now.Before(record.ExpiresAt) {
			delete(m.sessions, ticket)
		}
	}
}
