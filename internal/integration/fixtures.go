//go:build integration

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"b2b-orders-api/internal/domain"
)

// TestClient represents a test client with associated auth info
type TestClient struct {
	Client  *domain.Client
	Auth0ID string
	Email   string
}

// CreateTestClient creates a client in the database for testing
func CreateTestClient(t *testing.T, ctx context.Context, name string) *TestClient {
	t.Helper()

	clientID := uuid.New()
	holdedID := "holded-" + clientID.String()[:8]
	auth0ID := "auth0|test-" + clientID.String()[:8]
	email := name + "@test.example.com"

	client := &domain.Client{
		ID:          clientID,
		HoldedID:    holdedID,
		Auth0ID:     &auth0ID,
		Email:       email,
		CompanyName: name + " Corp",
		ContactName: name,
		Phone:       "+1234567890",
		VATType:     domain.VATTypeCIF,
		VATNumber:   "B12345678",
		IsActive:    true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Insert client
	_, err := testDB.Exec(ctx, `
		INSERT INTO clients (id, holded_id, auth0_id, email, company_name, contact_name, phone, vat_type, vat_number, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`, client.ID, client.HoldedID, client.Auth0ID, client.Email, client.CompanyName,
		client.ContactName, client.Phone, client.VATType, client.VATNumber,
		client.IsActive, client.CreatedAt, client.UpdatedAt)

	if err != nil {
		t.Fatalf("failed to create test client: %v", err)
	}

	// Register cleanup
	t.Cleanup(func() {
		cleanupClient(t, context.Background(), clientID.String())
	})

	return &TestClient{
		Client:  client,
		Auth0ID: auth0ID,
		Email:   email,
	}
}

// CreateTestClientUnlinked creates a client without an Auth0 ID (not yet signed up)
func CreateTestClientUnlinked(t *testing.T, ctx context.Context, name string) *domain.Client {
	t.Helper()

	clientID := uuid.New()
	holdedID := "holded-" + clientID.String()[:8]
	email := name + "@test.example.com"

	client := &domain.Client{
		ID:          clientID,
		HoldedID:    holdedID,
		Auth0ID:     nil, // Not linked
		Email:       email,
		CompanyName: name + " Corp",
		ContactName: name,
		IsActive:    true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Insert client
	_, err := testDB.Exec(ctx, `
		INSERT INTO clients (id, holded_id, email, company_name, contact_name, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, client.ID, client.HoldedID, client.Email, client.CompanyName,
		client.ContactName, client.IsActive, client.CreatedAt, client.UpdatedAt)

	if err != nil {
		t.Fatalf("failed to create test client: %v", err)
	}

	// Register cleanup
	t.Cleanup(func() {
		cleanupClient(t, context.Background(), clientID.String())
	})

	return client
}

// CreateTestProduct creates a product in the database for testing
func CreateTestProduct(t *testing.T, ctx context.Context, name string, price float64, stock int) *domain.Product {
	t.Helper()

	productID := uuid.New()
	holdedID := "holded-prod-" + productID.String()[:8]
	sku := "SKU-" + productID.String()[:8]

	product := &domain.Product{
		ID:               productID,
		HoldedID:         holdedID,
		SKU:              sku,
		Name:             name,
		Description:      "Test product: " + name,
		Category:         "Test Category",
		Price:            price,
		TaxRate:          21.0,
		StockQuantity:    stock,
		MinOrderQuantity: 1,
		IsActive:         true,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	// Insert product
	_, err := testDB.Exec(ctx, `
		INSERT INTO products (id, holded_id, sku, name, description, category, price, tax_rate, stock_quantity, min_order_quantity, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`, product.ID, product.HoldedID, product.SKU, product.Name, product.Description,
		product.Category, product.Price, product.TaxRate, product.StockQuantity,
		product.MinOrderQuantity, product.IsActive, product.CreatedAt, product.UpdatedAt)

	if err != nil {
		t.Fatalf("failed to create test product: %v", err)
	}

	// Register cleanup
	t.Cleanup(func() {
		cleanupProduct(t, context.Background(), productID.String())
	})

	return product
}

// CreateTestProductInactive creates an inactive product for testing
func CreateTestProductInactive(t *testing.T, ctx context.Context, name string) *domain.Product {
	t.Helper()

	productID := uuid.New()
	holdedID := "holded-prod-" + productID.String()[:8]
	sku := "SKU-" + productID.String()[:8]

	product := &domain.Product{
		ID:               productID,
		HoldedID:         holdedID,
		SKU:              sku,
		Name:             name,
		Description:      "Inactive test product: " + name,
		Category:         "Test Category",
		Price:            10.0,
		TaxRate:          21.0,
		StockQuantity:    100,
		MinOrderQuantity: 1,
		IsActive:         false, // Inactive
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	// Insert product
	_, err := testDB.Exec(ctx, `
		INSERT INTO products (id, holded_id, sku, name, description, category, price, tax_rate, stock_quantity, min_order_quantity, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`, product.ID, product.HoldedID, product.SKU, product.Name, product.Description,
		product.Category, product.Price, product.TaxRate, product.StockQuantity,
		product.MinOrderQuantity, product.IsActive, product.CreatedAt, product.UpdatedAt)

	if err != nil {
		t.Fatalf("failed to create test product: %v", err)
	}

	// Register cleanup
	t.Cleanup(func() {
		cleanupProduct(t, context.Background(), productID.String())
	})

	return product
}

// CreateTestProductWithMinQuantity creates a product with a minimum order quantity
func CreateTestProductWithMinQuantity(t *testing.T, ctx context.Context, name string, price float64, stock, minQty int) *domain.Product {
	t.Helper()

	productID := uuid.New()
	holdedID := "holded-prod-" + productID.String()[:8]
	sku := "SKU-" + productID.String()[:8]

	product := &domain.Product{
		ID:               productID,
		HoldedID:         holdedID,
		SKU:              sku,
		Name:             name,
		Description:      "Test product with min qty: " + name,
		Category:         "Test Category",
		Price:            price,
		TaxRate:          21.0,
		StockQuantity:    stock,
		MinOrderQuantity: minQty,
		IsActive:         true,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	// Insert product
	_, err := testDB.Exec(ctx, `
		INSERT INTO products (id, holded_id, sku, name, description, category, price, tax_rate, stock_quantity, min_order_quantity, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`, product.ID, product.HoldedID, product.SKU, product.Name, product.Description,
		product.Category, product.Price, product.TaxRate, product.StockQuantity,
		product.MinOrderQuantity, product.IsActive, product.CreatedAt, product.UpdatedAt)

	if err != nil {
		t.Fatalf("failed to create test product: %v", err)
	}

	// Register cleanup
	t.Cleanup(func() {
		cleanupProduct(t, context.Background(), productID.String())
	})

	return product
}

// CreateTestOrder creates an order in the database for testing
func CreateTestOrder(t *testing.T, ctx context.Context, clientID uuid.UUID, status domain.OrderStatus, items []OrderItemFixture) *domain.Order {
	t.Helper()

	orderID := uuid.New()

	order := &domain.Order{
		ID:        orderID,
		ClientID:  clientID,
		Status:    status,
		Notes:     "Test order",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Insert order with explicit empty strings for nullable text fields
	_, err := testDB.Exec(ctx, `
		INSERT INTO orders (id, client_id, status, notes, admin_notes, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, order.ID, order.ClientID, order.Status, order.Notes, "", order.CreatedAt, order.UpdatedAt)

	if err != nil {
		t.Fatalf("failed to create test order: %v", err)
	}

	// Insert order items
	for _, item := range items {
		itemID := uuid.New()
		_, err := testDB.Exec(ctx, `
			INSERT INTO order_items (id, order_id, product_id, quantity, unit_price, line_total)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, itemID, orderID, item.ProductID, item.Quantity, item.UnitPrice, item.LineTotal)

		if err != nil {
			t.Fatalf("failed to create test order item: %v", err)
		}

		order.Items = append(order.Items, domain.OrderItem{
			ID:        itemID,
			OrderID:   orderID,
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
			UnitPrice: item.UnitPrice,
			LineTotal: item.LineTotal,
		})
	}

	// Register cleanup
	t.Cleanup(func() {
		cleanupOrder(t, context.Background(), orderID.String())
	})

	return order
}

// OrderItemFixture represents an order item for fixtures
type OrderItemFixture struct {
	ProductID uuid.UUID
	Quantity  int
	UnitPrice *float64
	LineTotal *float64
}

// CreateTestDraftOrder creates a draft order (cart) for testing
func CreateTestDraftOrder(t *testing.T, ctx context.Context, clientID uuid.UUID) *domain.Order {
	t.Helper()
	return CreateTestOrder(t, ctx, clientID, domain.OrderStatusDraft, nil)
}

// CreateTestPendingOrder creates a pending order for testing
func CreateTestPendingOrder(t *testing.T, ctx context.Context, clientID uuid.UUID, items []OrderItemFixture) *domain.Order {
	t.Helper()
	return CreateTestOrder(t, ctx, clientID, domain.OrderStatusPending, items)
}
