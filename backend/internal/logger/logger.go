package logger

import (
	"io"
	"log/slog"
	"os"
)

// Config holds logger configuration
type Config struct {
	Level       string // "debug", "info", "warn", "error"
	Format      string // "json", "text"
	Output      io.Writer
	AddSource   bool
}

// New creates a new slog.Logger based on configuration
func New(cfg Config) *slog.Logger {
	if cfg.Output == nil {
		cfg.Output = os.Stdout
	}

	level := parseLevel(cfg.Level)

	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: cfg.AddSource,
	}

	var handler slog.Handler
	switch cfg.Format {
	case "text":
		handler = slog.NewTextHandler(cfg.Output, opts)
	default:
		handler = slog.NewJSONHandler(cfg.Output, opts)
	}

	return slog.New(handler)
}

// NewFromEnv creates a logger configured for the environment
func NewFromEnv(env string) *slog.Logger {
	cfg := Config{
		Level:  "info",
		Format: "json",
		Output: os.Stdout,
	}

	if env == "development" {
		cfg.Level = "debug"
		cfg.Format = "text"
		cfg.AddSource = true
	}

	return New(cfg)
}

func parseLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
