DROP INDEX IF EXISTS idx_epr_registration_requests_expires_at;
DROP INDEX IF EXISTS idx_epr_registration_requests_lifecycle_status;

ALTER TABLE epr_registration_requests
    DROP CONSTRAINT IF EXISTS chk_epr_registration_requests_lifecycle_status;

ALTER TABLE epr_registration_requests
    DROP COLUMN IF EXISTS last_reviewed_at,
    DROP COLUMN IF EXISTS suspension_reason,
    DROP COLUMN IF EXISTS lifecycle_status,
    DROP COLUMN IF EXISTS expires_at,
    DROP COLUMN IF EXISTS approved_at;
