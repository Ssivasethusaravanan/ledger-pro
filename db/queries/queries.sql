-- ============================================================================
-- LedgerPro: SQLC Queries
-- ============================================================================

-- ---------------------------------------------------------------------------
-- TENANTS
-- ---------------------------------------------------------------------------

-- name: CreateTenant :one
INSERT INTO tenants (name)
VALUES ($1)
RETURNING *;

-- ---------------------------------------------------------------------------
-- ACCOUNTS
-- ---------------------------------------------------------------------------

-- name: CreateAccount :one
INSERT INTO accounts (tenant_id, account_name, account_type, currency, metadata)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetAccountByID :one
SELECT * FROM accounts
WHERE id = $1 AND tenant_id = $2
LIMIT 1;

-- name: GetAccountByName :one
SELECT * FROM accounts
WHERE account_name = $1 AND tenant_id = $2
LIMIT 1;

-- name: ListAccounts :many
SELECT * FROM accounts
WHERE tenant_id = $1 AND id > @cursor::BIGINT
ORDER BY id ASC
LIMIT @page_size::INT;

-- name: CountAccounts :one
SELECT COUNT(*) FROM accounts
WHERE tenant_id = $1;

-- ---------------------------------------------------------------------------
-- TRANSACTIONS
-- ---------------------------------------------------------------------------

-- name: CreateTransaction :one
INSERT INTO transactions (tenant_id, idempotency_key, description, metadata)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetTransactionByID :one
SELECT * FROM transactions
WHERE id = $1 AND tenant_id = $2
LIMIT 1;

-- name: GetTransactionByIdempotencyKey :one
SELECT * FROM transactions
WHERE idempotency_key = $1 AND tenant_id = $2
LIMIT 1;

-- name: ListTransactions :many
SELECT * FROM transactions
WHERE tenant_id = $1 AND created_at < @cursor_created_at::TIMESTAMPTZ
ORDER BY created_at DESC
LIMIT @page_size::INT;

-- name: CountTransactions :one
SELECT COUNT(*) FROM transactions
WHERE tenant_id = $1;

-- ---------------------------------------------------------------------------
-- POSTINGS
-- ---------------------------------------------------------------------------

-- name: CreatePosting :one
INSERT INTO postings (tenant_id, transaction_id, account_id, amount, direction)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetPostingsByTransactionID :many
SELECT
    p.id,
    p.tenant_id,
    p.transaction_id,
    p.account_id,
    a.account_name,
    p.amount,
    p.direction,
    p.created_at
FROM postings p
JOIN accounts a ON a.id = p.account_id
WHERE p.transaction_id = $1 AND p.tenant_id = $2
ORDER BY p.id ASC;

-- name: GetPostingsByAccountID :many
SELECT
    p.id,
    p.tenant_id,
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
WHERE p.account_id = $1 AND p.tenant_id = $2
ORDER BY p.created_at DESC
LIMIT @page_size::INT OFFSET @page_offset::INT;

-- ---------------------------------------------------------------------------
-- BALANCE COMPUTATION
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
WHERE account_id = $1 AND tenant_id = $2;

-- ---------------------------------------------------------------------------
-- OUTBOX (Transactional Event Delivery)
-- ---------------------------------------------------------------------------

-- name: CreateOutboxEvent :one
INSERT INTO ledger_outbox (tenant_id, event_type, routing_key, payload)
VALUES ($1, $2, $3, $4)
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
INSERT INTO documents (tenant_id, transaction_id, filename, content_type, size_bytes, object_key)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetDocumentsByTransaction :many
SELECT * FROM documents
WHERE transaction_id = $1 AND tenant_id = $2
ORDER BY created_at ASC;

-- ---------------------------------------------------------------------------
-- USERS
-- ---------------------------------------------------------------------------

-- name: CreateUser :one
INSERT INTO users (tenant_id, email, password_hash, role)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetUserByEmail :one
SELECT * FROM users
WHERE email = $1
LIMIT 1;

