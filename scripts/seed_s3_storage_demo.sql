-- ============================================================================
-- S3 STORAGE DEMO DATA
-- Demonstrates the S3+Database architecture for container registry storage
-- ============================================================================

-- Insert a sample S3 storage backend
INSERT INTO storage_backends (
    id,
    name,
    backend_type,
    config,
    is_primary,
    is_active,
    encryption_enabled,
    lifecycle_policy
) VALUES (
    'a1b2c3d4-e5f6-7890-1234-567890abcdef',
    'Primary S3 Storage',
    's3',
    '{
        "bucket": "image-factory-blobs-prod",
        "region": "us-west-2",
        "endpoint": "s3.us-west-2.amazonaws.com",
        "access_key_id": "AKIA...",
        "secret_access_key": "[ENCRYPTED]",
        "path_style": false,
        "multipart_threshold": "100MB",
        "multipart_chunk_size": "32MB"
    }'::jsonb,
    true,
    true,
    true,
    '{
        "transition_to_ia": "30d",
        "transition_to_glacier": "90d", 
        "transition_to_deep_archive": "365d",
        "expiration": "2555d"
    }'::jsonb
) ON CONFLICT (id) DO NOTHING;

-- Sample blob data (represents actual blob storage in S3)
INSERT INTO image_blobs (
    id,
    digest,
    media_type,
    size_bytes,
    storage_backend_id,
    storage_path,
    compression_type,
    deduplication_key,
    reference_count,
    storage_tier,
    last_accessed_at,
    metadata
) VALUES 
(
    'b1234567-89ab-cdef-0123-456789abcdef',
    'sha256:8be619267e30a82be32eb87a08b70da1cc01f28c36f45b1b7c6b61b2e8c7f50f',
    'application/vnd.docker.image.rootfs.diff.tar.gzip',
    52428800, -- 50MB
    'a1b2c3d4-e5f6-7890-1234-567890abcdef',
    'blobs/sha256/8b/8be619267e30a82be32eb87a08b70da1cc01f28c36f45b1b7c6b61b2e8c7f50f',
    'gzip',
    'sha256:8be619267e30a82be32eb87a08b70da1cc01f28c36f45b1b7c6b61b2e8c7f50f',
    3, -- Referenced by 3 images
    'standard',
    NOW() - INTERVAL '2 hours',
    '{
        "uploaded_at": "2024-01-15T10:30:00Z",
        "uploader": "image-builder-service",
        "source_image": "ubuntu:22.04"
    }'::jsonb
),
(
    'b2345678-9abc-def0-1234-56789abcdef0',
    'sha256:6b4cf1b7e1e2e5c5a0c8f9d8e7f6g5h4i3j2k1l0m9n8o7p6q5r4s3t2u1v0w9x8',
    'application/vnd.docker.image.rootfs.diff.tar.gzip',
    25165824, -- 24MB  
    'a1b2c3d4-e5f6-7890-1234-567890abcdef',
    'blobs/sha256/6b/6b4cf1b7e1e2e5c5a0c8f9d8e7f6g5h4i3j2k1l0m9n8o7p6q5r4s3t2u1v0w9x8',
    'gzip',
    'sha256:6b4cf1b7e1e2e5c5a0c8f9d8e7f6g5h4i3j2k1l0m9n8o7p6q5r4s3t2u1v0w9x8',
    1, -- Referenced by 1 image
    'standard',
    NOW() - INTERVAL '1 hour',
    '{
        "uploaded_at": "2024-01-15T11:45:00Z",
        "uploader": "image-builder-service",
        "source_image": "alpine:3.19"
    }'::jsonb
) ON CONFLICT (id) DO NOTHING;

-- Sample manifest data (OCI/Docker manifests stored in database)
INSERT INTO image_manifests (
    id,
    image_id,
    manifest_type,
    digest,
    raw_manifest,
    parsed_manifest,
    platform,
    size_bytes,
    layer_count,
    storage_backend_id,
    storage_path
) VALUES (
    'm1234567-89ab-cdef-0123-456789abcdef',
    (SELECT id FROM catalog_images LIMIT 1), -- Reference existing image
    'application/vnd.docker.distribution.manifest.v2+json',
    'sha256:m1a2b3c4d5e6f7g8h9i0j1k2l3m4n5o6p7q8r9s0t1u2v3w4x5y6z7a8b9c0d1e2',
    '{
        "schemaVersion": 2,
        "mediaType": "application/vnd.docker.distribution.manifest.v2+json",
        "config": {
            "mediaType": "application/vnd.docker.container.image.v1+json",
            "size": 1469,
            "digest": "sha256:c1f2g3h4i5j6k7l8m9n0o1p2q3r4s5t6u7v8w9x0y1z2a3b4c5d6e7f8g9h0i1j2"
        },
        "layers": [
            {
                "mediaType": "application/vnd.docker.image.rootfs.diff.tar.gzip",
                "size": 52428800,
                "digest": "sha256:8be619267e30a82be32eb87a08b70da1cc01f28c36f45b1b7c6b61b2e8c7f50f"
            },
            {
                "mediaType": "application/vnd.docker.image.rootfs.diff.tar.gzip", 
                "size": 25165824,
                "digest": "sha256:6b4cf1b7e1e2e5c5a0c8f9d8e7f6g5h4i3j2k1l0m9n8o7p6q5r4s3t2u1v0w9x8"
            }
        ]
    }'::jsonb,
    '{
        "total_size": 77594624,
        "layer_digests": [
            "sha256:8be619267e30a82be32eb87a08b70da1cc01f28c36f45b1b7c6b61b2e8c7f50f",
            "sha256:6b4cf1b7e1e2e5c5a0c8f9d8e7f6g5h4i3j2k1l0m9n8o7p6q5r4s3t2u1v0w9x8"
        ],
        "config_digest": "sha256:c1f2g3h4i5j6k7l8m9n0o1p2q3r4s5t6u7v8w9x0y1z2a3b4c5d6e7f8g9h0i1j2"
    }'::jsonb,
    'linux/amd64',
    77594624,
    2,
    'a1b2c3d4-e5f6-7890-1234-567890abcdef',
    'manifests/sha256/m1/m1a2b3c4d5e6f7g8h9i0j1k2l3m4n5o6p7q8r9s0t1u2v3w4x5y6z7a8b9c0d1e2'
) ON CONFLICT (id) DO NOTHING;

-- Link layers to blobs (many-to-many relationship for deduplication)
INSERT INTO image_layer_blobs (
    id,
    manifest_id,
    blob_id,
    layer_index,
    layer_digest,
    layer_media_type,
    layer_size
) VALUES 
(
    'l1234567-89ab-cdef-0123-456789abcdef',
    'm1234567-89ab-cdef-0123-456789abcdef',
    'b1234567-89ab-cdef-0123-456789abcdef', 
    0,
    'sha256:8be619267e30a82be32eb87a08b70da1cc01f28c36f45b1b7c6b61b2e8c7f50f',
    'application/vnd.docker.image.rootfs.diff.tar.gzip',
    52428800
),
(
    'l2345678-9abc-def0-1234-56789abcdef0',
    'm1234567-89ab-cdef-0123-456789abcdef',
    'b2345678-9abc-def0-1234-56789abcdef0',
    1,
    'sha256:6b4cf1b7e1e2e5c5a0c8f9d8e7f6g5h4i3j2k1l0m9n8o7p6q5r4s3t2u1v0w9x8',
    'application/vnd.docker.image.rootfs.diff.tar.gzip',
    25165824
) ON CONFLICT (id) DO NOTHING;

-- Storage metrics for cost tracking
INSERT INTO storage_metrics (
    id,
    storage_backend_id,
    metric_type,
    metric_date,
    total_objects,
    total_size_bytes,
    storage_class_breakdown,
    cost_breakdown,
    bandwidth_usage
) VALUES (
    's1234567-89ab-cdef-0123-456789abcdef',
    'a1b2c3d4-e5f6-7890-1234-567890abcdef',
    'daily',
    CURRENT_DATE,
    2,
    77594624, -- Total size of our 2 blobs
    '{
        "standard": 2,
        "intelligent_tiering": 0,
        "glacier": 0
    }'::jsonb,
    '{
        "storage": 1.79,
        "requests": 0.004,
        "data_transfer": 0.02
    }'::jsonb,
    '{
        "ingress_bytes": 77594624,
        "egress_bytes": 155189248
    }'::jsonb
) ON CONFLICT (id) DO NOTHING;

-- Update catalog image metadata to link to the manifest and storage backend
UPDATE catalog_images
SET
    size_bytes = 77594624,
    metadata = COALESCE(metadata, '{}'::jsonb)
        || jsonb_build_object(
            'primary_manifest_id', 'm1234567-89ab-cdef-0123-456789abcdef',
            'total_blob_size', 77594624,
            'unique_blob_size', 77594624,
            'blob_count', 2,
            'storage_backend_id', 'a1b2c3d4-e5f6-7890-1234-567890abcdef',
            'storage_cost_estimate', 1.79,
            'transfer_cost_estimate', 0.02
        )
WHERE id = (SELECT id FROM catalog_images LIMIT 1);

-- Update container_registries to use S3 storage backend
UPDATE container_registries 
SET 
    storage_backend_id = 'a1b2c3d4-e5f6-7890-1234-567890abcdef',
    storage_prefix = 'registry-prod/',
    blob_mount_enabled = true,
    cross_repository_blob_mount = true,
    gc_policy = '{
        "enabled": true,
        "retention_days": 90,
        "keep_minimum_tags": 5,
        "delete_untagged_after_days": 7
    }'::jsonb
WHERE id = (SELECT id FROM container_registries LIMIT 1);

-- ============================================================================
-- QUERY EXAMPLES: Demonstrating S3+Database Benefits
-- ============================================================================

-- 1. Find all blobs that can be deduplicated (used by multiple images)
-- SELECT digest, reference_count, size_bytes, 
--        size_bytes * (reference_count - 1) as savings_bytes
-- FROM image_blobs 
-- WHERE reference_count > 1
-- ORDER BY savings_bytes DESC;

-- 2. Calculate storage costs by registry
-- SELECT cr.name as registry_name,
--        COUNT(DISTINCT ib.id) as blob_count,
--        SUM(ib.size_bytes) as total_size,
--        SUM(i.storage_cost_estimate) as monthly_cost
-- FROM container_registries cr
-- JOIN catalog_images i ON (i.metadata->>'storage_backend_id')::uuid = cr.storage_backend_id
-- JOIN image_manifests im ON im.image_id = i.id  
-- JOIN image_layer_blobs ilb ON ilb.manifest_id = im.id
-- JOIN image_blobs ib ON ib.id = ilb.blob_id
-- GROUP BY cr.id, cr.name;

-- 3. Find orphaned blobs (no longer referenced by any manifest)
-- SELECT ib.digest, ib.size_bytes, ib.storage_path
-- FROM image_blobs ib
-- LEFT JOIN image_layer_blobs ilb ON ilb.blob_id = ib.id
-- WHERE ilb.id IS NULL;

-- 4. Storage efficiency report
-- SELECT 
--     'Total Blob Storage' as metric,
--     SUM(size_bytes) as bytes,
--     COUNT(*) as count
-- FROM image_blobs
-- UNION ALL
-- SELECT 
--     'Deduplicated Storage Savings',
--     SUM(size_bytes * (reference_count - 1)),
--     SUM(reference_count - 1)
-- FROM image_blobs 
-- WHERE reference_count > 1;
