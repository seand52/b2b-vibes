package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"b2b-orders-api/internal/config"
	"b2b-orders-api/internal/database"
	"b2b-orders-api/internal/clients/holded"
	"b2b-orders-api/internal/clients/s3"
	"b2b-orders-api/internal/handlers"
	"b2b-orders-api/internal/logger"
	"b2b-orders-api/internal/middleware"
	"b2b-orders-api/internal/repository/postgres"
	"b2b-orders-api/internal/server"
	"b2b-orders-api/internal/service/auth"
	"b2b-orders-api/internal/service/cart"
	"b2b-orders-api/internal/service/order"
	"b2b-orders-api/internal/service/sync"
)

func main() {
	// Load .env file if present (ignore error if not found)
	_ = godotenv.Load()

	// Load config from environment variables
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	log := logger.NewFromEnv(cfg.Server.Environment)

	if err := run(cfg, log); err != nil {
		log.Error("application error", "error", err)
		os.Exit(1)
	}
}

func run(cfg *config.Config, log *slog.Logger) error {
	ctx := context.Background()

	// Connect to database
	db, err := database.New(ctx, database.Config{
		URL:          cfg.DB.URL,
		MaxOpenConns: cfg.DB.MaxOpenConns,
		MaxIdleConns: cfg.DB.MaxIdleConns,
	})
	if err != nil {
		return err
	}
	defer db.Close()

	log.Info("connected to database")

	// Initialize external clients
	var holdedClient holded.ClientInterface
	if cfg.IsDevelopment() && cfg.Holded.APIKey == "" {
		log.Info("using mock Holded client (no API key provided)")
		holdedClient = holded.NewMockClient()
	} else {
		holdedClient = holded.NewClient(holded.Config{
			APIKey:  cfg.Holded.APIKey,
			BaseURL: cfg.Holded.BaseURL,
		})
	}

	// Initialize S3 client
	s3Client, err := s3.NewClient(ctx, s3.Config{
		Region:    cfg.S3.Region,
		Bucket:    cfg.S3.Bucket,
		AccessKey: cfg.S3.AccessKey,
		SecretKey: cfg.S3.SecretKey,
	})
	if err != nil {
		return err
	}

	log.Info("initialized S3 client")

	// Initialize repositories
	productRepo := postgres.NewProductRepository(db)
	productImageRepo := postgres.NewProductImageRepository(db)
	clientRepo := postgres.NewClientRepository(db)
	orderRepo := postgres.NewOrderRepository(db)
	syncStateRepo := postgres.NewSyncStateRepository(db)

	// Initialize services
	authService := auth.NewService(clientRepo, log)
	orderService := order.NewService(orderRepo, productRepo, clientRepo, holdedClient, log)
	cartService := cart.NewService(orderRepo, productRepo, log)
	productSyncer := sync.NewProductSyncer(holdedClient, s3Client, productRepo, productImageRepo, syncStateRepo, log)
	clientSyncer := sync.NewClientSyncer(holdedClient, clientRepo, syncStateRepo, log)

	// Initialize auth middleware
	authMiddleware, err := middleware.NewAuthMiddleware(cfg.Auth0, log)
	if err != nil {
		return err
	}

	// Initialize handlers
	productHandler := handlers.NewProductHandler(productRepo, productImageRepo, log)
	orderHandler := handlers.NewOrderHandler(orderService, authService, log)
	cartHandler := handlers.NewCartHandler(cartService, authService, log)
	adminHandler := handlers.NewAdminHandler(orderService, clientRepo, log)
	syncHandler := handlers.NewSyncHandler(productSyncer, clientSyncer, log)
	healthHandler := handlers.NewHealthHandler(db, cfg.Server.Environment)

	// Create and start server
	srv := server.New(server.ServerDeps{
		Config:         cfg,
		DB:             db,
		Logger:         log,
		ProductHandler: productHandler,
		OrderHandler:   orderHandler,
		CartHandler:    cartHandler,
		AdminHandler:   adminHandler,
		SyncHandler:    syncHandler,
		HealthHandler:  healthHandler,
		AuthMiddleware: authMiddleware,
	})

	// Graceful shutdown
	shutdownCh := make(chan os.Signal, 1)
	signal.Notify(shutdownCh, os.Interrupt, syscall.SIGTERM)

	errCh := make(chan error, 1)
	go func() {
		if err := srv.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-shutdownCh:
		log.Info("received shutdown signal")

		shutdownCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			return err
		}
	}

	log.Info("server stopped")
	return nil
}
