-- ============================================================================
-- LedgerPro: Transactional Outbox Table
-- Migration: 000002_add_outbox (DOWN)
-- ============================================================================

BEGIN;

DROP TABLE IF EXISTS ledger_outbox;
DROP TYPE IF EXISTS outbox_status;

COMMIT;
