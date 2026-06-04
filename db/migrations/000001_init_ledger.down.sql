-- ============================================================================
-- LedgerPro: Double-Entry Financial Ledger Schema
-- Migration: 000001_init_ledger (DOWN)
-- 
-- Drops all objects in reverse dependency order.
-- ============================================================================

BEGIN;

-- 1. Drop triggers first (they depend on functions and tables)
DROP TRIGGER IF EXISTS trg_check_balance         ON postings;
DROP TRIGGER IF EXISTS trg_postings_immutable     ON postings;
DROP TRIGGER IF EXISTS trg_transactions_immutable ON transactions;

-- 2. Drop functions
DROP FUNCTION IF EXISTS fn_check_transaction_balance();
DROP FUNCTION IF EXISTS fn_prevent_mutation();

-- 3. Drop tables (child → parent order)
DROP TABLE IF EXISTS postings;
DROP TABLE IF EXISTS transactions;
DROP TABLE IF EXISTS accounts;

-- 4. Drop enum types
DROP TYPE IF EXISTS posting_direction;
DROP TYPE IF EXISTS account_type;

COMMIT;
