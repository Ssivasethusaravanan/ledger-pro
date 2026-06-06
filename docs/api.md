# LedgerPro API Reference

> **Base URL (Production):** `https://ledger-pro-api.onrender.com`  
> **Swagger UI:** `https://ledger-pro-api.onrender.com/swagger/index.html`  
> **Version:** `2.0.0`

---

## Table of Contents

1. [Authentication](#authentication)
2. [Accounts](#accounts)
3. [Transactions](#transactions)
4. [Documents](#documents)
5. [Observability](#observability)
6. [Error Handling (RFC 7807)](#error-handling-rfc-7807)
7. [Rate Limiting](#rate-limiting)
8. [Pagination](#pagination)

---

## Authentication

LedgerPro supports **two authentication methods**, designed for different use cases:

| Method | Use Case | How It Works |
|--------|----------|--------------|
| **API Key** (`X-API-Key` header) | Server-to-Server, CLI, Swagger UI | Pass your key in the `X-API-Key` header or as `Authorization: Bearer <key>` |
| **Session Cookie** (`session_token`) | Browser / Web Dashboard | Call `POST /v1/auth/login` with your API key → receive an HTTP-only, Secure, SameSite=Strict cookie |

### RBAC Roles

| Role | Permissions |
|------|-------------|
| `admin` | Full read + write access to all endpoints |
| `write` | Create accounts, transactions, upload documents + read |
| `read` | Read-only access to accounts, transactions, documents |

### API Key Format

API keys are configured server-side via the `API_KEYS` environment variable:
```
API_KEYS=sk-admin-master-key-xyz:admin,sk-dashboard-readonly-123:read
```

When authenticating, pass **only the key part** (before the `:`):
```
X-API-Key: sk-admin-master-key-xyz
```

---

### `POST /v1/auth/login`

Authenticates via API key and sets a secure HTTP-only `session_token` cookie. This is the primary authentication method for browser-based frontends.

**Request Body:**
```json
{
  "api_key": "sk-admin-master-key-xyz"
}
```

**Response (200 OK):**
```json
{
  "role": "admin"
}
```

**Cookie Set:**
```
Set-Cookie: session_token=<random-base64-token>; Path=/; HttpOnly; Secure; SameSite=Strict; Max-Age=86400
```

**Security Properties:**
- `HttpOnly` — JavaScript cannot access the token (XSS defense)
- `Secure` — Cookie is only sent over HTTPS
- `SameSite=Strict` — Cookie is never sent cross-origin (CSRF defense)
- Sessions are stored in Redis with a 24-hour TTL and auto-refresh on each request

---

### `POST /v1/auth/logout`

Revokes the active session and clears the session cookie.

**Response:** `204 No Content`

---

### `GET /v1/auth/me`

Returns the currently authenticated user's role. Requires either a valid session cookie or API key.

**Response (200 OK):**
```json
{
  "role": "admin"
}
```

---

## Accounts

Accounts are the fundamental building blocks of the double-entry ledger. Each account has a **type** that determines its accounting behavior.

### Account Types

| Type | Description | Example |
|------|-------------|---------|
| `asset` | Resources owned by the business | Cash, Bank Account, Equipment |
| `liability` | Debts owed by the business | Loans, Accounts Payable |
| `equity` | Owner's stake in the business | Owner's Capital, Retained Earnings |
| `revenue` | Income earned | Sales Revenue, Service Income |
| `expense` | Costs incurred | Rent, Salaries, Utilities |

### Supported Currencies

All **ISO 4217** currency codes are supported (167 currencies). Common examples: `INR`, `USD`, `EUR`, `GBP`, `JPY`, `AUD`, `CAD`, `CHF`, `SGD`.

If no currency is specified, `INR` is used as the default.

> **Important:** All monetary amounts are stored as **integers in the smallest currency unit** (e.g., paisa for INR, cents for USD). This eliminates floating-point rounding errors.

---

### `POST /v1/accounts`

Creates a new ledger account.

**Required Role:** `admin` or `write`

**Request Body:**
```json
{
  "account_name": "Operating Bank Account",
  "account_type": "asset",
  "currency": "INR",
  "metadata": {
    "bank": "HDFC",
    "branch": "Chennai Main"
  }
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `account_name` | string | ✅ | Unique name (1–255 characters) |
| `account_type` | string | ✅ | One of: `asset`, `liability`, `equity`, `revenue`, `expense` |
| `currency` | string | ❌ | ISO 4217 code (default: `INR`) |
| `metadata` | object | ❌ | Arbitrary JSON metadata (default: `{}`) |

**Response (201 Created):**
```json
{
  "id": 1,
  "account_name": "Operating Bank Account",
  "account_type": "asset",
  "currency": "INR",
  "metadata": {
    "bank": "HDFC",
    "branch": "Chennai Main"
  },
  "created_at": "2026-06-04T18:00:00Z"
}
```

**Possible Errors:**
- `400` — Invalid request body or invalid currency code
- `422` — Validation errors (missing name, invalid type)
- `500` — Duplicate account name (UNIQUE constraint)

---

### `GET /v1/accounts`

Returns a paginated list of all accounts.

**Required Role:** `admin`, `write`, or `read`

**Query Parameters:**

| Param | Type | Default | Description |
|-------|------|---------|-------------|
| `cursor` | integer | `0` | ID-based cursor for pagination |
| `page_size` | integer | `20` | Items per page (max: 100) |

**Response (200 OK):**
```json
{
  "data": [
    {
      "id": 1,
      "account_name": "Operating Bank Account",
      "account_type": "asset",
      "currency": "INR",
      "metadata": {},
      "created_at": "2026-06-04T18:00:00Z"
    }
  ],
  "next_cursor": 1,
  "page_size": 20
}
```

---

### `GET /v1/accounts/{id}`

Retrieves a single account by its numeric ID.

**Required Role:** `admin`, `write`, or `read`

**Response (200 OK):**
```json
{
  "id": 1,
  "account_name": "Operating Bank Account",
  "account_type": "asset",
  "currency": "INR",
  "metadata": {},
  "created_at": "2026-06-04T18:00:00Z"
}
```

**Possible Errors:**
- `400` — Invalid account ID (must be an integer)
- `404` — Account not found

---

### `GET /v1/accounts/{id}/balance`

Computes the **real-time balance** of an account by aggregating all its postings. There is no cached or materialized balance — it is always calculated fresh from the immutable posting records.

**Required Role:** `admin`, `write`, or `read`

**Response (200 OK):**
```json
{
  "account_id": 1,
  "total_debits": 500000,
  "total_credits": 200000,
  "net_balance": 300000,
  "currency": "INR"
}
```

> **Reading the balance:** For `asset` and `expense` accounts, the balance is `total_debits - total_credits`. For `liability`, `equity`, and `revenue` accounts, the balance is `total_credits - total_debits`. The `net_balance` field returns `total_debits - total_credits` in all cases — the client should interpret it based on account type.

---

## Transactions

Transactions are the core of LedgerPro's double-entry bookkeeping system. Every transaction consists of **two or more postings** where the total debits must exactly equal the total credits.

### Key Properties

- **Immutable** — Transactions and postings cannot be updated or deleted (enforced by database triggers)
- **Idempotent** — Every transaction requires a unique `Idempotency-Key` header; replaying the same key returns the original response
- **SERIALIZABLE Isolation** — All transactions run at the highest PostgreSQL isolation level
- **Distributed Locking** — Redis-based distributed locks prevent concurrent duplicate processing
- **Atomically Linked Events** — Each transaction writes an event to the transactional outbox (same DB transaction), guaranteeing at-least-once delivery to RabbitMQ

---

### `POST /v1/transactions`

Creates a new double-entry financial transaction.

**Required Role:** `admin` or `write`

**Required Headers:**

| Header | Type | Required | Description |
|--------|------|----------|-------------|
| `Idempotency-Key` | string | ✅ | Unique key (1–255 chars, typically a UUID) to prevent duplicate processing |

**Request Body:**
```json
{
  "description": "Client invoice payment — INV-2024-001",
  "metadata": {
    "invoice_number": "INV-2024-001",
    "client": "Acme Corp"
  },
  "postings": [
    {
      "account_id": 1,
      "amount": 150000,
      "direction": "debit"
    },
    {
      "account_id": 2,
      "amount": 150000,
      "direction": "credit"
    }
  ]
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `description` | string | ❌ | Human-readable description (max 1000 chars) |
| `metadata` | object | ❌ | Arbitrary JSON metadata |
| `postings` | array | ✅ | Array of posting entries (min: 2, max: 100) |
| `postings[].account_id` | integer | ✅ | Target account ID (must exist) |
| `postings[].amount` | integer | ✅ | Amount in smallest currency unit (must be > 0) |
| `postings[].direction` | string | ✅ | Either `"debit"` or `"credit"` |

**Validation Rules:**
1. At least 2 postings required (one debit, one credit)
2. Maximum 100 postings per transaction
3. All amounts must be positive integers
4. Sum of all debit amounts must exactly equal sum of all credit amounts
5. All referenced account IDs must exist

**Response (201 Created):**
```json
{
  "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "idempotency_key": "550e8400-e29b-41d4-a716-446655440000",
  "description": "Client invoice payment — INV-2024-001",
  "metadata": {
    "invoice_number": "INV-2024-001",
    "client": "Acme Corp"
  },
  "postings": [
    {
      "id": 1,
      "transaction_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
      "account_id": 1,
      "amount": 150000,
      "direction": "debit",
      "created_at": "2026-06-04T18:00:00Z"
    },
    {
      "id": 2,
      "transaction_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
      "account_id": 2,
      "amount": 150000,
      "direction": "credit",
      "created_at": "2026-06-04T18:00:00Z"
    }
  ],
  "created_at": "2026-06-04T18:00:00Z"
}
```

**Possible Errors:**

| Status | Type | Cause |
|--------|------|-------|
| `400` | Bad Request | Missing idempotency key, invalid body, empty postings, invalid amount or direction |
| `409` | Duplicate Idempotency Key | A transaction with this idempotency key already exists |
| `422` | Unbalanced Transaction | Total debits ≠ total credits |
| `503` | Lock Acquisition Failed | Could not acquire distributed lock (system under load — retry) |

---

### `GET /v1/transactions/{id}`

Retrieves a single transaction with all its postings.

**Required Role:** `admin`, `write`, or `read`

**Path Parameter:** `id` — UUID of the transaction

**Response (200 OK):**
```json
{
  "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "idempotency_key": "550e8400-e29b-41d4-a716-446655440000",
  "description": "Client invoice payment",
  "metadata": {},
  "postings": [
    {
      "id": 1,
      "transaction_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
      "account_id": 1,
      "amount": 150000,
      "direction": "debit",
      "created_at": "2026-06-04T18:00:00Z"
    }
  ],
  "created_at": "2026-06-04T18:00:00Z"
}
```

---

### `GET /v1/transactions`

Returns a paginated list of transactions (newest first).

**Required Role:** `admin`, `write`, or `read`

**Query Parameters:**

| Param | Type | Default | Description |
|-------|------|---------|-------------|
| `cursor` | string (RFC3339) | current time | Timestamp-based cursor for keyset pagination |
| `page_size` | integer | `20` | Items per page (max: 100) |

**Response (200 OK):**
```json
{
  "data": [
    {
      "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
      "idempotency_key": "unique-key-123",
      "description": "Payment received",
      "metadata": {},
      "created_at": "2026-06-04T18:00:00Z"
    }
  ],
  "next_cursor": "2026-06-04T17:59:59.999999Z",
  "page_size": 20
}
```

---

## Documents

LedgerPro supports attaching files (receipts, invoices, contracts) directly to transactions. Files are streamed to **Cloudflare R2** (S3-compatible object storage) and metadata is recorded in PostgreSQL.

---

### `POST /v1/transactions/{id}/documents`

Upload a file and attach it to a transaction.

**Required Role:** `admin` or `write`  
**Content-Type:** `multipart/form-data`  
**Max File Size:** 10 MB

**Form Fields:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `file` | binary | ✅ | The file to upload |

**Example (cURL):**
```bash
curl -X POST https://ledger-pro-api.onrender.com/v1/transactions/{txn_id}/documents \
  -H "X-API-Key: sk-admin-master-key-xyz" \
  -F "file=@receipt.pdf"
```

**Response (201 Created):**
```json
{
  "id": "doc-uuid-here",
  "transaction_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "filename": "receipt.pdf",
  "content_type": "application/pdf",
  "size_bytes": 102400,
  "download_url": "https://r2.cloudflarestorage.com/sflix/docs/txn_id/doc_id_receipt.pdf?X-Amz-Signature=...",
  "created_at": "2026-06-04T18:00:00Z"
}
```

> The `download_url` is a **presigned URL** valid for 15 minutes. After expiry, retrieve a fresh URL via the list endpoint.

---

### `GET /v1/transactions/{id}/documents`

List all documents attached to a transaction, with fresh presigned download URLs.

**Required Role:** `admin`, `write`, or `read`

**Response (200 OK):**
```json
{
  "data": [
    {
      "id": "doc-uuid-here",
      "transaction_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
      "filename": "receipt.pdf",
      "content_type": "application/pdf",
      "size_bytes": 102400,
      "download_url": "https://r2.cloudflarestorage.com/sflix/docs/...",
      "created_at": "2026-06-04T18:00:00Z"
    }
  ]
}
```

---

## Observability

### `GET /v1/health`

Deep health check that verifies connectivity to all dependencies. This endpoint is **public** (no authentication required).

**Response (200 OK):**
```json
{
  "status": "healthy",
  "service": "ledger_pro",
  "version": "2.0.0",
  "uptime": "3h45m12s",
  "dependencies": {
    "postgresql": {
      "status": "healthy",
      "latency": "2.3ms"
    },
    "postgresql_pool": {
      "status": "info",
      "latency": "total=25 acquired=3 idle=22"
    },
    "redis": {
      "status": "healthy",
      "latency": "1.1ms"
    }
  }
}
```

If any dependency is down, the status changes to `"degraded"` and the HTTP status is `503 Service Unavailable`.

### Distributed Tracing

Every request is assigned a **Trace ID** (UUID v4). You can:
- **Pass your own:** Include an `X-Trace-ID` header and the server will use it
- **Auto-generated:** If not provided, the server generates one

The trace ID appears in:
- Response header: `X-Trace-ID`
- All log entries for the request
- All error responses in the `trace_id` field

### Structured Logging

All logs are emitted in **JSON format** with fields:
```json
{
  "time": "2026-06-04T18:00:00Z",
  "level": "INFO",
  "source": {"function": "...", "file": "...", "line": 42},
  "msg": "http request",
  "method": "POST",
  "path": "/v1/accounts",
  "status": 201,
  "duration": "3.2ms",
  "trace_id": "abc-123",
  "remote_addr": "1.2.3.4:12345",
  "user_agent": "curl/8.0"
}
```

---

## Error Handling (RFC 7807)

All errors follow the **RFC 7807 Problem Details** standard with Content-Type `application/problem+json`.

**Error Response Structure:**
```json
{
  "type": "https://ledgerpro.dev/errors/bad-request",
  "title": "Bad Request",
  "status": 400,
  "detail": "invalid account id — must be an integer",
  "instance": "/v1/accounts/abc",
  "trace_id": "550e8400-e29b-41d4-a716-446655440000"
}
```

**Validation Errors (422)** include detailed field-level failures:
```json
{
  "type": "https://ledgerpro.dev/errors/validation-error",
  "title": "Validation Error",
  "status": 422,
  "detail": "2 validation error(s) found",
  "instance": "/v1/accounts",
  "trace_id": "...",
  "invalid_params": [
    { "name": "account_name", "reason": "account_name is required" },
    { "name": "account_type", "reason": "account_type is required (asset, liability, equity, revenue, expense)" }
  ]
}
```

### Error Type Reference

| HTTP Status | Problem Type | When It Occurs |
|-------------|-------------|----------------|
| `400` | `/errors/bad-request` | Malformed JSON, invalid parameters |
| `401` | `/errors/unauthorized` | Missing or invalid API key / session |
| `403` | `/errors/forbidden` | Authenticated but insufficient role |
| `404` | `/errors/account-not-found` | Account ID does not exist |
| `404` | `/errors/transaction-not-found` | Transaction ID does not exist |
| `409` | `/errors/duplicate-idempotency-key` | Idempotency key already used |
| `422` | `/errors/validation-error` | Field validation failures |
| `422` | `/errors/unbalanced-transaction` | Debits ≠ Credits |
| `429` | `/errors/rate-limit-exceeded` | Too many requests |
| `500` | `/errors/internal-error` | Unexpected server error |
| `503` | `/errors/lock-acquisition-failed` | Distributed lock contention |
| `503` | `/errors/service-unavailable` | Dependency down |

---

## Rate Limiting

LedgerPro implements a **Redis sliding-window rate limiter** per client identity (API key first, then IP fallback).

**Default Configuration:**
- **100 requests** per **1-minute** sliding window

**Rate Limit Headers (included on every response):**

| Header | Description |
|--------|-------------|
| `X-RateLimit-Limit` | Maximum requests allowed in the window |
| `X-RateLimit-Remaining` | Requests remaining in the current window |
| `X-RateLimit-Reset` | Unix timestamp when the window resets |

**When exceeded (429 Too Many Requests):**
```json
{
  "type": "https://ledgerpro.dev/errors/rate-limit-exceeded",
  "title": "Rate Limit Exceeded",
  "status": 429,
  "detail": "rate limit of 100 requests per 1m0s exceeded — retry after 1m0s"
}
```

The `Retry-After` header is also set with the window duration in seconds.

> **Fail-Open:** If Redis is temporarily unavailable, the rate limiter allows all traffic through to avoid blocking legitimate requests.

---

## Pagination

LedgerPro uses **cursor-based (keyset) pagination** for high-performance, consistent results.

### Accounts Pagination
- Uses **ID-based cursors** (integer)
- Pass `?cursor=<last_id>&page_size=20`
- The response includes `next_cursor` — pass it as `cursor` in the next request

### Transactions Pagination
- Uses **timestamp-based cursors** (RFC3339Nano)
- Pass `?cursor=<timestamp>&page_size=20`
- Sorted newest-first by `created_at`

**Example pagination flow:**
```
GET /v1/accounts?page_size=10              → returns data + next_cursor=10
GET /v1/accounts?cursor=10&page_size=10    → returns data + next_cursor=20
GET /v1/accounts?cursor=20&page_size=10    → returns data + next_cursor=25 (last page)
```

---

## Complete Endpoint Summary

| Method | Path | Auth | Role | Description |
|--------|------|------|------|-------------|
| `GET` | `/v1/health` | ❌ Public | — | Deep health check |
| `GET` | `/swagger/*` | ❌ Public | — | Interactive Swagger UI |
| `POST` | `/v1/auth/login` | ❌ Public | — | Login and get session cookie |
| `POST` | `/v1/auth/logout` | ✅ | Any | Logout and clear session |
| `GET` | `/v1/auth/me` | ✅ | Any | Get current user role |
| `POST` | `/v1/accounts` | ✅ | admin, write | Create account |
| `GET` | `/v1/accounts` | ✅ | admin, write, read | List accounts |
| `GET` | `/v1/accounts/{id}` | ✅ | admin, write, read | Get account |
| `GET` | `/v1/accounts/{id}/balance` | ✅ | admin, write, read | Get account balance |
| `POST` | `/v1/transactions` | ✅ | admin, write | Create transaction |
| `GET` | `/v1/transactions` | ✅ | admin, write, read | List transactions |
| `GET` | `/v1/transactions/{id}` | ✅ | admin, write, read | Get transaction |
| `POST` | `/v1/transactions/{id}/documents` | ✅ | admin, write | Upload document |
| `GET` | `/v1/transactions/{id}/documents` | ✅ | admin, write, read | List documents |
