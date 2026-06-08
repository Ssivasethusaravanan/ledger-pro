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

// Swagger DTOs (used only for API documentation generation)
type swaggAccount struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Currency    string `json:"currency"`
	CreatedAt   string `json:"created_at"`
}

type swaggTransaction struct {
	ID             string `json:"id"`
	IdempotencyKey string `json:"idempotency_key"`
	Description    string `json:"description"`
	CreatedAt      string `json:"created_at"`
}

var startTime = time.Now()

// HealthCheck provides a deep health check of all dependencies.
// @Summary Health Check
// @Description Checks connectivity to Postgres, Redis, RabbitMQ, and R2.
// @Tags Observability
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 503 {object} ProblemDetail
// @Router /v1/health [get]
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

// CreateAccount handles the creation of a new ledger account.
// @Summary Create Account
// @Description Creates a new double-entry ledger account
// @Tags Accounts
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param request body service.CreateAccountRequest true "Account details"
// @Success 201 {object} swaggAccount
// @Failure 400 {object} ProblemDetail
// @Failure 401 {object} ProblemDetail
// @Router /v1/accounts [post]
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

	account, err := h.svc.CreateAccount(r.Context(), GetTenantID(r.Context()), req)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "create account failed", slog.String("error", err.Error()))
		WriteProblem(w, r, InternalError("failed to create account"))
		return
	}

	writeJSON(w, http.StatusCreated, account)
}

// GetAccount retrieves a single account by ID.
// @Summary Get Account
// @Description Retrieves an account by its ID
// @Tags Accounts
// @Produce json
// @Security ApiKeyAuth
// @Param id path string true "Account ID"
// @Success 200 {object} swaggAccount
// @Failure 401 {object} ProblemDetail
// @Failure 404 {object} ProblemDetail
// @Router /v1/accounts/{id} [get]
func (h *Handler) GetAccount(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		WriteProblem(w, r, BadRequest("invalid account id — must be an integer"))
		return
	}

	account, err := h.svc.GetAccount(r.Context(), GetTenantID(r.Context()), id)
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

// ListAccounts returns a paginated list of all accounts.
// @Summary List Accounts
// @Description Retrieves a paginated list of accounts
// @Tags Accounts
// @Produce json
// @Security ApiKeyAuth
// @Param cursor query string false "Pagination cursor"
// @Param limit query int false "Number of items to return" default(50)
// @Success 200 {array} swaggAccount
// @Failure 401 {object} ProblemDetail
// @Router /v1/accounts [get]
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

	accounts, err := h.svc.ListAccounts(r.Context(), GetTenantID(r.Context()), cursor, int32(pageSize))
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

// GetAccountBalance computes the current balance of an account from its postings.
// @Summary Get Account Balance
// @Description Computes the real-time balance of an account
// @Tags Accounts
// @Produce json
// @Security ApiKeyAuth
// @Param id path string true "Account ID"
// @Success 200 {object} service.BalanceResponse
// @Failure 401 {object} ProblemDetail
// @Failure 404 {object} ProblemDetail
// @Router /v1/accounts/{id}/balance [get]
func (h *Handler) GetAccountBalance(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		WriteProblem(w, r, BadRequest("invalid account id — must be an integer"))
		return
	}

	balance, err := h.svc.GetAccountBalance(r.Context(), GetTenantID(r.Context()), id)
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

// GetAccountPostings retrieves a paginated list of postings for an account.
// @Summary Get Account Postings
// @Description Retrieves postings for an account
// @Tags Accounts
// @Produce json
// @Security ApiKeyAuth
// @Param id path string true "Account ID"
// @Param offset query int false "Pagination offset" default(0)
// @Param limit query int false "Number of items to return" default(50)
// @Success 200 {array} db.GetPostingsByAccountIDRow
// @Failure 401 {object} ProblemDetail
// @Router /v1/accounts/{id}/postings [get]
func (h *Handler) GetAccountPostings(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		WriteProblem(w, r, BadRequest("invalid account id — must be an integer"))
		return
	}

	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		WriteProblem(w, r, BadRequest("page_size must not exceed 100"))
		return
	}

	postings, err := h.svc.ListAccountPostings(r.Context(), GetTenantID(r.Context()), id, int32(offset), int32(pageSize))
	if err != nil {
		WriteProblem(w, r, InternalError("failed to list account postings"))
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data":      postings,
		"offset":    offset + len(postings),
		"page_size": pageSize,
	})
}

// ---------------------------------------------------------------------------
// Transaction Handlers
// ---------------------------------------------------------------------------

// CreateTransaction processes a new double-entry transaction atomically.
// @Summary Create Transaction
// @Description Processes a double-entry transaction
// @Tags Transactions
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param Idempotency-Key header string true "Idempotency Key (UUID)"
// @Param request body service.CreateTransactionRequest true "Transaction details"
// @Success 201 {object} swaggTransaction
// @Failure 400 {object} ProblemDetail
// @Failure 401 {object} ProblemDetail
// @Failure 409 {object} ProblemDetail
// @Router /v1/transactions [post]
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

	txnResponse, err := h.svc.CreateTransaction(r.Context(), GetTenantID(r.Context()), req)
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

// GetTransaction retrieves a single transaction by ID.
// @Summary Get Transaction
// @Description Retrieves a transaction by its ID
// @Tags Transactions
// @Produce json
// @Security ApiKeyAuth
// @Param id path string true "Transaction ID"
// @Success 200 {object} swaggTransaction
// @Failure 401 {object} ProblemDetail
// @Failure 404 {object} ProblemDetail
// @Router /v1/transactions/{id} [get]
func (h *Handler) GetTransaction(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		WriteProblem(w, r, BadRequest("invalid transaction id — must be a valid UUID"))
		return
	}

	txnResponse, err := h.svc.GetTransaction(r.Context(), GetTenantID(r.Context()), id)
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

// ListTransactions returns a paginated list of transactions.
// @Summary List Transactions
// @Description Retrieves a paginated list of transactions
// @Tags Transactions
// @Produce json
// @Security ApiKeyAuth
// @Param cursor query string false "Pagination cursor"
// @Param limit query int false "Number of items to return" default(50)
// @Success 200 {array} swaggTransaction
// @Failure 401 {object} ProblemDetail
// @Router /v1/transactions [get]
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

	txns, err := h.svc.ListTransactions(r.Context(), GetTenantID(r.Context()), cursorTime, int32(pageSize))
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
