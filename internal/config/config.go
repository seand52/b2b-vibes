package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all application configuration
type Config struct {
	Server  ServerConfig
	DB      DatabaseConfig
	Auth0   Auth0Config
	Holded  HoldedConfig
	S3      S3Config
	Sync    SyncConfig
	CORS    CORSConfig
}

type ServerConfig struct {
	Host         string
	Port         int
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	Environment  string
}

type DatabaseConfig struct {
	URL          string
	MaxOpenConns int
	MaxIdleConns int
}

type Auth0Config struct {
	Domain    string
	Audience  string
	RoleClaim string // Auth0 custom claim containing roles (e.g., "https://myapp.com/roles")
}

type HoldedConfig struct {
	APIKey  string
	BaseURL string
}

type S3Config struct {
	Region    string
	Bucket    string
	AccessKey string
	SecretKey string
}

type SyncConfig struct {
	IntervalMinutes int
}

type CORSConfig struct {
	AllowedOrigins []string
}

// Load reads configuration from environment variables
func Load() (*Config, error) {
	env := getEnv("ENV", "development")

	cfg := &Config{
		Server: ServerConfig{
			Host:         getEnv("SERVER_HOST", "0.0.0.0"),
			Port:         getEnvInt("PORT", 8080),
			ReadTimeout:  getEnvDuration("SERVER_READ_TIMEOUT", 10*time.Second),
			WriteTimeout: getEnvDuration("SERVER_WRITE_TIMEOUT", 10*time.Second),
			Environment:  env,
		},
		DB: DatabaseConfig{
			URL:          getEnv("DATABASE_URL", ""),
			MaxOpenConns: getEnvInt("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns: getEnvInt("DB_MAX_IDLE_CONNS", 5),
		},
		Auth0: Auth0Config{
			Domain:    getEnv("AUTH0_DOMAIN", ""),
			Audience:  getEnv("AUTH0_AUDIENCE", ""),
			RoleClaim: getEnv("AUTH0_ROLE_CLAIM", ""),
		},
		Holded: HoldedConfig{
			APIKey:  getEnv("HOLDED_API_KEY", ""),
			BaseURL: getEnv("HOLDED_BASE_URL", "https://api.holded.com/api/invoicing/v1"),
		},
		S3: S3Config{
			Region:    getEnv("AWS_REGION", "eu-west-1"),
			Bucket:    getEnv("AWS_S3_BUCKET", ""),
			AccessKey: getEnv("AWS_ACCESS_KEY_ID", ""),
			SecretKey: getEnv("AWS_SECRET_ACCESS_KEY", ""),
		},
		Sync: SyncConfig{
			IntervalMinutes: getEnvInt("SYNC_INTERVAL_MINUTES", 15),
		},
		CORS: CORSConfig{
			AllowedOrigins: getCORSOrigins(env),
		},
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) validate() error {
	var errs []string

	// Always required
	if c.DB.URL == "" {
		errs = append(errs, "DATABASE_URL is required")
	}
	if c.Auth0.Domain == "" {
		errs = append(errs, "AUTH0_DOMAIN is required")
	}
	if c.Auth0.Audience == "" {
		errs = append(errs, "AUTH0_AUDIENCE is required")
	}

	// Production-only requirements
	if c.IsProduction() {
		if c.S3.Bucket == "" {
			errs = append(errs, "AWS_S3_BUCKET is required in production")
		}
		if c.Holded.APIKey == "" {
			errs = append(errs, "HOLDED_API_KEY is required in production")
		}
		if len(c.CORS.AllowedOrigins) == 0 {
			errs = append(errs, "CORS_ALLOWED_ORIGINS is required in production")
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("config validation failed: %s", strings.Join(errs, "; "))
	}
	return nil
}

// IsDevelopment returns true if running in development mode
func (c *Config) IsDevelopment() bool {
	return c.Server.Environment == "development"
}

// IsProduction returns true if running in production mode
func (c *Config) IsProduction() bool {
	return c.Server.Environment == "production"
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

func getCORSOrigins(env string) []string {
	if originsEnv := os.Getenv("CORS_ALLOWED_ORIGINS"); originsEnv != "" {
		origins := strings.Split(originsEnv, ",")
		for i, origin := range origins {
			origins[i] = strings.TrimSpace(origin)
		}
		return origins
	}

	if env == "development" {
		return []string{
			"http://localhost:3000",
			"http://localhost:5173",
			"http://localhost:8080",
		}
	}

	return []string{}
}
