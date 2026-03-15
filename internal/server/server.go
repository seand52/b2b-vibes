package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"

	"b2b-orders-api/internal/config"
	"b2b-orders-api/internal/handlers"
	"b2b-orders-api/internal/middleware"
)

// Server represents the HTTP server
type Server struct {
	router     *chi.Mux
	httpServer *http.Server
	db         *pgxpool.Pool
	config     *config.Config
	logger     *slog.Logger

	// Handlers
	productHandler *handlers.ProductHandler
	orderHandler   *handlers.OrderHandler
	cartHandler    *handlers.CartHandler
	adminHandler   *handlers.AdminHandler
	syncHandler    *handlers.SyncHandler
	healthHandler  *handlers.HealthHandler

	// Middleware
	authMiddleware *middleware.AuthMiddleware
}

// ServerDeps contains all dependencies needed to create the server
type ServerDeps struct {
	Config         *config.Config
	DB             *pgxpool.Pool
	Logger         *slog.Logger
	ProductHandler *handlers.ProductHandler
	OrderHandler   *handlers.OrderHandler
	CartHandler    *handlers.CartHandler
	AdminHandler   *handlers.AdminHandler
	SyncHandler    *handlers.SyncHandler
	HealthHandler  *handlers.HealthHandler
	AuthMiddleware *middleware.AuthMiddleware
}

// New creates a new server instance
func New(deps ServerDeps) *Server {
	s := &Server{
		router:         chi.NewRouter(),
		db:             deps.DB,
		config:         deps.Config,
		logger:         deps.Logger,
		productHandler: deps.ProductHandler,
		orderHandler:   deps.OrderHandler,
		cartHandler:    deps.CartHandler,
		adminHandler:   deps.AdminHandler,
		syncHandler:    deps.SyncHandler,
		healthHandler:  deps.HealthHandler,
		authMiddleware: deps.AuthMiddleware,
	}

	s.setupMiddleware()
	s.setupRoutes()

	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", deps.Config.Server.Host, deps.Config.Server.Port),
		Handler:      s.router,
		ReadTimeout:  deps.Config.Server.ReadTimeout,
		WriteTimeout: deps.Config.Server.WriteTimeout,
		IdleTimeout:  time.Minute,
	}

	return s
}

func (s *Server) setupMiddleware() {
	s.router.Use(chimiddleware.RequestID)
	s.router.Use(chimiddleware.RealIP)
	s.router.Use(chimiddleware.Logger)
	s.router.Use(chimiddleware.Recoverer)
	s.router.Use(chimiddleware.Timeout(60 * time.Second))

	// Request body size limit (1MB)
	s.router.Use(func(next http.Handler) http.Handler {
		return http.MaxBytesHandler(next, 1<<20) // 1MB
	})

	allowCredentials := len(s.config.CORS.AllowedOrigins) > 0 && s.config.CORS.AllowedOrigins[0] != "*"

	s.router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   s.config.CORS.AllowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Request-ID"},
		ExposedHeaders:   []string{"X-Request-ID"},
		AllowCredentials: allowCredentials,
		MaxAge:           300,
	}))
}

func (s *Server) setupRoutes() {
	// Health checks (public, no auth)
	s.router.Get("/health", s.healthHandler.Full)
	s.router.Get("/health/live", s.healthHandler.Live)
	s.router.Get("/health/ready", s.healthHandler.Ready)

	// API v1 routes
	s.router.Route("/api/v1", func(r chi.Router) {
		// Public routes - product browsing (still requires auth to identify client)
		// 100 requests per minute per user
		r.Route("/products", func(r chi.Router) {
			r.Use(s.authMiddleware.Authenticate)
			r.Use(middleware.RateLimitByUser(100))
			r.Get("/", s.productHandler.List)
			r.Get("/{id}", s.productHandler.Get)
		})

		// Client routes - orders
		// Reads: 100/min, Writes: 20/min
		r.Route("/orders", func(r chi.Router) {
			r.Use(s.authMiddleware.Authenticate)
			r.With(middleware.RateLimitByUser(20)).Post("/", s.orderHandler.Create)
			r.With(middleware.RateLimitByUser(100)).Get("/", s.orderHandler.List)
			r.With(middleware.RateLimitByUser(100)).Get("/{id}", s.orderHandler.Get)
			r.With(middleware.RateLimitByUser(20)).Post("/{id}/cancel", s.orderHandler.Cancel)
		})

		// Client routes - cart (draft orders)
		// Reads: 100/min, Writes: 20/min
		r.Route("/cart", func(r chi.Router) {
			r.Use(s.authMiddleware.Authenticate)
			r.With(middleware.RateLimitByUser(100)).Get("/", s.cartHandler.GetCart)
			r.With(middleware.RateLimitByUser(20)).Post("/", s.cartHandler.CreateCart)
			r.With(middleware.RateLimitByUser(20)).Delete("/", s.cartHandler.DiscardCart)

			r.Route("/items", func(r chi.Router) {
				r.With(middleware.RateLimitByUser(20)).Post("/", s.cartHandler.AddItem)
				r.With(middleware.RateLimitByUser(20)).Put("/", s.cartHandler.SetItems)
				r.With(middleware.RateLimitByUser(20)).Put("/{product_id}", s.cartHandler.UpdateItemQuantity)
				r.With(middleware.RateLimitByUser(20)).Delete("/{product_id}", s.cartHandler.RemoveItem)
			})

			r.With(middleware.RateLimitByUser(20)).Put("/notes", s.cartHandler.UpdateNotes)
			r.With(middleware.RateLimitByUser(20)).Post("/submit", s.cartHandler.Submit)
		})

		// Admin routes
		r.Route("/admin", func(r chi.Router) {
			r.Use(s.authMiddleware.Authenticate)
			r.Use(s.authMiddleware.RequireAdmin)
			r.Use(middleware.RateLimitByUser(50))

			r.Route("/orders", func(r chi.Router) {
				r.Get("/", s.adminHandler.ListOrders)
				r.Get("/{id}", s.adminHandler.GetOrder)
				r.Post("/{id}/approve", s.adminHandler.ApproveOrder)
				r.Post("/{id}/reject", s.adminHandler.RejectOrder)
			})

			r.Route("/clients", func(r chi.Router) {
				r.Get("/", s.adminHandler.ListClients)
				r.Get("/{id}", s.adminHandler.GetClient)
			})

			r.Route("/sync", func(r chi.Router) {
				// Override: only 5/min for sync operations
				r.Use(middleware.RateLimitByUser(5))
				r.Post("/products", s.syncHandler.SyncProducts)
				r.Post("/clients", s.syncHandler.SyncClients)
			})
		})
	})
}

// Start starts the HTTP server
func (s *Server) Start() error {
	s.logger.Info("starting server", "addr", s.httpServer.Addr)
	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("shutting down server")
	return s.httpServer.Shutdown(ctx)
}
