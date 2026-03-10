ALTER TABLE epr_registration_requests
    ADD COLUMN IF NOT EXISTS approved_at TIMESTAMP WITH TIME ZONE,
    ADD COLUMN IF NOT EXISTS expires_at TIMESTAMP WITH TIME ZONE,
    ADD COLUMN IF NOT EXISTS lifecycle_status VARCHAR(32) NOT NULL DEFAULT 'active',
    ADD COLUMN IF NOT EXISTS suspension_reason TEXT,
    ADD COLUMN IF NOT EXISTS last_reviewed_at TIMESTAMP WITH TIME ZONE;

ALTER TABLE epr_registration_requests
    DROP CONSTRAINT IF EXISTS chk_epr_registration_requests_lifecycle_status;

ALTER TABLE epr_registration_requests
    ADD CONSTRAINT chk_epr_registration_requests_lifecycle_status
        CHECK (lifecycle_status IN ('active', 'expiring', 'expired', 'suspended'));

CREATE INDEX IF NOT EXISTS idx_epr_registration_requests_lifecycle_status
    ON epr_registration_requests (lifecycle_status, updated_at DESC);

CREATE INDEX IF NOT EXISTS idx_epr_registration_requests_expires_at
    ON epr_registration_requests (expires_at)
    WHERE expires_at IS NOT NULL;

UPDATE epr_registration_requests
SET approved_at = COALESCE(approved_at, decided_at, updated_at),
    last_reviewed_at = COALESCE(last_reviewed_at, decided_at, updated_at)
WHERE status = 'approved';
