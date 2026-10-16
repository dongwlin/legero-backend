package v2

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/dongwlin/legero-backend/internal/domain"
	"github.com/dongwlin/legero-backend/internal/infra/identity"
	"github.com/dongwlin/legero-backend/internal/service"
)

func TestOrderDeleteWrapsUnifiedEnvelope(t *testing.T) {
	orderID := uuid.New()
	removed := false
	router := newTestRouter(authStub{}, orderStub{
		removeFn: func(_ context.Context, _ domain.Actor, id uuid.UUID) error {
			require.Equal(t, orderID, id)
			removed = true
			return nil
		},
	}, statsStub{})

	recorder := doRequest(router, http.MethodDelete, "/api/v2/orders/"+orderID.String(), "", authHeader)

	require.True(t, removed)
	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Header().Get("Content-Type"), "application/json")
	require.JSONEq(t, `{"code":"0","message":"success","data":null}`, recorder.Body.String())
}

func TestOrderResponsesPreserveV2Payload(t *testing.T) {
	location := time.FixedZone("business", 8*60*60)
	createdAt := time.Date(2026, 10, 15, 23, 30, 0, 0, time.UTC)
	completedAt := createdAt.Add(20 * time.Minute)
	staple, packaging, packagingMethod := int16(3), int16(2), int16(1)
	customPrice := 2500
	order := domain.Order{
		ID:                   uuid.MustParse("5bdcd4d3-a07e-4b04-bb32-8a4adbb088cf"),
		Version:              9,
		DisplayNo:            "031",
		StapleTypeCode:       &staple,
		SizeCode:             4,
		CustomSizePriceCents: &customPrice,
		StapleAmountCode:     3,
		ExtraStapleUnits:     2,
		FriedEggCount:        1,
		TofuSkewerCount:      2,
		SelectedMeatCodes:    []int16{1, 6},
		GreensCode:           2,
		ScallionCode:         4,
		PepperCode:           3,
		DiningMethodCode:     2,
		PackagingCode:        &packaging,
		PackagingMethodCode:  &packagingMethod,
		TotalPriceCents:      3300,
		StapleStepStatusCode: 3,
		MeatStepStatusCode:   3,
		Note:                 "少盐",
		CreatedAt:            createdAt,
		UpdatedAt:            completedAt,
		CompletedAt:          &completedAt,
	}
	const expectedOrder = `{
		"id":"5bdcd4d3-a07e-4b04-bb32-8a4adbb088cf","version":9,"displayNo":"031",
		"stapleTypeCode":3,"sizeCode":4,"customSizePriceCents":2500,"stapleAmountCode":3,
		"extraStapleUnits":2,"friedEggCount":1,"tofuSkewerCount":2,"selectedMeatCodes":[1,6],
		"greensCode":2,"scallionCode":4,"pepperCode":3,"diningMethodCode":2,
		"packagingCode":2,"packagingMethodCode":1,"totalPriceCents":3300,
		"stapleStepStatusCode":3,"meatStepStatusCode":3,"note":"少盐",
		"createdAt":"2026-10-16T07:30:00+08:00","updatedAt":"2026-10-16T07:50:00+08:00",
		"completedAt":"2026-10-16T07:50:00+08:00"
	}`
	bootstrap := domain.BootstrapData{ActiveOrders: []domain.Order{order}}
	orders := orderStub{
		listFn: func(context.Context, domain.Actor, domain.ListOrdersQuery) (*domain.ListOrdersResult, error) {
			return &domain.ListOrdersResult{Items: []domain.Order{order}}, nil
		},
		createFn: func(context.Context, domain.Actor, domain.CreateOrdersInput) ([]domain.Order, error) {
			return []domain.Order{order}, nil
		},
		updateFn: func(context.Context, domain.Actor, uuid.UUID, domain.UpdateOrderInput) (*domain.Order, error) {
			return &order, nil
		},
		toggleStepFn: func(context.Context, domain.Actor, uuid.UUID, domain.ToggleStepInput) (*domain.Order, error) {
			return &order, nil
		},
		toggleServedFn: func(context.Context, domain.Actor, uuid.UUID, domain.ToggleServedInput) (*domain.Order, error) {
			return &order, nil
		},
	}
	auth := authStub{
		loginFn: func(context.Context, service.LoginRequest) (*domain.LoginResult, error) {
			return &domain.LoginResult{Bootstrap: bootstrap}, nil
		},
		bootstrapFn: func(context.Context, *identity.Context) (*domain.BootstrapData, error) {
			return &bootstrap, nil
		},
	}
	router := newTestRouterWithLocation(auth, orders, statsStub{}, location)
	orderPath := "/api/v2/orders/" + order.ID.String()
	for _, tt := range []struct {
		name, method, path, body, field string
	}{
		{"list", http.MethodGet, "/api/v2/orders", "", "items"},
		{"create", http.MethodPost, "/api/v2/orders", `{}`, "items"},
		{"update", http.MethodPut, orderPath, `{}`, "item"},
		{"toggle step", http.MethodPost, orderPath + "/actions/toggle-step", `{}`, "item"},
		{"toggle served", http.MethodPost, orderPath + "/actions/toggle-served", `{}`, "item"},
		{"login", http.MethodPost, "/api/v2/auth/login", `{}`, "activeOrders"},
		{"bootstrap", http.MethodGet, "/api/v2/bootstrap", "", "activeOrders"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			recorder := doRequest(router, tt.method, tt.path, tt.body, authHeader)
			require.Equal(t, http.StatusOK, recorder.Code)
			var body envelope
			require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
			require.Equal(t, "0", body.Code)
			var data map[string]json.RawMessage
			require.NoError(t, json.Unmarshal(body.Data, &data))
			item := data[tt.field]
			if tt.field != "item" {
				var items []json.RawMessage
				require.NoError(t, json.Unmarshal(item, &items))
				require.Len(t, items, 1)
				item = items[0]
			}
			require.JSONEq(t, expectedOrder, string(item))
		})
	}
}

func TestOrderResponsesPreserveNullAndEmptyFields(t *testing.T) {
	item := toOrderDTO(domain.Order{}, time.UTC)
	encoded, err := json.Marshal(item)
	require.NoError(t, err)
	var fields map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(encoded, &fields))
	for _, field := range []string{"stapleTypeCode", "customSizePriceCents", "packagingCode", "packagingMethodCode", "completedAt"} {
		require.Equal(t, "null", string(fields[field]), field)
	}
	require.Equal(t, "[]", string(fields["selectedMeatCodes"]))
	encoded, err = json.Marshal(toOrderDTOs(nil, time.UTC))
	require.NoError(t, err)
	require.Equal(t, "[]", string(encoded))
}

func TestOrderMutationRequestsMapToDomain(t *testing.T) {
	staple, packaging, packagingMethod := int16(3), int16(2), int16(1)
	customPrice := 2500
	form := domain.OrderFormInput{
		StapleTypeCode:       &staple,
		SizeCode:             4,
		CustomSizePriceCents: &customPrice,
		StapleAmountCode:     3,
		ExtraStapleUnits:     2,
		FriedEggCount:        1,
		TofuSkewerCount:      2,
		SelectedMeatCodes:    []int16{1, 6},
		GreensCode:           2,
		ScallionCode:         4,
		PepperCode:           3,
		DiningMethodCode:     2,
		PackagingCode:        &packaging,
		PackagingMethodCode:  &packagingMethod,
		Note:                 " 少盐 ",
	}
	const formJSON = `{"stapleTypeCode":3,"sizeCode":4,"customSizePriceCents":2500,"stapleAmountCode":3,
		"extraStapleUnits":2,"friedEggCount":1,"tofuSkewerCount":2,"selectedMeatCodes":[1,6],
		"greensCode":2,"scallionCode":4,"pepperCode":3,"diningMethodCode":2,
		"packagingCode":2,"packagingMethodCode":1,"note":" 少盐 "}`
	version := int64(8)
	updatedAt := time.Date(2026, 10, 16, 8, 0, 0, 0, time.FixedZone("client", 8*60*60))
	const concurrencyJSON = `"expectedVersion":8,"expectedUpdatedAt":"2026-10-16T08:00:00+08:00"`
	order := domain.Order{ID: uuid.New()}
	path := "/api/v2/orders/" + order.ID.String()
	for _, name := range []string{"create", "update", "toggle step", "toggle served", "clear"} {
		t.Run(name, func(t *testing.T) {
			called := false
			orders := orderStub{}
			method, target, payload := http.MethodPost, "", ""
			switch name {
			case "create":
				target, payload = "/api/v2/orders", `{"quantity":2,"form":`+formJSON+`}`
				orders.createFn = func(_ context.Context, _ domain.Actor, input domain.CreateOrdersInput) ([]domain.Order, error) {
					called = true
					require.Equal(t, domain.CreateOrdersInput{Quantity: 2, Form: form}, input)
					return []domain.Order{order}, nil
				}
			case "update":
				method, target, payload = http.MethodPut, path, `{"form":`+formJSON+`,`+concurrencyJSON+`}`
				orders.updateFn = func(_ context.Context, _ domain.Actor, id uuid.UUID, input domain.UpdateOrderInput) (*domain.Order, error) {
					called = true
					require.Equal(t, order.ID, id)
					require.Equal(t, form, input.Form)
					require.Equal(t, &version, input.ExpectedVersion)
					require.WithinDuration(t, updatedAt, *input.ExpectedUpdatedAt, 0)
					return &order, nil
				}
			case "toggle step":
				target, payload = path+"/actions/toggle-step", `{"step":"meat",`+concurrencyJSON+`}`
				orders.toggleStepFn = func(_ context.Context, _ domain.Actor, id uuid.UUID, input domain.ToggleStepInput) (*domain.Order, error) {
					called = true
					require.Equal(t, order.ID, id)
					require.Equal(t, "meat", input.Step)
					require.Equal(t, &version, input.ExpectedVersion)
					require.WithinDuration(t, updatedAt, *input.ExpectedUpdatedAt, 0)
					return &order, nil
				}
			case "toggle served":
				target, payload = path+"/actions/toggle-served", `{`+concurrencyJSON+`}`
				orders.toggleServedFn = func(_ context.Context, _ domain.Actor, id uuid.UUID, input domain.ToggleServedInput) (*domain.Order, error) {
					called = true
					require.Equal(t, order.ID, id)
					require.Equal(t, &version, input.ExpectedVersion)
					require.WithinDuration(t, updatedAt, *input.ExpectedUpdatedAt, 0)
					return &order, nil
				}
			case "clear":
				target, payload = "/api/v2/orders/actions/clear", `{"confirm":true,"mode":"before_today"}`
				orders.clearFn = func(_ context.Context, _ domain.Actor, confirm bool, mode domain.ClearWorkspaceMode) (int, error) {
					called = true
					require.True(t, confirm)
					require.Equal(t, domain.ClearWorkspaceModeBeforeToday, mode)
					return 2, nil
				}
			}
			router := newTestRouter(authStub{}, orders, statsStub{})
			recorder := doRequest(router, method, target, payload, authHeader)
			require.Equal(t, http.StatusOK, recorder.Code)
			require.True(t, called)
		})
	}
}
