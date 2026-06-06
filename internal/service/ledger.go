// Package service implements the core business logic for the LedgerPro
// financial ledger. It orchestrates distributed locking, database transactions,
// idempotency checks, and transactional outbox writes for guaranteed event
// delivery via a double-entry bookkeeping model.
package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/bsm/redislock"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"ledger_pro/internal/db"
)

// ---------------------------------------------------------------------------
// Error Sentinels
// ---------------------------------------------------------------------------

var (
	ErrIdempotencyKeyRequired = errors.New("idempotency key is required")
	ErrIdempotencyKeyConflict = errors.New("a transaction with this idempotency key already exists")
	ErrUnbalancedTransaction  = errors.New("sum of debits must equal sum of credits")
	ErrEmptyPostings          = errors.New("transaction must have at least two postings")
	ErrInvalidAmount          = errors.New("posting amount must be positive")
	ErrAccountNotFound        = errors.New("account not found")
	ErrTransactionNotFound    = errors.New("transaction not found")
	ErrLockAcquisitionFailed  = errors.New("failed to acquire distributed lock — try again")
	ErrInvalidDirection       = errors.New("invalid posting direction")
)

// ---------------------------------------------------------------------------
// Request/Response DTOs
// ---------------------------------------------------------------------------

// CreateAccountRequest holds the parameters for creating a new ledger account.
type CreateAccountRequest struct {
	AccountName string          `json:"account_name"`
	AccountType db.AccountType  `json:"account_type"`
	Currency    string          `json:"currency"`
	Metadata    json.RawMessage `json:"metadata,omitempty"`
}

// PostingEntry represents a single debit or credit leg in a transaction request.
type PostingEntry struct {
	AccountID int64              `json:"account_id"`
	Amount    int64              `json:"amount"`
	Direction db.PostingDirection `json:"direction"`
}

// CreateTransactionRequest holds the parameters for creating a new double-entry transaction.
type CreateTransactionRequest struct {
	IdempotencyKey string          `json:"idempotency_key"`
	Description    string          `json:"description"`
	Metadata       json.RawMessage `json:"metadata,omitempty"`
	Postings       []PostingEntry  `json:"postings"`
}

// TransactionResponse is the enriched response returned after creating a transaction.
type TransactionResponse struct {
	ID             uuid.UUID                          `json:"id"`
	IdempotencyKey string                             `json:"idempotency_key"`
	Description    string                             `json:"description"`
	Metadata       json.RawMessage                    `json:"metadata"`
	Postings       []db.GetPostingsByTransactionIDRow  `json:"postings"`
	CreatedAt      pgtype.Timestamptz                 `json:"created_at"`
}

// BalanceResponse holds the balance breakdown for a single account.
type BalanceResponse struct {
	AccountID    int64  `json:"account_id"`
	TotalDebits  int64  `json:"total_debits"`
	TotalCredits int64  `json:"total_credits"`
	NetBalance   int64  `json:"net_balance"`
	Currency     string `json:"currency"`
}

// DocumentResponse represents an attached document.
type DocumentResponse struct {
	ID            uuid.UUID          `json:"id"`
	TransactionID uuid.UUID          `json:"transaction_id"`
	Filename      string             `json:"filename"`
	ContentType   string             `json:"content_type"`
	SizeBytes     int64              `json:"size_bytes"`
	DownloadURL   string             `json:"download_url"`
	CreatedAt     pgtype.Timestamptz `json:"created_at"`
}

// ---------------------------------------------------------------------------
// Outbox Event Payloads
// ---------------------------------------------------------------------------

// TransactionCreatedEvent is published to the message broker after a successful transaction.
type TransactionCreatedEvent struct {
	EventID        string          `json:"event_id"`
	EventType      string          `json:"event_type"`
	TransactionID  uuid.UUID       `json:"transaction_id"`
	IdempotencyKey string          `json:"idempotency_key"`
	Description    string          `json:"description"`
	Postings       []PostingEntry  `json:"postings"`
	Metadata       json.RawMessage `json:"metadata"`
	CreatedAt      time.Time       `json:"created_at"`
}

// ---------------------------------------------------------------------------
// LedgerService
// ---------------------------------------------------------------------------

// LedgerService encapsulates the business logic for the financial ledger.
// It owns the database pool, Redis client, distributed lock client, and
// writes events to the transactional outbox instead of directly to RabbitMQ.
type LedgerService struct {
	pool           *pgxpool.Pool
	queries        *db.Queries
	rdb            *redis.Client
	locker         *redislock.Client
	logger         *slog.Logger
	storageService *StorageService

	// Configuration
	idempotencyTTL time.Duration
}

// NewLedgerService constructs a LedgerService with all dependencies injected.
// Note: RabbitMQ channel is no longer held here — events go through the outbox.
func NewLedgerService(
	pool *pgxpool.Pool,
	rdb *redis.Client,
	logger *slog.Logger,
	storageService *StorageService,
	idempotencyTTL time.Duration,
) *LedgerService {
	return &LedgerService{
		pool:           pool,
		queries:        db.New(pool),
		rdb:            rdb,
		locker:         redislock.New(rdb),
		logger:         logger,
		storageService: storageService,
		idempotencyTTL: idempotencyTTL,
	}
}

// Queries returns the underlying SQLC queries object (used by outbox worker).
func (s *LedgerService) Queries() *db.Queries {
	return s.queries
}

// Pool returns the underlying pgxpool (used for health checks).
func (s *LedgerService) Pool() *pgxpool.Pool {
	return s.pool
}

// RedisClient returns the underlying Redis client (used for health checks).
func (s *LedgerService) RedisClient() *redis.Client {
	return s.rdb
}

// ---------------------------------------------------------------------------
// Account Operations
// ---------------------------------------------------------------------------

// CreateAccount creates a new ledger account.
func (s *LedgerService) CreateAccount(ctx context.Context, req CreateAccountRequest) (db.Account, error) {
	metadata := req.Metadata
	if metadata == nil {
		metadata = json.RawMessage(`{}`)
	}

	account, err := s.queries.CreateAccount(ctx, db.CreateAccountParams{
		AccountName: req.AccountName,
		AccountType: req.AccountType,
		Currency:    req.Currency,
		Metadata:    metadata,
	})
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to create account",
			slog.String("account_name", req.AccountName),
			slog.String("error", err.Error()),
		)
		return db.Account{}, fmt.Errorf("create account: %w", err)
	}

	s.logger.InfoContext(ctx, "account created",
		slog.Int64("account_id", account.ID),
		slog.String("account_name", account.AccountName),
	)
	return account, nil
}

// GetAccount retrieves an account by its ID.
func (s *LedgerService) GetAccount(ctx context.Context, id int64) (db.Account, error) {
	account, err := s.queries.GetAccountByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.Account{}, ErrAccountNotFound
		}
		return db.Account{}, fmt.Errorf("get account: %w", err)
	}
	return account, nil
}

// ListAccounts retrieves a paginated list of accounts.
func (s *LedgerService) ListAccounts(ctx context.Context, cursor int64, pageSize int32) ([]db.Account, error) {
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	accounts, err := s.queries.ListAccounts(ctx, db.ListAccountsParams{
		Cursor:   cursor,
		PageSize: pageSize,
	})
	if err != nil {
		return nil, fmt.Errorf("list accounts: %w", err)
	}
	return accounts, nil
}

// GetAccountBalance computes the real-time balance for an account.
func (s *LedgerService) GetAccountBalance(ctx context.Context, accountID int64) (BalanceResponse, error) {
	// First verify the account exists
	account, err := s.queries.GetAccountByID(ctx, accountID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return BalanceResponse{}, ErrAccountNotFound
		}
		return BalanceResponse{}, fmt.Errorf("get account for balance: %w", err)
	}

	balance, err := s.queries.GetAccountBalance(ctx, accountID)
	if err != nil {
		return BalanceResponse{}, fmt.Errorf("get account balance: %w", err)
	}

	return BalanceResponse{
		AccountID:    accountID,
		TotalDebits:  balance.TotalDebits,
		TotalCredits: balance.TotalCredits,
		NetBalance:   balance.NetBalance,
		Currency:     account.Currency,
	}, nil
}

// ListAccountPostings retrieves a paginated list of postings for a specific account.
func (s *LedgerService) ListAccountPostings(ctx context.Context, accountID int64, offset int32, pageSize int32) ([]db.GetPostingsByAccountIDRow, error) {
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}

	postings, err := s.queries.GetPostingsByAccountID(ctx, db.GetPostingsByAccountIDParams{
		AccountID:  accountID,
		PageSize:   pageSize,
		PageOffset: offset,
	})
	if err != nil {
		return nil, fmt.Errorf("list account postings: %w", err)
	}

	return postings, nil
}

// ---------------------------------------------------------------------------
// Transaction Operations (Core Double-Entry Logic)
// ---------------------------------------------------------------------------

// CreateTransaction executes a double-entry financial transaction.
//
// The flow is:
//  1. Validate the request (non-empty, balanced debits/credits).
//  2. Acquire a distributed lock on the idempotency key via Redlock.
//  3. Check Redis for an existing idempotency entry (replay if found).
//  4. Begin a PostgreSQL transaction (SERIALIZABLE isolation).
//  5. Insert the transaction record.
//  6. Insert all postings (the deferred constraint trigger validates balance at COMMIT).
//  7. Write event to the transactional outbox (same TX).
//  8. Commit the database transaction.
//  9. Store the idempotency key in Redis with the configured TTL.
//  10. Release the distributed lock.
func (s *LedgerService) CreateTransaction(ctx context.Context, req CreateTransactionRequest) (*TransactionResponse, error) {
	// -----------------------------------------------------------------------
	// Validation
	// -----------------------------------------------------------------------
	if req.IdempotencyKey == "" {
		return nil, ErrIdempotencyKeyRequired
	}
	if len(req.Postings) < 2 {
		return nil, ErrEmptyPostings
	}

	// Pre-validate: debits must equal credits
	var debitSum, creditSum int64
	for _, p := range req.Postings {
		if p.Amount <= 0 {
			return nil, ErrInvalidAmount
		}
		switch p.Direction {
		case db.PostingDirectionDebit:
			debitSum += p.Amount
		case db.PostingDirectionCredit:
			creditSum += p.Amount
		default:
			return nil, fmt.Errorf("%w: %s", ErrInvalidDirection, p.Direction)
		}
	}
	if debitSum != creditSum {
		return nil, fmt.Errorf("%w: debits=%d credits=%d", ErrUnbalancedTransaction, debitSum, creditSum)
	}

	// -----------------------------------------------------------------------
	// Step 1: Acquire distributed lock
	// -----------------------------------------------------------------------
	lockKey := fmt.Sprintf("ledger:lock:%s", req.IdempotencyKey)
	lock, err := s.locker.Obtain(ctx, lockKey, 30*time.Second, &redislock.Options{
		RetryStrategy: redislock.LimitRetry(redislock.LinearBackoff(100*time.Millisecond), 5),
	})
	if err != nil {
		s.logger.WarnContext(ctx, "failed to acquire distributed lock",
			slog.String("idempotency_key", req.IdempotencyKey),
			slog.String("error", err.Error()),
		)
		return nil, ErrLockAcquisitionFailed
	}
	defer func() {
		if releaseErr := lock.Release(ctx); releaseErr != nil {
			s.logger.WarnContext(ctx, "failed to release distributed lock",
				slog.String("idempotency_key", req.IdempotencyKey),
				slog.String("error", releaseErr.Error()),
			)
		}
	}()

	// -----------------------------------------------------------------------
	// Step 2: Check idempotency (replay if key exists)
	// -----------------------------------------------------------------------
	idempotencyRedisKey := fmt.Sprintf("ledger:idempotency:%s", req.IdempotencyKey)
	cached, err := s.rdb.Get(ctx, idempotencyRedisKey).Result()
	if err == nil && cached != "" {
		// Key exists — deserialize and replay the cached response
		var replay TransactionResponse
		if jsonErr := json.Unmarshal([]byte(cached), &replay); jsonErr == nil {
			s.logger.InfoContext(ctx, "idempotency replay",
				slog.String("idempotency_key", req.IdempotencyKey),
				slog.String("transaction_id", replay.ID.String()),
			)
			return &replay, nil
		}
		// If unmarshal fails, fall through to create a new transaction
		s.logger.WarnContext(ctx, "corrupted idempotency cache — proceeding with new transaction",
			slog.String("idempotency_key", req.IdempotencyKey),
		)
	} else if err != nil && !errors.Is(err, redis.Nil) {
		// Redis error that is NOT a cache miss — log and continue (fail open)
		s.logger.WarnContext(ctx, "redis get failed — proceeding without idempotency cache",
			slog.String("error", err.Error()),
		)
	}

	// -----------------------------------------------------------------------
	// Step 3: Begin database transaction
	// -----------------------------------------------------------------------
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{
		IsoLevel: pgx.Serializable,
	})
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		// Rollback is a no-op if the transaction was already committed.
		_ = tx.Rollback(ctx)
	}()

	qtx := s.queries.WithTx(tx)

	// -----------------------------------------------------------------------
	// Step 4: Insert the transaction record
	// -----------------------------------------------------------------------
	metadata := req.Metadata
	if metadata == nil {
		metadata = json.RawMessage(`{}`)
	}

	txRecord, err := qtx.CreateTransaction(ctx, db.CreateTransactionParams{
		IdempotencyKey: req.IdempotencyKey,
		Description:    req.Description,
		Metadata:       metadata,
	})
	if err != nil {
		return nil, fmt.Errorf("insert transaction: %w", err)
	}

	// -----------------------------------------------------------------------
	// Step 5: Insert all postings
	// -----------------------------------------------------------------------
	for i, p := range req.Postings {
		_, err := qtx.CreatePosting(ctx, db.CreatePostingParams{
			TransactionID: txRecord.ID,
			AccountID:     p.AccountID,
			Amount:        p.Amount,
			Direction:     p.Direction,
		})
		if err != nil {
			return nil, fmt.Errorf("insert posting[%d]: %w", i, err)
		}
	}

	// -----------------------------------------------------------------------
	// Step 6: Write event to transactional outbox (same ACID transaction)
	// -----------------------------------------------------------------------
	event := TransactionCreatedEvent{
		EventID:        uuid.New().String(),
		EventType:      "transaction.created",
		TransactionID:  txRecord.ID,
		IdempotencyKey: txRecord.IdempotencyKey,
		Description:    txRecord.Description,
		Postings:       req.Postings,
		Metadata:       txRecord.Metadata,
		CreatedAt:      txRecord.CreatedAt.Time,
	}

	outboxPayload, err := BuildOutboxPayload(event)
	if err != nil {
		return nil, fmt.Errorf("build outbox payload: %w", err)
	}

	_, err = qtx.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{
		EventType:  event.EventType,
		RoutingKey: "transaction.created",
		Payload:    outboxPayload,
	})
	if err != nil {
		return nil, fmt.Errorf("insert outbox event: %w", err)
	}

	// -----------------------------------------------------------------------
	// Step 7: Commit (deferred constraint trigger validates balance)
	// -----------------------------------------------------------------------
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	s.logger.InfoContext(ctx, "transaction committed (outbox event queued)",
		slog.String("transaction_id", txRecord.ID.String()),
		slog.String("idempotency_key", req.IdempotencyKey),
		slog.Int("posting_count", len(req.Postings)),
	)

	// -----------------------------------------------------------------------
	// Step 8: Fetch the full postings for the response
	// -----------------------------------------------------------------------
	postings, err := s.queries.GetPostingsByTransactionID(ctx, txRecord.ID)
	if err != nil {
		s.logger.WarnContext(ctx, "failed to fetch postings after commit — non-fatal",
			slog.String("error", err.Error()),
		)
	}

	response := &TransactionResponse{
		ID:             txRecord.ID,
		IdempotencyKey: txRecord.IdempotencyKey,
		Description:    txRecord.Description,
		Metadata:       txRecord.Metadata,
		Postings:       postings,
		CreatedAt:      txRecord.CreatedAt,
	}

	// -----------------------------------------------------------------------
	// Step 9: Store idempotency key in Redis (best-effort)
	// -----------------------------------------------------------------------
	if responseJSON, jsonErr := json.Marshal(response); jsonErr == nil {
		if setErr := s.rdb.Set(ctx, idempotencyRedisKey, responseJSON, s.idempotencyTTL).Err(); setErr != nil {
			s.logger.WarnContext(ctx, "failed to cache idempotency key — non-fatal",
				slog.String("error", setErr.Error()),
			)
		}
	}

	return response, nil
}

// GetTransaction retrieves a transaction and its postings by ID.
func (s *LedgerService) GetTransaction(ctx context.Context, id uuid.UUID) (*TransactionResponse, error) {
	txRecord, err := s.queries.GetTransactionByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTransactionNotFound
		}
		return nil, fmt.Errorf("get transaction: %w", err)
	}

	postings, err := s.queries.GetPostingsByTransactionID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get postings: %w", err)
	}

	return &TransactionResponse{
		ID:             txRecord.ID,
		IdempotencyKey: txRecord.IdempotencyKey,
		Description:    txRecord.Description,
		Metadata:       txRecord.Metadata,
		Postings:       postings,
		CreatedAt:      txRecord.CreatedAt,
	}, nil
}

// ListTransactions retrieves a paginated list of transactions.
func (s *LedgerService) ListTransactions(ctx context.Context, cursorTime time.Time, pageSize int32) ([]db.Transaction, error) {
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}

	cursor := pgtype.Timestamptz{
		Time:  cursorTime,
		Valid: true,
	}

	txns, err := s.queries.ListTransactions(ctx, db.ListTransactionsParams{
		CursorCreatedAt: cursor,
		PageSize:        pageSize,
	})
	if err != nil {
		return nil, fmt.Errorf("list transactions: %w", err)
	}
	return txns, nil
}

// ---------------------------------------------------------------------------
// Document Operations
// ---------------------------------------------------------------------------

// AttachDocument streams a file to R2 and records its metadata in the database.
func (s *LedgerService) AttachDocument(
	ctx context.Context,
	transactionID uuid.UUID,
	filename string,
	contentType string,
	sizeBytes int64,
	file io.Reader,
) (DocumentResponse, error) {
	// 1. Verify transaction exists
	_, err := s.queries.GetTransactionByID(ctx, transactionID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return DocumentResponse{}, ErrTransactionNotFound
		}
		return DocumentResponse{}, fmt.Errorf("verify transaction: %w", err)
	}

	// 2. Generate unique object key (e.g. docs/txn_id/uuid_filename)
	docID := uuid.New()
	objectKey := fmt.Sprintf("docs/%s/%s_%s", transactionID.String(), docID.String(), filename)

	// 3. Upload to R2 (streams directly, doesn't buffer in memory)
	if err := s.storageService.UploadFile(ctx, file, objectKey, contentType); err != nil {
		return DocumentResponse{}, fmt.Errorf("upload to storage: %w", err)
	}

	// 4. Save metadata in Postgres
	doc, err := s.queries.CreateDocument(ctx, db.CreateDocumentParams{
		TransactionID: transactionID,
		Filename:      filename,
		ContentType:   contentType,
		SizeBytes:     sizeBytes,
		ObjectKey:     objectKey,
	})
	if err != nil {
		// Ideally we would delete from R2 here on failure, but for simplicity we let it orphan or rely on R2 lifecycle policies
		return DocumentResponse{}, fmt.Errorf("save document metadata: %w", err)
	}

	// 5. Generate a presigned URL immediately for the response
	downloadURL, err := s.storageService.GetPresignedDownloadURL(ctx, objectKey)
	if err != nil {
		s.logger.WarnContext(ctx, "failed to generate presigned url for new document", slog.String("error", err.Error()))
	}

	s.logger.InfoContext(ctx, "document attached to transaction",
		slog.String("transaction_id", transactionID.String()),
		slog.String("document_id", doc.ID.String()),
	)

	return DocumentResponse{
		ID:            doc.ID,
		TransactionID: doc.TransactionID,
		Filename:      doc.Filename,
		ContentType:   doc.ContentType,
		SizeBytes:     doc.SizeBytes,
		DownloadURL:   downloadURL,
		CreatedAt:     doc.CreatedAt,
	}, nil
}

// ListTransactionDocuments returns all documents attached to a transaction, with fresh download URLs.
func (s *LedgerService) ListTransactionDocuments(ctx context.Context, transactionID uuid.UUID) ([]DocumentResponse, error) {
	docs, err := s.queries.GetDocumentsByTransaction(ctx, transactionID)
	if err != nil {
		return nil, fmt.Errorf("get documents: %w", err)
	}

	responses := make([]DocumentResponse, len(docs))
	for i, doc := range docs {
		downloadURL, err := s.storageService.GetPresignedDownloadURL(ctx, doc.ObjectKey)
		if err != nil {
			s.logger.WarnContext(ctx, "failed to generate presigned url", slog.String("document_id", doc.ID.String()))
		}

		responses[i] = DocumentResponse{
			ID:            doc.ID,
			TransactionID: doc.TransactionID,
			Filename:      doc.Filename,
			ContentType:   doc.ContentType,
			SizeBytes:     doc.SizeBytes,
			DownloadURL:   downloadURL,
			CreatedAt:     doc.CreatedAt,
		}
	}

	return responses, nil
}
