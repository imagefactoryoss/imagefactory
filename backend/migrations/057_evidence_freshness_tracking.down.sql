ALTER TABLE catalog_image_metadata
    DROP CONSTRAINT IF EXISTS chk_catalog_image_metadata_layers_evidence_status,
    DROP CONSTRAINT IF EXISTS chk_catalog_image_metadata_sbom_evidence_status,
    DROP CONSTRAINT IF EXISTS chk_catalog_image_metadata_vulnerability_evidence_status;

ALTER TABLE catalog_image_metadata
    DROP COLUMN IF EXISTS vulnerability_evidence_updated_at,
    DROP COLUMN IF EXISTS vulnerability_evidence_build_id,
    DROP COLUMN IF EXISTS vulnerability_evidence_status,
    DROP COLUMN IF EXISTS sbom_evidence_updated_at,
    DROP COLUMN IF EXISTS sbom_evidence_build_id,
    DROP COLUMN IF EXISTS sbom_evidence_status,
    DROP COLUMN IF EXISTS layers_evidence_updated_at,
    DROP COLUMN IF EXISTS layers_evidence_build_id,
    DROP COLUMN IF EXISTS layers_evidence_status;
