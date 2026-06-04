-- ============================================================================
-- LedgerPro: Double-Entry Financial Ledger Schema
-- Migration: 000001_init_ledger (UP)
-- 
-- Invariants enforced:
--   1. Every posting amount > 0 (direction expressed via enum, not sign)
--   2. SUM(debits) = SUM(credits) per transaction (deferred constraint trigger)
--   3. postings and transactions are immutable (no UPDATE/DELETE)
--   4. idempotency_key is unique per transaction
-- ============================================================================

BEGIN;

-- ---------------------------------------------------------------------------
-- 1. ENUM TYPES
-- ---------------------------------------------------------------------------

CREATE TYPE account_type AS ENUM (
    'asset',
    'liability',
    'equity',
    'revenue',
    'expense'
);

CREATE TYPE posting_direction AS ENUM (
    'debit',
    'credit'
);

-- ---------------------------------------------------------------------------
-- 2. TABLES
-- ---------------------------------------------------------------------------

CREATE TABLE accounts (
    id            BIGSERIAL       PRIMARY KEY,
    account_name  TEXT            NOT NULL UNIQUE,
    account_type  account_type    NOT NULL,
    currency      TEXT            NOT NULL DEFAULT 'INR',
    metadata      JSONB           DEFAULT '{}',
    created_at    TIMESTAMPTZ     NOT NULL DEFAULT now()
);

COMMENT ON TABLE accounts IS 'Chart of accounts — each row is a unique ledger account.';
COMMENT ON COLUMN accounts.currency IS 'ISO 4217 currency code. Amounts are stored in smallest unit (e.g., paisa for INR).';

CREATE TABLE transactions (
    id                UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    idempotency_key   TEXT            NOT NULL UNIQUE,
    description       TEXT            NOT NULL DEFAULT '',
    metadata          JSONB           DEFAULT '{}',
    created_at        TIMESTAMPTZ     NOT NULL DEFAULT now()
);

COMMENT ON TABLE transactions IS 'A financial transaction — groups one or more postings that must balance.';
COMMENT ON COLUMN transactions.idempotency_key IS 'Client-provided key to prevent duplicate transaction creation.';

CREATE TABLE postings (
    id                BIGSERIAL           PRIMARY KEY,
    transaction_id    UUID                NOT NULL REFERENCES transactions(id) ON DELETE RESTRICT,
    account_id        BIGINT              NOT NULL REFERENCES accounts(id) ON DELETE RESTRICT,
    amount            BIGINT              NOT NULL CHECK (amount > 0),
    direction         posting_direction   NOT NULL,
    created_at        TIMESTAMPTZ         NOT NULL DEFAULT now()
);

COMMENT ON TABLE postings IS 'Individual debit/credit legs of a transaction. Append-only — never updated or deleted.';
COMMENT ON COLUMN postings.amount IS 'Positive integer in smallest currency unit (e.g., paisa). Direction is expressed via the direction enum.';

-- ---------------------------------------------------------------------------
-- 3. INDEXES
-- ---------------------------------------------------------------------------

CREATE INDEX idx_postings_transaction_id ON postings (transaction_id);
CREATE INDEX idx_postings_account_id     ON postings (account_id);
CREATE INDEX idx_transactions_created_at ON transactions (created_at DESC);
CREATE INDEX idx_accounts_created_at     ON accounts (created_at DESC);

-- ---------------------------------------------------------------------------
-- 4. IMMUTABILITY TRIGGERS
-- ---------------------------------------------------------------------------

-- Prevent any UPDATE or DELETE on the postings table.
CREATE OR REPLACE FUNCTION fn_prevent_mutation()
RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION 'immutable table — UPDATE and DELETE operations are forbidden on %', TG_TABLE_NAME
        USING ERRCODE = 'restrict_violation';
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_postings_immutable
    BEFORE UPDATE OR DELETE ON postings
    FOR EACH ROW
    EXECUTE FUNCTION fn_prevent_mutation();

CREATE TRIGGER trg_transactions_immutable
    BEFORE UPDATE OR DELETE ON transactions
    FOR EACH ROW
    EXECUTE FUNCTION fn_prevent_mutation();

-- ---------------------------------------------------------------------------
-- 5. BALANCED TRANSACTION CONSTRAINT TRIGGER (DEFERRED)
-- ---------------------------------------------------------------------------
-- This fires at COMMIT time. For each affected transaction, it asserts that
-- the sum of debit amounts equals the sum of credit amounts.
-- Using DEFERRABLE INITIALLY DEFERRED so all postings for a transaction
-- can be inserted before the check runs.
-- ---------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION fn_check_transaction_balance()
RETURNS TRIGGER AS $$
DECLARE
    v_debit_sum  BIGINT;
    v_credit_sum BIGINT;
BEGIN
    SELECT
        COALESCE(SUM(CASE WHEN direction = 'debit'  THEN amount ELSE 0 END), 0),
        COALESCE(SUM(CASE WHEN direction = 'credit' THEN amount ELSE 0 END), 0)
    INTO v_debit_sum, v_credit_sum
    FROM postings
    WHERE transaction_id = NEW.transaction_id;

    IF v_debit_sum <> v_credit_sum THEN
        RAISE EXCEPTION 'transaction % is unbalanced: debits (%) ≠ credits (%)',
            NEW.transaction_id, v_debit_sum, v_credit_sum
            USING ERRCODE = 'check_violation';
    END IF;

    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE CONSTRAINT TRIGGER trg_check_balance
    AFTER INSERT ON postings
    DEFERRABLE INITIALLY DEFERRED
    FOR EACH ROW
    EXECUTE FUNCTION fn_check_transaction_balance();

COMMIT;
