CREATE TABLE IF NOT EXISTS system_bootstrap_state (
    id UUID PRIMARY KEY,
    status VARCHAR(50) NOT NULL CHECK (status IN ('not_started', 'admin_password_issued', 'setup_in_progress', 'setup_complete')),
    setup_required BOOLEAN NOT NULL DEFAULT true,
    seed_version INTEGER NOT NULL DEFAULT 1,
    initial_admin_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    initial_admin_password_issued_at TIMESTAMPTZ,
    setup_completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_system_bootstrap_state_setup_required ON system_bootstrap_state(setup_required);

CREATE UNIQUE INDEX IF NOT EXISTS idx_system_bootstrap_state_single_row
    ON system_bootstrap_state ((true));
