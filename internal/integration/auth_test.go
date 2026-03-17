//go:build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"b2b-orders-api/internal/domain"
)

func TestAuth_MissingCredentials(t *testing.T) {
	endpoints := []struct {
		method string
		path   string
		body   map[string]interface{}
	}{
		{http.MethodGet, "/api/v1/products", nil},
		{http.MethodGet, "/api/v1/products/" + uuid.New().String(), nil},
		{http.MethodGet, "/api/v1/orders", nil},
		{http.MethodPost, "/api/v1/orders", map[string]interface{}{"items": []interface{}{}}},
		{http.MethodGet, "/api/v1/cart", nil},
		{http.MethodPost, "/api/v1/cart", nil},
		{http.MethodGet, "/api/v1/admin/orders", nil},
		{http.MethodGet, "/api/v1/admin/clients", nil},
	}

	for _, ep := range endpoints {
		t.Run(ep.method+" "+ep.path+" returns 401 without auth", func(t *testing.T) {
			var body io.Reader
			if ep.body != nil {
				bodyBytes, _ := json.Marshal(ep.body)
				body = bytes.NewReader(bodyBytes)
			}

			resp := doRequest(t, ep.method, ep.path, body, nil)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
		})
	}
}

func TestAuth_AdminOnlyEndpoints(t *testing.T) {
	ctx := context.Background()

	// Create a regular (non-admin) user
	client := CreateTestClient(t, ctx, "AuthRegularClient")
	regularHeaders := authHeaders(client.Auth0ID, client.Email)

	endpoints := []struct {
		method string
		path   string
		body   map[string]interface{}
	}{
		{http.MethodGet, "/api/v1/admin/orders", nil},
		{http.MethodGet, "/api/v1/admin/orders/" + uuid.New().String(), nil},
		{http.MethodPost, "/api/v1/admin/orders/" + uuid.New().String() + "/approve", map[string]interface{}{"approved_by": "test"}},
		{http.MethodPost, "/api/v1/admin/orders/" + uuid.New().String() + "/reject", map[string]interface{}{"reason": "test"}},
		{http.MethodGet, "/api/v1/admin/clients", nil},
		{http.MethodGet, "/api/v1/admin/clients/" + uuid.New().String(), nil},
	}

	for _, ep := range endpoints {
		t.Run(ep.method+" "+ep.path+" returns 403 for non-admin", func(t *testing.T) {
			var body io.Reader
			if ep.body != nil {
				bodyBytes, _ := json.Marshal(ep.body)
				body = bytes.NewReader(bodyBytes)
			}

			resp := doRequest(t, ep.method, ep.path, body, regularHeaders)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusForbidden, resp.StatusCode)
		})
	}
}

func TestAuth_DataIsolation(t *testing.T) {
	ctx := context.Background()

	// Create two separate clients
	client1 := CreateTestClient(t, ctx, "AuthIsolationClient1")
	client2 := CreateTestClient(t, ctx, "AuthIsolationClient2")

	product := CreateTestProduct(t, ctx, "Auth Isolation Product", 20.00, 100)

	// Client 1 creates an order
	price := 20.00
	lineTotal := 40.00
	order1 := CreateTestOrder(t, ctx, client1.Client.ID, domain.OrderStatusPending, []OrderItemFixture{
		{ProductID: product.ID, Quantity: 2, UnitPrice: &price, LineTotal: &lineTotal},
	})

	t.Run("client cannot view another client's order", func(t *testing.T) {
		headers := authHeaders(client2.Auth0ID, client2.Email)

		resp := doRequest(t, http.MethodGet, "/api/v1/orders/"+order1.ID.String(), nil, headers)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("client cannot cancel another client's order", func(t *testing.T) {
		headers := authHeaders(client2.Auth0ID, client2.Email)

		resp := doRequest(t, http.MethodPost, "/api/v1/orders/"+order1.ID.String()+"/cancel", nil, headers)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("client only sees their own orders in list", func(t *testing.T) {
		// Create order for client2
		CreateTestOrder(t, ctx, client2.Client.ID, domain.OrderStatusPending, nil)

		headers := authHeaders(client2.Auth0ID, client2.Email)

		resp := doRequest(t, http.MethodGet, "/api/v1/orders", nil, headers)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var orders []map[string]interface{}
		parseJSON(t, resp, &orders)

		// Should not contain client1's order
		for _, order := range orders {
			assert.NotEqual(t, order1.ID.String(), order["id"])
		}
	})
}

func TestAuth_CartIsolation(t *testing.T) {
	ctx := context.Background()

	client1 := CreateTestClient(t, ctx, "AuthCartIsoClient1")
	client2 := CreateTestClient(t, ctx, "AuthCartIsoClient2")

	product := CreateTestProduct(t, ctx, "Auth Cart Product", 15.00, 100)

	// Client1 creates cart with item
	headers1 := authHeaders(client1.Auth0ID, client1.Email)
	doRequest(t, http.MethodPost, "/api/v1/cart", nil, headers1).Body.Close()

	body, _ := json.Marshal(map[string]interface{}{
		"product_id": product.ID.String(),
		"quantity":   5,
	})
	doRequest(t, http.MethodPost, "/api/v1/cart/items", bytes.NewReader(body), headers1).Body.Close()

	t.Run("client cannot see another client's cart", func(t *testing.T) {
		headers2 := authHeaders(client2.Auth0ID, client2.Email)

		// Client2 should get 404 (no cart) not client1's cart
		resp := doRequest(t, http.MethodGet, "/api/v1/cart", nil, headers2)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("each client has their own cart", func(t *testing.T) {
		headers2 := authHeaders(client2.Auth0ID, client2.Email)

		// Create cart for client2
		createResp := doRequest(t, http.MethodPost, "/api/v1/cart", nil, headers2)
		require.Equal(t, http.StatusCreated, createResp.StatusCode, "cart creation should succeed")
		createResp.Body.Close()

		// Get client2's cart
		resp := doRequest(t, http.MethodGet, "/api/v1/cart", nil, headers2)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var cart map[string]interface{}
		parseJSON(t, resp, &cart)

		// Should be empty (not have client1's items)
		items := cart["items"].([]interface{})
		assert.Len(t, items, 0)
	})
}

func TestAuth_AdminCanAccessAllOrders(t *testing.T) {
	ctx := context.Background()

	admin := CreateTestClient(t, ctx, "AuthAdminAccessAdmin")
	adminHeaders := adminAuthHeaders(admin.Auth0ID, admin.Email)

	client := CreateTestClient(t, ctx, "AuthAdminAccessClient")
	product := CreateTestProduct(t, ctx, "Auth Admin Product", 25.00, 100)

	price := 25.00
	lineTotal := 50.00
	order := CreateTestOrder(t, ctx, client.Client.ID, domain.OrderStatusPending, []OrderItemFixture{
		{ProductID: product.ID, Quantity: 2, UnitPrice: &price, LineTotal: &lineTotal},
	})

	t.Run("admin can view any client's order", func(t *testing.T) {
		resp := doRequest(t, http.MethodGet, "/api/v1/admin/orders/"+order.ID.String(), nil, adminHeaders)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		parseJSON(t, resp, &result)

		assert.Equal(t, order.ID.String(), result["id"])
	})

	t.Run("admin can view any client", func(t *testing.T) {
		resp := doRequest(t, http.MethodGet, "/api/v1/admin/clients/"+client.Client.ID.String(), nil, adminHeaders)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		parseJSON(t, resp, &result)

		assert.Equal(t, client.Client.ID.String(), result["id"])
	})
}
