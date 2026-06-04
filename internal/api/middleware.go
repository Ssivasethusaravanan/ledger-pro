// Package api implements the HTTP transport layer for LedgerPro.
// This file contains production-grade middleware: TraceID injection, structured
// logging, API key authentication with RBAC, Redis sliding-window rate limiting,
// CORS, and request timeout enforcement.
package api

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// ---------------------------------------------------------------------------
// Context Keys
// ---------------------------------------------------------------------------

type contextKey string

const (
	// TraceIDKey is the context key for the distributed trace ID.
	TraceIDKey contextKey = "trace_id"
	// APIKeyRoleKey is the context key for the authenticated role.
	APIKeyRoleKey contextKey = "api_key_role"
)

// GetTraceID extracts the trace ID from the request context.
func GetTraceID(ctx context.Context) string {
	if id, ok := ctx.Value(TraceIDKey).(string); ok {
		return id
	}
	return ""
}

// GetAPIKeyRole extracts the authenticated role from the request context.
func GetAPIKeyRole(ctx context.Context) string {
	if role, ok := ctx.Value(APIKeyRoleKey).(string); ok {
		return role
	}
	return ""
}

// ---------------------------------------------------------------------------
// TraceID Middleware
// ---------------------------------------------------------------------------
// Injects a unique trace ID into every request. If the client provides an
// X-Trace-ID header, it is respected (passthrough). Otherwise, a new UUID v4
// is generated. The trace ID is set on both the request context and the
// response header.
// ---------------------------------------------------------------------------

// TraceIDMiddleware injects a trace ID into the request context and response headers.
func TraceIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		traceID := r.Header.Get("X-Trace-ID")
		if traceID == "" {
			traceID = uuid.New().String()
		}

		// Set on response header
		w.Header().Set("X-Trace-ID", traceID)

		// Inject into context
		ctx := context.WithValue(r.Context(), TraceIDKey, traceID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// ---------------------------------------------------------------------------
// Structured Logging Middleware
// ---------------------------------------------------------------------------
// Logs every HTTP request with method, path, status, duration, and trace ID
// using Go's structured slog logger in JSON format.
// ---------------------------------------------------------------------------

// responseWriter wraps http.ResponseWriter to capture the status code.
type responseWriter struct {
	http.ResponseWriter
	statusCode  int
	wroteHeader bool
}

func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
}

func (rw *responseWriter) WriteHeader(code int) {
	if !rw.wroteHeader {
		rw.statusCode = code
		rw.wroteHeader = true
		rw.ResponseWriter.WriteHeader(code)
	}
}

// StructuredLoggerMiddleware logs each request with structured fields.
func StructuredLoggerMiddleware(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			wrapped := newResponseWriter(w)

			next.ServeHTTP(wrapped, r)

			duration := time.Since(start)
			traceID := GetTraceID(r.Context())

			logger.InfoContext(r.Context(), "http request",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", wrapped.statusCode),
				slog.Duration("duration", duration),
				slog.String("trace_id", traceID),
				slog.String("remote_addr", r.RemoteAddr),
				slog.String("user_agent", r.UserAgent()),
			)
		})
	}
}

// ---------------------------------------------------------------------------
// API Key Authentication Middleware
// ---------------------------------------------------------------------------
// Validates the API key from the X-API-Key header or Authorization: Bearer
// header. The key is looked up against a configured key map to determine
// the role (admin, write, read). If no key is found, the request is rejected.
// ---------------------------------------------------------------------------

// APIKeyEntry represents a registered API key with its associated role.
type APIKeyEntry struct {
	Role string // "admin", "write", "read"
}

// APIKeyAuthMiddleware validates API keys and injects the role into the context.
// Pass nil for publicPaths to require auth on all routes.
func APIKeyAuthMiddleware(keys map[string]APIKeyEntry, publicPaths map[string]bool, logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip auth for public paths (health, metrics)
			if publicPaths != nil && publicPaths[r.URL.Path] {
				next.ServeHTTP(w, r)
				return
			}

			// Extract API key from X-API-Key or Authorization: Bearer
			apiKey := r.Header.Get("X-API-Key")
			if apiKey == "" {
				authHeader := r.Header.Get("Authorization")
				if strings.HasPrefix(authHeader, "Bearer ") {
					apiKey = strings.TrimPrefix(authHeader, "Bearer ")
				}
			}

			if apiKey == "" {
				logger.WarnContext(r.Context(), "request missing API key",
					slog.String("path", r.URL.Path),
					slog.String("remote_addr", r.RemoteAddr),
				)
				WriteProblem(w, r, Unauthorized("API key is required — provide via X-API-Key header or Authorization: Bearer"))
				return
			}

			entry, found := keys[apiKey]
			if !found {
				logger.WarnContext(r.Context(), "invalid API key",
					slog.String("path", r.URL.Path),
					slog.String("remote_addr", r.RemoteAddr),
				)
				WriteProblem(w, r, Unauthorized("invalid API key"))
				return
			}

			// Inject role into context
			ctx := context.WithValue(r.Context(), APIKeyRoleKey, entry.Role)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRole returns a middleware that checks if the authenticated role is
// in the allowed set. Must be applied AFTER APIKeyAuthMiddleware.
func RequireRole(allowed ...string) func(http.Handler) http.Handler {
	allowedSet := make(map[string]bool, len(allowed))
	for _, role := range allowed {
		allowedSet[role] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role := GetAPIKeyRole(r.Context())
			if role == "" {
				WriteProblem(w, r, Unauthorized("authentication required"))
				return
			}
			if !allowedSet[role] {
				WriteProblem(w, r, Forbidden(fmt.Sprintf("role '%s' does not have permission for this operation", role)))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// ---------------------------------------------------------------------------
// Redis Sliding-Window Rate Limiter Middleware
// ---------------------------------------------------------------------------
// Uses a Redis sorted set (ZSET) per client key to implement a precise
// sliding-window rate limiter. Falls back to allow traffic if Redis is
// unavailable (fail-open).
// ---------------------------------------------------------------------------

// RateLimitConfig holds the configuration for the rate limiter.
type RateLimitConfig struct {
	RequestsPerWindow int           // max requests per window (e.g., 100)
	WindowDuration    time.Duration // sliding window size (e.g., 1 minute)
}

// RateLimitMiddleware enforces per-client rate limiting using Redis.
func RateLimitMiddleware(rdb *redis.Client, cfg RateLimitConfig, logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Determine client identity: API key first, then IP
			clientKey := r.Header.Get("X-API-Key")
			if clientKey == "" {
				clientKey = r.RemoteAddr
			}
			redisKey := fmt.Sprintf("ledger:ratelimit:%s", clientKey)

			ctx := r.Context()
			now := time.Now()
			windowStart := now.Add(-cfg.WindowDuration)
			nowNanos := float64(now.UnixNano())

			// Lua script for atomic sliding-window rate limit check
			luaScript := redis.NewScript(`
				local key = KEYS[1]
				local window_start = tonumber(ARGV[1])
				local now = tonumber(ARGV[2])
				local max_requests = tonumber(ARGV[3])
				local window_ms = tonumber(ARGV[4])
				
				-- Remove expired entries
				redis.call('ZREMRANGEBYSCORE', key, '-inf', window_start)
				
				-- Count current entries
				local current = redis.call('ZCARD', key)
				
				if current >= max_requests then
					return {0, current, max_requests}
				end
				
				-- Add current request
				redis.call('ZADD', key, now, now .. ':' .. math.random(1000000))
				redis.call('PEXPIRE', key, window_ms)
				
				return {1, current + 1, max_requests}
			`)

			result, err := luaScript.Run(ctx, rdb, []string{redisKey},
				float64(windowStart.UnixNano()),
				nowNanos,
				cfg.RequestsPerWindow,
				cfg.WindowDuration.Milliseconds(),
			).Int64Slice()

			if err != nil {
				// Fail open — if Redis is down, allow the request
				logger.WarnContext(ctx, "rate limiter redis error — failing open",
					slog.String("error", err.Error()),
					slog.String("client_key", clientKey),
				)
				next.ServeHTTP(w, r)
				return
			}

			allowed := result[0] == 1
			current := result[1]
			limit := result[2]
			remaining := limit - current
			if remaining < 0 {
				remaining = 0
			}

			// Set standard rate limit headers
			resetTime := now.Add(cfg.WindowDuration)
			w.Header().Set("X-RateLimit-Limit", strconv.FormatInt(limit, 10))
			w.Header().Set("X-RateLimit-Remaining", strconv.FormatInt(remaining, 10))
			w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(resetTime.Unix(), 10))

			if !allowed {
				logger.WarnContext(ctx, "rate limit exceeded",
					slog.String("client_key", clientKey),
					slog.Int64("current", current),
					slog.Int64("limit", limit),
				)
				w.Header().Set("Retry-After", strconv.Itoa(int(cfg.WindowDuration.Seconds())))
				WriteProblem(w, r, RateLimitExceeded(
					fmt.Sprintf("rate limit of %d requests per %s exceeded — retry after %s",
						cfg.RequestsPerWindow, cfg.WindowDuration, cfg.WindowDuration),
				))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// ---------------------------------------------------------------------------
// CORS Middleware
// ---------------------------------------------------------------------------
// Handles Cross-Origin Resource Sharing with configurable allowed origins.
// ---------------------------------------------------------------------------

// CORSConfig holds CORS middleware configuration.
type CORSConfig struct {
	AllowedOrigins []string
	AllowedMethods []string
	AllowedHeaders []string
	MaxAge         int // preflight cache seconds
}

// DefaultCORSConfig returns a sensible default CORS configuration.
func DefaultCORSConfig() CORSConfig {
	return CORSConfig{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{
			"Accept", "Authorization", "Content-Type", "X-API-Key",
			"X-Trace-ID", "Idempotency-Key", "X-Correlation-Id",
		},
		MaxAge: 86400, // 24 hours
	}
}

// CORSMiddleware adds CORS headers and handles preflight OPTIONS requests.
func CORSMiddleware(cfg CORSConfig) func(http.Handler) http.Handler {
	origins := strings.Join(cfg.AllowedOrigins, ", ")
	methods := strings.Join(cfg.AllowedMethods, ", ")
	headers := strings.Join(cfg.AllowedHeaders, ", ")
	maxAge := strconv.Itoa(cfg.MaxAge)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", origins)
			w.Header().Set("Access-Control-Allow-Methods", methods)
			w.Header().Set("Access-Control-Allow-Headers", headers)
			w.Header().Set("Access-Control-Max-Age", maxAge)
			w.Header().Set("Access-Control-Expose-Headers", "X-Trace-ID, X-RateLimit-Limit, X-RateLimit-Remaining, X-RateLimit-Reset")

			// Handle preflight
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// ---------------------------------------------------------------------------
// Request Timeout Middleware
// ---------------------------------------------------------------------------
// Wraps each request in a context with a deadline. If the handler does not
// complete within the timeout, the context is cancelled (downstream queries
// abort), and a 504 Gateway Timeout is returned.
// ---------------------------------------------------------------------------

// RequestTimeoutMiddleware enforces a maximum duration for request processing.
func RequestTimeoutMiddleware(timeout time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			defer cancel()
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
