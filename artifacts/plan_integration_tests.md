# Integration Test Plan: B2B Orders API

**Created:** 2026-03-17
**Status:** Proposed
**Author:** QA Engineer

## Overview

Add a full integration test suite using testcontainers-go to spin up real PostgreSQL and test the complete API flows with actual HTTP requests.

---

## Test Infrastructure

### Dependencies to Add

```go
// go.mod additions
github.com/testcontainers/testcontainers-go v0.31.0
github.com/testcontainers/testcontainers-go/modules/postgres v0.31.0
```

### Directory Structure

```
internal/
└── integration/
    ├── setup_test.go        # TestMain, container lifecycle, helpers
    ├── health_test.go       # Health endpoint tests
    ├── products_test.go     # Product browsing tests
    ├── cart_test.go         # Cart workflow tests
    ├── orders_test.go       # Order creation and cancellation
    ├── admin_test.go        # Admin approval/rejection flows
    ├── auth_test.go         # Authorization boundary tests
    └── fixtures.go          # Test data factories
```

### Build Tag

All integration tests use build tag to separate from unit tests:

```go
//go:build integration

package integration
```

### Makefile Target

```makefile
test-integration:
	CGO_ENABLED=0 go test -tags=integration -v ./internal/integration/...

test-all:
	CGO_ENABLED=0 go test ./...
	CGO_ENABLED=0 go test -tags=integration -v ./internal/integration/...
```

---

## Test Setup (`setup_test.go`)

### Container Lifecycle

```go
var (
    testDB        *pgxpool.Pool
    testServer    *httptest.Server
    testClient    *http.Client
    pgContainer   testcontainers.Container
)

func TestMain(m *testing.M) {
    ctx := context.Background()

    // 1. Start PostgreSQL container
    pgContainer, testDB = startPostgres(ctx)

    // 2. Run migrations
    runMigrations(testDB)

    // 3. Start test server (with mock auth)
    testServer = startTestServer(testDB)
    testClient = testServer.Client()

    // 4. Run tests
    code := m.Run()

    // 5. Cleanup
    testServer.Close()
    testDB.Close()
    pgContainer.Terminate(ctx)

    os.Exit(code)
}
```

### Auth Strategy: Bypass Middleware

For integration tests, we bypass Auth0 and inject test claims directly:

```go
// TestAuthMiddleware bypasses real Auth0 and injects claims from headers
type TestAuthMiddleware struct{}

func (m *TestAuthMiddleware) Authenticate(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Read test headers
        userID := r.Header.Get("X-Test-User-ID")      // Auth0 subject
        email := r.Header.Get("X-Test-Email")
        isAdmin := r.Header.Get("X-Test-Admin") == "true"

        if userID == "" {
            http.Error(w, "Unauthorized", 401)
            return
        }

        // Inject into context (same as real middleware)
        ctx := context.WithValue(r.Context(), auth0IDKey, userID)
        ctx = context.WithValue(ctx, emailKey, email)
        ctx = context.WithValue(ctx, adminKey, isAdmin)

        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

### Helper Functions

```go
// makeRequest creates an authenticated request
func makeRequest(method, path string, body interface{}, opts ...RequestOption) *http.Request

// asClient sets client auth headers
func asClient(clientID, email string) RequestOption

// asAdmin sets admin auth headers
func asAdmin(email string) RequestOption

// parseResponse decodes JSON response
func parseResponse[T any](t *testing.T, resp *http.Response) T

// assertStatus checks response status code
func assertStatus(t *testing.T, resp *http.Response, expected int)

// cleanupOrders deletes all orders (call in t.Cleanup)
func cleanupOrders(t *testing.T)
```

---

## Test Data Fixtures (`fixtures.go`)

### Seeded Test Data

```go
var (
    // Clients (pre-seeded, linked to Auth0 IDs)
    TestClient1 = Client{
        ID:       uuid.MustParse("11111111-1111-1111-1111-111111111111"),
        Auth0ID:  "auth0|client1",
        Email:    "client1@example.com",
        Company:  "Acme Corp",
        HoldedID: "holded-client-1",
    }
    TestClient2 = Client{
        ID:       uuid.MustParse("22222222-2222-2222-2222-222222222222"),
        Auth0ID:  "auth0|client2",
        Email:    "client2@example.com",
        Company:  "Beta Inc",
        HoldedID: "holded-client-2",
    }
    TestAdmin = Client{
        ID:       uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
        Auth0ID:  "auth0|admin",
        Email:    "admin@example.com",
        IsAdmin:  true,
    }

    // Products (pre-seeded)
    TestProduct1 = Product{
        ID:               uuid.MustParse("p1111111-1111-1111-1111-111111111111"),
        SKU:              "WIDGET-001",
        Name:             "Industrial Widget",
        Price:            99.99,
        StockQuantity:    100,
        MinOrderQuantity: 1,
        IsActive:         true,
    }
    TestProduct2 = Product{
        ID:               uuid.MustParse("p2222222-2222-2222-2222-222222222222"),
        SKU:              "GADGET-002",
        Name:             "Premium Gadget",
        Price:            249.99,
        StockQuantity:    50,
        MinOrderQuantity: 5,
        IsActive:         true,
    }
    TestProductOutOfStock = Product{
        ID:               uuid.MustParse("p3333333-3333-3333-3333-333333333333"),
        SKU:              "RARE-003",
        Name:             "Rare Item",
        Price:            999.99,
        StockQuantity:    0,  // Out of stock
        MinOrderQuantity: 1,
        IsActive:         true,
    }
    TestProductInactive = Product{
        ID:               uuid.MustParse("p4444444-4444-4444-4444-444444444444"),
        SKU:              "OLD-004",
        Name:             "Discontinued Product",
        IsActive:         false,
    }
)

// seedTestData inserts fixtures into database
func seedTestData(db *pgxpool.Pool) error
```

---

## Test Suites

### 1. Health Tests (`health_test.go`)

| Test | Description | Expected |
|------|-------------|----------|
| `TestHealth_Live` | GET /health/live | 200, `{"status":"alive"}` |
| `TestHealth_Ready` | GET /health/ready | 200, database check passes |
| `TestHealth_Full` | GET /health | 200, includes version and checks |

**No authentication required.**

---

### 2. Product Tests (`products_test.go`)

| Test | Description | Expected |
|------|-------------|----------|
| `TestProducts_List` | GET /api/v1/products | 200, returns active products only |
| `TestProducts_List_Unauthenticated` | GET without auth | 401 Unauthorized |
| `TestProducts_GetByID` | GET /api/v1/products/{id} | 200, returns product with images |
| `TestProducts_GetByID_NotFound` | GET with invalid ID | 404 Not Found |
| `TestProducts_GetByID_Inactive` | GET inactive product | 404 Not Found |

---

### 3. Cart Tests (`cart_test.go`)

| Test | Description | Expected |
|------|-------------|----------|
| `TestCart_CreateNew` | POST /api/v1/cart | 201, creates draft order |
| `TestCart_GetExisting` | POST when draft exists | 200, returns existing |
| `TestCart_GetWithDetails` | GET /api/v1/cart | 200, enriched response with products |
| `TestCart_AddItem` | POST /api/v1/cart/items | 200, item added |
| `TestCart_AddItem_InvalidProduct` | Add non-existent product | 400 Bad Request |
| `TestCart_AddItem_InactiveProduct` | Add inactive product | 400 Bad Request |
| `TestCart_UpdateQuantity` | PUT /api/v1/cart/items/{id} | 200, quantity updated |
| `TestCart_UpdateQuantity_Zero` | Set quantity to 0 | 200, item removed |
| `TestCart_RemoveItem` | DELETE /api/v1/cart/items/{id} | 200, item removed |
| `TestCart_SetItems` | PUT /api/v1/cart/items | 200, replaces all items |
| `TestCart_UpdateNotes` | PUT /api/v1/cart/notes | 200, notes updated |
| `TestCart_Discard` | DELETE /api/v1/cart | 204, cart deleted |
| `TestCart_Submit` | POST /api/v1/cart/submit | 201, order created as pending |
| `TestCart_Submit_Empty` | Submit empty cart | 400 Bad Request |
| `TestCart_Submit_InsufficientStock` | Submit with out-of-stock item | 409 Conflict |
| `TestCart_Submit_BelowMinQuantity` | Quantity below minimum | 400 Bad Request |

**Full Workflow Test:**

```go
func TestCart_FullWorkflow(t *testing.T) {
    // 1. Create cart
    // 2. Add multiple items
    // 3. Update one quantity
    // 4. Remove one item
    // 5. Add notes
    // 6. Get cart and verify totals
    // 7. Submit
    // 8. Verify order created with correct items
    // 9. Verify cart is gone
}
```

---

### 4. Order Tests (`orders_test.go`)

| Test | Description | Expected |
|------|-------------|----------|
| `TestOrders_Create` | POST /api/v1/orders | 201, creates pending order |
| `TestOrders_Create_EmptyItems` | Create with no items | 400 Bad Request |
| `TestOrders_Create_InvalidProduct` | Invalid product ID | 400 Bad Request |
| `TestOrders_List` | GET /api/v1/orders | 200, returns client's orders |
| `TestOrders_List_FilterByStatus` | GET with ?status=pending | 200, filtered results |
| `TestOrders_GetByID` | GET /api/v1/orders/{id} | 200, returns order |
| `TestOrders_GetByID_NotOwned` | Get another client's order | 404 Not Found |
| `TestOrders_Cancel` | POST /api/v1/orders/{id}/cancel | 204, status=cancelled |
| `TestOrders_Cancel_NotPending` | Cancel approved order | 409 Conflict |
| `TestOrders_Cancel_NotOwned` | Cancel another's order | 404 Not Found |

---

### 5. Admin Tests (`admin_test.go`)

| Test | Description | Expected |
|------|-------------|----------|
| `TestAdmin_ListOrders` | GET /api/v1/admin/orders | 200, all orders |
| `TestAdmin_ListOrders_NotAdmin` | List as regular client | 403 Forbidden |
| `TestAdmin_GetOrder` | GET /api/v1/admin/orders/{id} | 200, any order |
| `TestAdmin_ApproveOrder` | POST .../approve | 200, status=approved |
| `TestAdmin_ApproveOrder_NotPending` | Approve cancelled order | 409 Conflict |
| `TestAdmin_ApproveOrder_MissingApprovedBy` | No approved_by field | 400 Bad Request |
| `TestAdmin_RejectOrder` | POST .../reject | 204, status=rejected |
| `TestAdmin_RejectOrder_MissingReason` | No reason field | 400 Bad Request |
| `TestAdmin_ListClients` | GET /api/v1/admin/clients | 200, all clients |
| `TestAdmin_GetClient` | GET /api/v1/admin/clients/{id} | 200, client details |

**Approval Workflow Test:**

```go
func TestAdmin_ApprovalWorkflow(t *testing.T) {
    // 1. Client creates order
    // 2. Admin lists orders, finds it
    // 3. Admin approves with approved_by
    // 4. Verify status=approved, approved_at set
    // 5. Client can see approved status
}
```

---

### 6. Authorization Boundary Tests (`auth_test.go`)

| Test | Description | Expected |
|------|-------------|----------|
| `TestAuth_NoToken_Products` | GET /api/v1/products without auth | 401 |
| `TestAuth_NoToken_Cart` | Any cart endpoint without auth | 401 |
| `TestAuth_NoToken_Orders` | Any order endpoint without auth | 401 |
| `TestAuth_ClientAccessOtherOrder` | Client A gets Client B's order | 404 |
| `TestAuth_ClientAccessAdmin` | Client accesses /admin/* | 403 |
| `TestAuth_ClientCancelOtherOrder` | Client A cancels Client B's order | 404 |
| `TestAuth_IsolatedCarts` | Client A can't see Client B's cart | Separate carts |

---

## Test Isolation Strategy

### Per-Test Cleanup

```go
func TestOrders_Create(t *testing.T) {
    t.Cleanup(func() {
        cleanupOrdersForClient(t, TestClient1.ID)
    })

    // Test code...
}
```

### Parallel Execution

Tests within a file can run in parallel if they use different clients:

```go
func TestCart_AddItem(t *testing.T) {
    t.Parallel()  // Uses TestClient1
    // ...
}

func TestOrders_List(t *testing.T) {
    t.Parallel()  // Uses TestClient2
    // ...
}
```

### Database Transactions (Alternative)

For better isolation, wrap each test in a transaction and rollback:

```go
func withTx(t *testing.T, fn func(tx pgx.Tx)) {
    tx, _ := testDB.Begin(context.Background())
    t.Cleanup(func() { tx.Rollback(context.Background()) })
    fn(tx)
}
```

---

## Implementation Order

### Phase 1: Infrastructure
1. Add testcontainers dependencies
2. Create `setup_test.go` with container lifecycle
3. Create test auth middleware bypass
4. Create fixtures and seed data
5. Add Makefile target

### Phase 2: Core Tests
1. Health tests (simplest, validates setup)
2. Product tests (read-only)
3. Cart tests (full CRUD)
4. Order tests (create, list, cancel)

### Phase 3: Admin & Auth
1. Admin tests (approval workflow)
2. Authorization boundary tests

### Phase 4: Polish
1. Full workflow tests (end-to-end scenarios)
2. Edge cases and error conditions
3. Performance baseline tests (optional)

---

## Expected Test Count

| Suite | Test Count |
|-------|------------|
| Health | 3 |
| Products | 5 |
| Cart | 16 + 1 workflow |
| Orders | 10 |
| Admin | 10 + 1 workflow |
| Auth | 7 |
| **Total** | **~53 tests** |

---

## Success Criteria

- [ ] All tests pass with `make test-integration`
- [ ] Tests complete in < 60 seconds
- [ ] No flaky tests (run 3x to verify)
- [ ] No shared state between tests
- [ ] Coverage of all API endpoints
- [ ] Coverage of all error conditions
- [ ] Clear test names that document behavior
