//go:build integration

package integration

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProducts_List(t *testing.T) {
	ctx := context.Background()

	// Create test client for authentication
	client := CreateTestClient(t, ctx, "ProductListClient")

	// Create some test products
	product1 := CreateTestProduct(t, ctx, "Test Product 1", 10.00, 100)
	product2 := CreateTestProduct(t, ctx, "Test Product 2", 20.00, 50)

	t.Run("lists active products", func(t *testing.T) {
		resp := doRequest(t, http.MethodGet, "/api/v1/products", nil, authHeaders(client.Auth0ID, client.Email))
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var products []map[string]interface{}
		parseJSON(t, resp, &products)

		// Should contain at least our test products
		require.GreaterOrEqual(t, len(products), 2)

		// Find our test products
		var found1, found2 bool
		for _, p := range products {
			if p["id"] == product1.ID.String() {
				found1 = true
				assert.Equal(t, product1.Name, p["name"])
			}
			if p["id"] == product2.ID.String() {
				found2 = true
				assert.Equal(t, product2.Name, p["name"])
			}
		}
		assert.True(t, found1, "product1 should be in list")
		assert.True(t, found2, "product2 should be in list")
	})

	t.Run("excludes inactive products", func(t *testing.T) {
		inactiveProduct := CreateTestProductInactive(t, ctx, "Inactive Product")

		resp := doRequest(t, http.MethodGet, "/api/v1/products", nil, authHeaders(client.Auth0ID, client.Email))
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var products []map[string]interface{}
		parseJSON(t, resp, &products)

		// Inactive product should not be in list
		for _, p := range products {
			assert.NotEqual(t, inactiveProduct.ID.String(), p["id"], "inactive product should not be listed")
		}
	})

	t.Run("requires authentication", func(t *testing.T) {
		resp := doRequest(t, http.MethodGet, "/api/v1/products", nil, nil)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
}

func TestProducts_Get(t *testing.T) {
	ctx := context.Background()

	// Create test client for authentication
	client := CreateTestClient(t, ctx, "ProductGetClient")

	// Create a test product
	product := CreateTestProduct(t, ctx, "Get Test Product", 25.50, 75)

	t.Run("gets product by ID", func(t *testing.T) {
		resp := doRequest(t, http.MethodGet, "/api/v1/products/"+product.ID.String(), nil, authHeaders(client.Auth0ID, client.Email))
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		parseJSON(t, resp, &result)

		assert.Equal(t, product.ID.String(), result["id"])
		assert.Equal(t, product.Name, result["name"])
		assert.Equal(t, product.SKU, result["sku"])
		assert.Equal(t, product.Price, result["price"])
		assert.Equal(t, float64(product.StockQuantity), result["stock_quantity"])
	})

	t.Run("returns 404 for non-existent product", func(t *testing.T) {
		fakeID := uuid.New().String()
		resp := doRequest(t, http.MethodGet, "/api/v1/products/"+fakeID, nil, authHeaders(client.Auth0ID, client.Email))
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("returns 404 for inactive product", func(t *testing.T) {
		inactiveProduct := CreateTestProductInactive(t, ctx, "Inactive Get Product")

		resp := doRequest(t, http.MethodGet, "/api/v1/products/"+inactiveProduct.ID.String(), nil, authHeaders(client.Auth0ID, client.Email))
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("returns 400 for invalid UUID", func(t *testing.T) {
		resp := doRequest(t, http.MethodGet, "/api/v1/products/not-a-uuid", nil, authHeaders(client.Auth0ID, client.Email))
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("requires authentication", func(t *testing.T) {
		resp := doRequest(t, http.MethodGet, "/api/v1/products/"+product.ID.String(), nil, nil)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
}
