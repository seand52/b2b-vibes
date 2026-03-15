package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Version should be set at build time via ldflags
var Version = "dev"

// HealthResponse represents the full health check response
type HealthResponse struct {
	Status      string                 `json:"status"`
	Version     string                 `json:"version"`
	Environment string                 `json:"environment,omitempty"`
	Checks      map[string]HealthCheck `json:"checks,omitempty"`
}

// HealthCheck represents an individual dependency check
type HealthCheck struct {
	Status    string `json:"status"`
	LatencyMs int64  `json:"latency_ms,omitempty"`
	Error     string `json:"error,omitempty"`
}

// HealthHandler holds dependencies for health checks
type HealthHandler struct {
	db          *pgxpool.Pool
	environment string
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(db *pgxpool.Pool, environment string) *HealthHandler {
	return &HealthHandler{
		db:          db,
		environment: environment,
	}
}

// Live handles liveness probe - just checks if process is responding
func (h *HealthHandler) Live(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "alive"})
}

// Ready handles readiness probe - checks critical dependencies
func (h *HealthHandler) Ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Check database
	start := time.Now()
	err := h.db.Ping(ctx)
	latency := time.Since(start).Milliseconds()

	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(HealthResponse{
			Status:  "not_ready",
			Version: Version,
			Checks: map[string]HealthCheck{
				"database": {Status: "down", Error: "connection failed"},
			},
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(HealthResponse{
		Status:  "ready",
		Version: Version,
		Checks: map[string]HealthCheck{
			"database": {Status: "up", LatencyMs: latency},
		},
	})
}

// Full handles comprehensive health check with all dependency status
func (h *HealthHandler) Full(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	checks := make(map[string]HealthCheck)
	allHealthy := true

	// Check database
	start := time.Now()
	err := h.db.Ping(ctx)
	latency := time.Since(start).Milliseconds()

	if err != nil {
		checks["database"] = HealthCheck{Status: "down", Error: "connection failed"}
		allHealthy = false
	} else {
		checks["database"] = HealthCheck{Status: "up", LatencyMs: latency}
	}

	status := "healthy"
	statusCode := http.StatusOK
	if !allHealthy {
		status = "unhealthy"
		statusCode = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(HealthResponse{
		Status:      status,
		Version:     Version,
		Environment: h.environment,
		Checks:      checks,
	})
}

// Health is the legacy health check for backwards compatibility
// Deprecated: Use Full, Ready, or Live instead
func Health(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		if err := db.Ping(ctx); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]string{"status": "unhealthy"})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
	}
}
