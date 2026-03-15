# ADR: Production Readiness Improvements

**Status:** Accepted
**Date:** 2026-03-16
**Deciders:** Principal Architect
**Technical Story:** Prepare B2B Orders API for production deployment

## Context

The B2B Orders API has mature core functionality (auth, database, business logic, integrations) but lacks several production-hardening features. This ADR documents architectural decisions for the remaining gaps.

## Decisions

### 1. CORS Configuration Strategy

**Decision:** Environment-specific CORS with explicit origin allowlists.

**Options Considered:**

| Option | Pros | Cons |
|--------|------|------|
| A. Wildcard (`*`) | Simple | Incompatible with credentials; security risk |
| B. Environment variable list | Flexible, secure | Requires config management |
| C. Dynamic origin validation | Most flexible | Complex; potential bypass bugs |

**Choice:** Option B

**Implementation:**
```go
// Config addition
type CORSConfig struct {
    AllowedOrigins []string // From CORS_ALLOWED_ORIGINS env var (comma-separated)
}

// Defaults
// Development: ["http://localhost:3000", "http://localhost:5173"]
// Production: Must be explicitly configured
```

**Rationale:** Explicit allowlists prevent security misconfigurations. Environment variables allow different configs per deployment without code changes.

---

### 2. Rate Limiting Strategy

**Decision:** Tiered rate limiting with per-user identification for authenticated endpoints.

**Options Considered:**

| Option | Pros | Cons |
|--------|------|------|
| A. Global rate limit | Simple | Unfair to legitimate users during attack |
| B. Per-IP only | Identifies abusers | Shared IPs (NAT) cause false positives |
| C. Per-user (JWT subject) | Fair to legitimate users | Requires auth context |
| D. Tiered (B + C) | Best of both worlds | More complex |

**Choice:** Option D - Tiered approach

**Implementation:**
- Unauthenticated endpoints: Per-IP limiting
- Authenticated endpoints: Per-user (JWT `sub` claim) limiting
- Use `go-chi/httprate` for in-memory rate limiting (sufficient for single-instance)

**Limits:**

| Endpoint Pattern | Limit | Window | Key |
|------------------|-------|--------|-----|
| `POST /api/v1/orders` | 20 | 1 min | User ID |
| `GET /api/v1/*` | 100 | 1 min | User ID |
| `POST /api/v1/admin/sync/*` | 5 | 1 min | User ID |
| `/health*` | 1000 | 1 min | IP |

**Rationale:** Per-user limiting prevents one compromised account from affecting others. Tiered limits reflect endpoint sensitivity (writes more restricted than reads).

---

### 3. Pagination Strategy

**Decision:** Cursor-based pagination for list endpoints.

**Options Considered:**

| Option | Pros | Cons |
|--------|------|------|
| A. Offset/limit | Simple; familiar | Breaks with inserts; O(n) skip cost |
| B. Keyset/cursor | Stable; performant | Slightly more complex |
| C. Page tokens (opaque) | Hides implementation | Harder to debug |

**Choice:** Option B - Keyset pagination with transparent cursors

**Implementation:**
```
GET /api/v1/orders?limit=25&after=2024-01-15T10:30:00Z_uuid

Response:
{
  "data": [...],
  "pagination": {
    "limit": 25,
    "has_more": true,
    "next_cursor": "2024-01-15T09:00:00Z_abc123"
  }
}
```

**Cursor Format:** `{created_at}_{id}` - composite key for deterministic ordering

**Rationale:** Order data changes frequently (new orders, status updates). Cursor-based pagination provides stable results regardless of concurrent writes. Using `created_at` + `id` ensures uniqueness and natural chronological ordering.

---

### 4. Health Check Design

**Decision:** Kubernetes-compatible probe endpoints with dependency health aggregation.

**Options Considered:**

| Option | Pros | Cons |
|--------|------|------|
| A. Single `/health` | Simple | Can't distinguish liveness vs readiness |
| B. `/health/live` + `/health/ready` | K8s native | Two endpoints to maintain |
| C. Single endpoint with query params | Flexible | Non-standard |

**Choice:** Option B with enhanced `/health` for full status

**Implementation:**

| Endpoint | Purpose | Response Time | Checks |
|----------|---------|---------------|--------|
| `GET /health/live` | Liveness | <10ms | Process responding |
| `GET /health/ready` | Readiness | <100ms | DB ping |
| `GET /health` | Full status | <500ms | All deps + metadata |

**Response Format:**
```json
{
  "status": "healthy",
  "version": "1.2.3",
  "environment": "production",
  "checks": {
    "database": {"status": "up", "latency_ms": 2},
    "holded": {"status": "up", "last_sync": "2024-01-15T10:00:00Z"}
  }
}
```

**Rationale:** Kubernetes uses liveness probes to restart unhealthy pods and readiness probes to route traffic. Separating them allows the app to signal "I'm alive but not ready for traffic" during startup or dependency outages.

---

### 5. Request Size Limiting

**Decision:** 1MB global limit with endpoint-specific overrides.

**Implementation:**
```go
// Global middleware
r.Use(func(next http.Handler) http.Handler {
    return http.MaxBytesHandler(next, 1<<20) // 1MB
})
```

**Rationale:** Order creation payloads are small (JSON with product IDs and quantities). 1MB is generous for normal use while preventing memory exhaustion attacks. No endpoint currently needs larger payloads.

---

### 6. Database Indexing Strategy

**Decision:** Add indexes for common query patterns identified in repository code.

**Indexes:**
```sql
-- Order queries (most common)
CREATE INDEX idx_orders_client_status ON orders(client_id, status);
CREATE INDEX idx_orders_created_at ON orders(created_at DESC);
CREATE INDEX idx_orders_status ON orders(status) WHERE status NOT IN ('delivered', 'cancelled', 'rejected');

-- Product queries
CREATE INDEX idx_products_active ON products(is_active) WHERE is_active = true;
CREATE INDEX idx_products_category ON products(category) WHERE is_active = true;

-- Client lookup (auth flow)
CREATE INDEX idx_clients_email ON clients(email);

-- Image retrieval
CREATE INDEX idx_product_images_product ON product_images(product_id, display_order);
```

**Rationale:** Partial indexes (WHERE clauses) reduce index size and improve write performance. Active products and non-terminal orders are the primary query targets.

---

### 7. Configuration Validation

**Decision:** Fail-fast validation with environment-aware requirements.

**Implementation:**
```go
func (c *Config) validate() error {
    var errs []string

    // Always required
    required := map[string]string{
        "DATABASE_URL":   c.DB.URL,
        "AUTH0_DOMAIN":   c.Auth0.Domain,
        "AUTH0_AUDIENCE": c.Auth0.Audience,
    }

    // Production-only requirements
    if c.IsProduction() {
        required["AWS_S3_BUCKET"] = c.S3.Bucket
        required["HOLDED_API_KEY"] = c.Holded.APIKey
        required["CORS_ALLOWED_ORIGINS"] = strings.Join(c.CORS.AllowedOrigins, ",")
    }

    for name, value := range required {
        if value == "" {
            errs = append(errs, name+" is required")
        }
    }

    if len(errs) > 0 {
        return fmt.Errorf("config validation failed: %s", strings.Join(errs, "; "))
    }
    return nil
}
```

**Rationale:** Development mode allows mock clients and permissive CORS. Production mode requires all integrations to be explicitly configured, preventing accidental deployment with missing credentials.

---

## Implementation Priority

| Priority | Item | Effort | Dependencies |
|----------|------|--------|--------------|
| P0 | CORS configuration | 2h | None |
| P0 | Config validation | 2h | None |
| P1 | Request size limits | 1h | None |
| P1 | Rate limiting | 4h | None |
| P1 | Health check enhancements | 3h | None |
| P2 | Database indexes | 2h | None |
| P2 | Pagination | 10h | Repository interface changes |

## Consequences

### Positive
- Secure by default (no wildcard CORS in production)
- Resilient to abuse (rate limiting, size limits)
- Kubernetes-ready (proper health probes)
- Scalable queries (pagination, indexes)
- Fast failure on misconfiguration

### Negative
- Additional configuration required for deployment
- Rate limiting may need tuning based on actual usage patterns
- Pagination requires client changes for list endpoints

### Risks
- Rate limits too aggressive → legitimate users blocked (mitigate: start permissive, tighten based on data)
- Cursor pagination complexity → bugs in edge cases (mitigate: comprehensive tests)

## References

- [go-chi/httprate](https://github.com/go-chi/httprate) - Rate limiting middleware
- [Kubernetes Probes](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/)
- [Pagination Best Practices](https://www.citusdata.com/blog/2016/03/30/five-ways-to-paginate/)
