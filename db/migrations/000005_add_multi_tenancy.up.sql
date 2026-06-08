BEGIN;

-- 1. Create tenants table
CREATE TABLE tenants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- 2. Create a default tenant for existing data
INSERT INTO tenants (id, name) VALUES ('00000000-0000-0000-0000-000000000000', 'Default Tenant');

-- 3. Add tenant_id to users
ALTER TABLE users ADD COLUMN tenant_id UUID REFERENCES tenants(id) ON DELETE RESTRICT;
UPDATE users SET tenant_id = '00000000-0000-0000-0000-000000000000' WHERE tenant_id IS NULL;
ALTER TABLE users ALTER COLUMN tenant_id SET NOT NULL;
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_email_key;
ALTER TABLE users ADD CONSTRAINT users_tenant_email_key UNIQUE (tenant_id, email);

-- 4. Add tenant_id to accounts
ALTER TABLE accounts ADD COLUMN tenant_id UUID REFERENCES tenants(id) ON DELETE RESTRICT;
UPDATE accounts SET tenant_id = '00000000-0000-0000-0000-000000000000' WHERE tenant_id IS NULL;
ALTER TABLE accounts ALTER COLUMN tenant_id SET NOT NULL;
ALTER TABLE accounts DROP CONSTRAINT IF EXISTS accounts_account_name_key;
ALTER TABLE accounts ADD CONSTRAINT accounts_tenant_account_name_key UNIQUE (tenant_id, account_name);

-- 5. Add tenant_id to transactions
ALTER TABLE transactions ADD COLUMN tenant_id UUID REFERENCES tenants(id) ON DELETE RESTRICT;
UPDATE transactions SET tenant_id = '00000000-0000-0000-0000-000000000000' WHERE tenant_id IS NULL;
ALTER TABLE transactions ALTER COLUMN tenant_id SET NOT NULL;
ALTER TABLE transactions DROP CONSTRAINT IF EXISTS transactions_idempotency_key_key;
ALTER TABLE transactions ADD CONSTRAINT transactions_tenant_idempotency_key_key UNIQUE (tenant_id, idempotency_key);

-- 6. Add tenant_id to postings
ALTER TABLE postings ADD COLUMN tenant_id UUID REFERENCES tenants(id) ON DELETE RESTRICT;
UPDATE postings SET tenant_id = '00000000-0000-0000-0000-000000000000' WHERE tenant_id IS NULL;
ALTER TABLE postings ALTER COLUMN tenant_id SET NOT NULL;

-- 7. Add tenant_id to documents
ALTER TABLE documents ADD COLUMN tenant_id UUID REFERENCES tenants(id) ON DELETE RESTRICT;
UPDATE documents SET tenant_id = '00000000-0000-0000-0000-000000000000' WHERE tenant_id IS NULL;
ALTER TABLE documents ALTER COLUMN tenant_id SET NOT NULL;

-- 8. Add tenant_id to ledger_outbox
ALTER TABLE ledger_outbox ADD COLUMN tenant_id UUID REFERENCES tenants(id) ON DELETE RESTRICT;
UPDATE ledger_outbox SET tenant_id = '00000000-0000-0000-0000-000000000000' WHERE tenant_id IS NULL;
ALTER TABLE ledger_outbox ALTER COLUMN tenant_id SET NOT NULL;

-- Add indexes for tenant filtering
CREATE INDEX idx_users_tenant_id ON users(tenant_id);
CREATE INDEX idx_accounts_tenant_id ON accounts(tenant_id);
CREATE INDEX idx_transactions_tenant_id ON transactions(tenant_id);
CREATE INDEX idx_postings_tenant_id ON postings(tenant_id);
CREATE INDEX idx_documents_tenant_id ON documents(tenant_id);
CREATE INDEX idx_ledger_outbox_tenant_id ON ledger_outbox(tenant_id);

COMMIT;
