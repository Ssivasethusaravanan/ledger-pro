-- ============================================================================
-- LedgerPro: Documents Table
-- Migration: 000003_add_documents (UP)
-- ============================================================================

BEGIN;

CREATE TABLE documents (
    id              UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    transaction_id  UUID            NOT NULL REFERENCES transactions(id) ON DELETE CASCADE,
    filename        TEXT            NOT NULL,
    content_type    TEXT            NOT NULL,
    size_bytes      BIGINT          NOT NULL,
    object_key      TEXT            NOT NULL UNIQUE,
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT now()
);

CREATE INDEX idx_documents_transaction_id ON documents(transaction_id);

COMMENT ON TABLE documents IS 'Metadata for files (receipts/invoices) attached to transactions and stored in R2.';

COMMIT;
