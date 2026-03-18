//go:build integration

package integration

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHealth_Live(t *testing.T) {
	resp := doRequest(t, http.MethodGet, "/health/live", nil, nil)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]string
	parseJSON(t, resp, &result)

	assert.Equal(t, "alive", result["status"])
}

func TestHealth_Ready(t *testing.T) {
	resp := doRequest(t, http.MethodGet, "/health/ready", nil, nil)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result struct {
		Status  string `json:"status"`
		Version string `json:"version"`
		Checks  map[string]struct {
			Status    string `json:"status"`
			LatencyMs int64  `json:"latency_ms"`
		} `json:"checks"`
	}
	parseJSON(t, resp, &result)

	assert.Equal(t, "ready", result.Status)
	require.Contains(t, result.Checks, "database")
	assert.Equal(t, "up", result.Checks["database"].Status)
}

func TestHealth_Full(t *testing.T) {
	resp := doRequest(t, http.MethodGet, "/health", nil, nil)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result struct {
		Status      string `json:"status"`
		Version     string `json:"version"`
		Environment string `json:"environment"`
		Checks      map[string]struct {
			Status    string `json:"status"`
			LatencyMs int64  `json:"latency_ms"`
		} `json:"checks"`
	}
	parseJSON(t, resp, &result)

	assert.Equal(t, "healthy", result.Status)
	assert.Equal(t, "test", result.Environment)
	require.Contains(t, result.Checks, "database")
	assert.Equal(t, "up", result.Checks["database"].Status)
}
