# LedgerPro API Reference

This document outlines the core endpoints for the LedgerPro system. 

## Authentication
LedgerPro supports two methods of authentication:
1. **Server-to-Server**: Include your API Key in the `X-API-Key` header or as a `Bearer` token in the `Authorization` header.
2. **Browser (Web UI)**: Use the `POST /v1/auth/login` endpoint to receive a secure HTTP-only `session_token` cookie. The browser will automatically attach this cookie to subsequent requests.

### Authentication Endpoints

#### `POST /v1/auth/login`
Authenticates a user via API key and sets a secure `session_token` cookie.
**Request Body (JSON):**
```json
{
  "api_key": "sk-admin-xyz"
}
```
**Response (200 OK):**
```json
{
  "role": "admin"
}
```

#### `POST /v1/auth/logout`
Revokes the session and clears the cookie.
**Response (204 No Content)**

#### `GET /v1/auth/me`
Returns the current authenticated role.
**Response (200 OK):**
```json
{
  "role": "admin"
}
```

---

## Accounts

#### `POST /v1/accounts`
Create a new ledger account.
**Request Body:**
```json
{
  "name": "Cash",
  "type": "asset",
  "description": "Main operating bank account",
  "currency": "USD"
}
```
**Response (201 Created):**
```json
{
  "id": "e44...76f",
  "name": "Cash",
  "type": "asset",
  "description": "Main operating bank account",
  "currency": "USD",
  "created_at": "2024-05-15T10:00:00Z"
}
```

#### `GET /v1/accounts`
List all ledger accounts (supports pagination via `?cursor=` and `?limit=`).
**Response (200 OK):** Array of account objects.

#### `GET /v1/accounts/{id}/balance`
Get the mathematically computed balance for a specific account.
**Response (200 OK):**
```json
{
  "account_id": "e44...76f",
  "balance": "15000",
  "currency": "USD"
}
```

---

## Transactions

#### `POST /v1/transactions`
Record a double-entry transaction. Must include an Idempotency-Key header.
**Headers:**
- `Idempotency-Key: <uuid>` (Required)

**Request Body:**
```json
{
  "reference": "INV-2024-001",
  "description": "Client invoice payment",
  "postings": [
    {
      "account_id": "<cash_account_id>",
      "amount": "150.00",
      "direction": "debit"
    },
    {
      "account_id": "<revenue_account_id>",
      "amount": "150.00",
      "direction": "credit"
    }
  ]
}
```
**Response (201 Created):** Transaction details with computed posting entries.

---

## Documents (Cloudflare R2 Integration)

#### `POST /v1/transactions/{id}/documents`
Attach a file (receipt, invoice) to a transaction. Use `multipart/form-data`.
**Form Fields:**
- `file`: The binary file being uploaded.
- `document_type`: String (e.g., "receipt", "invoice").

#### `GET /v1/transactions/{id}/documents`
List all documents attached to a specific transaction, returning presigned download URLs.
**Response (200 OK):**
```json
[
  {
    "id": "doc_uuid",
    "filename": "receipt.pdf",
    "file_size": 102400,
    "content_type": "application/pdf",
    "document_type": "receipt",
    "url": "https://<r2-domain>/sflix/receipt.pdf?X-Amz-Signature=..."
  }
]
```

## Error Handling
The API strictly follows **RFC 7807 (Problem Details for HTTP APIs)**. If an error occurs (e.g., debits do not equal credits), you will receive a response like this:
```json
{
  "type": "https://ledgerpro.local/probs/bad-request",
  "title": "Bad Request",
  "status": 400,
  "detail": "transaction postings do not balance (debits: 150.00, credits: 100.00)",
  "instance": "/v1/transactions",
  "trace_id": "e44...76f"
}
```
