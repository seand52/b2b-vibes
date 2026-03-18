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

func TestOrders_Create(t *testing.T) {
	ctx := context.Background()

	t.Run("creates order directly", func(t *testing.T) {
		client := CreateTestClient(t, ctx, "OrderCreateClient")
		product := CreateTestProduct(t, ctx, "Order Create Product", 25.00, 100)
		headers := authHeaders(client.Auth0ID, client.Email)

		body, _ := json.Marshal(map[string]interface{}{
			"items": []map[string]interface{}{
				{"product_id": product.ID.String(), "quantity": 3},
			},
			"notes": "Test order via direct API",
		})

		resp := doRequest(t, http.MethodPost, "/api/v1/orders", bytes.NewReader(body), headers)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var order map[string]interface{}
		parseJSON(t, resp, &order)

		assert.Equal(t, "pending", order["status"])
		assert.NotEmpty(t, order["id"])

		items := order["items"].([]interface{})
		require.Len(t, items, 1)

		item := items[0].(map[string]interface{})
		assert.Equal(t, product.ID.String(), item["product_id"])
		assert.Equal(t, float64(3), item["quantity"])
	})

	t.Run("rejects empty items", func(t *testing.T) {
		client := CreateTestClient(t, ctx, "OrderEmptyClient")
		headers := authHeaders(client.Auth0ID, client.Email)

		body, _ := json.Marshal(map[string]interface{}{
			"items": []map[string]interface{}{},
		})

		resp := doRequest(t, http.MethodPost, "/api/v1/orders", bytes.NewReader(body), headers)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("rejects non-existent product", func(t *testing.T) {
		client := CreateTestClient(t, ctx, "OrderNoProductClient")
		headers := authHeaders(client.Auth0ID, client.Email)

		body, _ := json.Marshal(map[string]interface{}{
			"items": []map[string]interface{}{
				{"product_id": uuid.New().String(), "quantity": 1},
			},
		})

		resp := doRequest(t, http.MethodPost, "/api/v1/orders", bytes.NewReader(body), headers)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("rejects insufficient stock", func(t *testing.T) {
		client := CreateTestClient(t, ctx, "OrderInsufficientClient")
		product := CreateTestProduct(t, ctx, "Low Stock Order Product", 10.00, 5)
		headers := authHeaders(client.Auth0ID, client.Email)

		body, _ := json.Marshal(map[string]interface{}{
			"items": []map[string]interface{}{
				{"product_id": product.ID.String(), "quantity": 10},
			},
		})

		resp := doRequest(t, http.MethodPost, "/api/v1/orders", bytes.NewReader(body), headers)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusConflict, resp.StatusCode)
	})

	t.Run("requires authentication", func(t *testing.T) {
		body, _ := json.Marshal(map[string]interface{}{
			"items": []map[string]interface{}{
				{"product_id": uuid.New().String(), "quantity": 1},
			},
		})

		resp := doRequest(t, http.MethodPost, "/api/v1/orders", bytes.NewReader(body), nil)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
}

func TestOrders_List(t *testing.T) {
	ctx := context.Background()

	client := CreateTestClient(t, ctx, "OrderListClient")
	product := CreateTestProduct(t, ctx, "Order List Product", 15.00, 100)
	headers := authHeaders(client.Auth0ID, client.Email)

	// Create a pending order
	price := 15.00
	lineTotal := 30.00
	CreateTestOrder(t, ctx, client.Client.ID, domain.OrderStatusPending, []OrderItemFixture{
		{ProductID: product.ID, Quantity: 2, UnitPrice: &price, LineTotal: &lineTotal},
	})

	t.Run("lists client's orders", func(t *testing.T) {
		resp := doRequest(t, http.MethodGet, "/api/v1/orders", nil, headers)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var orders []map[string]interface{}
		parseJSON(t, resp, &orders)

		require.GreaterOrEqual(t, len(orders), 1)

		// All orders should belong to this client
		for _, order := range orders {
			assert.Equal(t, "pending", order["status"])
		}
	})

	t.Run("filters by status", func(t *testing.T) {
		// Create an approved order
		CreateTestOrder(t, ctx, client.Client.ID, domain.OrderStatusApproved, nil)

		resp := doRequest(t, http.MethodGet, "/api/v1/orders?status=approved", nil, headers)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var orders []map[string]interface{}
		parseJSON(t, resp, &orders)

		for _, order := range orders {
			assert.Equal(t, "approved", order["status"])
		}
	})

	t.Run("returns empty list for client with no orders", func(t *testing.T) {
		newClient := CreateTestClient(t, ctx, "OrderListEmptyClient")
		headers := authHeaders(newClient.Auth0ID, newClient.Email)

		resp := doRequest(t, http.MethodGet, "/api/v1/orders", nil, headers)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var orders []map[string]interface{}
		parseJSON(t, resp, &orders)

		assert.Len(t, orders, 0)
	})
}

func TestOrders_Get(t *testing.T) {
	ctx := context.Background()

	client := CreateTestClient(t, ctx, "OrderGetClient")
	product := CreateTestProduct(t, ctx, "Order Get Product", 20.00, 100)
	headers := authHeaders(client.Auth0ID, client.Email)

	price := 20.00
	lineTotal := 40.00
	order := CreateTestOrder(t, ctx, client.Client.ID, domain.OrderStatusPending, []OrderItemFixture{
		{ProductID: product.ID, Quantity: 2, UnitPrice: &price, LineTotal: &lineTotal},
	})

	t.Run("gets order by ID", func(t *testing.T) {
		resp := doRequest(t, http.MethodGet, "/api/v1/orders/"+order.ID.String(), nil, headers)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		parseJSON(t, resp, &result)

		assert.Equal(t, order.ID.String(), result["id"])
		assert.Equal(t, "pending", result["status"])
	})

	t.Run("returns 404 for non-existent order", func(t *testing.T) {
		resp := doRequest(t, http.MethodGet, "/api/v1/orders/"+uuid.New().String(), nil, headers)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("returns 404 for another client's order", func(t *testing.T) {
		// Create another client with their own order
		otherClient := CreateTestClient(t, ctx, "OrderGetOtherClient")
		otherOrder := CreateTestOrder(t, ctx, otherClient.Client.ID, domain.OrderStatusPending, nil)

		// Try to access other client's order
		resp := doRequest(t, http.MethodGet, "/api/v1/orders/"+otherOrder.ID.String(), nil, headers)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("returns 400 for invalid UUID", func(t *testing.T) {
		resp := doRequest(t, http.MethodGet, "/api/v1/orders/not-a-uuid", nil, headers)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

func TestOrders_Cancel(t *testing.T) {
	ctx := context.Background()

	t.Run("cancels pending order", func(t *testing.T) {
		client := CreateTestClient(t, ctx, "OrderCancelClient")
		product := CreateTestProduct(t, ctx, "Order Cancel Product", 10.00, 100)
		headers := authHeaders(client.Auth0ID, client.Email)

		price := 10.00
		lineTotal := 20.00
		order := CreateTestOrder(t, ctx, client.Client.ID, domain.OrderStatusPending, []OrderItemFixture{
			{ProductID: product.ID, Quantity: 2, UnitPrice: &price, LineTotal: &lineTotal},
		})

		resp := doRequest(t, http.MethodPost, "/api/v1/orders/"+order.ID.String()+"/cancel", nil, headers)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNoContent, resp.StatusCode)

		// Verify order is now cancelled
		getResp := doRequest(t, http.MethodGet, "/api/v1/orders/"+order.ID.String(), nil, headers)
		defer getResp.Body.Close()

		var result map[string]interface{}
		parseJSON(t, getResp, &result)
		assert.Equal(t, "cancelled", result["status"])
	})

	t.Run("cannot cancel approved order", func(t *testing.T) {
		client := CreateTestClient(t, ctx, "OrderCancelApprovedClient")
		headers := authHeaders(client.Auth0ID, client.Email)

		order := CreateTestOrder(t, ctx, client.Client.ID, domain.OrderStatusApproved, nil)

		resp := doRequest(t, http.MethodPost, "/api/v1/orders/"+order.ID.String()+"/cancel", nil, headers)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusConflict, resp.StatusCode)
	})

	t.Run("cannot cancel another client's order", func(t *testing.T) {
		client := CreateTestClient(t, ctx, "OrderCancelOwnerClient")
		otherClient := CreateTestClient(t, ctx, "OrderCancelOtherClient")
		headers := authHeaders(client.Auth0ID, client.Email)

		otherOrder := CreateTestOrder(t, ctx, otherClient.Client.ID, domain.OrderStatusPending, nil)

		resp := doRequest(t, http.MethodPost, "/api/v1/orders/"+otherOrder.ID.String()+"/cancel", nil, headers)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("returns 404 for non-existent order", func(t *testing.T) {
		client := CreateTestClient(t, ctx, "OrderCancelNotFoundClient")
		headers := authHeaders(client.Auth0ID, client.Email)

		resp := doRequest(t, http.MethodPost, "/api/v1/orders/"+uuid.New().String()+"/cancel", nil, headers)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
}
