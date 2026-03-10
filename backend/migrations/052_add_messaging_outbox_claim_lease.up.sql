ALTER TABLE messaging_outbox
    ADD COLUMN IF NOT EXISTS claim_owner TEXT NULL,
    ADD COLUMN IF NOT EXISTS claim_expires_at TIMESTAMPTZ NULL;

CREATE INDEX IF NOT EXISTS idx_messaging_outbox_claim_lease
    ON messaging_outbox (claim_expires_at)
    WHERE published_at IS NULL;

