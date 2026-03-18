//go:build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"b2b-orders-api/internal/domain"
)

func TestAdmin_ListOrders(t *testing.T) {
	ctx := context.Background()

	// Create admin user
	admin := CreateTestClient(t, ctx, "AdminListOrdersAdmin")
	adminHeaders := adminAuthHeaders(admin.Auth0ID, admin.Email)

	// Create regular clients with orders
	client1 := CreateTestClient(t, ctx, "AdminListOrdersClient1")
	client2 := CreateTestClient(t, ctx, "AdminListOrdersClient2")

	CreateTestOrder(t, ctx, client1.Client.ID, domain.OrderStatusPending, nil)
	CreateTestOrder(t, ctx, client2.Client.ID, domain.OrderStatusPending, nil)
	CreateTestOrder(t, ctx, client1.Client.ID, domain.OrderStatusApproved, nil)

	t.Run("lists all orders", func(t *testing.T) {
		resp := doRequest(t, http.MethodGet, "/api/v1/admin/orders", nil, adminHeaders)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var orders []map[string]interface{}
		parseJSON(t, resp, &orders)

		// Should have at least our 3 test orders
		require.GreaterOrEqual(t, len(orders), 3)
	})

	t.Run("filters by status", func(t *testing.T) {
		resp := doRequest(t, http.MethodGet, "/api/v1/admin/orders?status=pending", nil, adminHeaders)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var orders []map[string]interface{}
		parseJSON(t, resp, &orders)

		for _, order := range orders {
			assert.Equal(t, "pending", order["status"])
		}
	})

	t.Run("requires admin role", func(t *testing.T) {
		client := CreateTestClient(t, ctx, "AdminListOrdersRegular")
		regularHeaders := authHeaders(client.Auth0ID, client.Email)

		resp := doRequest(t, http.MethodGet, "/api/v1/admin/orders", nil, regularHeaders)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	})
}

func TestAdmin_GetOrder(t *testing.T) {
	ctx := context.Background()

	admin := CreateTestClient(t, ctx, "AdminGetOrderAdmin")
	adminHeaders := adminAuthHeaders(admin.Auth0ID, admin.Email)

	client := CreateTestClient(t, ctx, "AdminGetOrderClient")
	product := CreateTestProduct(t, ctx, "Admin Get Product", 30.00, 100)

	price := 30.00
	lineTotal := 60.00
	order := CreateTestOrder(t, ctx, client.Client.ID, domain.OrderStatusPending, []OrderItemFixture{
		{ProductID: product.ID, Quantity: 2, UnitPrice: &price, LineTotal: &lineTotal},
	})

	t.Run("gets any order", func(t *testing.T) {
		resp := doRequest(t, http.MethodGet, "/api/v1/admin/orders/"+order.ID.String(), nil, adminHeaders)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		parseJSON(t, resp, &result)

		assert.Equal(t, order.ID.String(), result["id"])
	})

	t.Run("returns 404 for non-existent order", func(t *testing.T) {
		resp := doRequest(t, http.MethodGet, "/api/v1/admin/orders/"+uuid.New().String(), nil, adminHeaders)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
}

func TestAdmin_ApproveOrder(t *testing.T) {
	ctx := context.Background()

	admin := CreateTestClient(t, ctx, "AdminApproveAdmin")
	adminHeaders := adminAuthHeaders(admin.Auth0ID, admin.Email)

	t.Run("approves pending order", func(t *testing.T) {
		client := CreateTestClient(t, ctx, "AdminApproveClient")
		product := CreateTestProduct(t, ctx, "Admin Approve Product", 25.00, 100)

		price := 25.00
		lineTotal := 50.00
		order := CreateTestOrder(t, ctx, client.Client.ID, domain.OrderStatusPending, []OrderItemFixture{
			{ProductID: product.ID, Quantity: 2, UnitPrice: &price, LineTotal: &lineTotal},
		})

		body, _ := json.Marshal(map[string]interface{}{
			"approved_by": "admin@example.com",
		})

		resp := doRequest(t, http.MethodPost, "/api/v1/admin/orders/"+order.ID.String()+"/approve", bytes.NewReader(body), adminHeaders)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		parseJSON(t, resp, &result)

		assert.Equal(t, "approved", result["status"])
		assert.NotNil(t, result["approved_at"])
	})

	t.Run("requires approved_by field", func(t *testing.T) {
		client := CreateTestClient(t, ctx, "AdminApproveNoByClient")
		order := CreateTestOrder(t, ctx, client.Client.ID, domain.OrderStatusPending, nil)

		body, _ := json.Marshal(map[string]interface{}{})

		resp := doRequest(t, http.MethodPost, "/api/v1/admin/orders/"+order.ID.String()+"/approve", bytes.NewReader(body), adminHeaders)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("cannot approve non-pending order", func(t *testing.T) {
		client := CreateTestClient(t, ctx, "AdminApproveApprovedClient")
		order := CreateTestOrder(t, ctx, client.Client.ID, domain.OrderStatusApproved, nil)

		body, _ := json.Marshal(map[string]interface{}{
			"approved_by": "admin@example.com",
		})

		resp := doRequest(t, http.MethodPost, "/api/v1/admin/orders/"+order.ID.String()+"/approve", bytes.NewReader(body), adminHeaders)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusConflict, resp.StatusCode)
	})

	t.Run("returns 404 for non-existent order", func(t *testing.T) {
		body, _ := json.Marshal(map[string]interface{}{
			"approved_by": "admin@example.com",
		})

		resp := doRequest(t, http.MethodPost, "/api/v1/admin/orders/"+uuid.New().String()+"/approve", bytes.NewReader(body), adminHeaders)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
}

func TestAdmin_RejectOrder(t *testing.T) {
	ctx := context.Background()

	admin := CreateTestClient(t, ctx, "AdminRejectAdmin")
	adminHeaders := adminAuthHeaders(admin.Auth0ID, admin.Email)

	t.Run("rejects pending order", func(t *testing.T) {
		client := CreateTestClient(t, ctx, "AdminRejectClient")
		order := CreateTestOrder(t, ctx, client.Client.ID, domain.OrderStatusPending, nil)

		body, _ := json.Marshal(map[string]interface{}{
			"reason": "Out of stock",
		})

		resp := doRequest(t, http.MethodPost, "/api/v1/admin/orders/"+order.ID.String()+"/reject", bytes.NewReader(body), adminHeaders)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNoContent, resp.StatusCode)

		// Verify order is rejected
		getResp := doRequest(t, http.MethodGet, "/api/v1/admin/orders/"+order.ID.String(), nil, adminHeaders)
		defer getResp.Body.Close()

		var result map[string]interface{}
		parseJSON(t, getResp, &result)
		assert.Equal(t, "rejected", result["status"])
	})

	t.Run("requires reason field", func(t *testing.T) {
		client := CreateTestClient(t, ctx, "AdminRejectNoReasonClient")
		order := CreateTestOrder(t, ctx, client.Client.ID, domain.OrderStatusPending, nil)

		body, _ := json.Marshal(map[string]interface{}{})

		resp := doRequest(t, http.MethodPost, "/api/v1/admin/orders/"+order.ID.String()+"/reject", bytes.NewReader(body), adminHeaders)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("cannot reject non-pending order", func(t *testing.T) {
		client := CreateTestClient(t, ctx, "AdminRejectApprovedClient")
		order := CreateTestOrder(t, ctx, client.Client.ID, domain.OrderStatusApproved, nil)

		body, _ := json.Marshal(map[string]interface{}{
			"reason": "Changed my mind",
		})

		resp := doRequest(t, http.MethodPost, "/api/v1/admin/orders/"+order.ID.String()+"/reject", bytes.NewReader(body), adminHeaders)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusConflict, resp.StatusCode)
	})
}

func TestAdmin_ListClients(t *testing.T) {
	ctx := context.Background()

	admin := CreateTestClient(t, ctx, "AdminListClientsAdmin")
	adminHeaders := adminAuthHeaders(admin.Auth0ID, admin.Email)

	// Create some test clients
	CreateTestClient(t, ctx, "AdminListClient1")
	CreateTestClient(t, ctx, "AdminListClient2")

	t.Run("lists all clients", func(t *testing.T) {
		resp := doRequest(t, http.MethodGet, "/api/v1/admin/clients", nil, adminHeaders)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var clients []map[string]interface{}
		parseJSON(t, resp, &clients)

		// Should have at least our test clients
		require.GreaterOrEqual(t, len(clients), 3)
	})

	t.Run("requires admin role", func(t *testing.T) {
		client := CreateTestClient(t, ctx, "AdminListClientsRegular")
		regularHeaders := authHeaders(client.Auth0ID, client.Email)

		resp := doRequest(t, http.MethodGet, "/api/v1/admin/clients", nil, regularHeaders)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	})
}

func TestAdmin_GetClient(t *testing.T) {
	ctx := context.Background()

	admin := CreateTestClient(t, ctx, "AdminGetClientAdmin")
	adminHeaders := adminAuthHeaders(admin.Auth0ID, admin.Email)

	client := CreateTestClient(t, ctx, "AdminGetClientTarget")

	t.Run("gets client by ID", func(t *testing.T) {
		resp := doRequest(t, http.MethodGet, "/api/v1/admin/clients/"+client.Client.ID.String(), nil, adminHeaders)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		parseJSON(t, resp, &result)

		assert.Equal(t, client.Client.ID.String(), result["id"])
		assert.Equal(t, client.Client.Email, result["email"])
	})

	t.Run("returns 404 for non-existent client", func(t *testing.T) {
		resp := doRequest(t, http.MethodGet, "/api/v1/admin/clients/"+uuid.New().String(), nil, adminHeaders)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("returns 400 for invalid UUID", func(t *testing.T) {
		resp := doRequest(t, http.MethodGet, "/api/v1/admin/clients/not-a-uuid", nil, adminHeaders)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}
