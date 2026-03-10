DROP INDEX IF EXISTS idx_messaging_outbox_claim_lease;

ALTER TABLE messaging_outbox
    DROP COLUMN IF EXISTS claim_expires_at,
    DROP COLUMN IF EXISTS claim_owner;
