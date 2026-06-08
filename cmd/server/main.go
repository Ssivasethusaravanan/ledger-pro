// Package main is the entry point for the LedgerPro financial ledger server.
// It wires all dependencies together, runs database migrations on startup,
// initializes the outbox worker, and manages graceful shutdown of all
// connections and background goroutines.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	_ "github.com/joho/godotenv/autoload"
	"github.com/jackc/pgx/v5/pgxpool"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"

	"ledger_pro/db/migrations"
	"ledger_pro/internal/api"
	"ledger_pro/internal/db"
	"ledger_pro/internal/service"
)

// @title LedgerPro API
// @version 1.0
// @description A FAANG-grade double-entry ledger system.
// @host ledger-pro-api.onrender.com
// @BasePath /
// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name X-API-Key
func main() {
	// -----------------------------------------------------------------------
	// Logger
	// -----------------------------------------------------------------------
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level:     parseLogLevel(getEnv("LOG_LEVEL", "info")),
		AddSource: true,
	}))
	slog.SetDefault(logger)

	logger.Info("starting LedgerPro server",
		slog.String("version", "2.0.0"),
		slog.String("go", "1.22+"),
	)

	// -----------------------------------------------------------------------
	// Configuration from Environment
	// -----------------------------------------------------------------------
	cfg := loadConfig()

	// -----------------------------------------------------------------------
	// Run Database Migrations
	// -----------------------------------------------------------------------
	runMigrations(cfg.DatabaseURL, logger)

	// -----------------------------------------------------------------------
	// PostgreSQL Connection Pool (Supabase)
	// -----------------------------------------------------------------------
	ctx := context.Background()

	poolConfig, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		logger.Error("failed to parse database URL", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Production-grade pool settings
	poolConfig.MaxConns = 25
	poolConfig.MinConns = 5
	poolConfig.MaxConnLifetime = 30 * time.Minute
	poolConfig.MaxConnIdleTime = 5 * time.Minute
	poolConfig.HealthCheckPeriod = 1 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		logger.Error("failed to create database pool", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer pool.Close()

	// Verify connectivity
	if err := pool.Ping(ctx); err != nil {
		logger.Error("failed to ping database", slog.String("error", err.Error()))
		os.Exit(1)
	}
	logger.Info("connected to PostgreSQL (Supabase)",
		slog.Int("max_conns", int(poolConfig.MaxConns)),
	)

	// -----------------------------------------------------------------------
	// Redis Client
	// -----------------------------------------------------------------------
	redisOpt, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		logger.Error("failed to parse Redis URL", slog.String("error", err.Error()))
		os.Exit(1)
	}

	redisOpt.DialTimeout = 5 * time.Second
	redisOpt.ReadTimeout = 3 * time.Second
	redisOpt.WriteTimeout = 3 * time.Second
	redisOpt.PoolSize = 20
	redisOpt.MinIdleConns = 5

	rdb := redis.NewClient(redisOpt)
	defer rdb.Close()

	if err := rdb.Ping(ctx).Err(); err != nil {
		logger.Error("failed to ping Redis", slog.String("error", err.Error()))
		os.Exit(1)
	}
	logger.Info("connected to Redis")

	// -----------------------------------------------------------------------
	// RabbitMQ Connection & Channel
	// -----------------------------------------------------------------------
	amqpConn, err := amqp.Dial(cfg.RabbitMQURL)
	if err != nil {
		logger.Error("failed to connect to RabbitMQ", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer amqpConn.Close()

	amqpChan, err := amqpConn.Channel()
	if err != nil {
		logger.Error("failed to open RabbitMQ channel", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer amqpChan.Close()

	// Declare the exchange for ledger events
	err = amqpChan.ExchangeDeclare(
		"ledger.events", // name
		"topic",         // type
		true,            // durable
		false,           // auto-deleted
		false,           // internal
		false,           // no-wait
		nil,             // arguments
	)
	if err != nil {
		logger.Error("failed to declare RabbitMQ exchange", slog.String("error", err.Error()))
		os.Exit(1)
	}
	logger.Info("connected to RabbitMQ", slog.String("exchange", "ledger.events"))

	// -----------------------------------------------------------------------
	// Storage Service (Cloudflare R2)
	// -----------------------------------------------------------------------
	storageCfg := service.StorageConfig{
		AccountID:       cfg.R2AccountID,
		AccessKeyID:     cfg.R2AccessKeyID,
		SecretAccessKey: cfg.R2SecretAccessKey,
		BucketName:      cfg.R2BucketName,
		Endpoint:        cfg.R2Endpoint,
	}
	storageService, err := service.NewStorageService(ctx, storageCfg, logger)
	if err != nil {
		logger.Error("failed to initialize storage service", slog.String("error", err.Error()))
		os.Exit(1)
	}
	logger.Info("connected to Cloudflare R2", slog.String("bucket", cfg.R2BucketName))

	// -----------------------------------------------------------------------
	// Service Layer
	// -----------------------------------------------------------------------
	idempotencyTTL, err := time.ParseDuration(cfg.IdempotencyTTL)
	if err != nil {
		idempotencyTTL = 24 * time.Hour
	}

	// Service no longer takes RabbitMQ channel — events go through outbox
	ledgerService := service.NewLedgerService(pool, rdb, logger, storageService, idempotencyTTL)

	// -----------------------------------------------------------------------
	// Authentication Service (Session Storage)
	// -----------------------------------------------------------------------
	authService := service.NewAuthService(rdb, 24*time.Hour)

	// -----------------------------------------------------------------------
	// Outbox Worker (Background Event Publisher)
	// -----------------------------------------------------------------------
	outboxCfg := service.DefaultOutboxWorkerConfig()
	outboxWorker := service.NewOutboxWorker(
		db.New(pool),
		amqpChan,
		logger,
		outboxCfg,
	)
	outboxWorker.Start(ctx)

	// -----------------------------------------------------------------------
	// API Key Configuration
	// -----------------------------------------------------------------------
	apiKeys := loadAPIKeys()

	// -----------------------------------------------------------------------
	// Rate Limit Configuration
	// -----------------------------------------------------------------------
	rateLimitCfg := &api.RateLimitConfig{
		RequestsPerWindow: getEnvInt("RATE_LIMIT_REQUESTS", 100),
		WindowDuration:    getEnvDuration("RATE_LIMIT_WINDOW", 1*time.Minute),
	}

	// -----------------------------------------------------------------------
	// HTTP Router & Server
	// -----------------------------------------------------------------------
	routerCfg := api.RouterConfig{
		APIKeys:         apiKeys,
		RateLimitConfig: rateLimitCfg,
		CORSConfig:      nil, // use defaults
		RequestTimeout:  cfg.WriteTimeout,
	}

	router := api.NewRouter(ledgerService, authService, rdb, logger, routerCfg)

	server := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.ServerPort),
		Handler:      router,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  120 * time.Second,
	}

	// -----------------------------------------------------------------------
	// Graceful Shutdown
	// -----------------------------------------------------------------------
	// Start server in a goroutine
	serverErrors := make(chan error, 1)
	go func() {
		logger.Info("HTTP server listening",
			slog.String("addr", server.Addr),
		)
		serverErrors <- server.ListenAndServe()
	}()

	// Wait for interrupt signal or server error
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		logger.Error("server error", slog.String("error", err.Error()))

	case sig := <-shutdown:
		logger.Info("shutdown signal received",
			slog.String("signal", sig.String()),
		)

		// Give outstanding requests 30 seconds to complete
		shutdownCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		// Stop the outbox worker first (let it finish current batch)
		logger.Info("stopping outbox worker...")
		outboxWorker.Stop()

		if err := server.Shutdown(shutdownCtx); err != nil {
			logger.Error("graceful shutdown failed — forcing close",
				slog.String("error", err.Error()),
			)
			_ = server.Close()
		}

		logger.Info("HTTP server stopped")
	}

	// Deferred closes will run for pool, rdb, amqpChan, amqpConn
	logger.Info("LedgerPro shutdown complete")
}

// ---------------------------------------------------------------------------
// Migrations
// ---------------------------------------------------------------------------

func runMigrations(dbURL string, logger *slog.Logger) {
	d, err := iofs.New(migrations.FS, ".")
	if err != nil {
		logger.Error("failed to load embedded migrations", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// We need to parse the dbURL since golang-migrate might expect a slightly different format,
	// but generally standard postgres:// URLs work out of the box with the postgres driver.
	m, err := migrate.NewWithSourceInstance("iofs", d, dbURL)
	if err != nil {
		logger.Error("failed to initialize database migrations", slog.String("error", err.Error()))
		os.Exit(1)
	}

	logger.Info("running database migrations...")
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		logger.Error("failed to apply migrations", slog.String("error", err.Error()))
		os.Exit(1)
	}
	logger.Info("database migrations applied successfully (or already up to date)")
}

// ---------------------------------------------------------------------------
// Configuration
// ---------------------------------------------------------------------------

type config struct {
	DatabaseURL    string
	RedisURL       string
	RabbitMQURL    string
	ServerPort     string
	ReadTimeout    time.Duration
	WriteTimeout   time.Duration
	IdempotencyTTL string
	LogLevel       string

	// R2 Storage
	R2AccountID       string
	R2AccessKeyID     string
	R2SecretAccessKey string
	R2BucketName      string
	R2Endpoint        string
}

func loadConfig() config {
	readTimeout, _ := time.ParseDuration(getEnv("SERVER_READ_TIMEOUT", "10s"))
	writeTimeout, _ := time.ParseDuration(getEnv("SERVER_WRITE_TIMEOUT", "30s"))

	return config{
		DatabaseURL:    getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/ledger_pro?sslmode=disable"),
		RedisURL:       getEnv("REDIS_URL", "redis://localhost:6379/0"),
		RabbitMQURL:    getEnv("RABBITMQ_URL", "amqp://ledger:ledger_secret@localhost:5672/"),
		ServerPort:     getEnv("SERVER_PORT", "8080"),
		ReadTimeout:    readTimeout,
		WriteTimeout:   writeTimeout,
		IdempotencyTTL: getEnv("IDEMPOTENCY_TTL", "24h"),
		LogLevel:       getEnv("LOG_LEVEL", "info"),

		R2AccountID:       getEnv("R2_ACCOUNT_ID", ""),
		R2AccessKeyID:     getEnv("R2_ACCESS_KEY_ID", ""),
		R2SecretAccessKey: getEnv("R2_SECRET_ACCESS_KEY", ""),
		R2BucketName:      getEnv("R2_BUCKET_NAME", ""),
		R2Endpoint:        getEnv("R2_ENDPOINT", ""),
	}
}

// loadAPIKeys loads API keys from the API_KEYS environment variable.
// Format: "key1:role1,key2:role2" (e.g., "sk-abc123:admin,sk-xyz789:read")
// If empty, authentication is disabled (dev mode).
func loadAPIKeys() map[string]api.APIKeyEntry {
	raw := os.Getenv("API_KEYS")
	if raw == "" {
		return nil
	}

	keys := make(map[string]api.APIKeyEntry)
	pairs := strings.Split(raw, ",")
	for _, pair := range pairs {
		parts := strings.SplitN(strings.TrimSpace(pair), ":", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			role := strings.TrimSpace(parts[1])
			if key != "" && role != "" {
				keys[key] = api.APIKeyEntry{Role: role}
			}
		}
	}

	return keys
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	var n int
	fmt.Sscanf(v, "%d", &n)
	if n <= 0 {
		return fallback
	}
	return n
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	return d
}

func parseLogLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
