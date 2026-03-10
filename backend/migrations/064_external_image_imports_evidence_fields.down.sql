ALTER TABLE external_image_imports
    DROP COLUMN IF EXISTS source_image_digest,
    DROP COLUMN IF EXISTS sbom_evidence_json,
    DROP COLUMN IF EXISTS sbom_summary_json,
    DROP COLUMN IF EXISTS scan_summary_json,
    DROP COLUMN IF EXISTS policy_snapshot_json,
    DROP COLUMN IF EXISTS policy_reasons_json,
    DROP COLUMN IF EXISTS policy_decision;
