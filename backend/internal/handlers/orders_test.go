package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"b2b-orders-api/internal/domain"
	"b2b-orders-api/internal/repository"
	"b2b-orders-api/internal/service/auth"
	"b2b-orders-api/internal/service/order"
	"b2b-orders-api/internal/testutil"
)

func setupOrderHandler(
	orderRepo *testutil.MockOrderRepo,
	productRepo *testutil.MockProductRepo,
	clientRepo *testutil.MockClientRepo,
) *OrderHandler {
	orderSvc := order.NewService(orderRepo, productRepo, clientRepo, nil, testutil.NewDiscardLogger())
	authSvc := auth.NewService(clientRepo, testutil.NewDiscardLogger())
	return NewOrderHandler(orderSvc, authSvc, testutil.NewDiscardLogger())
}

func TestOrderHandler_Create(t *testing.T) {
	clientID := uuid.New()
	productID := uuid.New()

	client := &domain.Client{
		ID:       clientID,
		HoldedID: "holded-123",
		Email:    "test@example.com",
	}

	product := &domain.Product{
		ID:               productID,
		Name:             "Test Product",
		Price:            50.00,
		StockQuantity:    100,
		MinOrderQuantity: 1,
		IsActive:         true,
	}

	tests := []struct {
		name           string
		auth0ID        string
		email          string
		body           map[string]interface{}
		client         *domain.Client
		clientErr      error
		product        *domain.Product
		productErr     error
		wantStatusCode int
	}{
		{
			name:    "successful order creation",
			auth0ID: "auth0|123",
			email:   "test@example.com",
			body: map[string]interface{}{
				"items": []map[string]interface{}{
					{"product_id": productID.String(), "quantity": 5},
				},
				"notes": "Test order",
			},
			client:         client,
			product:        product,
			wantStatusCode: http.StatusCreated,
		},
		{
			name:    "empty items",
			auth0ID: "auth0|123",
			email:   "test@example.com",
			body: map[string]interface{}{
				"items": []map[string]interface{}{},
			},
			client:         client,
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:    "invalid product_id",
			auth0ID: "auth0|123",
			email:   "test@example.com",
			body: map[string]interface{}{
				"items": []map[string]interface{}{
					{"product_id": "not-a-uuid", "quantity": 5},
				},
			},
			client:         client,
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:    "product not found",
			auth0ID: "auth0|123",
			email:   "test@example.com",
			body: map[string]interface{}{
				"items": []map[string]interface{}{
					{"product_id": productID.String(), "quantity": 5},
				},
			},
			client:         client,
			productErr:     repository.ErrNotFound,
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "client not found",
			auth0ID:        "auth0|unknown",
			email:          "unknown@example.com",
			body:           map[string]interface{}{"items": []map[string]interface{}{}},
			clientErr:      repository.ErrNotFound,
			wantStatusCode: http.StatusUnauthorized,
		},
		{
			name:    "no auth context",
			auth0ID: "", // No auth
			email:   "",
			body: map[string]interface{}{
				"items": []map[string]interface{}{
					{"product_id": productID.String(), "quantity": 5},
				},
			},
			wantStatusCode: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orderRepo := &testutil.MockOrderRepo{}
			productRepo := &testutil.MockProductRepo{GetByIDResult: tt.product, GetByIDErr: tt.productErr}
			clientRepo := &testutil.MockClientRepo{
				GetByAuth0IDResult: tt.client,
				GetByAuth0IDErr:    tt.clientErr,
				GetByEmailResult:   tt.client,
				GetByEmailErr:      tt.clientErr,
			}
			handler := setupOrderHandler(orderRepo, productRepo, clientRepo)

			body, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			if tt.auth0ID != "" {
				req = testutil.WithAuthContext(req, tt.auth0ID, tt.email)
			}

			rec := httptest.NewRecorder()
			handler.Create(rec, req)

			assert.Equal(t, tt.wantStatusCode, rec.Code)

			if tt.wantStatusCode == http.StatusCreated {
				var response orderResponse
				err := json.NewDecoder(rec.Body).Decode(&response)
				require.NoError(t, err)
				assert.Equal(t, domain.OrderStatusPending, response.Status)
				assert.Len(t, response.Items, 1)
			}
		})
	}
}

func TestOrderHandler_List(t *testing.T) {
	clientID := uuid.New()
	orderID := uuid.New()

	client := &domain.Client{
		ID:       clientID,
		HoldedID: "holded-123",
		Email:    "test@example.com",
	}

	tests := []struct {
		name           string
		auth0ID        string
		email          string
		client         *domain.Client
		orders         []domain.Order
		listErr        error
		wantStatusCode int
		wantLen        int
	}{
		{
			name:    "successful list",
			auth0ID: "auth0|123",
			email:   "test@example.com",
			client:  client,
			orders: []domain.Order{
				{ID: orderID, ClientID: clientID, Status: domain.OrderStatusPending},
				{ID: uuid.New(), ClientID: clientID, Status: domain.OrderStatusApproved},
			},
			wantStatusCode: http.StatusOK,
			wantLen:        2,
		},
		{
			name:           "empty list",
			auth0ID:        "auth0|123",
			email:          "test@example.com",
			client:         client,
			orders:         []domain.Order{},
			wantStatusCode: http.StatusOK,
			wantLen:        0,
		},
		{
			name:           "no auth context",
			auth0ID:        "",
			email:          "",
			wantStatusCode: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orderRepo := &testutil.MockOrderRepo{ListByClientIDResult: tt.orders, ListByClientIDErr: tt.listErr}
			productRepo := &testutil.MockProductRepo{}
			clientRepo := &testutil.MockClientRepo{
				GetByAuth0IDResult: tt.client,
				GetByEmailResult:   tt.client,
			}
			handler := setupOrderHandler(orderRepo, productRepo, clientRepo)

			req := httptest.NewRequest(http.MethodGet, "/orders", nil)
			if tt.auth0ID != "" {
				req = testutil.WithAuthContext(req, tt.auth0ID, tt.email)
			}

			rec := httptest.NewRecorder()
			handler.List(rec, req)

			assert.Equal(t, tt.wantStatusCode, rec.Code)

			if tt.wantStatusCode == http.StatusOK {
				var response []orderResponse
				err := json.NewDecoder(rec.Body).Decode(&response)
				require.NoError(t, err)
				assert.Len(t, response, tt.wantLen)
			}
		})
	}
}

func TestOrderHandler_Get(t *testing.T) {
	clientID := uuid.New()
	orderID := uuid.New()
	otherClientID := uuid.New()

	client := &domain.Client{
		ID:       clientID,
		HoldedID: "holded-123",
		Email:    "test@example.com",
	}

	tests := []struct {
		name           string
		urlID          string
		auth0ID        string
		email          string
		client         *domain.Client
		order          *domain.Order
		orderErr       error
		wantStatusCode int
	}{
		{
			name:    "successful get",
			urlID:   orderID.String(),
			auth0ID: "auth0|123",
			email:   "test@example.com",
			client:  client,
			order: &domain.Order{
				ID:       orderID,
				ClientID: clientID,
				Status:   domain.OrderStatusPending,
			},
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "invalid uuid",
			urlID:          "not-a-uuid",
			auth0ID:        "auth0|123",
			email:          "test@example.com",
			client:         client,
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "order not found",
			urlID:          orderID.String(),
			auth0ID:        "auth0|123",
			email:          "test@example.com",
			client:         client,
			orderErr:       repository.ErrNotFound,
			wantStatusCode: http.StatusNotFound,
		},
		{
			name:    "order belongs to different client",
			urlID:   orderID.String(),
			auth0ID: "auth0|123",
			email:   "test@example.com",
			client:  client,
			order: &domain.Order{
				ID:       orderID,
				ClientID: otherClientID, // Different client
				Status:   domain.OrderStatusPending,
			},
			wantStatusCode: http.StatusNotFound,
		},
		{
			name:           "no auth context",
			urlID:          orderID.String(),
			auth0ID:        "",
			email:          "",
			wantStatusCode: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orderRepo := &testutil.MockOrderRepo{GetByIDResult: tt.order, GetByIDErr: tt.orderErr}
			productRepo := &testutil.MockProductRepo{}
			clientRepo := &testutil.MockClientRepo{
				GetByAuth0IDResult: tt.client,
				GetByEmailResult:   tt.client,
			}
			handler := setupOrderHandler(orderRepo, productRepo, clientRepo)

			r := chi.NewRouter()
			r.Get("/orders/{id}", handler.Get)

			req := httptest.NewRequest(http.MethodGet, "/orders/"+tt.urlID, nil)
			if tt.auth0ID != "" {
				req = testutil.WithAuthContext(req, tt.auth0ID, tt.email)
			}

			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)

			assert.Equal(t, tt.wantStatusCode, rec.Code)

			if tt.wantStatusCode == http.StatusOK {
				var response orderResponse
				err := json.NewDecoder(rec.Body).Decode(&response)
				require.NoError(t, err)
				assert.Equal(t, tt.order.ID, response.ID)
			}
		})
	}
}

func TestOrderHandler_Cancel(t *testing.T) {
	clientID := uuid.New()
	orderID := uuid.New()
	otherClientID := uuid.New()

	client := &domain.Client{
		ID:       clientID,
		HoldedID: "holded-123",
		Email:    "test@example.com",
	}

	tests := []struct {
		name           string
		urlID          string
		auth0ID        string
		email          string
		client         *domain.Client
		order          *domain.Order
		orderErr       error
		wantStatusCode int
	}{
		{
			name:    "successful cancellation",
			urlID:   orderID.String(),
			auth0ID: "auth0|123",
			email:   "test@example.com",
			client:  client,
			order: &domain.Order{
				ID:       orderID,
				ClientID: clientID,
				Status:   domain.OrderStatusPending,
			},
			wantStatusCode: http.StatusNoContent,
		},
		{
			name:           "invalid uuid",
			urlID:          "not-a-uuid",
			auth0ID:        "auth0|123",
			email:          "test@example.com",
			client:         client,
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "order not found",
			urlID:          orderID.String(),
			auth0ID:        "auth0|123",
			email:          "test@example.com",
			client:         client,
			orderErr:       repository.ErrNotFound,
			wantStatusCode: http.StatusNotFound,
		},
		{
			name:    "order belongs to different client",
			urlID:   orderID.String(),
			auth0ID: "auth0|123",
			email:   "test@example.com",
			client:  client,
			order: &domain.Order{
				ID:       orderID,
				ClientID: otherClientID,
				Status:   domain.OrderStatusPending,
			},
			wantStatusCode: http.StatusNotFound,
		},
		{
			name:    "order not pending",
			urlID:   orderID.String(),
			auth0ID: "auth0|123",
			email:   "test@example.com",
			client:  client,
			order: &domain.Order{
				ID:       orderID,
				ClientID: clientID,
				Status:   domain.OrderStatusApproved,
			},
			wantStatusCode: http.StatusConflict,
		},
		{
			name:           "no auth context",
			urlID:          orderID.String(),
			auth0ID:        "",
			email:          "",
			wantStatusCode: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orderRepo := &testutil.MockOrderRepo{GetByIDResult: tt.order, GetByIDErr: tt.orderErr}
			productRepo := &testutil.MockProductRepo{}
			clientRepo := &testutil.MockClientRepo{
				GetByAuth0IDResult: tt.client,
				GetByEmailResult:   tt.client,
			}
			handler := setupOrderHandler(orderRepo, productRepo, clientRepo)

			r := chi.NewRouter()
			r.Post("/orders/{id}/cancel", handler.Cancel)

			req := httptest.NewRequest(http.MethodPost, "/orders/"+tt.urlID+"/cancel", nil)
			if tt.auth0ID != "" {
				req = testutil.WithAuthContext(req, tt.auth0ID, tt.email)
			}

			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)

			assert.Equal(t, tt.wantStatusCode, rec.Code)
		})
	}
}
