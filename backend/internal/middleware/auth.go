package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	jwtmiddleware "github.com/auth0/go-jwt-middleware/v2"
	"github.com/auth0/go-jwt-middleware/v2/jwks"
	"github.com/auth0/go-jwt-middleware/v2/validator"

	"b2b-orders-api/internal/config"
)

// CustomClaims contains custom claims from Auth0 token
type CustomClaims struct {
	Email  string                 `json:"email"`
	Scope  string                 `json:"scope"`
	Extra  map[string]interface{} `json:"-"` // Captures additional claims
}

// UnmarshalJSON custom unmarshaler to capture all claims
func (c *CustomClaims) UnmarshalJSON(data []byte) error {
	// First unmarshal into a map to capture everything
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	// Extract known fields
	if email, ok := raw["email"].(string); ok {
		c.Email = email
	}
	if scope, ok := raw["scope"].(string); ok {
		c.Scope = scope
	}

	// Store the full map for role extraction later
	c.Extra = raw
	return nil
}

// Validate implements validator.CustomClaims
func (c CustomClaims) Validate(ctx context.Context) error {
	return nil
}

// GetRoles extracts roles from a specific claim name
func (c CustomClaims) GetRoles(claimName string) []string {
	if claimName == "" || c.Extra == nil {
		return nil
	}

	rolesRaw, ok := c.Extra[claimName]
	if !ok {
		return nil
	}

	// Handle []interface{} from JSON
	rolesSlice, ok := rolesRaw.([]interface{})
	if !ok {
		return nil
	}

	roles := make([]string, 0, len(rolesSlice))
	for _, r := range rolesSlice {
		if role, ok := r.(string); ok {
			roles = append(roles, role)
		}
	}
	return roles
}

// HasRole checks if the user has a specific role
func (c CustomClaims) HasRole(claimName, role string) bool {
	for _, r := range c.GetRoles(claimName) {
		if r == role {
			return true
		}
	}
	return false
}

// AuthMiddleware handles JWT validation using Auth0
type AuthMiddleware struct {
	middleware *jwtmiddleware.JWTMiddleware
	roleClaim  string
	logger     *slog.Logger
}

// roleContextKey is the key for storing roles in context
type roleContextKey struct{}

// NewAuthMiddleware creates a new Auth0 JWT middleware
func NewAuthMiddleware(cfg config.Auth0Config, logger *slog.Logger) (*AuthMiddleware, error) {
	issuerURL, err := url.Parse("https://" + cfg.Domain + "/")
	if err != nil {
		return nil, fmt.Errorf("parsing issuer URL: %w", err)
	}

	provider := jwks.NewCachingProvider(issuerURL, 5*time.Minute)

	jwtValidator, err := validator.New(
		provider.KeyFunc,
		validator.RS256,
		issuerURL.String(),
		[]string{cfg.Audience},
		validator.WithCustomClaims(func() validator.CustomClaims {
			return &CustomClaims{}
		}),
		validator.WithAllowedClockSkew(30*time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("creating validator: %w", err)
	}

	errorHandler := func(w http.ResponseWriter, r *http.Request, err error) {
		logger.Warn("JWT validation failed", "error", err, "path", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"unauthorized","message":"Invalid or missing token"}`))
	}

	middleware := jwtmiddleware.New(
		jwtValidator.ValidateToken,
		jwtmiddleware.WithErrorHandler(errorHandler),
	)

	return &AuthMiddleware{
		middleware: middleware,
		roleClaim:  cfg.RoleClaim,
		logger:     logger,
	}, nil
}

// Authenticate is a middleware that validates JWT tokens
func (a *AuthMiddleware) Authenticate(next http.Handler) http.Handler {
	return a.middleware.CheckJWT(next)
}

// GetValidatedClaims extracts validated claims from the request context
func GetValidatedClaims(ctx context.Context) (*validator.ValidatedClaims, error) {
	claims, ok := ctx.Value(jwtmiddleware.ContextKey{}).(*validator.ValidatedClaims)
	if !ok {
		return nil, fmt.Errorf("no claims in context")
	}
	return claims, nil
}

// GetCustomClaims extracts custom claims from the request context
func GetCustomClaims(ctx context.Context) (*CustomClaims, error) {
	validated, err := GetValidatedClaims(ctx)
	if err != nil {
		return nil, err
	}

	custom, ok := validated.CustomClaims.(*CustomClaims)
	if !ok {
		return nil, fmt.Errorf("invalid custom claims type")
	}
	return custom, nil
}

// GetAuth0ID extracts the Auth0 user ID (sub claim) from context
func GetAuth0ID(ctx context.Context) (string, error) {
	claims, err := GetValidatedClaims(ctx)
	if err != nil {
		return "", err
	}
	return claims.RegisteredClaims.Subject, nil
}

// GetEmail extracts the email from custom claims
func GetEmail(ctx context.Context) (string, error) {
	custom, err := GetCustomClaims(ctx)
	if err != nil {
		return "", err
	}
	return custom.Email, nil
}

// RequireAdmin is a middleware that checks for admin role
func (a *AuthMiddleware) RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		custom, err := GetCustomClaims(r.Context())
		if err != nil {
			a.logger.Warn("failed to get claims for admin check", "error", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte(`{"error":"forbidden","message":"Admin access required"}`))
			return
		}

		if !custom.HasRole(a.roleClaim, "admin") {
			a.logger.Warn("non-admin user attempted admin access",
				"email", custom.Email,
				"path", r.URL.Path,
			)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte(`{"error":"forbidden","message":"Admin access required"}`))
			return
		}

		next.ServeHTTP(w, r)
	})
}
