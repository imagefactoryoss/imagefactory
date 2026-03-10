ALTER TABLE catalog_image_metadata
    ADD COLUMN IF NOT EXISTS layers_evidence_status VARCHAR(16) NOT NULL DEFAULT 'unavailable',
    ADD COLUMN IF NOT EXISTS layers_evidence_build_id UUID,
    ADD COLUMN IF NOT EXISTS layers_evidence_updated_at TIMESTAMP WITH TIME ZONE,
    ADD COLUMN IF NOT EXISTS sbom_evidence_status VARCHAR(16) NOT NULL DEFAULT 'unavailable',
    ADD COLUMN IF NOT EXISTS sbom_evidence_build_id UUID,
    ADD COLUMN IF NOT EXISTS sbom_evidence_updated_at TIMESTAMP WITH TIME ZONE,
    ADD COLUMN IF NOT EXISTS vulnerability_evidence_status VARCHAR(16) NOT NULL DEFAULT 'unavailable',
    ADD COLUMN IF NOT EXISTS vulnerability_evidence_build_id UUID,
    ADD COLUMN IF NOT EXISTS vulnerability_evidence_updated_at TIMESTAMP WITH TIME ZONE;

ALTER TABLE catalog_image_metadata
    DROP CONSTRAINT IF EXISTS chk_catalog_image_metadata_layers_evidence_status,
    DROP CONSTRAINT IF EXISTS chk_catalog_image_metadata_sbom_evidence_status,
    DROP CONSTRAINT IF EXISTS chk_catalog_image_metadata_vulnerability_evidence_status;

ALTER TABLE catalog_image_metadata
    ADD CONSTRAINT chk_catalog_image_metadata_layers_evidence_status
        CHECK (layers_evidence_status IN ('fresh', 'stale', 'unavailable')),
    ADD CONSTRAINT chk_catalog_image_metadata_sbom_evidence_status
        CHECK (sbom_evidence_status IN ('fresh', 'stale', 'unavailable')),
    ADD CONSTRAINT chk_catalog_image_metadata_vulnerability_evidence_status
        CHECK (vulnerability_evidence_status IN ('fresh', 'stale', 'unavailable'));

UPDATE catalog_image_metadata m
SET
    layers_evidence_status = CASE
        WHEN EXISTS (SELECT 1 FROM catalog_image_layers l WHERE l.image_id = m.image_id) THEN 'stale'
        ELSE 'unavailable'
    END,
    layers_evidence_updated_at = CASE
        WHEN EXISTS (SELECT 1 FROM catalog_image_layers l WHERE l.image_id = m.image_id)
            THEN COALESCE(
                (SELECT MAX(le.updated_at) FROM catalog_image_layer_evidence le WHERE le.image_id = m.image_id),
                (SELECT MAX(l2.created_at) FROM catalog_image_layers l2 WHERE l2.image_id = m.image_id)
            )
        ELSE NULL
    END,
    sbom_evidence_status = CASE
        WHEN EXISTS (SELECT 1 FROM catalog_image_sbom s WHERE s.image_id = m.image_id) THEN 'stale'
        ELSE 'unavailable'
    END,
    sbom_evidence_updated_at = CASE
        WHEN EXISTS (SELECT 1 FROM catalog_image_sbom s WHERE s.image_id = m.image_id)
            THEN (SELECT MAX(s2.scan_timestamp) FROM catalog_image_sbom s2 WHERE s2.image_id = m.image_id)
        ELSE NULL
    END,
    vulnerability_evidence_status = CASE
        WHEN EXISTS (SELECT 1 FROM catalog_image_vulnerability_scans v WHERE v.image_id = m.image_id) THEN 'stale'
        ELSE 'unavailable'
    END,
    vulnerability_evidence_updated_at = CASE
        WHEN EXISTS (SELECT 1 FROM catalog_image_vulnerability_scans v WHERE v.image_id = m.image_id)
            THEN (
                SELECT MAX(COALESCE(v2.completed_at, v2.started_at))
                FROM catalog_image_vulnerability_scans v2
                WHERE v2.image_id = m.image_id
            )
        ELSE NULL
    END;
