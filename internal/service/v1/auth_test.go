package v1

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"aidanwoods.dev/go-paseto"

	"github.com/dongwlin/legero-backend/internal/apperr"
	"github.com/dongwlin/legero-backend/internal/domain"
	"github.com/dongwlin/legero-backend/internal/infra/config"
	"github.com/dongwlin/legero-backend/internal/infra/crypto"
	"github.com/dongwlin/legero-backend/internal/infra/database"
	"github.com/dongwlin/legero-backend/internal/infra/identity"
	"github.com/dongwlin/legero-backend/internal/repo"
	"github.com/dongwlin/legero-backend/internal/service"
	"github.com/dongwlin/legero-backend/migrations"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
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

// mockOrderLoader implements ActiveOrderLoader for tests.
type mockOrderLoader struct{}

func (m *mockOrderLoader) ListActive(_ context.Context, _ uuid.UUID) ([]domain.Order, error) {
	return []domain.Order{}, nil
}

// trackingOrderLoader records ListActive invocations so tests can assert that
// fail-closed paths never load order data.
type trackingOrderLoader struct {
	listActiveCalls int
}

func (l *trackingOrderLoader) ListActive(_ context.Context, _ uuid.UUID) ([]domain.Order, error) {
	l.listActiveCalls++
	return []domain.Order{}, nil
}

// defaultAccessLoader returns the production workspace-access loader: the
// repository bound to the given database handle. Keep in sync with the
// loader factory in ProvideAuth (internal/app/provider.go).
func defaultAccessLoader(db bun.IDB) service.WorkspaceAccessLoader {
	return repo.NewWorkspace(db)
}

// newTestService creates a Service wired to testDB with test-friendly settings.
func newTestService(t *testing.T, db *bun.DB) service.Auth {
	t.Helper()
	return newTestServiceWithLoader(t, db, &mockOrderLoader{})
}

// newTestServiceWithLoader creates a Service wired to testDB with a custom
// ActiveOrderLoader.
func newTestServiceWithLoader(t *testing.T, db *bun.DB, loader service.ActiveOrderLoader) service.Auth {
	t.Helper()
	return newTestServiceFull(t, db, loader, defaultAccessLoader)
}

// newTestServiceFull creates a Service wired to testDB with a custom
// ActiveOrderLoader and a custom workspace-access factory. The factory lets a
// test simulate roles the database schema does not permit without mutating the
// shared schema.
func newTestServiceFull(
	t *testing.T,
	db *bun.DB,
	loader service.ActiveOrderLoader,
	newAccessLoader func(bun.IDB) service.WorkspaceAccessLoader,
) service.Auth {
	t.Helper()

	keyBytes := make([]byte, 32)
	_, err := rand.Read(keyBytes)
	require.NoError(t, err)

	hasher := crypto.NewPasswordHasher(config.Argon2Config{
		MemoryKiB:   8 * 1024,
		Iterations:  1,
		Parallelism: 1,
		SaltLength:  16,
		KeyLength:   32,
	})

	svc, err := NewAuth(
		db,
		newAccessLoader,
		loader,
		hasher,
		time.UTC,
		15*time.Minute,
		7*24*time.Hour,
		keyBytes,
	)
	require.NoError(t, err)
	return svc
}

// TestNewAuth_RequiresAccessLoaderFactory guards the constructor's fail-fast
// validation: a nil workspace-access factory is a wiring mistake that must be
// reported at startup rather than deferred into a request-time panic.
func TestNewAuth_RequiresAccessLoaderFactory(t *testing.T) {
	keyBytes := make([]byte, 32)
	_, err := NewAuth(
		testDB,
		nil,
		&mockOrderLoader{},
		crypto.NewPasswordHasher(config.Argon2Config{
			MemoryKiB:   8 * 1024,
			Iterations:  1,
			Parallelism: 1,
			SaltLength:  16,
			KeyLength:   32,
		}),
		time.UTC,
		15*time.Minute,
		7*24*time.Hour,
		keyBytes,
	)
	require.Error(t, err)
	require.ErrorContains(t, err, "workspace access loader factory is required")
}

func createTestUser(t *testing.T, ctx context.Context, db bun.IDB, opts ...func(*domain.User)) uuid.UUID {
	t.Helper()

	user := domain.User{
		ID:           uuid.New(),
		Phone:        fmt.Sprintf("1%s", uuid.New().String()[:11]),
		PasswordHash: crypto.MustHashForTests("password123"),
		IsActive:     true,
		Version:      1,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	for _, opt := range opts {
		opt(&user)
	}

	if err := repo.NewUser(db).Insert(ctx, &user); err != nil {
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

// stubAccessLoader returns predefined Accesses from the workspace-access seam
// instead of querying the database. It lets tests simulate a workspace role
// the application does not yet support (the migration pins roles to
// owner/staff) without dropping the shared schema constraint.
type stubAccessLoader struct {
	primary *domain.Access
	access  *domain.Access

	// handles records the database handle the seam was bound to on each
	// factory invocation, in call order (the root DB for Login/Bootstrap, the
	// transaction for Refresh).
	handles []bun.IDB
}

func (l *stubAccessLoader) GetPrimaryAccess(_ context.Context, _ uuid.UUID) (*domain.Access, error) {
	return l.primary, nil
}

func (l *stubAccessLoader) GetAccess(_ context.Context, _, _ uuid.UUID) (*domain.Access, error) {
	return l.access, nil
}

// stubAccessLoaderFactory wraps a stub so it can be passed to newTestServiceFull.
// It records the database handle each invocation is bound to on the stub, so
// tests can assert which handle (root DB vs. transaction) a loader was built
// from.
func stubAccessLoaderFactory(loader *stubAccessLoader) func(bun.IDB) service.WorkspaceAccessLoader {
	return func(db bun.IDB) service.WorkspaceAccessLoader {
		loader.handles = append(loader.handles, db)
		return loader
	}
}

// ---------------------------------------------------------------------------
// AuthService.Login
// ---------------------------------------------------------------------------

func TestLogin_Success(t *testing.T) {
	ctx := context.Background()
	svc := newTestService(t, testDB)

	userID := createTestUser(t, ctx, testDB, func(u *domain.User) {
		u.Phone = "13800001001"
		u.IsActive = true
	})
	wsID := createTestWorkspace(t, ctx, testDB)
	createTestWorkspaceMember(t, ctx, testDB, userID, wsID, "owner")

	result, err := svc.Login(ctx, service.LoginRequest{Phone: "13800001001", Password: "password123"})
	require.NoError(t, err)
	require.NotNil(t, result)

	// Token pair.
	require.NotEmpty(t, result.TokenPair.AccessToken)
	require.NotEmpty(t, result.TokenPair.RefreshToken)
	require.True(t, result.TokenPair.AccessTokenExpiresAt.After(time.Now()))
	require.True(t, result.TokenPair.RefreshTokenExpiresAt.After(time.Now()))

	// Bootstrap data.
	require.Equal(t, userID, result.Bootstrap.User.ID)
	require.Equal(t, "13800001001", result.Bootstrap.User.Phone)
	require.Equal(t, domain.RoleOwner, result.Bootstrap.User.Role)
	require.Equal(t, wsID, result.Bootstrap.Workspace.ID)
	require.Equal(t, "test-workspace", result.Bootstrap.Workspace.Name)
	require.Contains(t, result.Bootstrap.Permissions, "orders:read")
	require.Contains(t, result.Bootstrap.Permissions, "orders:write")
	require.Contains(t, result.Bootstrap.Permissions, "orders:clear")
}

func TestLogin_InvalidPhone(t *testing.T) {
	ctx := context.Background()
	svc := newTestService(t, testDB)

	_, err := svc.Login(ctx, service.LoginRequest{Phone: "00000000000", Password: "password123"})
	require.Error(t, err)

	var appErr *apperr.AppError
	require.True(t, errors.As(err, &appErr))
	require.Equal(t, apperr.KindUnauthenticated, appErr.Kind)
}

func TestLogin_WrongPassword(t *testing.T) {
	ctx := context.Background()
	svc := newTestService(t, testDB)

	createTestUser(t, ctx, testDB, func(u *domain.User) {
		u.Phone = "13800001002"
		u.IsActive = true
	})

	_, err := svc.Login(ctx, service.LoginRequest{Phone: "13800001002", Password: "wrongpassword"})
	require.Error(t, err)

	var appErr *apperr.AppError
	require.True(t, errors.As(err, &appErr))
	require.Equal(t, apperr.KindUnauthenticated, appErr.Kind)
}

func TestLogin_InactiveUser(t *testing.T) {
	ctx := context.Background()
	svc := newTestService(t, testDB)

	createTestUser(t, ctx, testDB, func(u *domain.User) {
		u.Phone = "13800001003"
		u.IsActive = false
	})

	_, err := svc.Login(ctx, service.LoginRequest{Phone: "13800001003", Password: "password123"})
	require.Error(t, err)

	var appErr *apperr.AppError
	require.True(t, errors.As(err, &appErr))
	require.Equal(t, apperr.KindUnauthenticated, appErr.Kind)
}

func TestLogin_NoWorkspace(t *testing.T) {
	ctx := context.Background()
	svc := newTestService(t, testDB)

	createTestUser(t, ctx, testDB, func(u *domain.User) {
		u.Phone = "13800001004"
		u.IsActive = true
	})
	// No workspace member created.

	_, err := svc.Login(ctx, service.LoginRequest{Phone: "13800001004", Password: "password123"})
	require.Error(t, err)

	var appErr *apperr.AppError
	require.True(t, errors.As(err, &appErr))
	require.Equal(t, apperr.KindNotFound, appErr.Kind)
}

// ---------------------------------------------------------------------------
// AuthService.Refresh
// ---------------------------------------------------------------------------

func TestRefresh_Success(t *testing.T) {
	ctx := context.Background()
	svc := newTestService(t, testDB)

	userID := createTestUser(t, ctx, testDB, func(u *domain.User) {
		u.Phone = "13800002001"
		u.IsActive = true
	})
	wsID := createTestWorkspace(t, ctx, testDB)
	createTestWorkspaceMember(t, ctx, testDB, userID, wsID, "owner")

	loginResult, err := svc.Login(ctx, service.LoginRequest{Phone: "13800002001", Password: "password123"})
	require.NoError(t, err)

	pair, err := svc.Refresh(ctx, loginResult.TokenPair.RefreshToken)
	require.NoError(t, err)
	require.NotNil(t, pair)
	require.NotEmpty(t, pair.AccessToken)
	require.NotEmpty(t, pair.RefreshToken)
	require.NotEqual(t, loginResult.TokenPair.AccessToken, pair.AccessToken)
	require.NotEqual(t, loginResult.TokenPair.RefreshToken, pair.RefreshToken)
}

func TestRefresh_ExpiredToken(t *testing.T) {
	ctx := context.Background()

	// Create a service with a very short refresh TTL so the token is already expired.
	keyBytes := make([]byte, 32)
	_, err := rand.Read(keyBytes)
	require.NoError(t, err)

	hasher := crypto.NewPasswordHasher(config.Argon2Config{
		MemoryKiB:   8 * 1024,
		Iterations:  1,
		Parallelism: 1,
		SaltLength:  16,
		KeyLength:   32,
	})

	svc, err := NewAuth(
		testDB,
		defaultAccessLoader,
		&mockOrderLoader{},
		hasher,
		time.UTC,
		15*time.Minute,
		1*time.Millisecond, // very short refresh TTL
		keyBytes,
	)
	require.NoError(t, err)

	userID := createTestUser(t, ctx, testDB, func(u *domain.User) {
		u.Phone = "13800002002"
		u.IsActive = true
	})
	wsID := createTestWorkspace(t, ctx, testDB)
	createTestWorkspaceMember(t, ctx, testDB, userID, wsID, "owner")

	loginResult, err := svc.Login(ctx, service.LoginRequest{Phone: "13800002002", Password: "password123"})
	require.NoError(t, err)

	// Wait for the token to expire.
	time.Sleep(10 * time.Millisecond)

	_, err = svc.Refresh(ctx, loginResult.TokenPair.RefreshToken)
	require.Error(t, err)

	var appErr *apperr.AppError
	require.True(t, errors.As(err, &appErr))
	require.Equal(t, apperr.KindUnauthenticated, appErr.Kind)
}

func TestRefresh_RevokedToken(t *testing.T) {
	ctx := context.Background()
	svc := newTestService(t, testDB)

	userID := createTestUser(t, ctx, testDB, func(u *domain.User) {
		u.Phone = "13800002003"
		u.IsActive = true
	})
	wsID := createTestWorkspace(t, ctx, testDB)
	createTestWorkspaceMember(t, ctx, testDB, userID, wsID, "owner")

	loginResult, err := svc.Login(ctx, service.LoginRequest{Phone: "13800002003", Password: "password123"})
	require.NoError(t, err)

	// Revoke the refresh token by setting revoked_at.
	tokenHash := crypto.HashToken(loginResult.TokenPair.RefreshToken)
	now := time.Now()
	_, err = testDB.NewUpdate().
		Table("refresh_tokens").
		Set("revoked_at = ?", now).
		Where("token_hash = ?", tokenHash).
		Exec(ctx)
	require.NoError(t, err)

	_, err = svc.Refresh(ctx, loginResult.TokenPair.RefreshToken)
	require.Error(t, err)

	var appErr *apperr.AppError
	require.True(t, errors.As(err, &appErr))
	require.Equal(t, apperr.KindUnauthenticated, appErr.Kind)
}

// ---------------------------------------------------------------------------
// AuthService.Bootstrap
// ---------------------------------------------------------------------------

func TestBootstrap_Success(t *testing.T) {
	ctx := context.Background()
	svc := newTestService(t, testDB)

	userID := createTestUser(t, ctx, testDB, func(u *domain.User) {
		u.Phone = "13800003001"
		u.IsActive = true
	})
	wsID := createTestWorkspace(t, ctx, testDB)
	createTestWorkspaceMember(t, ctx, testDB, userID, wsID, "staff")

	authCtx := &identity.Context{
		UserID:      userID,
		WorkspaceID: wsID,
		Role:        string(domain.RoleStaff),
	}

	data, err := svc.Bootstrap(ctx, authCtx)
	require.NoError(t, err)
	require.NotNil(t, data)

	require.Equal(t, userID, data.User.ID)
	require.Equal(t, "13800003001", data.User.Phone)
	require.Equal(t, domain.RoleStaff, data.User.Role)
	require.Equal(t, wsID, data.Workspace.ID)
	require.Equal(t, "test-workspace", data.Workspace.Name)
	require.Contains(t, data.Permissions, "orders:read")
	require.Contains(t, data.Permissions, "orders:write")
	require.NotContains(t, data.Permissions, "orders:clear")
	require.NotNil(t, data.ActiveOrders)
	require.Len(t, data.ActiveOrders, 0)
}

func TestBootstrap_InactiveUser(t *testing.T) {
	ctx := context.Background()
	svc := newTestService(t, testDB)

	userID := createTestUser(t, ctx, testDB, func(u *domain.User) {
		u.Phone = "13800003002"
		u.IsActive = false
	})
	wsID := createTestWorkspace(t, ctx, testDB)
	createTestWorkspaceMember(t, ctx, testDB, userID, wsID, "owner")

	authCtx := &identity.Context{
		UserID:      userID,
		WorkspaceID: wsID,
		Role:        string(domain.RoleOwner),
	}

	_, err := svc.Bootstrap(ctx, authCtx)
	require.Error(t, err)

	var appErr *apperr.AppError
	require.True(t, errors.As(err, &appErr))
	require.Equal(t, apperr.KindUnauthenticated, appErr.Kind)
}

func TestBootstrap_NoWorkspace(t *testing.T) {
	ctx := context.Background()
	svc := newTestService(t, testDB)

	userID := createTestUser(t, ctx, testDB, func(u *domain.User) {
		u.Phone = "13800003003"
		u.IsActive = true
	})

	authCtx := &identity.Context{
		UserID:      userID,
		WorkspaceID: uuid.New(),
		Role:        string(domain.RoleOwner),
	}

	_, err := svc.Bootstrap(ctx, authCtx)
	require.Error(t, err)

	var appErr *apperr.AppError
	require.True(t, errors.As(err, &appErr))
	require.Equal(t, apperr.KindNotFound, appErr.Kind)
}

// ---------------------------------------------------------------------------
// AuthService.parseToken role claim validation
// ---------------------------------------------------------------------------

// buildTestToken encrypts a PASETO access token with the given role claim.
// When setRole is false, the role claim is omitted entirely.
func buildTestToken(t *testing.T, s *auth, role string, setRole bool) string {
	t.Helper()

	now := time.Now()
	token := paseto.NewToken()
	token.SetIssuedAt(now)
	token.SetNotBefore(now)
	token.SetExpiration(now.Add(time.Hour))
	token.SetSubject(uuid.New().String())
	token.SetJti(uuid.New().String())
	token.SetString("wid", uuid.New().String())
	if setRole {
		token.SetString("role", role)
	}
	token.SetString("typ", "access")

	return token.V4Encrypt(s.key, nil)
}

func TestParseToken_RejectsInvalidRole(t *testing.T) {
	svc := newTestService(t, testDB)
	s := svc.(*auth)

	tests := []struct {
		name    string
		role    string
		setRole bool
	}{
		{name: "unknown role", role: "admin", setRole: true},
		{name: "empty role", role: "", setRole: true},
		{name: "missing role", setRole: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw := buildTestToken(t, s, tt.role, tt.setRole)
			_, err := s.parseToken(raw, "access")
			require.Error(t, err)

			var appErr *apperr.AppError
			require.True(t, errors.As(err, &appErr))
			require.Equal(t, apperr.KindUnauthenticated, appErr.Kind)
			require.Equal(t, "invalid token role", appErr.Message)
		})
	}
}

func TestParseToken_ValidRoles(t *testing.T) {
	svc := newTestService(t, testDB)
	s := svc.(*auth)

	for _, role := range []domain.Role{domain.RoleOwner, domain.RoleStaff} {
		raw := buildTestToken(t, s, string(role), true)
		claims, err := s.parseToken(raw, "access")
		require.NoError(t, err)
		require.Equal(t, role, claims.Role)
	}
}

// ---------------------------------------------------------------------------
// AuthService fail-closed on unknown workspace roles
// ---------------------------------------------------------------------------

func TestLogin_UnknownRole_FailsClosed(t *testing.T) {
	ctx := context.Background()
	loader := &trackingOrderLoader{}

	userID := createTestUser(t, ctx, testDB, func(u *domain.User) {
		u.Phone = "13800004001"
		u.IsActive = true
	})

	// Simulate a workspace membership whose role the application does not yet
	// support via the access seam, so no invalid row is ever written to the
	// shared database.
	accesses := &stubAccessLoader{
		primary: &domain.Access{
			UserID:        userID,
			WorkspaceID:   uuid.New(),
			WorkspaceName: "test-workspace",
			Role:          "viewer",
		},
	}
	svc := newTestServiceFull(t, testDB, loader, stubAccessLoaderFactory(accesses))

	_, err := svc.Login(ctx, service.LoginRequest{Phone: "13800004001", Password: "password123"})
	require.Error(t, err)

	var appErr *apperr.AppError
	require.True(t, errors.As(err, &appErr))
	require.Equal(t, apperr.KindForbidden, appErr.Kind)

	// No orders were loaded and no tokens were issued for the unknown role.
	require.Zero(t, loader.listActiveCalls)
	refreshCount, err := testDB.NewSelect().Table("refresh_tokens").Where("user_id = ?", userID).Count(ctx)
	require.NoError(t, err)
	require.Zero(t, refreshCount)
}

func TestBootstrap_UnknownRole_Forbidden(t *testing.T) {
	ctx := context.Background()

	userID := createTestUser(t, ctx, testDB, func(u *domain.User) {
		u.Phone = "13800004002"
		u.IsActive = true
	})
	wsID := uuid.New()

	accesses := &stubAccessLoader{
		access: &domain.Access{
			UserID:        userID,
			WorkspaceID:   wsID,
			WorkspaceName: "test-workspace",
			Role:          "viewer",
		},
	}
	svc := newTestServiceFull(t, testDB, &mockOrderLoader{}, stubAccessLoaderFactory(accesses))

	authCtx := &identity.Context{
		UserID:      userID,
		WorkspaceID: wsID,
		Role:        "viewer",
	}

	_, err := svc.Bootstrap(ctx, authCtx)
	require.Error(t, err)

	var appErr *apperr.AppError
	require.True(t, errors.As(err, &appErr))
	require.Equal(t, apperr.KindForbidden, appErr.Kind)
}

func TestRefresh_DegradedRole_FailsClosed(t *testing.T) {
	ctx := context.Background()

	userID := createTestUser(t, ctx, testDB, func(u *domain.User) {
		u.Phone = "13800004003"
		u.IsActive = true
	})
	// The workspace row must exist: login persists a refresh token row that
	// references it via a foreign key.
	wsID := createTestWorkspace(t, ctx, testDB)

	// The workspace role degrades to one the application does not support
	// (e.g. a future DB role) after the original tokens were issued: login
	// sees the original role, refresh sees the degraded one.
	accesses := &stubAccessLoader{
		primary: &domain.Access{
			UserID:        userID,
			WorkspaceID:   wsID,
			WorkspaceName: "test-workspace",
			Role:          domain.RoleOwner,
		},
		access: &domain.Access{
			UserID:        userID,
			WorkspaceID:   wsID,
			WorkspaceName: "test-workspace",
			Role:          "viewer",
		},
	}
	svc := newTestServiceFull(t, testDB, &mockOrderLoader{}, stubAccessLoaderFactory(accesses))

	loginResult, err := svc.Login(ctx, service.LoginRequest{Phone: "13800004003", Password: "password123"})
	require.NoError(t, err)

	_, err = svc.Refresh(ctx, loginResult.TokenPair.RefreshToken)
	require.Error(t, err)

	var appErr *apperr.AppError
	require.True(t, errors.As(err, &appErr))
	require.Equal(t, apperr.KindForbidden, appErr.Kind)

	// No replacement pair was issued and the original token was not rotated.
	refreshCount, err := testDB.NewSelect().Table("refresh_tokens").Where("user_id = ?", userID).Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, refreshCount)

	var rotated bool
	err = testDB.NewRaw("SELECT rotated_at IS NOT NULL FROM refresh_tokens WHERE user_id = ? LIMIT 1", userID).Scan(ctx, &rotated)
	require.NoError(t, err)
	require.False(t, rotated)

	// The workspace-access seam must stay transaction-scoped: Login resolves
	// through the root DB, but Refresh must resolve through the same
	// transaction that locked and rotated the refresh token. Without this, a
	// regression to s.newAccessLoader(s.db) in Refresh would silently lose the
	// transaction binding while every regression test still passed.
	require.Len(t, accesses.handles, 2)
	rootDB, ok := accesses.handles[0].(*bun.DB)
	require.True(t, ok, "login should bind the access loader to the root database")
	require.Same(t, testDB, rootDB)
	_, ok = accesses.handles[1].(bun.Tx)
	require.True(t, ok, "refresh must bind the access loader to the transaction, not the root database")
}
