-- Migration: Add IAM policies tables

CREATE TABLE IF NOT EXISTS policies (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    description TEXT,
    type TEXT DEFAULT 'custom',
    document JSONB NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE,
    updated_at TIMESTAMP WITH TIME ZONE
);

CREATE TABLE IF NOT EXISTS user_policies (
    user_id BIGINT NOT NULL,
    policy_id BIGINT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE,
    PRIMARY KEY (user_id, policy_id),
    CONSTRAINT fk_user_policies_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT fk_user_policies_policy FOREIGN KEY (policy_id) REFERENCES policies(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS role_policies (
    role_id BIGINT NOT NULL,
    policy_id BIGINT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE,
    PRIMARY KEY (role_id, policy_id),
    CONSTRAINT fk_role_policies_role FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE CASCADE,
    CONSTRAINT fk_role_policies_policy FOREIGN KEY (policy_id) REFERENCES policies(id) ON DELETE CASCADE
);

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_policies_name ON policies(name);
