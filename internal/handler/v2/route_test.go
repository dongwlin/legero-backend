package v2

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/dongwlin/legero-backend/internal/domain"
	"github.com/dongwlin/legero-backend/internal/handler/v2/dto"
	"github.com/dongwlin/legero-backend/internal/infra/config"
	"github.com/dongwlin/legero-backend/internal/infra/identity"
	"github.com/dongwlin/legero-backend/internal/service"
)

type authStub struct {
	loginFn     func(context.Context, service.LoginRequest) (*domain.LoginResult, error)
	refreshFn   func(context.Context, string) (*domain.TokenPair, error)
	bootstrapFn func(context.Context, *identity.Context) (*domain.BootstrapData, error)
}

func (s authStub) Login(ctx context.Context, req service.LoginRequest) (*domain.LoginResult, error) {
	if s.loginFn == nil {
		return nil, nil
	}
	return s.loginFn(ctx, req)
}

func (s authStub) Refresh(ctx context.Context, token string) (*domain.TokenPair, error) {
	if s.refreshFn == nil {
		return nil, nil
	}
	return s.refreshFn(ctx, token)
}

func (s authStub) Bootstrap(ctx context.Context, authCtx *identity.Context) (*domain.BootstrapData, error) {
	if s.bootstrapFn == nil {
		return nil, nil
	}
	return s.bootstrapFn(ctx, authCtx)
}

func (s authStub) RequireAccessToken(context.Context, string) (*identity.Context, error) {
	return &identity.Context{
		UserID:      uuid.New(),
		WorkspaceID: uuid.New(),
		Role:        string(domain.RoleOwner),
	}, nil
}

type orderStub struct {
	listFn   func(context.Context, domain.Actor, domain.ListOrdersQuery) (*domain.ListOrdersResult, error)
	createFn func(context.Context, domain.Actor, domain.CreateOrdersInput) ([]domain.Order, error)
}

func (s orderStub) ListActive(context.Context, uuid.UUID) ([]domain.Order, error) {
	return nil, nil
}

func (s orderStub) List(ctx context.Context, actor domain.Actor, query domain.ListOrdersQuery) (*domain.ListOrdersResult, error) {
	if s.listFn == nil {
		return nil, nil
	}
	return s.listFn(ctx, actor, query)
}

func (s orderStub) CreateBatch(ctx context.Context, actor domain.Actor, input domain.CreateOrdersInput) ([]domain.Order, error) {
	if s.createFn == nil {
		return nil, nil
	}
	return s.createFn(ctx, actor, input)
}

func (s orderStub) UpdateForm(context.Context, domain.Actor, uuid.UUID, domain.UpdateOrderInput) (*domain.Order, error) {
	return nil, nil
}

func (s orderStub) ToggleStep(context.Context, domain.Actor, uuid.UUID, domain.ToggleStepInput) (*domain.Order, error) {
	return nil, nil
}

func (s orderStub) ToggleServed(context.Context, domain.Actor, uuid.UUID, domain.ToggleServedInput) (*domain.Order, error) {
	return nil, nil
}

func (s orderStub) Remove(context.Context, domain.Actor, uuid.UUID) error { return nil }

func (s orderStub) ClearWorkspace(context.Context, domain.Actor, bool, domain.ClearWorkspaceMode) (int, error) {
	return 0, nil
}

type statsStub struct{}

func (statsStub) Daily(context.Context, uuid.UUID, time.Time, time.Time) ([]domain.DailyRow, error) {
	return nil, nil
}

func (statsStub) Report(context.Context, uuid.UUID, domain.ReportQuery) (*domain.Report, error) {
	return nil, nil
}

// newTestRouter builds a gin engine with the real v2 route registration and
// middleware stack; only the services are stubbed.
func newTestRouter(authSvc service.Auth, orderSvc service.Order, statsSvc service.Stats) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterRoutes(
		router.Group("/api/v2"),
		authSvc,
		orderSvc,
		statsSvc,
		nil,
		nil,
		time.UTC,
		&config.Config{},
		nil,
	)
	return router
}

func doRequest(router *gin.Engine, method, target, body string, header map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	for key, value := range header {
		req.Header.Set(key, value)
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	return recorder
}

type envelope struct {
	Code    string          `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

var authHeader = map[string]string{"Authorization": "Bearer test-token"}

func TestOrderListWrapsUnifiedEnvelope(t *testing.T) {
	orderID := uuid.New()
	router := newTestRouter(authStub{}, orderStub{
		listFn: func(context.Context, domain.Actor, domain.ListOrdersQuery) (*domain.ListOrdersResult, error) {
			return &domain.ListOrdersResult{
				Items: []domain.Order{{ID: orderID, Version: 7}},
			}, nil
		},
	}, statsStub{})

	recorder := doRequest(router, http.MethodGet, "/api/v2/orders", "", authHeader)

	require.Equal(t, http.StatusOK, recorder.Code)

	var body envelope
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	require.Equal(t, "0", body.Code)
	require.Equal(t, "success", body.Message)

	var data dto.ListOrdersResponseData
	require.NoError(t, json.Unmarshal(body.Data, &data))
	require.Len(t, data.Items, 1)
	require.Equal(t, orderID.String(), data.Items[0].ID)
	require.Equal(t, int64(7), data.Items[0].Version)

}

func TestOrderCreateIsMutation(t *testing.T) {
	router := newTestRouter(authStub{}, orderStub{
		createFn: func(context.Context, domain.Actor, domain.CreateOrdersInput) ([]domain.Order, error) {
			return []domain.Order{{ID: uuid.New(), Version: 1}}, nil
		},
	}, statsStub{})

	recorder := doRequest(router, http.MethodPost, "/api/v2/orders",
		`{"quantity":1,"form":{}}`, authHeader)

	require.Equal(t, http.StatusOK, recorder.Code)
	var body envelope
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	require.Equal(t, "0", body.Code)
}

func TestLoginWrapsUnifiedEnvelope(t *testing.T) {
	now := time.Date(2026, 8, 21, 12, 0, 0, 0, time.UTC)
	router := newTestRouter(authStub{
		loginFn: func(context.Context, service.LoginRequest) (*domain.LoginResult, error) {
			return &domain.LoginResult{
				TokenPair: domain.TokenPair{
					AccessToken:           "access",
					AccessTokenExpiresAt:  now,
					RefreshToken:          "refresh",
					RefreshTokenExpiresAt: now.Add(time.Hour),
				},
			}, nil
		},
	}, orderStub{}, statsStub{})

	recorder := doRequest(router, http.MethodPost, "/api/v2/auth/login",
		`{"phone":"13800000001","password":"secret"}`, nil)

	require.Equal(t, http.StatusOK, recorder.Code)
	var body envelope
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	require.Equal(t, "0", body.Code)

	var data dto.LoginResponseData
	require.NoError(t, json.Unmarshal(body.Data, &data))
	require.Equal(t, "access", data.AccessToken)
	require.Equal(t, "Bearer", data.TokenType)
}

func TestValidationErrorRendersEnvelopeCode(t *testing.T) {
	router := newTestRouter(authStub{}, orderStub{}, statsStub{})

	recorder := doRequest(router, http.MethodGet, "/api/v2/stats/report", "", authHeader)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	var body envelope
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	require.Equal(t, "validation_failed", body.Code)
	require.Equal(t, "period must be one of day, week, month", body.Message)
	require.Equal(t, "null", string(body.Data))
}

func TestUnauthorizedRendersEnvelopeCode(t *testing.T) {
	router := newTestRouter(authStub{}, orderStub{}, statsStub{})

	recorder := doRequest(router, http.MethodGet, "/api/v2/orders", "", nil)

	require.Equal(t, http.StatusUnauthorized, recorder.Code)
	var body envelope
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	require.Equal(t, "unauthorized", body.Code)
	require.Equal(t, "null", string(body.Data))
}
