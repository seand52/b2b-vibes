package holded

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/go-resty/resty/v2"
)

const defaultTimeout = 10 * time.Second

// Client is the Holded API client
type Client struct {
	resty  *resty.Client
	logger *slog.Logger
}

// Config holds Holded client configuration
type Config struct {
	APIKey  string
	BaseURL string
	Timeout time.Duration
	Logger  *slog.Logger
}

// NewClient creates a new Holded API client
func NewClient(cfg Config) *Client {
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = defaultTimeout
	}

	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "https://api.holded.com/api/invoicing/v1"
	}

	r := resty.New().
		SetBaseURL(baseURL).
		SetTimeout(timeout).
		SetHeader("key", cfg.APIKey).
		SetHeader("Content-Type", "application/json").
		SetHeader("Accept", "application/json")

	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}

	return &Client{
		resty:  r,
		logger: logger,
	}
}

// APIError represents an error from the Holded API
type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("holded API error: %d - %s", e.StatusCode, e.Message)
}

func (c *Client) handleResponse(resp *resty.Response, err error) error {
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}

	if resp.IsError() {
		// Log full response for debugging, but don't expose to caller
		c.logger.Error("holded API error",
			"status", resp.StatusCode(),
			"path", resp.Request.URL,
			"body", string(resp.Body()),
		)

		return &APIError{
			StatusCode: resp.StatusCode(),
			Message:    "external service error",
		}
	}

	return nil
}

func (c *Client) get(ctx context.Context, path string, result any) error {
	resp, err := c.resty.R().
		SetContext(ctx).
		SetResult(result).
		Get(path)

	return c.handleResponse(resp, err)
}

func (c *Client) post(ctx context.Context, path string, body any, result any) error {
	resp, err := c.resty.R().
		SetContext(ctx).
		SetBody(body).
		SetResult(result).
		Post(path)

	return c.handleResponse(resp, err)
}
