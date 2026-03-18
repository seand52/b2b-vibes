package middleware

import (
	"net/http"
	"time"

	"github.com/go-chi/httprate"
)

// RateLimitByIP creates a rate limiter keyed by client IP
func RateLimitByIP(requestsPerMin int) func(http.Handler) http.Handler {
	return httprate.Limit(
		requestsPerMin,
		time.Minute,
		httprate.WithKeyFuncs(httprate.KeyByIP),
		httprate.WithLimitHandler(rateLimitExceededHandler),
	)
}

// RateLimitByUser creates a rate limiter keyed by authenticated user (JWT sub claim)
func RateLimitByUser(requestsPerMin int) func(http.Handler) http.Handler {
	return httprate.Limit(
		requestsPerMin,
		time.Minute,
		httprate.WithKeyFuncs(func(r *http.Request) (string, error) {
			// Extract user ID from JWT context
			auth0ID, err := GetAuth0ID(r.Context())
			if err != nil {
				// Fall back to IP if no auth
				return httprate.KeyByIP(r)
			}
			return auth0ID, nil
		}),
		httprate.WithLimitHandler(rateLimitExceededHandler),
	)
}

func rateLimitExceededHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusTooManyRequests)
	w.Write([]byte(`{"code":"RATE_LIMITED","message":"Too many requests. Please try again later."}`))
}
