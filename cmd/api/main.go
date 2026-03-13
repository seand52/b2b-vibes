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

	"b2b-orders-api/internal/config"
	"b2b-orders-api/internal/database"
	"b2b-orders-api/internal/logger"
	"b2b-orders-api/internal/server"
)

func main() {
	// Load config first to determine environment
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

	// Create and start server
	srv := server.New(cfg, db, log)

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
