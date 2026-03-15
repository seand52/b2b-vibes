# Health Check Implementation

## Overview

Enhanced health checks with Kubernetes-compatible liveness/readiness probes.

## Endpoints

### 1. Liveness Probe - `/health/live`

**Purpose**: Checks if the application process is responsive.

**Response**: Always returns 200 OK if the process is running.

```json
{
  "status": "alive"
}
```

**Kubernetes Configuration**:
```yaml
livenessProbe:
  httpGet:
    path: /health/live
    port: 8080
  initialDelaySeconds: 10
  periodSeconds: 10
  timeoutSeconds: 2
  failureThreshold: 3
```

### 2. Readiness Probe - `/health/ready`

**Purpose**: Checks if the application is ready to serve traffic (database is available).

**Response**:
- 200 OK if database is reachable
- 503 Service Unavailable if database is down

**Success Response**:
```json
{
  "status": "ready",
  "version": "v1.0.0",
  "checks": {
    "database": {
      "status": "up",
      "latency_ms": 15
    }
  }
}
```

**Failure Response**:
```json
{
  "status": "not_ready",
  "version": "v1.0.0",
  "checks": {
    "database": {
      "status": "down",
      "error": "connection failed"
    }
  }
}
```

**Kubernetes Configuration**:
```yaml
readinessProbe:
  httpGet:
    path: /health/ready
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 5
  timeoutSeconds: 5
  successThreshold: 1
  failureThreshold: 2
```

### 3. Full Health Check - `/health`

**Purpose**: Comprehensive health check with detailed dependency status, version, and environment info.

**Response**:
- 200 OK if all checks pass
- 503 Service Unavailable if any check fails

```json
{
  "status": "healthy",
  "version": "v1.0.0",
  "environment": "production",
  "checks": {
    "database": {
      "status": "up",
      "latency_ms": 15
    }
  }
}
```

## Design Decisions

### Why These Three Endpoints?

1. **Liveness** (`/health/live`): Kubernetes uses this to determine if the pod should be restarted. It only checks if the process is running, not if dependencies are available. This prevents restart loops when external dependencies (like the database) are temporarily unavailable.

2. **Readiness** (`/health/ready`): Kubernetes uses this to determine if the pod should receive traffic. It checks critical dependencies (database). If the database is down, the pod is removed from the service load balancer until it recovers.

3. **Full** (`/health`): Provides detailed status for monitoring systems, debugging, and operations teams. Includes version information and environment context.

### What's NOT Checked in Readiness?

External services like Holded and S3 are NOT checked in readiness probes because:
- They are not required for every request
- Checking them would make the readiness probe too sensitive
- Failures to these services should be handled gracefully at the application level
- These services may have their own rate limits or throttling

### Timeout Strategy

All database health checks use a 5-second timeout to prevent hanging probes. This is configured at the handler level, not at the database pool level, so it doesn't interfere with normal request handling.

## Version Information

The `Version` variable in `handlers/health.go` should be set at build time:

```bash
go build -ldflags="-X 'b2b-orders-api/internal/handlers.Version=v1.0.0'" ./cmd/api
```

## Backwards Compatibility

The legacy `Health()` function is preserved for backwards compatibility. It returns a simple status response:

```json
{
  "status": "healthy"
}
```

This endpoint is deprecated but maintained to avoid breaking existing clients.

## Testing

### Unit Tests

Located in `internal/handlers/health_test.go`:
- Tests for liveness probe (always succeeds)
- Tests for readiness probe with database success/failure
- Tests for full health check with all checks
- Tests for legacy health endpoint
- Tests for timeout handling

### Manual Testing

```bash
# Liveness check
curl http://localhost:8080/health/live

# Readiness check
curl http://localhost:8080/health/ready

# Full health check
curl http://localhost:8080/health
```

## Implementation Details

### File Changes

1. **`internal/handlers/health.go`**:
   - Added `HealthHandler` struct with database and environment dependencies
   - Implemented `Live()`, `Ready()`, and `Full()` methods
   - Added `HealthResponse` and `HealthCheck` types for structured responses
   - Preserved legacy `Health()` function for backwards compatibility

2. **`internal/server/server.go`**:
   - Added `healthHandler` field to `Server` struct
   - Added `HealthHandler` to `ServerDeps` struct
   - Updated `New()` to initialize health handler
   - Updated `setupRoutes()` to register three new endpoints

3. **`cmd/api/main.go`**:
   - Added health handler initialization
   - Passed health handler to server dependencies

4. **`internal/handlers/health_test.go`** (new):
   - Comprehensive test suite for all health check endpoints
   - Integration tests that can run against real database
   - Unit tests for liveness probe

## Monitoring Integration

Health check endpoints can be integrated with:

- **Kubernetes**: Liveness and readiness probes
- **Prometheus**: Scrape `/health` endpoint for metrics
- **Load Balancers**: Use `/health/ready` for backend health checks
- **Uptime Monitors**: Use `/health` for availability monitoring

## Security Considerations

Health check endpoints are intentionally **NOT** protected by authentication:
- They need to be accessible to Kubernetes control plane
- They need to be accessible to load balancers
- They don't expose sensitive information
- They have a 5-second timeout to prevent abuse

Rate limiting is also intentionally NOT applied to health checks to ensure probes always succeed when the application is healthy.
