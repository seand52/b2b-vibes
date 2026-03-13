# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

B2B Orders API - A Go-based REST API for managing orders from business clients. Built with chi router and following standard Go project layout.

### Business Context
This is a SaaS platform where:
- Business clients register accounts and place orders for products
- Clients interact directly with the API (their end customers do NOT use this app)
- Each order can contain multiple items with quantities and pricing
- Order status progresses through: pending → confirmed → shipped → delivered (or cancelled)
- Standard e-commerce flow: browse products → add to order → checkout → track status

### Current Implementation State
**Status**: Early development phase with in-memory data storage

**What's Implemented:**
- RESTful API structure with chi router
- CRUD endpoints for users, products, and orders
- Request/response models with validation structs
- Middleware stack (logging, recovery, timeouts, request ID tracking)
- Health check endpoint
- Order status management (pending, confirmed, shipped, delivered, cancelled)

**What's NOT Implemented (TODOs):**
- Database persistence (handlers return placeholder responses)
- Authentication and authorization
- Input validation and sanitization
- Business logic (order total calculation, inventory checks, status transition rules)
- Standardized error responses
- Rate limiting and request throttling
- CORS configuration for frontend clients

### Current Structure

```
b2b-orders-api/
├── cmd/
│   └── api/              # Main application entry point
│       └── main.go       # Server bootstrap
├── internal/
│   ├── handlers/         # HTTP request handlers
│   │   ├── users.go      # User CRUD endpoints
│   │   ├── products.go   # Product CRUD endpoints
│   │   └── orders.go     # Order CRUD endpoints
│   ├── models/           # Data models and request/response types
│   │   ├── user.go
│   │   ├── product.go
│   │   └── order.go
│   └── server/           # Server setup and routing
│       └── server.go     # Chi router configuration
└── go.mod
```

### Data Models

**User:**
- `ID` (string) - Unique identifier
- `Email` (string) - User email address
- `Name` (string) - Business/user name
- `CreatedAt`, `UpdatedAt` (time.Time) - Timestamps

**Product:**
- `ID` (string) - Unique identifier
- `Name` (string) - Product name
- `Description` (string) - Product description
- `Label` (string) - Product label for categorizing
- `Price` (float64) - Product price
- `StockQuantity` (int) - Available inventory
- `CreatedAt`, `UpdatedAt` (time.Time) - Timestamps

**Order:**
- `ID` (string) - Unique identifier
- `UserID` (string) - Reference to user
- `Items` ([]OrderItem) - Array of order items
- `TotalPrice` (float64) - Calculated total
- `Status` (OrderStatus) - Current order status
- `CreatedAt`, `UpdatedAt` (time.Time) - Timestamps

**OrderItem:**
- `ProductID` (string) - Reference to product
- `Quantity` (int) - Quantity ordered
- `Price` (float64) - Price at time of order

**OrderStatus:** `pending`, `confirmed`, `shipped`, `delivered`, `cancelled`

### API Endpoints

All endpoints are prefixed with `/api/v1`:

**Users:**
- `POST /api/v1/users` - Create a new user
- `GET /api/v1/users` - List all users
- `GET /api/v1/users/{id}` - Get user by ID
- `PUT /api/v1/users/{id}` - Update user
- `DELETE /api/v1/users/{id}` - Delete user

**Products:**
- `POST /api/v1/products` - Create a new product
- `GET /api/v1/products` - List all products
- `GET /api/v1/products/{id}` - Get product by ID
- `PUT /api/v1/products/{id}` - Update product
- `DELETE /api/v1/products/{id}` - Delete product

**Orders:**
- `POST /api/v1/orders` - Create a new order
- `GET /api/v1/orders` - List all orders
- `GET /api/v1/orders/{id}` - Get order by ID
- `PUT /api/v1/orders/{id}` - Update order (mainly status)
- `DELETE /api/v1/orders/{id}` - Delete order

**Health:**
- `GET /health` - Health check endpoint

### Response Formats

**Success Response:**
```json
{
  "id": "123",
  "name": "Product Name",
  "price": 99.99,
  ...
}
```

**Error Response (to be standardized):**
```json
{
  "error": "error message",
  "code": "ERROR_CODE",
  "details": {}
}
```

**List Response:**
```json
[
  { "id": "1", ... },
  { "id": "2", ... }
]
```

### Business Rules

**Order Creation:**
- Must include valid UserID
- Must contain at least one item
- ProductID in each item must reference existing product
- TotalPrice should be calculated server-side (sum of item.Price × item.Quantity)
- Initial status is always `pending`

**Order Status Transitions:**
- `pending` → `confirmed`, `cancelled`
- `confirmed` → `shipped`, `cancelled`
- `shipped` → `delivered`
- `delivered` → (terminal state)
- `cancelled` → (terminal state)

**Product Management:**
- Price must be positive
- StockQuantity must be non-negative
- When order is confirmed, reduce product stock accordingly (when implemented)

**User Management:**
- Email should be unique
- Email format validation required

## Development Commands

```bash
# Install dependencies
go mod tidy

# Run the API locally (default port 8080)
go run ./cmd/api

# Run on custom port
PORT=3000 go run ./cmd/api

# Build the application
go build -o bin/b2b-orders-api ./cmd/api

# Run tests
go test ./...

# Run tests with race detector
go test -race ./...

# Run a single test
go test ./path/to/package -run TestName

# Run benchmarks
go test -bench=. ./...

# Lint (requires golangci-lint)
golangci-lint run

# Format code
gofmt -w .
```

## Architecture Guidelines

This project follows the golang-pro skill patterns:

### Project Structure
- Follows standard Go project layout with `cmd/` and `internal/` directories
- `cmd/api/` contains the main application entry point
- `internal/handlers/` contains HTTP request handlers (one file per resource)
- `internal/models/` contains data models, request/response types
- `internal/server/` contains server setup, routing, and middleware configuration
- All handlers use chi's `chi.URLParam(r, "id")` for path parameters
- Handlers return placeholder responses with TODO comments for database integration

### Middleware Stack
Current middleware (applied in order):
1. `RequestID` - Adds unique request ID to context
2. `RealIP` - Detects real client IP from headers
3. `Logger` - Logs HTTP requests
4. `Recoverer` - Recovers from panics
5. `Timeout` - 60-second request timeout

**Future middleware considerations:**
- Authentication/JWT validation
- CORS headers
- Rate limiting per user/IP
- Request size limits
- API versioning
- Content-Type validation

### Concurrency Patterns
- All blocking operations must accept `context.Context` for cancellation
- Use goroutines with clear lifecycle management (don't leak goroutines)
- Propagate context through the call chain for timeouts and cancellation
- Use channels for communication between goroutines when appropriate

### Error Handling
- Handle all errors explicitly
- Wrap errors with context using `fmt.Errorf("%w", err)` for error chains
- Don't use panic for normal error handling
- Return errors rather than logging and continuing

### Testing Requirements
- Write table-driven tests with subtests (`t.Run`)
- Run tests with race detector (`-race` flag)
- Aim for 80%+ test coverage on business logic
- Use benchmarks for performance-critical code

### API Design
- Design small, focused interfaces
- Accept interfaces, return concrete types where appropriate
- Use functional options pattern for configuration
- Document all exported functions, types, and packages

### Security & Validation
**Input Validation (when implementing):**
- Validate all request bodies against expected schemas
- Sanitize user inputs to prevent injection attacks
- Validate ID formats (UUIDs, numeric IDs, etc.)
- Check for required fields and proper data types
- Validate email formats, price ranges, quantities

**Authentication (to be implemented):**
- Use JWT tokens for stateless authentication
- Protect all endpoints except `/health` and possibly product listing
- Include user ID in JWT claims for authorization
- Implement role-based access (e.g., admin vs. regular user)

**Authorization:**
- Users should only access their own orders
- Users can view all products
- Consider admin role for product/user management

**Other Security:**
- Use HTTPS in production
- Set appropriate CORS headers
- Implement rate limiting to prevent abuse
- Sanitize error messages (don't leak sensitive info)
- Log security events (failed auth, suspicious activity)

### Code Quality
- Run `gofmt` and `golangci-lint` before committing
- Using Go 1.25.4 - leverage modern features including generics
- Avoid reflection unless absolutely necessary for performance
- Keep handlers thin - move business logic to service layer when complexity grows

## Development Roadmap

### Phase 1: Core Functionality (Current)
- [x] Basic API structure and routing
- [x] Data models and request/response types
- [x] Placeholder CRUD endpoints
- [ ] Database integration (PostgreSQL recommended)
- [ ] Input validation and error handling
- [ ] Business logic implementation (order calculations, status transitions)

### Phase 2: Production Readiness
- [ ] Authentication (JWT)
- [ ] Authorization (user-specific data access)
- [ ] Comprehensive error responses
- [ ] Database migrations
- [ ] Unit and integration tests
- [ ] API documentation (OpenAPI/Swagger)

### Phase 3: Enhancement
- [ ] Rate limiting
- [ ] Pagination for list endpoints
- [ ] Filtering and sorting (e.g., orders by status, date range)
- [ ] Email notifications (order confirmations)
- [ ] Admin dashboard endpoints
- [ ] Metrics and monitoring

### When to Refactor

**Introduce Service Layer when:**
- Business logic in handlers exceeds 20-30 lines
- Logic needs to be shared across multiple handlers
- Complex validation or calculation logic emerges
- Testing handlers becomes difficult due to tight coupling

**Database Layer Pattern:**
- Use repository pattern for data access
- Keep database queries separate from business logic
- Use interfaces to allow mock implementations for testing
- Consider using `sqlx` or `pgx` for PostgreSQL

**Example structure with services:**
```
internal/
├── handlers/       # HTTP layer (thin, just request/response)
├── services/       # Business logic layer
├── repository/     # Data access layer
└── models/         # Shared data structures
```
