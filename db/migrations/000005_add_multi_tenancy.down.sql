BEGIN;

-- 1. Remove indexes
DROP INDEX IF EXISTS idx_ledger_outbox_tenant_id;
DROP INDEX IF EXISTS idx_documents_tenant_id;
DROP INDEX IF EXISTS idx_postings_tenant_id;
DROP INDEX IF EXISTS idx_transactions_tenant_id;
DROP INDEX IF EXISTS idx_accounts_tenant_id;
DROP INDEX IF EXISTS idx_users_tenant_id;

-- 2. Restore constraints and remove columns
ALTER TABLE ledger_outbox DROP COLUMN tenant_id;
ALTER TABLE documents DROP COLUMN tenant_id;
ALTER TABLE postings DROP COLUMN tenant_id;

ALTER TABLE transactions DROP CONSTRAINT transactions_tenant_idempotency_key_key;
ALTER TABLE transactions ADD CONSTRAINT transactions_idempotency_key_key UNIQUE(idempotency_key);
ALTER TABLE transactions DROP COLUMN tenant_id;

ALTER TABLE accounts DROP CONSTRAINT accounts_tenant_account_name_key;
ALTER TABLE accounts ADD CONSTRAINT accounts_account_name_key UNIQUE(account_name);
ALTER TABLE accounts DROP COLUMN tenant_id;

ALTER TABLE users DROP CONSTRAINT users_tenant_email_key;
ALTER TABLE users ADD CONSTRAINT users_email_key UNIQUE(email);
ALTER TABLE users DROP COLUMN tenant_id;

-- 3. Drop tenants table
DROP TABLE tenants;

COMMIT;
