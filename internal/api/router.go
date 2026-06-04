package api

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/redis/go-redis/v9"

	"ledger_pro/internal/service"
)

// RouterConfig holds configuration for the HTTP router and its middleware.
type RouterConfig struct {
	// APIKeys is a map of API key string → role entry for authentication.
	// If empty, authentication middleware is skipped (dev mode).
	APIKeys map[string]APIKeyEntry

	// RateLimitConfig controls the rate limiter. If nil, rate limiting is disabled.
	RateLimitConfig *RateLimitConfig

	// CORSConfig controls CORS. If nil, defaults are used.
	CORSConfig *CORSConfig

	// RequestTimeout is the maximum time a request handler may run.
	RequestTimeout time.Duration
}

// DefaultRouterConfig returns sensible defaults for development.
func DefaultRouterConfig() RouterConfig {
	return RouterConfig{
		APIKeys: nil, // no auth in dev
		RateLimitConfig: &RateLimitConfig{
			RequestsPerWindow: 100,
			WindowDuration:    1 * time.Minute,
		},
		RequestTimeout: 30 * time.Second,
	}
}

// NewRouter constructs the go-chi router with the full FAANG-grade middleware
// stack and all route registrations.
func NewRouter(
	svc *service.LedgerService,
	authSvc *service.AuthService,
	rdb *redis.Client,
	logger *slog.Logger,
	cfg RouterConfig,
) http.Handler {
	r := chi.NewRouter()

	// ---------------------------------------------------------------------------
	// Global Middleware Stack (order matters)
	// ---------------------------------------------------------------------------

	// 1. Recovery from panics → 500
	r.Use(middleware.Recoverer)

	// 2. Real IP extraction from X-Forwarded-For / X-Real-IP
	r.Use(middleware.RealIP)

	// 3. Chi's built-in request ID
	r.Use(middleware.RequestID)

	// 4. CORS handling
	if cfg.CORSConfig != nil {
		r.Use(CORSMiddleware(*cfg.CORSConfig))
	} else {
		r.Use(CORSMiddleware(DefaultCORSConfig()))
	}

	// 5. Distributed Trace ID injection
	r.Use(TraceIDMiddleware)

	// 6. Structured JSON logging
	r.Use(StructuredLoggerMiddleware(logger))

	// 7. Request timeout enforcement
	if cfg.RequestTimeout > 0 {
		r.Use(RequestTimeoutMiddleware(cfg.RequestTimeout))
	}

	// 8. Rate limiting (per-client via Redis)
	if cfg.RateLimitConfig != nil {
		r.Use(RateLimitMiddleware(rdb, *cfg.RateLimitConfig, logger))
	}

	// 9. API Key Authentication (skipped if no keys configured)
	publicPaths := map[string]bool{
		"/v1/health":     true,
		"/v1/metrics":    true,
		"/v1/auth/login": true,
	}
	if len(cfg.APIKeys) > 0 {
		r.Use(APIKeyAuthMiddleware(cfg.APIKeys, publicPaths, logger, authSvc))
	}

	// ---------------------------------------------------------------------------
	// Handler
	// ---------------------------------------------------------------------------
	h := NewHandler(svc, authSvc, logger)

	// ---------------------------------------------------------------------------
	// Routes
	// ---------------------------------------------------------------------------

	// Health & observability (public — no auth required)
	r.Get("/v1/health", h.HealthCheck)

	// Auth routes
	r.Route("/v1/auth", func(r chi.Router) {
		r.Post("/login", h.LoginHandler(cfg.APIKeys))
		r.Post("/logout", h.LogoutHandler)
		r.Get("/me", h.MeHandler)
	})

	// Account routes
	r.Route("/v1/accounts", func(r chi.Router) {
		// Write operations require admin or write role
		r.Group(func(r chi.Router) {
			if len(cfg.APIKeys) > 0 {
				r.Use(RequireRole("admin", "write"))
			}
			r.Post("/", h.CreateAccount)
		})

		// Read operations require any authenticated role
		r.Group(func(r chi.Router) {
			if len(cfg.APIKeys) > 0 {
				r.Use(RequireRole("admin", "write", "read"))
			}
			r.Get("/", h.ListAccounts)
			r.Get("/{id}", h.GetAccount)
			r.Get("/{id}/balance", h.GetAccountBalance)
		})
	})

	// Transaction routes
	r.Route("/v1/transactions", func(r chi.Router) {
		// Write operations require admin or write role
		r.Group(func(r chi.Router) {
			if len(cfg.APIKeys) > 0 {
				r.Use(RequireRole("admin", "write"))
			}
			r.Post("/", h.CreateTransaction)
			r.Post("/{id}/documents", h.AttachDocumentHandler)
		})

		// Read operations require any authenticated role
		r.Group(func(r chi.Router) {
			if len(cfg.APIKeys) > 0 {
				r.Use(RequireRole("admin", "write", "read"))
			}
			r.Get("/", h.ListTransactions)
			r.Get("/{id}", h.GetTransaction)
			r.Get("/{id}/documents", h.ListTransactionDocumentsHandler)
		})
	})

	return r
}
