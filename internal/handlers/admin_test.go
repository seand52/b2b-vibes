package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"b2b-orders-api/internal/domain"
	"b2b-orders-api/internal/repository"
	"b2b-orders-api/internal/service/order"
	"b2b-orders-api/internal/testutil"
)

func setupAdminHandler(orderRepo *testutil.MockOrderRepo, clientRepo *testutil.MockClientRepo) *AdminHandler {
	svc := order.NewService(orderRepo, nil, nil, nil, testutil.NewDiscardLogger())
	return NewAdminHandler(svc, clientRepo, testutil.NewDiscardLogger())
}

func TestAdminHandler_ListOrders(t *testing.T) {
	orderID := uuid.New()
	clientID := uuid.New()

	tests := []struct {
		name           string
		queryParams    string
		orders         []domain.Order
		listErr        error
		wantStatusCode int
		wantLen        int
	}{
		{
			name: "successful list",
			orders: []domain.Order{
				{ID: orderID, ClientID: clientID, Status: domain.OrderStatusPending},
				{ID: uuid.New(), ClientID: clientID, Status: domain.OrderStatusApproved},
			},
			wantStatusCode: http.StatusOK,
			wantLen:        2,
		},
		{
			name:           "empty list",
			orders:         []domain.Order{},
			wantStatusCode: http.StatusOK,
			wantLen:        0,
		},
		{
			name:        "with status filter",
			queryParams: "?status=pending",
			orders: []domain.Order{
				{ID: orderID, ClientID: clientID, Status: domain.OrderStatusPending},
			},
			wantStatusCode: http.StatusOK,
			wantLen:        1,
		},
		{
			name:           "repository error",
			listErr:        assert.AnError,
			wantStatusCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orderRepo := &testutil.MockOrderRepo{ListResult: tt.orders, ListErr: tt.listErr}
			handler := setupAdminHandler(orderRepo, nil)

			req := httptest.NewRequest(http.MethodGet, "/admin/orders"+tt.queryParams, nil)
			rec := httptest.NewRecorder()

			handler.ListOrders(rec, req)

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

func TestAdminHandler_GetOrder(t *testing.T) {
	orderID := uuid.New()
	clientID := uuid.New()

	tests := []struct {
		name           string
		urlID          string
		order          *domain.Order
		getErr         error
		wantStatusCode int
	}{
		{
			name:  "successful get",
			urlID: orderID.String(),
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
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "order not found",
			urlID:          uuid.New().String(),
			getErr:         repository.ErrNotFound,
			wantStatusCode: http.StatusNotFound,
		},
		{
			name:           "repository error",
			urlID:          orderID.String(),
			getErr:         assert.AnError,
			wantStatusCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orderRepo := &testutil.MockOrderRepo{GetByIDResult: tt.order, GetByIDErr: tt.getErr}
			handler := setupAdminHandler(orderRepo, nil)

			r := chi.NewRouter()
			r.Get("/admin/orders/{id}", handler.GetOrder)

			req := httptest.NewRequest(http.MethodGet, "/admin/orders/"+tt.urlID, nil)
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

func TestAdminHandler_ApproveOrder(t *testing.T) {
	orderID := uuid.New()

	tests := []struct {
		name           string
		urlID          string
		body           map[string]string
		wantStatusCode int
	}{
		{
			name:           "invalid uuid",
			urlID:          "not-a-uuid",
			body:           map[string]string{"approved_by": "admin@example.com"},
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "missing approved_by",
			urlID:          orderID.String(),
			body:           map[string]string{},
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "empty approved_by",
			urlID:          orderID.String(),
			body:           map[string]string{"approved_by": ""},
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "invalid json",
			urlID:          orderID.String(),
			body:           nil, // Will send invalid JSON
			wantStatusCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orderRepo := &testutil.MockOrderRepo{}
			handler := setupAdminHandler(orderRepo, nil)

			r := chi.NewRouter()
			r.Post("/admin/orders/{id}/approve", handler.ApproveOrder)

			var body []byte
			if tt.body != nil {
				body, _ = json.Marshal(tt.body)
			} else {
				body = []byte("invalid json")
			}

			req := httptest.NewRequest(http.MethodPost, "/admin/orders/"+tt.urlID+"/approve", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			assert.Equal(t, tt.wantStatusCode, rec.Code)
		})
	}
}

func TestAdminHandler_RejectOrder(t *testing.T) {
	orderID := uuid.New()

	tests := []struct {
		name           string
		urlID          string
		body           map[string]string
		order          *domain.Order
		rejectErr      error
		wantStatusCode int
	}{
		{
			name:  "successful rejection",
			urlID: orderID.String(),
			body:  map[string]string{"reason": "Out of stock"},
			order: &domain.Order{
				ID:     orderID,
				Status: domain.OrderStatusPending,
			},
			wantStatusCode: http.StatusNoContent,
		},
		{
			name:           "invalid uuid",
			urlID:          "not-a-uuid",
			body:           map[string]string{"reason": "Out of stock"},
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "missing reason",
			urlID:          orderID.String(),
			body:           map[string]string{},
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "empty reason",
			urlID:          orderID.String(),
			body:           map[string]string{"reason": ""},
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "order not found",
			urlID:          orderID.String(),
			body:           map[string]string{"reason": "Out of stock"},
			order:          nil,
			wantStatusCode: http.StatusNotFound,
		},
		{
			name:  "order not pending",
			urlID: orderID.String(),
			body:  map[string]string{"reason": "Too late"},
			order: &domain.Order{
				ID:     orderID,
				Status: domain.OrderStatusApproved,
			},
			wantStatusCode: http.StatusConflict,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var getErr error
			if tt.order == nil && tt.name == "order not found" {
				getErr = repository.ErrNotFound
			}
			orderRepo := &testutil.MockOrderRepo{
				GetByIDResult: tt.order,
				GetByIDErr:    getErr,
				RejectErr:     tt.rejectErr,
			}
			handler := setupAdminHandler(orderRepo, nil)

			r := chi.NewRouter()
			r.Post("/admin/orders/{id}/reject", handler.RejectOrder)

			body, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPost, "/admin/orders/"+tt.urlID+"/reject", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			assert.Equal(t, tt.wantStatusCode, rec.Code)
		})
	}
}

func TestAdminHandler_ListClients(t *testing.T) {
	clientID := uuid.New()

	tests := []struct {
		name           string
		queryParams    string
		clients        []domain.Client
		listErr        error
		wantStatusCode int
		wantLen        int
	}{
		{
			name: "successful list",
			clients: []domain.Client{
				{ID: clientID, Email: "client1@example.com", CompanyName: "Company 1", IsActive: true, CreatedAt: time.Now()},
				{ID: uuid.New(), Email: "client2@example.com", CompanyName: "Company 2", IsActive: true, CreatedAt: time.Now()},
			},
			wantStatusCode: http.StatusOK,
			wantLen:        2,
		},
		{
			name:           "empty list",
			clients:        []domain.Client{},
			wantStatusCode: http.StatusOK,
			wantLen:        0,
		},
		{
			name:        "with search filter",
			queryParams: "?search=company",
			clients: []domain.Client{
				{ID: clientID, Email: "client@company.com", CompanyName: "Company 1", IsActive: true, CreatedAt: time.Now()},
			},
			wantStatusCode: http.StatusOK,
			wantLen:        1,
		},
		{
			name:           "repository error",
			listErr:        assert.AnError,
			wantStatusCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clientRepo := &testutil.MockClientRepo{ListResult: tt.clients, ListErr: tt.listErr}
			handler := setupAdminHandler(nil, clientRepo)

			req := httptest.NewRequest(http.MethodGet, "/admin/clients"+tt.queryParams, nil)
			rec := httptest.NewRecorder()

			handler.ListClients(rec, req)

			assert.Equal(t, tt.wantStatusCode, rec.Code)

			if tt.wantStatusCode == http.StatusOK {
				var response []clientResponse
				err := json.NewDecoder(rec.Body).Decode(&response)
				require.NoError(t, err)
				assert.Len(t, response, tt.wantLen)
			}
		})
	}
}

func TestAdminHandler_GetClient(t *testing.T) {
	clientID := uuid.New()
	auth0ID := "auth0|123"

	tests := []struct {
		name           string
		urlID          string
		client         *domain.Client
		getErr         error
		wantStatusCode int
		wantLinked     bool
	}{
		{
			name:  "successful get - linked client",
			urlID: clientID.String(),
			client: &domain.Client{
				ID:          clientID,
				HoldedID:    "holded-123",
				Auth0ID:     &auth0ID,
				Email:       "client@example.com",
				CompanyName: "Test Company",
				IsActive:    true,
				CreatedAt:   time.Now(),
			},
			wantStatusCode: http.StatusOK,
			wantLinked:     true,
		},
		{
			name:  "successful get - unlinked client",
			urlID: clientID.String(),
			client: &domain.Client{
				ID:          clientID,
				HoldedID:    "holded-456",
				Auth0ID:     nil,
				Email:       "unlinked@example.com",
				CompanyName: "Unlinked Company",
				IsActive:    true,
				CreatedAt:   time.Now(),
			},
			wantStatusCode: http.StatusOK,
			wantLinked:     false,
		},
		{
			name:           "invalid uuid",
			urlID:          "not-a-uuid",
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "client not found",
			urlID:          uuid.New().String(),
			getErr:         repository.ErrNotFound,
			wantStatusCode: http.StatusNotFound,
		},
		{
			name:           "repository error",
			urlID:          clientID.String(),
			getErr:         assert.AnError,
			wantStatusCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clientRepo := &testutil.MockClientRepo{GetByIDResult: tt.client, GetByIDErr: tt.getErr}
			handler := setupAdminHandler(nil, clientRepo)

			r := chi.NewRouter()
			r.Get("/admin/clients/{id}", handler.GetClient)

			req := httptest.NewRequest(http.MethodGet, "/admin/clients/"+tt.urlID, nil)
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			assert.Equal(t, tt.wantStatusCode, rec.Code)

			if tt.wantStatusCode == http.StatusOK {
				var response clientResponse
				err := json.NewDecoder(rec.Body).Decode(&response)
				require.NoError(t, err)
				assert.Equal(t, tt.client.ID, response.ID)
				assert.Equal(t, tt.client.Email, response.Email)
				assert.Equal(t, tt.wantLinked, response.IsLinked)
			}
		})
	}
}
