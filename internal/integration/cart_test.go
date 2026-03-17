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
)

func TestCart_CreateAndGet(t *testing.T) {
	ctx := context.Background()

	client := CreateTestClient(t, ctx, "CartCreateClient")
	headers := authHeaders(client.Auth0ID, client.Email)

	t.Run("creates new cart", func(t *testing.T) {
		resp := doRequest(t, http.MethodPost, "/api/v1/cart", nil, headers)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var cart map[string]interface{}
		parseJSON(t, resp, &cart)

		assert.Equal(t, "draft", cart["status"])
		assert.NotEmpty(t, cart["id"])
		items, ok := cart["items"].([]interface{})
		require.True(t, ok)
		assert.Len(t, items, 0)
	})

	t.Run("returns existing cart on subsequent create", func(t *testing.T) {
		// First create
		resp1 := doRequest(t, http.MethodPost, "/api/v1/cart", nil, headers)
		defer resp1.Body.Close()

		var cart1 map[string]interface{}
		parseJSON(t, resp1, &cart1)
		cartID := cart1["id"]

		// Second create should return same cart
		resp2 := doRequest(t, http.MethodPost, "/api/v1/cart", nil, headers)
		defer resp2.Body.Close()

		assert.Equal(t, http.StatusOK, resp2.StatusCode) // OK, not Created

		var cart2 map[string]interface{}
		parseJSON(t, resp2, &cart2)
		assert.Equal(t, cartID, cart2["id"])
	})

	t.Run("gets cart", func(t *testing.T) {
		resp := doRequest(t, http.MethodGet, "/api/v1/cart", nil, headers)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var cart map[string]interface{}
		parseJSON(t, resp, &cart)
		assert.Equal(t, "draft", cart["status"])
	})
}

func TestCart_AddItem(t *testing.T) {
	ctx := context.Background()

	client := CreateTestClient(t, ctx, "CartAddItemClient")
	product := CreateTestProduct(t, ctx, "Cart Add Product", 15.00, 100)
	headers := authHeaders(client.Auth0ID, client.Email)

	// Create cart first
	resp := doRequest(t, http.MethodPost, "/api/v1/cart", nil, headers)
	resp.Body.Close()

	t.Run("adds item to cart", func(t *testing.T) {
		body, _ := json.Marshal(map[string]interface{}{
			"product_id": product.ID.String(),
			"quantity":   5,
		})

		resp := doRequest(t, http.MethodPost, "/api/v1/cart/items", bytes.NewReader(body), headers)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var cart map[string]interface{}
		parseJSON(t, resp, &cart)

		items := cart["items"].([]interface{})
		require.Len(t, items, 1)

		item := items[0].(map[string]interface{})
		assert.Equal(t, product.ID.String(), item["product_id"])
		assert.Equal(t, float64(5), item["quantity"])
		assert.Equal(t, product.Name, item["product_name"])
	})

	t.Run("increments quantity when adding same product", func(t *testing.T) {
		body, _ := json.Marshal(map[string]interface{}{
			"product_id": product.ID.String(),
			"quantity":   3,
		})

		resp := doRequest(t, http.MethodPost, "/api/v1/cart/items", bytes.NewReader(body), headers)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var cart map[string]interface{}
		parseJSON(t, resp, &cart)

		items := cart["items"].([]interface{})
		require.Len(t, items, 1)

		item := items[0].(map[string]interface{})
		assert.Equal(t, float64(8), item["quantity"]) // 5 + 3
	})

	t.Run("rejects invalid product ID", func(t *testing.T) {
		body, _ := json.Marshal(map[string]interface{}{
			"product_id": "not-a-uuid",
			"quantity":   1,
		})

		resp := doRequest(t, http.MethodPost, "/api/v1/cart/items", bytes.NewReader(body), headers)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("rejects non-existent product", func(t *testing.T) {
		body, _ := json.Marshal(map[string]interface{}{
			"product_id": uuid.New().String(),
			"quantity":   1,
		})

		resp := doRequest(t, http.MethodPost, "/api/v1/cart/items", bytes.NewReader(body), headers)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("rejects inactive product", func(t *testing.T) {
		inactiveProduct := CreateTestProductInactive(t, ctx, "Inactive Cart Product")

		body, _ := json.Marshal(map[string]interface{}{
			"product_id": inactiveProduct.ID.String(),
			"quantity":   1,
		})

		resp := doRequest(t, http.MethodPost, "/api/v1/cart/items", bytes.NewReader(body), headers)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("rejects zero quantity", func(t *testing.T) {
		body, _ := json.Marshal(map[string]interface{}{
			"product_id": product.ID.String(),
			"quantity":   0,
		})

		resp := doRequest(t, http.MethodPost, "/api/v1/cart/items", bytes.NewReader(body), headers)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("rejects negative quantity", func(t *testing.T) {
		body, _ := json.Marshal(map[string]interface{}{
			"product_id": product.ID.String(),
			"quantity":   -1,
		})

		resp := doRequest(t, http.MethodPost, "/api/v1/cart/items", bytes.NewReader(body), headers)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

func TestCart_UpdateItemQuantity(t *testing.T) {
	ctx := context.Background()

	client := CreateTestClient(t, ctx, "CartUpdateClient")
	product := CreateTestProduct(t, ctx, "Cart Update Product", 25.00, 100)
	headers := authHeaders(client.Auth0ID, client.Email)

	// Create cart and add item
	doRequest(t, http.MethodPost, "/api/v1/cart", nil, headers).Body.Close()

	addBody, _ := json.Marshal(map[string]interface{}{
		"product_id": product.ID.String(),
		"quantity":   5,
	})
	doRequest(t, http.MethodPost, "/api/v1/cart/items", bytes.NewReader(addBody), headers).Body.Close()

	t.Run("updates item quantity", func(t *testing.T) {
		body, _ := json.Marshal(map[string]interface{}{
			"quantity": 10,
		})

		resp := doRequest(t, http.MethodPut, "/api/v1/cart/items/"+product.ID.String(), bytes.NewReader(body), headers)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var cart map[string]interface{}
		parseJSON(t, resp, &cart)

		items := cart["items"].([]interface{})
		require.Len(t, items, 1)

		item := items[0].(map[string]interface{})
		assert.Equal(t, float64(10), item["quantity"])
	})

	t.Run("removes item when quantity is 0", func(t *testing.T) {
		body, _ := json.Marshal(map[string]interface{}{
			"quantity": 0,
		})

		resp := doRequest(t, http.MethodPut, "/api/v1/cart/items/"+product.ID.String(), bytes.NewReader(body), headers)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var cart map[string]interface{}
		parseJSON(t, resp, &cart)

		items := cart["items"].([]interface{})
		assert.Len(t, items, 0)
	})

	t.Run("returns 404 for item not in cart", func(t *testing.T) {
		body, _ := json.Marshal(map[string]interface{}{
			"quantity": 5,
		})

		resp := doRequest(t, http.MethodPut, "/api/v1/cart/items/"+uuid.New().String(), bytes.NewReader(body), headers)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
}

func TestCart_RemoveItem(t *testing.T) {
	ctx := context.Background()

	client := CreateTestClient(t, ctx, "CartRemoveClient")
	product := CreateTestProduct(t, ctx, "Cart Remove Product", 30.00, 100)
	headers := authHeaders(client.Auth0ID, client.Email)

	// Create cart and add item
	doRequest(t, http.MethodPost, "/api/v1/cart", nil, headers).Body.Close()

	addBody, _ := json.Marshal(map[string]interface{}{
		"product_id": product.ID.String(),
		"quantity":   5,
	})
	doRequest(t, http.MethodPost, "/api/v1/cart/items", bytes.NewReader(addBody), headers).Body.Close()

	t.Run("removes item from cart", func(t *testing.T) {
		resp := doRequest(t, http.MethodDelete, "/api/v1/cart/items/"+product.ID.String(), nil, headers)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var cart map[string]interface{}
		parseJSON(t, resp, &cart)

		items := cart["items"].([]interface{})
		assert.Len(t, items, 0)
	})
}

func TestCart_SetItems(t *testing.T) {
	ctx := context.Background()

	client := CreateTestClient(t, ctx, "CartSetItemsClient")
	product1 := CreateTestProduct(t, ctx, "Cart Set Product 1", 10.00, 100)
	product2 := CreateTestProduct(t, ctx, "Cart Set Product 2", 20.00, 100)
	headers := authHeaders(client.Auth0ID, client.Email)

	// Create cart
	doRequest(t, http.MethodPost, "/api/v1/cart", nil, headers).Body.Close()

	t.Run("sets all items at once", func(t *testing.T) {
		body, _ := json.Marshal(map[string]interface{}{
			"items": []map[string]interface{}{
				{"product_id": product1.ID.String(), "quantity": 2},
				{"product_id": product2.ID.String(), "quantity": 3},
			},
		})

		resp := doRequest(t, http.MethodPut, "/api/v1/cart/items", bytes.NewReader(body), headers)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var cart map[string]interface{}
		parseJSON(t, resp, &cart)

		items := cart["items"].([]interface{})
		assert.Len(t, items, 2)

		summary := cart["summary"].(map[string]interface{})
		assert.Equal(t, float64(2), summary["item_count"])
		assert.Equal(t, float64(5), summary["total_units"])
		// Subtotal: 2*10 + 3*20 = 80
		assert.Equal(t, float64(80), summary["subtotal"])
	})

	t.Run("replaces existing items", func(t *testing.T) {
		body, _ := json.Marshal(map[string]interface{}{
			"items": []map[string]interface{}{
				{"product_id": product1.ID.String(), "quantity": 1},
			},
		})

		resp := doRequest(t, http.MethodPut, "/api/v1/cart/items", bytes.NewReader(body), headers)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var cart map[string]interface{}
		parseJSON(t, resp, &cart)

		items := cart["items"].([]interface{})
		assert.Len(t, items, 1)
	})
}

func TestCart_UpdateNotes(t *testing.T) {
	ctx := context.Background()

	client := CreateTestClient(t, ctx, "CartNotesClient")
	headers := authHeaders(client.Auth0ID, client.Email)

	// Create cart
	doRequest(t, http.MethodPost, "/api/v1/cart", nil, headers).Body.Close()

	t.Run("updates cart notes", func(t *testing.T) {
		body, _ := json.Marshal(map[string]interface{}{
			"notes": "Please deliver before 5 PM",
		})

		resp := doRequest(t, http.MethodPut, "/api/v1/cart/notes", bytes.NewReader(body), headers)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var cart map[string]interface{}
		parseJSON(t, resp, &cart)

		assert.Equal(t, "Please deliver before 5 PM", cart["notes"])
	})
}

func TestCart_Submit(t *testing.T) {
	ctx := context.Background()

	t.Run("submits cart as pending order", func(t *testing.T) {
		client := CreateTestClient(t, ctx, "CartSubmitClient")
		product := CreateTestProduct(t, ctx, "Cart Submit Product", 50.00, 100)
		headers := authHeaders(client.Auth0ID, client.Email)

		// Create cart and add item
		doRequest(t, http.MethodPost, "/api/v1/cart", nil, headers).Body.Close()

		addBody, _ := json.Marshal(map[string]interface{}{
			"product_id": product.ID.String(),
			"quantity":   2,
		})
		doRequest(t, http.MethodPost, "/api/v1/cart/items", bytes.NewReader(addBody), headers).Body.Close()

		// Submit cart
		resp := doRequest(t, http.MethodPost, "/api/v1/cart/submit", nil, headers)
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
		assert.Equal(t, float64(2), item["quantity"])
	})

	t.Run("rejects empty cart submission", func(t *testing.T) {
		client := CreateTestClient(t, ctx, "CartEmptySubmitClient")
		headers := authHeaders(client.Auth0ID, client.Email)

		// Create empty cart
		doRequest(t, http.MethodPost, "/api/v1/cart", nil, headers).Body.Close()

		// Try to submit
		resp := doRequest(t, http.MethodPost, "/api/v1/cart/submit", nil, headers)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("rejects submission when insufficient stock", func(t *testing.T) {
		client := CreateTestClient(t, ctx, "CartInsufficientStockClient")
		product := CreateTestProduct(t, ctx, "Low Stock Product", 10.00, 5) // Only 5 in stock
		headers := authHeaders(client.Auth0ID, client.Email)

		// Create cart and add item with quantity > stock
		doRequest(t, http.MethodPost, "/api/v1/cart", nil, headers).Body.Close()

		addBody, _ := json.Marshal(map[string]interface{}{
			"product_id": product.ID.String(),
			"quantity":   10, // More than available
		})
		doRequest(t, http.MethodPost, "/api/v1/cart/items", bytes.NewReader(addBody), headers).Body.Close()

		// Try to submit
		resp := doRequest(t, http.MethodPost, "/api/v1/cart/submit", nil, headers)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusConflict, resp.StatusCode)
	})

	t.Run("rejects submission when below minimum quantity", func(t *testing.T) {
		client := CreateTestClient(t, ctx, "CartMinQtyClient")
		product := CreateTestProductWithMinQuantity(t, ctx, "Min Qty Product", 10.00, 100, 5) // Min 5
		headers := authHeaders(client.Auth0ID, client.Email)

		// Create cart and add item with quantity < min
		doRequest(t, http.MethodPost, "/api/v1/cart", nil, headers).Body.Close()

		addBody, _ := json.Marshal(map[string]interface{}{
			"product_id": product.ID.String(),
			"quantity":   2, // Less than minimum
		})
		doRequest(t, http.MethodPost, "/api/v1/cart/items", bytes.NewReader(addBody), headers).Body.Close()

		// Try to submit
		resp := doRequest(t, http.MethodPost, "/api/v1/cart/submit", nil, headers)
		defer resp.Body.Close()

		// Should fail due to minimum quantity
		assert.NotEqual(t, http.StatusCreated, resp.StatusCode)
	})
}

func TestCart_Discard(t *testing.T) {
	ctx := context.Background()

	client := CreateTestClient(t, ctx, "CartDiscardClient")
	headers := authHeaders(client.Auth0ID, client.Email)

	t.Run("discards cart", func(t *testing.T) {
		// Create cart
		doRequest(t, http.MethodPost, "/api/v1/cart", nil, headers).Body.Close()

		// Discard
		resp := doRequest(t, http.MethodDelete, "/api/v1/cart", nil, headers)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNoContent, resp.StatusCode)

		// Getting cart should now return 404
		getResp := doRequest(t, http.MethodGet, "/api/v1/cart", nil, headers)
		defer getResp.Body.Close()

		assert.Equal(t, http.StatusNotFound, getResp.StatusCode)
	})

	t.Run("returns 404 when no cart exists", func(t *testing.T) {
		client2 := CreateTestClient(t, ctx, "CartDiscardNoCartClient")
		headers2 := authHeaders(client2.Auth0ID, client2.Email)

		resp := doRequest(t, http.MethodDelete, "/api/v1/cart", nil, headers2)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
}

func TestCart_NoCart(t *testing.T) {
	ctx := context.Background()

	client := CreateTestClient(t, ctx, "CartNoCartClient")
	headers := authHeaders(client.Auth0ID, client.Email)

	t.Run("returns 404 when getting non-existent cart", func(t *testing.T) {
		resp := doRequest(t, http.MethodGet, "/api/v1/cart", nil, headers)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
}
