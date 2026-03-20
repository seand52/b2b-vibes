# ADR: Order Item Count Optimization

**Status:** Proposed
**Date:** 2026-03-20
**Author:** Architect

## Context

The `/admin/clients` page displays order history for each client, showing item counts per order. The current implementation:

1. Fetches orders matching filter criteria
2. Fetches **all item rows** for those orders via `WHERE order_id IN (...)`
3. Maps items back to orders and counts them client-side

### Current Code Path
```
ListOrders → List() → getItemsForOrders() → returns full OrderItem structs
```

### Problems Identified

| Issue | Impact |
|-------|--------|
| Full item data transferred when only COUNT needed | Wasted bandwidth, memory |
| No index on `order_items.order_id` | Full table scans |
| IN clause with many IDs | Query planner struggles at scale |
| O(orders) + O(items) data transfer | Linear scaling with data volume |

### Scale Projections

| Orders | Items/Order | Total Items | Current Approach | With Optimization |
|--------|-------------|-------------|------------------|-------------------|
| 100 | 5 | 500 | ~50KB | ~4KB |
| 1,000 | 5 | 5,000 | ~500KB | ~40KB |
| 10,000 | 5 | 50,000 | ~5MB | ~400KB |

## Decision Drivers

1. **Read performance** - Admin pages should load quickly
2. **Accuracy** - Item counts must be correct
3. **Implementation effort** - Prefer incremental improvements
4. **Maintainability** - Avoid complex synchronization logic

## Options Considered

### Option A: Denormalized `item_count` Column

Add columns to `orders` table:
```sql
ALTER TABLE orders ADD COLUMN item_count INTEGER DEFAULT 0;
ALTER TABLE orders ADD COLUMN total_quantity INTEGER DEFAULT 0;
```

Maintain via triggers or application logic.

| Pros | Cons |
|------|------|
| O(1) read performance | Write overhead |
| No joins needed | Drift risk if updates fail |
| Simple queries | Requires migration + backfill |
| Industry standard pattern | Two sources of truth |

### Option B: COUNT Subquery

```sql
SELECT o.*,
       (SELECT COUNT(*) FROM order_items WHERE order_id = o.id) as item_count
FROM orders o
WHERE ...
```

| Pros | Cons |
|------|------|
| Always accurate | Correlated subquery per row |
| No schema change | Slower than denormalized |
| Simple implementation | Needs index to perform well |

### Option C: JOIN with GROUP BY

```sql
SELECT o.*, COUNT(oi.id) as item_count
FROM orders o
LEFT JOIN order_items oi ON o.id = oi.order_id
WHERE ...
GROUP BY o.id
```

| Pros | Cons |
|------|------|
| Single query | Changes result structure |
| Accurate | Repository code changes |
| Good PostgreSQL optimization | Slightly complex |

### Option D: Materialized View

```sql
CREATE MATERIALIZED VIEW order_summaries AS
SELECT order_id, COUNT(*) as item_count, SUM(quantity) as total_quantity
FROM order_items
GROUP BY order_id;
```

| Pros | Cons |
|------|------|
| Very fast reads | Staleness |
| Can include more aggregates | Refresh complexity |
| | PostgreSQL-specific |

### Option E: Add Index Only

```sql
CREATE INDEX idx_order_items_order_id ON order_items(order_id);
```

Keep current implementation, just optimize.

| Pros | Cons |
|------|------|
| Minimal change | Still transfers full item data |
| Quick win | Doesn't solve core inefficiency |

## Trade-off Matrix

| Criteria | A: Denorm | B: Subquery | C: JOIN | D: MatView | E: Index |
|----------|-----------|-------------|---------|------------|----------|
| Read Performance | 5 | 3 | 4 | 5 | 2 |
| Write Complexity | 3 | 5 | 5 | 3 | 5 |
| Data Accuracy | 3 | 5 | 5 | 4 | 5 |
| Implementation | 3 | 4 | 4 | 2 | 5 |
| Maintenance | 3 | 5 | 5 | 2 | 5 |
| Scalability | 5 | 3 | 4 | 5 | 2 |
| **Total** | **22** | **25** | **27** | **21** | **24** |

## Decision

**Phased approach:**

### Phase 1 (Immediate)

1. **Add missing index** (critical, do regardless):
   ```sql
   CREATE INDEX idx_order_items_order_id ON order_items(order_id);
   ```

2. **Use COUNT subquery for list operations:**
   - Modify `List()` to use subquery for counts
   - Only fetch full items for detail views (`GetByID`)
   - Add `ItemCount` and `TotalQuantity` fields to Order struct

### Phase 2 (If scale requires, >10K orders)

1. Add denormalized columns with database trigger
2. Backfill existing data
3. Update application to use denormalized columns

## Implementation Notes

### New List Query (Phase 1)

```sql
SELECT
    o.id, o.client_id, o.status, o.notes, o.admin_notes,
    o.holded_invoice_id, o.approved_at, o.approved_by,
    o.rejected_at, o.rejection_reason, o.created_at, o.updated_at,
    (SELECT COUNT(*) FROM order_items WHERE order_id = o.id) as item_count,
    (SELECT COALESCE(SUM(quantity), 0) FROM order_items WHERE order_id = o.id) as total_quantity
FROM orders o
WHERE ...
ORDER BY o.created_at DESC
```

### Domain Model Change

```go
type Order struct {
    // ... existing fields
    ItemCount     int  `json:"item_count"`      // Populated by List queries
    TotalQuantity int  `json:"total_quantity"`  // Populated by List queries
    Items         []OrderItem `json:"items,omitempty"` // Populated by GetByID
}
```

### API Response Change

List endpoints return `item_count` and `total_quantity`.
Detail endpoints return full `items` array.

## Consequences

### Positive
- 10-100x reduction in data transfer for list views
- Accurate counts without schema changes
- Clear upgrade path to Phase 2 if needed
- Missing index fixes all item queries

### Negative
- Two query patterns (list vs detail)
- Slightly more complex repository code
- Subqueries add small overhead vs denormalized

### Neutral
- Frontend already expects `items.length` - needs update to use `item_count`

## References

- [PostgreSQL Subquery Performance](https://www.postgresql.org/docs/current/queries-table-expressions.html#QUERIES-SUBQUERIES)
- [Denormalization Patterns](https://www.mongodb.com/blog/post/6-rules-of-thumb-for-mongodb-schema-design)
