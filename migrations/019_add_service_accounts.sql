-- Service Accounts: programmatic identities for API access (AWS IAM Access Keys equivalent)

CREATE TABLE IF NOT EXISTS service_accounts (
    id              SERIAL PRIMARY KEY,
    name            VARCHAR(255) NOT NULL UNIQUE,
    description     TEXT DEFAULT '',
    project_id      INTEGER REFERENCES projects(id) ON DELETE SET NULL,
    created_by_id   INTEGER NOT NULL DEFAULT 0,
    access_key_id   VARCHAR(32) NOT NULL UNIQUE,
    secret_hash     TEXT NOT NULL,
    is_active       BOOLEAN NOT NULL DEFAULT TRUE,
    last_used_at    TIMESTAMP,
    expires_at      TIMESTAMP,
    created_at      TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_service_accounts_project_id ON service_accounts(project_id);
CREATE INDEX IF NOT EXISTS idx_service_accounts_access_key ON service_accounts(access_key_id);

-- Join table: service_account <-> role (M2M)
CREATE TABLE IF NOT EXISTS service_account_roles (
    service_account_id  INTEGER NOT NULL REFERENCES service_accounts(id) ON DELETE CASCADE,
    role_id             INTEGER NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    created_at          TIMESTAMP NOT NULL DEFAULT NOW(),
    PRIMARY KEY (service_account_id, role_id)
);

-- Join table: service_account <-> policy (M2M)
CREATE TABLE IF NOT EXISTS service_account_policies (
    service_account_id  INTEGER NOT NULL REFERENCES service_accounts(id) ON DELETE CASCADE,
    policy_id           INTEGER NOT NULL REFERENCES policies(id) ON DELETE CASCADE,
    created_at          TIMESTAMP NOT NULL DEFAULT NOW(),
    PRIMARY KEY (service_account_id, policy_id)
);
