# ADR: Cart / Draft Order System

**Status:** Proposed
**Date:** 2026-03-16
**Deciders:** Principal Architect
**Technical Story:** Enable customers to build and modify orders before submitting for approval

## Context

Currently, the B2B Orders API only supports creating orders in a single atomic operation:
- `POST /orders` creates an order directly in `pending` status
- Once created, orders cannot be modified (only cancelled)
- No "shopping cart" experience for customers

**User Story:**
> As a business client, I want to browse products, add them to a cart, adjust quantities, remove items, and review my order before submitting it for admin approval.

## Decision

### Option Analysis

| Option | Description | Pros | Cons |
|--------|-------------|------|------|
| **A. New "draft" status** | Add `draft` as first order status | Simple; reuses existing Order model | Pollutes order history with abandoned drafts |
| **B. Separate Cart entity** | New `carts` table, convert to order on submit | Clean separation; carts can be abandoned | Two similar data models; conversion logic |
| **C. Draft order with cleanup** | Option A + scheduled cleanup of stale drafts | Balance of simplicity and cleanliness | Adds background job dependency |

**Decision: Option A (New "draft" status) with soft delete**

Rationale:
- Simplest implementation (extends existing model)
- Consistent API patterns
- Abandoned drafts can be soft-deleted or have an `abandoned_at` timestamp
- Frontend already understands the Order model
- One active draft per client enforced at service level

---

## Design

### New Order Status Flow

```
                    ┌─────────────┐
                    │   draft     │  ← Customer building order
                    └──────┬──────┘
                           │ submit
                           ▼
┌───────────────────────────────────────────────────────────────┐
│                    EXISTING FLOW                               │
│  ┌─────────┐      ┌──────────┐      ┌───────────┐             │
│  │ pending │─────▶│ approved │─────▶│  shipped  │──▶delivered │
│  └────┬────┘      └──────────┘      └───────────┘             │
│       │                                                        │
│       ├──▶ rejected                                            │
│       └──▶ cancelled                                           │
└───────────────────────────────────────────────────────────────┘
```

**Status Transitions:**
- `draft` → `pending` (customer submits)
- `draft` → `cancelled` (customer abandons)
- All existing transitions remain unchanged

### API Design

#### Cart (Draft Order) Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/api/v1/cart` | Create or get current draft |
| `GET` | `/api/v1/cart` | Get current draft with product details |
| `PUT` | `/api/v1/cart/items` | Set all items (full replace) |
| `POST` | `/api/v1/cart/items` | Add item to cart |
| `PUT` | `/api/v1/cart/items/{product_id}` | Update item quantity |
| `DELETE` | `/api/v1/cart/items/{product_id}` | Remove item from cart |
| `PUT` | `/api/v1/cart/notes` | Update order notes |
| `POST` | `/api/v1/cart/submit` | Submit draft → pending |
| `DELETE` | `/api/v1/cart` | Discard draft |

#### Request/Response Schemas

**GET /api/v1/cart Response:**
```json
{
  "id": "uuid",
  "status": "draft",
  "notes": "string",
  "created_at": "2024-01-15T10:00:00Z",
  "updated_at": "2024-01-15T10:30:00Z",
  "items": [
    {
      "product_id": "uuid",
      "product_name": "Widget Pro",
      "product_sku": "WGT-001",
      "quantity": 5,
      "unit_price": 29.99,
      "line_total": 149.95,
      "stock_available": 100,
      "min_order_quantity": 1,
      "in_stock": true
    }
  ],
  "summary": {
    "subtotal": 149.95,
    "tax_rate": 21.0,
    "tax_amount": 31.49,
    "total": 181.44,
    "item_count": 1,
    "total_units": 5
  }
}
```

**POST /api/v1/cart/items Request:**
```json
{
  "product_id": "uuid",
  "quantity": 5
}
```

**PUT /api/v1/cart/items/{product_id} Request:**
```json
{
  "quantity": 10
}
```

**POST /api/v1/cart/submit Response:**
```json
{
  "id": "uuid",
  "status": "pending",
  "submitted_at": "2024-01-15T11:00:00Z"
}
```

### Business Rules

1. **One draft per client**: Each client can have at most one active draft order
2. **Stock validation**: Validate stock on submit, not on add (to allow wishlist-style behavior)
3. **Price snapshot**: Capture prices at submit time (not when adding to cart)
4. **Draft expiration**: Drafts older than 30 days auto-marked as `abandoned` (optional background job)
5. **Quantity limits**: Enforce `min_order_quantity` and available stock at submit
6. **Empty cart**: Cannot submit empty draft

### Database Changes

**orders table changes:**
```sql
-- Add new status value (no schema change needed, status is VARCHAR)
-- New status: 'draft'

-- Add submitted_at timestamp
ALTER TABLE orders ADD COLUMN submitted_at TIMESTAMPTZ;

-- Add index for finding client's draft
CREATE INDEX idx_orders_client_draft ON orders(client_id, status)
    WHERE status = 'draft';
```

**order_items changes:**
```sql
-- Add price snapshot (captured at submit time)
ALTER TABLE order_items ADD COLUMN unit_price DECIMAL(12,4);
ALTER TABLE order_items ADD COLUMN line_total DECIMAL(12,4);
```

### Domain Model Changes

```go
// OrderStatus additions
const (
    OrderStatusDraft OrderStatus = "draft"
    // ... existing statuses
)

// Update transitions
func (s OrderStatus) CanTransitionTo(target OrderStatus) bool {
    transitions := map[OrderStatus][]OrderStatus{
        OrderStatusDraft:     {OrderStatusPending, OrderStatusCancelled},
        OrderStatusPending:   {OrderStatusApproved, OrderStatusRejected, OrderStatusCancelled},
        // ... existing
    }
}

// OrderItem additions
type OrderItem struct {
    // ... existing fields
    UnitPrice  *float64  // Captured at submit
    LineTotal  *float64  // Captured at submit
}
```

### Service Layer

**New CartService** (or extend OrderService):

```go
type CartService interface {
    // Get or create draft for client
    GetOrCreateDraft(ctx context.Context, clientID uuid.UUID) (*domain.Order, error)

    // Get draft with enriched product data
    GetDraftWithDetails(ctx context.Context, clientID uuid.UUID) (*CartResponse, error)

    // Item operations
    AddItem(ctx context.Context, clientID uuid.UUID, productID uuid.UUID, quantity int) error
    UpdateItemQuantity(ctx context.Context, clientID uuid.UUID, productID uuid.UUID, quantity int) error
    RemoveItem(ctx context.Context, clientID uuid.UUID, productID uuid.UUID) error
    SetItems(ctx context.Context, clientID uuid.UUID, items []ItemRequest) error

    // Notes
    UpdateNotes(ctx context.Context, clientID uuid.UUID, notes string) error

    // Submit (draft → pending)
    Submit(ctx context.Context, clientID uuid.UUID) (*domain.Order, error)

    // Discard
    DiscardDraft(ctx context.Context, clientID uuid.UUID) error
}
```

---

## Consequences

### Positive
- Intuitive shopping cart UX for customers
- Reuses existing Order model (no new entity)
- Clear separation: drafts are editable, pending+ are immutable
- Price and stock captured at submission = predictable invoicing
- Simple frontend integration (same order model)

### Negative
- Draft orders appear in order history (mitigated by filtering)
- Abandoned drafts accumulate (mitigated by cleanup job or `abandoned_at`)
- Additional API endpoints to maintain

### Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Stale drafts accumulate | Medium | Low | Background cleanup job; `abandoned_at` timestamp |
| Race condition on submit | Low | Medium | Optimistic locking on order version |
| Stock oversold | Medium | High | Final stock check at submit; reserve stock option |

---

## Implementation Plan

### Phase 1: Core Cart (MVP)
1. Add `draft` status to domain model
2. Add `submitted_at` column to orders
3. Add price columns to order_items
4. Create CartService with basic operations
5. Add cart endpoints to handlers
6. Add cart routes to server

### Phase 2: Enrichment
1. Cart response with product details and pricing
2. Stock availability display
3. Price calculations (subtotal, tax, total)

### Phase 3: Polish
1. Cart cleanup background job (optional)
2. Stock reservation on submit (optional)
3. Cart merge on login (optional, for anonymous carts)

---

## API Summary

| Endpoint | Method | Description | Auth |
|----------|--------|-------------|------|
| `/api/v1/cart` | GET | Get current cart with details | Client |
| `/api/v1/cart` | POST | Create new cart (or return existing) | Client |
| `/api/v1/cart` | DELETE | Discard current cart | Client |
| `/api/v1/cart/items` | POST | Add item to cart | Client |
| `/api/v1/cart/items` | PUT | Replace all items | Client |
| `/api/v1/cart/items/{product_id}` | PUT | Update item quantity | Client |
| `/api/v1/cart/items/{product_id}` | DELETE | Remove item | Client |
| `/api/v1/cart/notes` | PUT | Update notes | Client |
| `/api/v1/cart/submit` | POST | Submit cart → pending order | Client |

---

## Appendix: Existing vs New Endpoints

### Existing Order Endpoints (unchanged)
```
POST   /api/v1/orders           # Still works (creates pending directly)
GET    /api/v1/orders           # List orders (filter out drafts by default?)
GET    /api/v1/orders/{id}      # Get order
POST   /api/v1/orders/{id}/cancel  # Cancel order
```

### New Cart Endpoints
```
GET    /api/v1/cart             # Get/create draft
POST   /api/v1/cart             # Explicit create
DELETE /api/v1/cart             # Discard
POST   /api/v1/cart/items       # Add item
PUT    /api/v1/cart/items       # Replace items
PUT    /api/v1/cart/items/{id}  # Update quantity
DELETE /api/v1/cart/items/{id}  # Remove item
PUT    /api/v1/cart/notes       # Update notes
POST   /api/v1/cart/submit      # Submit to pending
```

**Recommendation:** Keep existing `POST /orders` for programmatic/API clients who want atomic order creation. Cart endpoints are for interactive UI flows.
