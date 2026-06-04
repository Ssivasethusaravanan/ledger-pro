package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"ledger_pro/internal/service"
)

// ---------------------------------------------------------------------------
// Handler
// ---------------------------------------------------------------------------

// Handler holds the HTTP handler methods and their service dependency.
type Handler struct {
	svc    *service.LedgerService
	auth   *service.AuthService
	pool   *pgxpool.Pool
	rdb    *redis.Client
	logger *slog.Logger
}

// NewHandler constructs a new Handler with the given services and logger.
func NewHandler(svc *service.LedgerService, auth *service.AuthService, logger *slog.Logger) *Handler {
	return &Handler{
		svc:    svc,
		auth:   auth,
		pool:   svc.Pool(),
		rdb:    svc.RedisClient(),
		logger: logger,
	}
}

// ---------------------------------------------------------------------------
// JSON Helpers
// ---------------------------------------------------------------------------

// writeJSON writes a JSON response with the given status code.
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if data != nil {
		_ = json.NewEncoder(w).Encode(data)
	}
}

// ---------------------------------------------------------------------------
// Deep Health Check
// ---------------------------------------------------------------------------

// DependencyHealth represents the health status of a single dependency.
type DependencyHealth struct {
	Status  string `json:"status"`
	Latency string `json:"latency"`
	Error   string `json:"error,omitempty"`
}

// HealthResponse is the full health check response.
type HealthResponse struct {
	Status       string                      `json:"status"`
	Service      string                      `json:"service"`
	Version      string                      `json:"version"`
	Uptime       string                      `json:"uptime"`
	Dependencies map[string]DependencyHealth `json:"dependencies"`
}

var startTime = time.Now()

// HealthCheck performs deep diagnostic checks on all dependencies.
// Returns 200 if all dependencies are healthy, 503 if any critical one is down.
func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	deps := make(map[string]DependencyHealth)
	allHealthy := true

	// PostgreSQL health check
	pgStart := time.Now()
	pgErr := h.pool.Ping(ctx)
	pgLatency := time.Since(pgStart)
	if pgErr != nil {
		deps["postgresql"] = DependencyHealth{
			Status:  "unhealthy",
			Latency: pgLatency.String(),
			Error:   pgErr.Error(),
		}
		allHealthy = false
	} else {
		stat := h.pool.Stat()
		deps["postgresql"] = DependencyHealth{
			Status:  "healthy",
			Latency: pgLatency.String(),
		}
		// Add pool stats as a separate entry
		deps["postgresql_pool"] = DependencyHealth{
			Status: "info",
			Latency: fmt.Sprintf("total=%d acquired=%d idle=%d",
				stat.TotalConns(), stat.AcquiredConns(), stat.IdleConns()),
		}
	}

	// Redis health check
	redisStart := time.Now()
	redisErr := h.rdb.Ping(ctx).Err()
	redisLatency := time.Since(redisStart)
	if redisErr != nil {
		deps["redis"] = DependencyHealth{
			Status:  "unhealthy",
			Latency: redisLatency.String(),
			Error:   redisErr.Error(),
		}
		allHealthy = false
	} else {
		deps["redis"] = DependencyHealth{
			Status:  "healthy",
			Latency: redisLatency.String(),
		}
	}

	status := "healthy"
	httpStatus := http.StatusOK
	if !allHealthy {
		status = "degraded"
		httpStatus = http.StatusServiceUnavailable
	}

	writeJSON(w, httpStatus, HealthResponse{
		Status:       status,
		Service:      "ledger_pro",
		Version:      "2.0.0",
		Uptime:       time.Since(startTime).Round(time.Second).String(),
		Dependencies: deps,
	})
}

// ---------------------------------------------------------------------------
// Account Handlers
// ---------------------------------------------------------------------------

// CreateAccount handles POST /v1/accounts
func (h *Handler) CreateAccount(w http.ResponseWriter, r *http.Request) {
	var req service.CreateAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteProblem(w, r, BadRequest("invalid request body: "+err.Error()))
		return
	}

	// Default currency
	if req.Currency == "" {
		req.Currency = "INR"
	}

	// Validate
	if vr := ValidateCreateAccountRequest(req); vr.HasErrors() {
		WriteProblem(w, r, vr.ToProblem())
		return
	}

	account, err := h.svc.CreateAccount(r.Context(), req)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "create account failed", slog.String("error", err.Error()))
		WriteProblem(w, r, InternalError("failed to create account"))
		return
	}

	writeJSON(w, http.StatusCreated, account)
}

// GetAccount handles GET /v1/accounts/{id}
func (h *Handler) GetAccount(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		WriteProblem(w, r, BadRequest("invalid account id — must be an integer"))
		return
	}

	account, err := h.svc.GetAccount(r.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrAccountNotFound) {
			WriteProblem(w, r, AccountNotFound(fmt.Sprintf("account with id %d does not exist", id)))
			return
		}
		WriteProblem(w, r, InternalError("failed to get account"))
		return
	}

	writeJSON(w, http.StatusOK, account)
}

// ListAccounts handles GET /v1/accounts
func (h *Handler) ListAccounts(w http.ResponseWriter, r *http.Request) {
	cursor, _ := strconv.ParseInt(r.URL.Query().Get("cursor"), 10, 64)
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		WriteProblem(w, r, BadRequest("page_size must not exceed 100"))
		return
	}

	accounts, err := h.svc.ListAccounts(r.Context(), cursor, int32(pageSize))
	if err != nil {
		WriteProblem(w, r, InternalError("failed to list accounts"))
		return
	}

	// Build next cursor
	var nextCursor int64
	if len(accounts) > 0 {
		nextCursor = accounts[len(accounts)-1].ID
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data":        accounts,
		"next_cursor": nextCursor,
		"page_size":   pageSize,
	})
}

// GetAccountBalance handles GET /v1/accounts/{id}/balance
func (h *Handler) GetAccountBalance(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		WriteProblem(w, r, BadRequest("invalid account id — must be an integer"))
		return
	}

	balance, err := h.svc.GetAccountBalance(r.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrAccountNotFound) {
			WriteProblem(w, r, AccountNotFound(fmt.Sprintf("account with id %d does not exist", id)))
			return
		}
		WriteProblem(w, r, InternalError("failed to get balance"))
		return
	}

	writeJSON(w, http.StatusOK, balance)
}

// ---------------------------------------------------------------------------
// Transaction Handlers
// ---------------------------------------------------------------------------

// CreateTransaction handles POST /v1/transactions
// Requires an Idempotency-Key header.
func (h *Handler) CreateTransaction(w http.ResponseWriter, r *http.Request) {
	// Extract idempotency key from header
	idempotencyKey := r.Header.Get("Idempotency-Key")

	var req service.CreateTransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteProblem(w, r, BadRequest("invalid request body: "+err.Error()))
		return
	}

	// Override with header value (header is the source of truth)
	req.IdempotencyKey = idempotencyKey

	// Validate
	if vr := ValidateCreateTransactionRequest(req); vr.HasErrors() {
		WriteProblem(w, r, vr.ToProblem())
		return
	}

	txnResponse, err := h.svc.CreateTransaction(r.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrIdempotencyKeyRequired):
			WriteProblem(w, r, BadRequest(err.Error()))
		case errors.Is(err, service.ErrIdempotencyKeyConflict):
			WriteProblem(w, r, DuplicateIdempotencyKey(err.Error()))
		case errors.Is(err, service.ErrUnbalancedTransaction):
			WriteProblem(w, r, UnbalancedTransaction(err.Error(),
				InvalidParam{Name: "postings", Reason: "debit and credit legs must balance mathematically"},
			))
		case errors.Is(err, service.ErrEmptyPostings):
			WriteProblem(w, r, BadRequest(err.Error()))
		case errors.Is(err, service.ErrInvalidAmount):
			WriteProblem(w, r, BadRequest(err.Error()))
		case errors.Is(err, service.ErrInvalidDirection):
			WriteProblem(w, r, BadRequest(err.Error()))
		case errors.Is(err, service.ErrLockAcquisitionFailed):
			WriteProblem(w, r, LockAcquisitionFailed("could not acquire distributed lock — the system is under load, please retry"))
		default:
			h.logger.ErrorContext(r.Context(), "create transaction failed",
				slog.String("idempotency_key", idempotencyKey),
				slog.String("error", err.Error()),
			)
			WriteProblem(w, r, InternalError("failed to create transaction"))
		}
		return
	}

	writeJSON(w, http.StatusCreated, txnResponse)
}

// GetTransaction handles GET /v1/transactions/{id}
func (h *Handler) GetTransaction(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		WriteProblem(w, r, BadRequest("invalid transaction id — must be a valid UUID"))
		return
	}

	txnResponse, err := h.svc.GetTransaction(r.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrTransactionNotFound) {
			WriteProblem(w, r, TransactionNotFound(fmt.Sprintf("transaction with id %s does not exist", idStr)))
			return
		}
		WriteProblem(w, r, InternalError("failed to get transaction"))
		return
	}

	writeJSON(w, http.StatusOK, txnResponse)
}

// ListTransactions handles GET /v1/transactions
func (h *Handler) ListTransactions(w http.ResponseWriter, r *http.Request) {
	cursorStr := r.URL.Query().Get("cursor")
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		WriteProblem(w, r, BadRequest("page_size must not exceed 100"))
		return
	}

	var cursorTime time.Time
	if cursorStr != "" {
		var err error
		cursorTime, err = time.Parse(time.RFC3339Nano, cursorStr)
		if err != nil {
			WriteProblem(w, r, BadRequest("invalid cursor — must be RFC3339 timestamp"))
			return
		}
	} else {
		cursorTime = time.Now().Add(time.Second) // future to get latest first
	}

	txns, err := h.svc.ListTransactions(r.Context(), cursorTime, int32(pageSize))
	if err != nil {
		WriteProblem(w, r, InternalError("failed to list transactions"))
		return
	}

	// Build next cursor from the last item's created_at
	var nextCursor string
	if len(txns) > 0 && txns[len(txns)-1].CreatedAt.Valid {
		nextCursor = txns[len(txns)-1].CreatedAt.Time.Format(time.RFC3339Nano)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data":        txns,
		"next_cursor": nextCursor,
		"page_size":   pageSize,
	})
}
