# Production Readiness Plan: B2B Orders API

**Created:** 2026-03-16
**Status:** Draft
**Author:** Principal Architect Agent

## Executive Summary

The B2B Orders API is significantly more mature than initial documentation suggested. Core business logic, authentication, authorization, database persistence, and external integrations are fully implemented. However, several gaps remain before the system is production-ready. This plan identifies and prioritizes these gaps.

---

## Current State Assessment

### Implemented (Production-Ready)

| Component | Status | Notes |
|-----------|--------|-------|
| Core API Structure | **Complete** | Chi router, clean layered architecture |
| Database Layer | **Complete** | PostgreSQL + pgx/v5, repository pattern, migrations |
| Authentication | **Complete** | Auth0 JWT with JWKS caching, custom claims |
| Authorization | **Complete** | Role-based (admin check), ownership verification |
| Input Validation | **Complete** | Handler, service, and domain layers |
| Error Handling | **Complete** | Standardized error codes and responses |
| Business Logic | **Complete** | Order creation, status transitions, stock checks |
| External Integrations | **Complete** | Holded ERP (products, contacts, invoices), AWS S3 |
| Testing | **Good** | 12 test files, table-driven tests, mocks |
| Logging | **Complete** | Structured logging with slog (JSON in prod) |
| Graceful Shutdown | **Complete** | Signal handling, 30s drain |
| Configuration | **Complete** | Environment variables with validation |

### Gaps Requiring Attention

| Gap | Severity | Effort | Impact |
|-----|----------|--------|--------|
| **P0: CORS Configuration** | Critical | Low | Security vulnerability |
| **P0: Config Validation** | Critical | Low | Silent failures in production |
| **P1: Rate Limiting** | High | Medium | DoS protection |
| **P1: Request Size Limits** | High | Low | Memory exhaustion protection |
| **P1: Health Check Enhancements** | High | Low | Kubernetes readiness/liveness |
| **P2: Pagination** | Medium | Medium | Performance at scale |
| **P2: API Documentation** | Medium | Medium | Developer experience |
| **P2: Database Indexes** | Medium | Low | Query performance |
| **P3: Metrics & Observability** | Low | High | Operational visibility |
| **P3: Automated Background Sync** | Low | Medium | Currently manual-only |

---

## P0: Critical (Must Fix Before Production)

### 1. CORS Configuration (Security)

**Current State:** `AllowedOrigins: []string{"*"}` with `AllowCredentials: true`

**Risk:** This combination is invalid per CORS spec and a security vulnerability. Browsers will reject credentials with wildcard origins, but misconfigured proxies may not.

**Location:** `internal/server/server.go:85-92`

**Recommendation:**
- Environment-specific CORS origins
- Development: `localhost:*` patterns
- Production: Explicit list of frontend domains
- Remove or conditionally set `AllowCredentials` based on actual need

**Effort:** 2-4 hours

---

### 2. Configuration Validation (Reliability)

**Current State:** Only `DATABASE_URL` is validated. Missing `AUTH0_DOMAIN`, `AUTH0_AUDIENCE`, `AWS_S3_BUCKET` will cause runtime panics.

**Location:** `internal/config/config.go:98-103`

**Recommendation:**
```go
func (c *Config) validate() error {
    var errs []string
    if c.DB.URL == "" {
        errs = append(errs, "DATABASE_URL is required")
    }
    if c.Auth0.Domain == "" {
        errs = append(errs, "AUTH0_DOMAIN is required")
    }
    if c.Auth0.Audience == "" {
        errs = append(errs, "AUTH0_AUDIENCE is required")
    }
    // Production-only validations
    if c.IsProduction() {
        if c.S3.Bucket == "" {
            errs = append(errs, "AWS_S3_BUCKET is required in production")
        }
        if c.Holded.APIKey == "" {
            errs = append(errs, "HOLDED_API_KEY is required in production")
        }
    }
    if len(errs) > 0 {
        return fmt.Errorf("config validation failed: %s", strings.Join(errs, "; "))
    }
    return nil
}
```

**Effort:** 1-2 hours

---

## P1: High Priority (Required for Production)

### 3. Rate Limiting

**Current State:** No rate limiting. API is vulnerable to DoS and abuse.

**Recommendation:** Use `go-chi/httprate` middleware with tiered limits:

| Endpoint Pattern | Limit | Window |
|------------------|-------|--------|
| `/health` | 1000/min | 1 min |
| `/api/v1/products*` | 100/min | 1 min |
| `/api/v1/orders` POST | 20/min | 1 min |
| `/api/v1/admin/*` | 50/min | 1 min |

**Implementation:**
- Per-IP limiting for unauthenticated endpoints
- Per-user (JWT subject) limiting for authenticated endpoints
- Return `Retry-After` header on 429

**Effort:** 4-6 hours

---

### 4. Request Size Limits

**Current State:** No body size limits. Large payloads can exhaust memory.

**Recommendation:**
```go
// Add to middleware stack
r.Use(chimiddleware.SetHeader("Content-Type", "application/json"))
r.Use(func(next http.Handler) http.Handler {
    return http.MaxBytesHandler(next, 1<<20) // 1MB limit
})
```

**Effort:** 1 hour

---

### 5. Health Check Enhancements

**Current State:** Single `/health` endpoint checks DB connectivity.

**Recommendation:** Kubernetes-compatible probes:

| Endpoint | Purpose | Checks |
|----------|---------|--------|
| `/health/live` | Liveness probe | Process is running |
| `/health/ready` | Readiness probe | DB + Auth0 JWKS reachable |
| `/health` | Full status | All dependencies + version info |

**Response Format:**
```json
{
  "status": "healthy",
  "version": "1.0.0",
  "checks": {
    "database": {"status": "up", "latency_ms": 2},
    "auth0": {"status": "up"}
  }
}
```

**Effort:** 3-4 hours

---

## P2: Medium Priority (Should Have)

### 6. Pagination for List Endpoints

**Current State:** List endpoints return all records.

**Affected Endpoints:**
- `GET /api/v1/products`
- `GET /api/v1/orders`
- `GET /api/v1/admin/orders`
- `GET /api/v1/admin/clients`

**Recommendation:** Cursor-based pagination (better for real-time data):

```
GET /api/v1/orders?limit=25&after=<cursor>

Response:
{
  "data": [...],
  "pagination": {
    "has_more": true,
    "next_cursor": "eyJpZCI6IjEyMyJ9"
  }
}
```

**Alternative:** Offset-based (simpler, worse performance at scale):
```
GET /api/v1/orders?page=1&per_page=25
```

**Effort:** 8-12 hours (all endpoints + repository changes)

---

### 7. API Documentation (OpenAPI/Swagger)

**Current State:** No API documentation beyond CLAUDE.md.

**Recommendation:** Generate OpenAPI 3.0 spec using `swaggo/swag`:

1. Add annotations to handlers
2. Generate `docs/swagger.json`
3. Serve Swagger UI at `/docs` (development only)

**Effort:** 6-8 hours

---

### 8. Database Indexes

**Current State:** Only primary keys and explicit unique constraints indexed.

**Recommended Indexes:**

```sql
-- Order queries by client and status
CREATE INDEX idx_orders_client_status ON orders(client_id, status);
CREATE INDEX idx_orders_created_at ON orders(created_at DESC);

-- Product queries by category and active status
CREATE INDEX idx_products_category_active ON products(category, is_active) WHERE is_active = true;
CREATE INDEX idx_products_sku ON products(sku);

-- Client email lookup (for auth linking)
CREATE INDEX idx_clients_email ON clients(email);

-- Product images by product
CREATE INDEX idx_product_images_product ON product_images(product_id, display_order);
```

**Effort:** 2-3 hours (new migration file)

---

## P3: Nice to Have (Post-Launch)

### 9. Metrics & Observability

**Components:**
- Prometheus metrics endpoint (`/metrics`)
- Request duration histograms
- Error rate counters
- Database pool stats
- Custom business metrics (orders/hour, sync lag)

**Effort:** 12-16 hours

---

### 10. Automated Background Sync

**Current State:** Product/client sync is manual via admin endpoints.

**Recommendation:**
- Background goroutine with configurable interval (`SYNC_INTERVAL_MINUTES`)
- Leader election if running multiple instances
- Circuit breaker for Holded API failures

**Effort:** 8-12 hours

---

## Implementation Roadmap

### Week 1: Critical Security & Reliability
- [ ] P0: CORS configuration
- [ ] P0: Config validation
- [ ] P1: Request size limits

### Week 2: Protection & Operations
- [ ] P1: Rate limiting
- [ ] P1: Health check enhancements
- [ ] P2: Database indexes

### Week 3: Developer Experience
- [ ] P2: Pagination (prioritize orders endpoint)
- [ ] P2: API documentation

### Post-Launch
- [ ] P3: Metrics & observability
- [ ] P3: Automated background sync
- [ ] P2: Remaining pagination endpoints

---

## Architecture Notes

### What's Done Well

1. **Clean Layered Architecture**
   - Handlers → Services → Repositories
   - Domain models separate from DTOs
   - Interfaces for testability

2. **Auth Design**
   - Auth0 integration with JWKS caching
   - Proper separation of authentication and authorization
   - Client linking flow (Auth0 ID → existing Holded contact)

3. **Error Handling**
   - Consistent error codes
   - Proper HTTP status mapping
   - Wrapped errors with context

4. **Testing Strategy**
   - Table-driven tests
   - Mock implementations for all repositories
   - Context injection for auth testing

### Minor Recommendations

1. **CLAUDE.md Accuracy**: Update to reflect actual implementation state (remove references to "placeholder" and "TODO" handlers)

2. **Order Status Validation**: Consider adding database-level CHECK constraint for valid status values

3. **S3 URL Generation**: Current implementation generates public URLs; consider signed URLs for private content

4. **Holded Rate Limits**: Add retry logic with backoff for Holded API calls

---

## Decision Log

| Decision | Rationale |
|----------|-----------|
| Cursor-based pagination | Better for real-time order data; offset pagination breaks with new inserts |
| Per-user rate limiting | Prevents single compromised client from affecting others |
| Separate liveness/readiness | Kubernetes best practice; allows pod restart vs. traffic routing |
| swaggo for OpenAPI | Generates from annotations; keeps docs in sync with code |

---

## Appendix: Files to Modify

| File | Changes |
|------|---------|
| `internal/server/server.go` | CORS config, request limits |
| `internal/config/config.go` | Validation expansion |
| `internal/handlers/health.go` | Enhanced health checks |
| `internal/repository/interfaces.go` | Add pagination params to List methods |
| `internal/repository/postgres/*.go` | Implement pagination queries |
| `migrations/002_add_indexes.up.sql` | New migration for indexes |
| New: `internal/middleware/ratelimit.go` | Rate limiting middleware |
