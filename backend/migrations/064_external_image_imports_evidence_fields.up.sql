ALTER TABLE external_image_imports
    ADD COLUMN IF NOT EXISTS policy_decision VARCHAR(32),
    ADD COLUMN IF NOT EXISTS policy_reasons_json TEXT,
    ADD COLUMN IF NOT EXISTS policy_snapshot_json TEXT,
    ADD COLUMN IF NOT EXISTS scan_summary_json TEXT,
    ADD COLUMN IF NOT EXISTS sbom_summary_json TEXT,
    ADD COLUMN IF NOT EXISTS sbom_evidence_json TEXT,
    ADD COLUMN IF NOT EXISTS source_image_digest VARCHAR(255);
