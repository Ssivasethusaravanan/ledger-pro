<p align="center">
  <h1 align="center">LedgerPro</h1>
  <p align="center">
    A FAANG-grade, production-ready Double-Entry Financial Ledger API built with Go.
    <br />
    <a href="docs/api.md"><strong>API Reference →</strong></a>
    &nbsp;·&nbsp;
    <a href="https://ledger-pro-api.onrender.com/swagger/index.html"><strong>Swagger UI →</strong></a>
    &nbsp;·&nbsp;
    <a href="https://ledger-pro-api.onrender.com/v1/health"><strong>Health Check →</strong></a>
  </p>
</p>

---

## What is LedgerPro?

LedgerPro is a **financial source of truth** — an API that tracks every monetary movement in your application using **double-entry bookkeeping**, the same system used by banks and Fortune 500 companies since the 15th century.

Every transaction in LedgerPro creates a balanced set of **debits and credits** that mathematically must sum to zero. This makes it impossible for money to appear or disappear from the system. Balances are never stored — they are always computed in real-time from the immutable posting records.

---

## Features at a Glance

| Category | Feature | Description |
|----------|---------|-------------|
| 📚 **Core Ledger** | Double-Entry Bookkeeping | Every transaction has balanced debit/credit postings |
| 📚 **Core Ledger** | Append-Only Immutability | Postings and transactions can never be updated or deleted |
| 📚 **Core Ledger** | Real-Time Balance Computation | No cached balances — always computed fresh from postings |
| 📚 **Core Ledger** | ISO 4217 Currencies | Supports all 167 active currency codes |
| 📚 **Core Ledger** | Integer Arithmetic | Amounts stored in smallest unit (paisa/cents) — no floating point |
| 🔐 **Security** | API Key + RBAC | Role-based access control (admin / write / read) |
| 🔐 **Security** | HTTP-Only Session Cookies | Secure browser auth (XSS + CSRF defense) |
| 🔐 **Security** | Redis Session Store | Stateful sessions with auto-refresh TTL |
| ⚡ **Reliability** | Idempotency Keys | Duplicate transaction prevention with Redis + distributed locks |
| ⚡ **Reliability** | SERIALIZABLE Isolation | Highest PostgreSQL isolation level for transactions |
| ⚡ **Reliability** | Distributed Locking | Redis-based Redlock for concurrent request protection |
| ⚡ **Reliability** | Transactional Outbox | Guaranteed at-least-once event delivery to RabbitMQ |
| 📊 **Observability** | Structured JSON Logging | Every request logged with method, path, status, duration, trace ID |
| 📊 **Observability** | Distributed Tracing | Auto-generated or client-provided X-Trace-ID on every request |
| 📊 **Observability** | Deep Health Checks | Real-time dependency status with latency measurements |
| 🛡️ **Protection** | Redis Sliding-Window Rate Limiter | Per-client rate limiting with standard headers |
| 🛡️ **Protection** | Request Timeout Enforcement | Configurable max request duration |
| 🛡️ **Protection** | CORS | Configurable cross-origin resource sharing |
| 🛡️ **Protection** | RFC 7807 Error Responses | Standardized problem details for all errors |
| 📎 **Documents** | Cloudflare R2 File Storage | Attach receipts/invoices to transactions |
| 📎 **Documents** | Presigned Download URLs | Secure, time-limited file access (15 min) |
| 📦 **Events** | RabbitMQ Event Streaming | `transaction.created` events published reliably |
| 📦 **Events** | Outbox Worker | Background goroutine with exponential backoff retries |

---

## Architecture

```
┌────────────────────────────────────────────────────────────────────┐
│                        HTTP CLIENT                                 │
│              (Swagger UI / Browser / Server-to-Server)             │
└──────────────────┬─────────────────────────────────────────────────┘
                   │
                   ▼
┌──────────────────────────────────────────────────────────────────────┐
│                    MIDDLEWARE STACK (chi router)                      │
│  ┌──────────┐ ┌─────────┐ ┌──────────┐ ┌──────┐ ┌───────────────┐  │
│  │ Recoverer│ │ Real IP │ │ Trace ID │ │ CORS │ │ Structured    │  │
│  │          │ │         │ │ Injection│ │      │ │ JSON Logging  │  │
│  └──────────┘ └─────────┘ └──────────┘ └──────┘ └───────────────┘  │
│  ┌──────────────────┐ ┌─────────────────┐ ┌───────────────────────┐ │
│  │ Request Timeout  │ │ Rate Limiter    │ │ API Key Auth + RBAC   │ │
│  │ (30s default)    │ │ (Redis ZSET)    │ │ (key or cookie)       │ │
│  └──────────────────┘ └─────────────────┘ └───────────────────────┘ │
└──────────────────┬───────────────────────────────────────────────────┘
                   │
                   ▼
┌──────────────────────────────────────────────────────────────────────┐
│                        HANDLER LAYER                                 │
│  Accounts ─ Transactions ─ Documents ─ Auth ─ Health                 │
└──────────────────┬───────────────────────────────────────────────────┘
                   │
                   ▼
┌──────────────────────────────────────────────────────────────────────┐
│                        SERVICE LAYER                                 │
│  LedgerService ─ AuthService ─ StorageService                        │
│                                                                      │
│  • Distributed Lock Acquisition (Redlock)                            │
│  • Idempotency Check (Redis GET)                                     │
│  • PostgreSQL Transaction (SERIALIZABLE)                             │
│  • Outbox Event Write (same ACID TX)                                 │
│  • Idempotency Cache (Redis SET with TTL)                            │
└──────────────────┬───────────────────────────────────────────────────┘
                   │
        ┌──────────┼───────────┬─────────────────┐
        ▼          ▼           ▼                 ▼
┌────────────┐ ┌────────┐ ┌──────────┐ ┌──────────────┐
│ PostgreSQL │ │ Redis  │ │ RabbitMQ │ │ Cloudflare   │
│ (Supabase) │ │(Render)│ │(CloudAMQ)│ │ R2 Storage   │
│            │ │        │ │          │ │              │
│ • accounts │ │ • locks│ │ • events │ │ • documents  │
│ • txns     │ │ • rates│ │   topic  │ │ • presigned  │
│ • postings │ │ • idem │ │   exchange│ │   URLs       │
│ • outbox   │ │ • sess │ │          │ │              │
│ • documents│ │        │ │          │ │              │
└────────────┘ └────────┘ └──────────┘ └──────────────┘
                               ▲
                               │
                   ┌───────────┘
                   │
         ┌─────────────────┐
         │  OUTBOX WORKER   │
         │  (background)    │
         │                  │
         │ Polls outbox     │
         │ every 200ms      │
         │ Batch size: 50   │
         │ Exp. backoff     │
         │ Max 5 retries    │
         └─────────────────┘
```

---

## Tech Stack

| Component | Technology | Purpose |
|-----------|-----------|---------|
| Language | **Go 1.22+** | High-performance, concurrent server |
| Database | **PostgreSQL** (Supabase) | ACID-compliant ledger storage |
| DB Driver | **pgx/v5** + **SQLC** | Type-safe, high-performance queries |
| Cache/Lock | **Redis** (Render) | Rate limiting, distributed locks, sessions, idempotency |
| Message Broker | **RabbitMQ** (CloudAMQP) | Event streaming with publisher confirms |
| Object Storage | **Cloudflare R2** | S3-compatible document storage |
| HTTP Router | **chi/v5** | Lightweight, composable middleware |
| Auth | **bsm/redislock** | Distributed lock algorithm |
| Container | **Docker** (scratch) | Minimal, secure production image |
| Deployment | **Render** | Auto-deploy from GitHub |

---

## API Endpoints

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| `GET` | `/v1/health` | Public | Deep health check with dependency status |
| `GET` | `/swagger/*` | Public | Interactive Swagger UI |
| `POST` | `/v1/auth/login` | Public | Login → get secure session cookie |
| `POST` | `/v1/auth/logout` | ✅ | Logout → revoke session |
| `GET` | `/v1/auth/me` | ✅ | Get current authenticated role |
| `POST` | `/v1/accounts` | ✅ admin/write | Create a ledger account |
| `GET` | `/v1/accounts` | ✅ any role | List accounts (paginated) |
| `GET` | `/v1/accounts/{id}` | ✅ any role | Get single account |
| `GET` | `/v1/accounts/{id}/balance` | ✅ any role | Compute real-time balance |
| `POST` | `/v1/transactions` | ✅ admin/write | Create double-entry transaction |
| `GET` | `/v1/transactions` | ✅ any role | List transactions (paginated) |
| `GET` | `/v1/transactions/{id}` | ✅ any role | Get transaction with postings |
| `POST` | `/v1/transactions/{id}/documents` | ✅ admin/write | Upload file to R2 |
| `GET` | `/v1/transactions/{id}/documents` | ✅ any role | List documents with download URLs |

> 📖 For detailed request/response payloads, see the [full API reference](docs/api.md).

---

## Quick Start

### Prerequisites

- **Docker** and **Docker Compose**
- **Go 1.22+**
- **golang-migrate**: `go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest`

### 1. Start Infrastructure

```bash
docker-compose up -d
```

This starts **Redis** and **RabbitMQ** locally. PostgreSQL is hosted on Supabase.

### 2. Configure Environment

```bash
cp .env.example .env
```

Fill in your credentials:
- `DATABASE_URL` — Supabase PostgreSQL connection string
- `REDIS_URL` — Redis connection string
- `RABBITMQ_URL` — RabbitMQ connection string
- `API_KEYS` — Your API keys in format `key:role,key:role`
- `R2_*` — Cloudflare R2 credentials (optional, for document uploads)

### 3. Run Migrations

```bash
migrate -path db/migrations -database "$DATABASE_URL" up
```

### 4. Start the Server

```bash
go run ./cmd/server/main.go
```

The server starts on `http://localhost:8080`.

### 5. Test It

```bash
# Health check (no auth)
curl http://localhost:8080/v1/health

# Create an account (with API key)
curl -X POST http://localhost:8080/v1/accounts \
  -H "Content-Type: application/json" \
  -H "X-API-Key: sk-admin-master-key-xyz" \
  -d '{"account_name": "Cash", "account_type": "asset", "currency": "INR"}'
```

---

## Transaction Flow (Step by Step)

Here's what happens when you `POST /v1/transactions`:

```
1. ✅ Validate request (balanced debits/credits, valid amounts)
2. 🔒 Acquire distributed lock on idempotency key (Redlock, 30s TTL)
3. 🔍 Check Redis for existing idempotency entry (replay if found)
4. 🗄️ BEGIN PostgreSQL transaction (SERIALIZABLE isolation)
5. 📝 INSERT transaction record
6. 📝 INSERT all posting records
7. 📤 INSERT outbox event (same ACID transaction)
8. ✅ COMMIT (deferred constraint trigger validates balance)
9. 💾 Cache idempotency key in Redis (24h TTL)
10. 🔓 Release distributed lock
```

The **Outbox Worker** (background goroutine) then:
```
1. Polls outbox table every 200ms (batch of 50)
2. Publishes each event to RabbitMQ (topic exchange)
3. Marks event as "published" on success
4. Retries with exponential backoff on failure (max 5 retries)
5. Marks as "failed" after exhausting retries (dead letter)
```

---

## Production Deployment

LedgerPro is deployed on **Render** with a minimal Docker image:

```dockerfile
# Multi-stage build → ~10MB final image
FROM golang:1.26-alpine AS builder
RUN CGO_ENABLED=0 go build -ldflags="-w -s" -o /go/bin/ledger_pro ./cmd/server/main.go

FROM scratch
COPY --from=builder /go/bin/ledger_pro /ledger_pro
ENTRYPOINT ["/ledger_pro"]
```

### Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `DATABASE_URL` | ✅ | — | PostgreSQL connection string |
| `REDIS_URL` | ✅ | — | Redis connection string |
| `RABBITMQ_URL` | ✅ | — | RabbitMQ AMQP URL |
| `API_KEYS` | ❌ | — | API keys (`key:role,key:role`). Empty = auth disabled |
| `SERVER_PORT` | ❌ | `8080` | HTTP server port |
| `SERVER_READ_TIMEOUT` | ❌ | `10s` | HTTP read timeout |
| `SERVER_WRITE_TIMEOUT` | ❌ | `30s` | HTTP write timeout |
| `IDEMPOTENCY_TTL` | ❌ | `24h` | How long idempotency keys are cached |
| `RATE_LIMIT_REQUESTS` | ❌ | `100` | Max requests per window |
| `RATE_LIMIT_WINDOW` | ❌ | `1m` | Sliding window duration |
| `R2_ACCOUNT_ID` | ❌ | — | Cloudflare R2 account ID |
| `R2_ACCESS_KEY_ID` | ❌ | — | R2 access key |
| `R2_SECRET_ACCESS_KEY` | ❌ | — | R2 secret key |
| `R2_BUCKET_NAME` | ❌ | — | R2 bucket name |
| `R2_ENDPOINT` | ❌ | — | R2 endpoint URL |
| `LOG_LEVEL` | ❌ | `info` | Log level (debug/info/warn/error) |

---

## Database Schema

```
┌─────────────────┐     ┌─────────────────────┐     ┌──────────────────┐
│    accounts      │     │    transactions       │     │    postings       │
├─────────────────┤     ├─────────────────────┤     ├──────────────────┤
│ id (BIGSERIAL)  │     │ id (UUID)            │◄────│ transaction_id   │
│ account_name    │◄────│ idempotency_key      │     │ account_id ──────┤►
│ account_type    │     │ description          │     │ amount (BIGINT)  │
│ currency        │     │ metadata (JSONB)     │     │ direction (ENUM) │
│ metadata (JSONB)│     │ created_at           │     │ created_at       │
│ created_at      │     └─────────────────────┘     └──────────────────┘
└─────────────────┘
        ▲                                                    │
        │            ┌───────────────────┐                   │
        └────────────│ Balance is always  │◄─────────────────┘
                     │ computed from      │
                     │ SUM(postings)      │
                     └───────────────────┘

┌─────────────────────┐     ┌─────────────────────┐
│   ledger_outbox      │     │   documents          │
├─────────────────────┤     ├─────────────────────┤
│ id (UUID)           │     │ id (UUID)            │
│ event_type          │     │ transaction_id       │
│ routing_key         │     │ filename             │
│ payload (JSONB)     │     │ content_type         │
│ status (ENUM)       │     │ size_bytes           │
│ retry_count         │     │ object_key (R2 path) │
│ last_error          │     │ created_at           │
│ created_at          │     └─────────────────────┘
│ published_at        │
└─────────────────────┘
```

### Integrity Guarantees

- **Immutability Triggers**: `UPDATE` and `DELETE` on `postings` and `transactions` raise exceptions
- **Balance Constraint**: A deferred constraint trigger on `postings` validates `SUM(debits) = SUM(credits)` at `COMMIT` time
- **Unique Constraints**: `account_name` and `idempotency_key` are unique

---

## License

MIT
