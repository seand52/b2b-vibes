//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	jwtmiddleware "github.com/auth0/go-jwt-middleware/v2"
	"github.com/auth0/go-jwt-middleware/v2/validator"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"b2b-orders-api/internal/clients/holded"
	"b2b-orders-api/internal/handlers"
	"b2b-orders-api/internal/middleware"
	postgresrepo "b2b-orders-api/internal/repository/postgres"
	"b2b-orders-api/internal/service/auth"
	"b2b-orders-api/internal/service/cart"
	"b2b-orders-api/internal/service/order"
)

// mockHoldedClient implements order.HoldedClient for testing
type mockHoldedClient struct{}

func (m *mockHoldedClient) CreateInvoice(ctx context.Context, req *holded.CreateInvoiceRequest) (*holded.Invoice, error) {
	return &holded.Invoice{
		ID:         "test-invoice-" + fmt.Sprintf("%d", time.Now().UnixNano()),
		InvoiceNum: "INV-TEST-001",
	}, nil
}

// Test constants
const (
	testRoleClaim = "https://test.example.com/roles"
)

// Package-level test resources
var (
	testDB     *pgxpool.Pool
	testServer *httptest.Server
	testRouter *chi.Mux
	testLogger *slog.Logger

	// Repositories
	productRepo      *postgresrepo.ProductRepository
	productImageRepo *postgresrepo.ProductImageRepository
	clientRepo       *postgresrepo.ClientRepository
	orderRepo        *postgresrepo.OrderRepository

	// Services
	authService  *auth.Service
	orderService *order.Service
	cartService  *cart.Service

	// Handlers
	healthHandler  *handlers.HealthHandler
	productHandler *handlers.ProductHandler
	orderHandler   *handlers.OrderHandler
	cartHandler    *handlers.CartHandler
	adminHandler   *handlers.AdminHandler
)

func TestMain(m *testing.M) {
	ctx := context.Background()

	// Create logger that discards output for tests
	testLogger = slog.New(slog.NewTextHandler(io.Discard, nil))

	// Start PostgreSQL container
	pgContainer, err := tcpostgres.Run(ctx,
		"postgres:16-alpine",
		tcpostgres.WithDatabase("testdb"),
		tcpostgres.WithUsername("test"),
		tcpostgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		log.Fatalf("failed to start postgres container: %v", err)
	}

	// Get connection string
	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		log.Fatalf("failed to get connection string: %v", err)
	}

	// Connect to database
	poolConfig, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		log.Fatalf("failed to parse pool config: %v", err)
	}

	testDB, err = pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	// Run migrations
	if err := runMigrations(ctx, testDB); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}

	// Initialize repositories
	productRepo = postgresrepo.NewProductRepository(testDB)
	productImageRepo = postgresrepo.NewProductImageRepository(testDB)
	clientRepo = postgresrepo.NewClientRepository(testDB)
	orderRepo = postgresrepo.NewOrderRepository(testDB)

	// Initialize services
	authService = auth.NewService(clientRepo, testLogger)
	mockHolded := &mockHoldedClient{}
	orderService = order.NewService(orderRepo, productRepo, clientRepo, mockHolded, testLogger)
	cartService = cart.NewService(orderRepo, productRepo, testLogger)

	// Initialize handlers
	healthHandler = handlers.NewHealthHandler(testDB, "test")
	productHandler = handlers.NewProductHandler(productRepo, productImageRepo, testLogger)
	orderHandler = handlers.NewOrderHandler(orderService, authService, testLogger)
	cartHandler = handlers.NewCartHandler(cartService, authService, testLogger)
	adminHandler = handlers.NewAdminHandler(orderService, clientRepo, testLogger)

	// Setup router with test auth middleware
	testRouter = setupTestRouter()
	testServer = httptest.NewServer(testRouter)

	// Run tests
	code := m.Run()

	// Cleanup
	testServer.Close()
	testDB.Close()

	if err := pgContainer.Terminate(ctx); err != nil {
		log.Printf("failed to terminate container: %v", err)
	}

	os.Exit(code)
}

func runMigrations(ctx context.Context, db *pgxpool.Pool) error {
	// Find project root by looking for go.mod
	dir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	// Walk up to find project root
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return fmt.Errorf("could not find project root (go.mod)")
		}
		dir = parent
	}

	migrationsDir := filepath.Join(dir, "migrations")

	// Read and apply migrations in order
	files, err := filepath.Glob(filepath.Join(migrationsDir, "*.up.sql"))
	if err != nil {
		return fmt.Errorf("globbing migrations: %w", err)
	}

	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("reading migration %s: %w", file, err)
		}

		if _, err := db.Exec(ctx, string(content)); err != nil {
			return fmt.Errorf("executing migration %s: %w", file, err)
		}
	}

	return nil
}

func setupTestRouter() *chi.Mux {
	r := chi.NewRouter()

	// Basic middleware
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.Timeout(30 * time.Second))

	// Health routes (no auth)
	r.Get("/health", healthHandler.Full)
	r.Get("/health/live", healthHandler.Live)
	r.Get("/health/ready", healthHandler.Ready)

	// API routes with test auth middleware
	r.Route("/api/v1", func(r chi.Router) {
		// Products
		r.Route("/products", func(r chi.Router) {
			r.Use(testAuthMiddleware)
			r.Get("/", productHandler.List)
			r.Get("/{id}", productHandler.Get)
		})

		// Orders
		r.Route("/orders", func(r chi.Router) {
			r.Use(testAuthMiddleware)
			r.Post("/", orderHandler.Create)
			r.Get("/", orderHandler.List)
			r.Get("/{id}", orderHandler.Get)
			r.Post("/{id}/cancel", orderHandler.Cancel)
		})

		// Cart
		r.Route("/cart", func(r chi.Router) {
			r.Use(testAuthMiddleware)
			r.Get("/", cartHandler.GetCart)
			r.Post("/", cartHandler.CreateCart)
			r.Delete("/", cartHandler.DiscardCart)

			r.Route("/items", func(r chi.Router) {
				r.Post("/", cartHandler.AddItem)
				r.Put("/", cartHandler.SetItems)
				r.Put("/{product_id}", cartHandler.UpdateItemQuantity)
				r.Delete("/{product_id}", cartHandler.RemoveItem)
			})

			r.Put("/notes", cartHandler.UpdateNotes)
			r.Post("/submit", cartHandler.Submit)
		})

		// Admin routes
		r.Route("/admin", func(r chi.Router) {
			r.Use(testAuthMiddleware)
			r.Use(testAdminMiddleware)

			r.Route("/orders", func(r chi.Router) {
				r.Get("/", adminHandler.ListOrders)
				r.Get("/{id}", adminHandler.GetOrder)
				r.Post("/{id}/approve", adminHandler.ApproveOrder)
				r.Post("/{id}/reject", adminHandler.RejectOrder)
			})

			r.Route("/clients", func(r chi.Router) {
				r.Get("/", adminHandler.ListClients)
				r.Get("/{id}", adminHandler.GetClient)
			})
		})
	})

	return r
}

// testAuthMiddleware reads test headers to simulate authentication
// Headers: X-Test-User-ID, X-Test-Email, X-Test-Admin
func testAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := r.Header.Get("X-Test-User-ID")
		email := r.Header.Get("X-Test-Email")

		if userID == "" || email == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"unauthorized","message":"Missing authentication headers"}`))
			return
		}

		// Store in context using the same keys as real auth middleware
		ctx := withTestAuth(r.Context(), userID, email, r.Header.Get("X-Test-Admin") == "true")
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// testAdminMiddleware checks for admin privileges
func testAdminMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		isAdmin := r.Header.Get("X-Test-Admin")
		if isAdmin != "true" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte(`{"error":"forbidden","message":"Admin access required"}`))
			return
		}
		next.ServeHTTP(w, r)
	})
}

// withTestAuth adds test auth info to context using the same format as Auth0 JWT middleware
func withTestAuth(ctx context.Context, auth0ID, email string, isAdmin bool) context.Context {
	extra := make(map[string]interface{})
	if isAdmin {
		extra[testRoleClaim] = []interface{}{"admin"}
	}

	claims := &validator.ValidatedClaims{
		RegisteredClaims: validator.RegisteredClaims{
			Subject: auth0ID,
		},
		CustomClaims: &middleware.CustomClaims{
			Email: email,
			Extra: extra,
		},
	}

	return context.WithValue(ctx, jwtmiddleware.ContextKey{}, claims)
}

// Helper functions for tests

// doRequest makes a request to the test server
func doRequest(t *testing.T, method, path string, body io.Reader, headers map[string]string) *http.Response {
	t.Helper()

	req, err := http.NewRequest(method, testServer.URL+path, body)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}

	return resp
}

// parseJSON parses JSON response body
func parseJSON(t *testing.T, resp *http.Response, v interface{}) {
	t.Helper()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}
	defer resp.Body.Close()

	if err := json.Unmarshal(body, v); err != nil {
		t.Fatalf("failed to parse JSON: %v\nBody: %s", err, string(body))
	}
}

// authHeaders returns authentication headers for a test client
func authHeaders(auth0ID, email string) map[string]string {
	return map[string]string{
		"X-Test-User-ID": auth0ID,
		"X-Test-Email":   email,
	}
}

// adminAuthHeaders returns authentication headers for an admin user
func adminAuthHeaders(auth0ID, email string) map[string]string {
	return map[string]string{
		"X-Test-User-ID": auth0ID,
		"X-Test-Email":   email,
		"X-Test-Admin":   "true",
	}
}

// cleanupClient removes all data for a client (for test isolation)
func cleanupClient(t *testing.T, ctx context.Context, clientID string) {
	t.Helper()

	// Delete all orders for the client (cascades to order items)
	_, err := testDB.Exec(ctx, `DELETE FROM orders WHERE client_id = $1`, clientID)
	if err != nil {
		t.Logf("warning: failed to cleanup orders for client %s: %v", clientID, err)
	}

	// Delete the client
	_, err = testDB.Exec(ctx, `DELETE FROM clients WHERE id = $1`, clientID)
	if err != nil {
		t.Logf("warning: failed to cleanup client %s: %v", clientID, err)
	}
}

// cleanupProduct removes a product (for test isolation)
func cleanupProduct(t *testing.T, ctx context.Context, productID string) {
	t.Helper()

	// Delete product images first (cascade should handle this but be explicit)
	_, err := testDB.Exec(ctx, `DELETE FROM product_images WHERE product_id = $1`, productID)
	if err != nil {
		t.Logf("warning: failed to cleanup product images for product %s: %v", productID, err)
	}

	// Delete the product
	_, err = testDB.Exec(ctx, `DELETE FROM products WHERE id = $1`, productID)
	if err != nil {
		t.Logf("warning: failed to cleanup product %s: %v", productID, err)
	}
}

// cleanupOrder removes an order (for test isolation)
func cleanupOrder(t *testing.T, ctx context.Context, orderID string) {
	t.Helper()

	_, err := testDB.Exec(ctx, `DELETE FROM orders WHERE id = $1`, orderID)
	if err != nil {
		t.Logf("warning: failed to cleanup order %s: %v", orderID, err)
	}
}
