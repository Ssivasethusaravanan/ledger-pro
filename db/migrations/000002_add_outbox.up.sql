-- ============================================================================
-- LedgerPro: Transactional Outbox Table
-- Migration: 000002_add_outbox (UP)
--
-- The Outbox Pattern ensures At-Least-Once event delivery. Events are written
-- to this table within the same PostgreSQL ACID transaction as the ledger
-- postings. A background worker polls and publishes them to RabbitMQ.
-- ============================================================================

BEGIN;

-- ---------------------------------------------------------------------------
-- ENUM: outbox event status
-- ---------------------------------------------------------------------------

CREATE TYPE outbox_status AS ENUM (
    'pending',
    'published',
    'failed'
);

-- ---------------------------------------------------------------------------
-- TABLE: ledger_outbox
-- ---------------------------------------------------------------------------

CREATE TABLE ledger_outbox (
    id              UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type      TEXT            NOT NULL,
    routing_key     TEXT            NOT NULL DEFAULT '',
    payload         JSONB           NOT NULL,
    status          outbox_status   NOT NULL DEFAULT 'pending',
    retry_count     INT             NOT NULL DEFAULT 0,
    max_retries     INT             NOT NULL DEFAULT 5,
    last_error      TEXT            DEFAULT '',
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT now(),
    processed_at    TIMESTAMPTZ
);

COMMENT ON TABLE ledger_outbox IS 'Transactional outbox for guaranteed event delivery. Written within the same ACID TX as ledger postings.';

-- ---------------------------------------------------------------------------
-- INDEXES
-- ---------------------------------------------------------------------------

-- The worker queries pending events ordered by creation time
CREATE INDEX idx_outbox_status_created
    ON ledger_outbox (status, created_at ASC)
    WHERE status = 'pending';

-- Failed events for retry/dead-letter inspection
CREATE INDEX idx_outbox_failed
    ON ledger_outbox (status, retry_count)
    WHERE status = 'failed';

-- Immutability trigger: once published, outbox rows are immutable
-- (only status transitions from pending → published/failed are allowed via queries)
-- We enforce this at the application layer, not a trigger, because the worker
-- needs to UPDATE status. A trigger here would block the worker.

COMMIT;
