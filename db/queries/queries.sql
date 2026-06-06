-- ============================================================================
-- LedgerPro: SQLC Queries
-- 
-- Naming convention: [Action][Entity] with SQLC annotations
-- All balance computations are real-time aggregations — no materialized column.
-- ============================================================================

-- ---------------------------------------------------------------------------
-- ACCOUNTS
-- ---------------------------------------------------------------------------

-- name: CreateAccount :one
INSERT INTO accounts (account_name, account_type, currency, metadata)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetAccountByID :one
SELECT * FROM accounts
WHERE id = $1
LIMIT 1;

-- name: GetAccountByName :one
SELECT * FROM accounts
WHERE account_name = $1
LIMIT 1;

-- name: ListAccounts :many
SELECT * FROM accounts
WHERE id > @cursor::BIGINT
ORDER BY id ASC
LIMIT @page_size::INT;

-- name: CountAccounts :one
SELECT COUNT(*) FROM accounts;

-- ---------------------------------------------------------------------------
-- TRANSACTIONS
-- ---------------------------------------------------------------------------

-- name: CreateTransaction :one
INSERT INTO transactions (idempotency_key, description, metadata)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetTransactionByID :one
SELECT * FROM transactions
WHERE id = $1
LIMIT 1;

-- name: GetTransactionByIdempotencyKey :one
SELECT * FROM transactions
WHERE idempotency_key = $1
LIMIT 1;

-- name: ListTransactions :many
SELECT * FROM transactions
WHERE created_at < @cursor_created_at::TIMESTAMPTZ
ORDER BY created_at DESC
LIMIT @page_size::INT;

-- name: CountTransactions :one
SELECT COUNT(*) FROM transactions;

-- ---------------------------------------------------------------------------
-- POSTINGS
-- ---------------------------------------------------------------------------

-- name: CreatePosting :one
INSERT INTO postings (transaction_id, account_id, amount, direction)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetPostingsByTransactionID :many
SELECT
    p.id,
    p.transaction_id,
    p.account_id,
    a.account_name,
    p.amount,
    p.direction,
    p.created_at
FROM postings p
JOIN accounts a ON a.id = p.account_id
WHERE p.transaction_id = $1
ORDER BY p.id ASC;

-- name: GetPostingsByAccountID :many
SELECT
    p.id,
    p.transaction_id,
    t.description AS transaction_description,
    p.account_id,
    a.account_name,
    p.amount,
    p.direction,
    p.created_at
FROM postings p
JOIN accounts a ON a.id = p.account_id
JOIN transactions t ON t.id = p.transaction_id
WHERE p.account_id = $1
ORDER BY p.created_at DESC
LIMIT @page_size::INT OFFSET @page_offset::INT;

-- ---------------------------------------------------------------------------
-- BALANCE COMPUTATION
-- ---------------------------------------------------------------------------
-- Returns the net balance for an account as:
--   SUM(debits) - SUM(credits)
-- 
-- For asset/expense accounts: positive balance = normal (debit-normal).
-- For liability/equity/revenue accounts: negative balance = normal (credit-normal).
-- The caller interprets sign based on account_type.
-- ---------------------------------------------------------------------------

-- name: GetAccountBalance :one
SELECT
    COALESCE(SUM(CASE WHEN direction = 'debit'  THEN amount ELSE 0 END), 0)::BIGINT AS total_debits,
    COALESCE(SUM(CASE WHEN direction = 'credit' THEN amount ELSE 0 END), 0)::BIGINT AS total_credits,
    (
        COALESCE(SUM(CASE WHEN direction = 'debit'  THEN amount ELSE 0 END), 0) -
        COALESCE(SUM(CASE WHEN direction = 'credit' THEN amount ELSE 0 END), 0)
    )::BIGINT AS net_balance
FROM postings
WHERE account_id = $1;

-- ---------------------------------------------------------------------------
-- OUTBOX (Transactional Event Delivery)
-- ---------------------------------------------------------------------------

-- name: CreateOutboxEvent :one
INSERT INTO ledger_outbox (event_type, routing_key, payload)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetPendingOutboxEvents :many
SELECT * FROM ledger_outbox
WHERE status = 'pending'
ORDER BY created_at ASC
LIMIT @batch_size::INT;

-- name: MarkOutboxEventPublished :exec
UPDATE ledger_outbox
SET status = 'published', processed_at = now()
WHERE id = $1 AND status = 'pending';

-- name: MarkOutboxEventFailed :exec
UPDATE ledger_outbox
SET status = 'failed', last_error = $2, processed_at = now()
WHERE id = $1;

-- name: IncrementOutboxRetry :exec
UPDATE ledger_outbox
SET retry_count = retry_count + 1, last_error = $2
WHERE id = $1 AND status = 'pending';

-- name: RequeueFailedOutboxEvents :exec
UPDATE ledger_outbox
SET status = 'pending', retry_count = 0, last_error = '', processed_at = NULL
WHERE status = 'failed' AND retry_count < max_retries;

-- ---------------------------------------------------------------------------
-- DOCUMENTS (R2 Storage Attachments)
-- ---------------------------------------------------------------------------

-- name: CreateDocument :one
INSERT INTO documents (transaction_id, filename, content_type, size_bytes, object_key)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetDocumentsByTransaction :many
SELECT * FROM documents
WHERE transaction_id = $1
ORDER BY created_at ASC;

-- ---------------------------------------------------------------------------
-- USERS
-- ---------------------------------------------------------------------------

-- name: CreateUser :one
INSERT INTO users (email, password_hash, role)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetUserByEmail :one
SELECT * FROM users
WHERE email = $1
LIMIT 1;

