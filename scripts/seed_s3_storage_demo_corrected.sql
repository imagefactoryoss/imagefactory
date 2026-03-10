-- ============================================================================
-- S3 STORAGE DEMO DATA (CORRECTED)
-- Demonstrates the S3+Database architecture for container registry storage
-- ============================================================================

-- Get required IDs first
WITH sample_data AS (
    SELECT 
        c.id as company_id,
        o.id as org_unit_id, 
        u.id as user_id,
        cr.id as registry_id,
        i.id as image_id
    FROM companies c
    CROSS JOIN org_units o
    CROSS JOIN users u  
    CROSS JOIN container_registries cr
    CROSS JOIN catalog_images i
    LIMIT 1
),

-- Insert a sample S3 storage backend
storage_backend_insert AS (
    INSERT INTO storage_backends (
        id,
        company_id,
        org_unit_id,
        name,
        backend_type,
        is_default,
        is_active,
        s3_bucket,
        s3_region,
        s3_access_key_id,
        s3_secret_access_key,
        s3_endpoint_url,
        s3_force_path_style,
        default_storage_class,
        lifecycle_policies,
        encryption_config,
        description,
        created_by
    ) SELECT 
        'a1b2c3d4-e5f6-7890-1234-567890abcdef'::uuid,
        sample_data.company_id,
        sample_data.org_unit_id,
        'Primary S3 Storage',
        's3',
        true,
        true,
        'image-factory-blobs-prod',
        'us-west-2', 
        'AKIA...',
        '[ENCRYPTED]',
        's3.us-west-2.amazonaws.com',
        false,
        'STANDARD',
        '{
            "transition_to_ia": "30d",
            "transition_to_glacier": "90d", 
            "transition_to_deep_archive": "365d",
            "expiration": "2555d"
        }'::jsonb,
        '{"enabled": true, "algorithm": "AES256"}'::jsonb,
        'Primary S3 storage backend for container registry blobs',
        sample_data.user_id
    FROM sample_data
    ON CONFLICT (id) DO NOTHING
    RETURNING id
),

-- Sample blob data (represents actual blob storage in S3)
blob_insert AS (
    INSERT INTO image_blobs (
        id,
        digest,
        media_type,
        size_bytes,
        compression,
        deduplication_key,
        reference_count,
        last_accessed_at,
        metadata
    ) VALUES 
    (
        'b1234567-89ab-cdef-0123-456789abcdef'::uuid,
        'sha256:8be619267e30a82be32eb87a08b70da1cc01f28c36f45b1b7c6b61b2e8c7f50f',
        'application/vnd.docker.image.rootfs.diff.tar.gzip',
        52428800, -- 50MB
        'gzip',
        'sha256:8be619267e30a82be32eb87a08b70da1cc01f28c36f45b1b7c6b61b2e8c7f50f',
        3, -- Referenced by 3 images
        NOW() - INTERVAL '2 hours',
        '{
            "uploaded_at": "2024-01-15T10:30:00Z",
            "uploader": "image-builder-service",
            "source_image": "ubuntu:22.04"
        }'::jsonb
    ),
    (
        'b2345678-9abc-def0-1234-56789abcdef0'::uuid,
        'sha256:6b4cf1b7e1e2e5c5a0c8f9d8e7f6g5h4i3j2k1l0m9n8o7p6q5r4s3t2u1v0w9x8',
        'application/vnd.docker.image.rootfs.diff.tar.gzip',
        25165824, -- 24MB  
        'gzip',
        'sha256:6b4cf1b7e1e2e5c5a0c8f9d8e7f6g5h4i3j2k1l0m9n8o7p6q5r4s3t2u1v0w9x8',
        1, -- Referenced by 1 image
        NOW() - INTERVAL '1 hour',
        '{
            "uploaded_at": "2024-01-15T11:45:00Z",
            "uploader": "image-builder-service",
            "source_image": "alpine:3.19"
        }'::jsonb
    ) ON CONFLICT (id) DO NOTHING
    RETURNING id
),

-- Sample manifest data (OCI/Docker manifests stored in database)
manifest_insert AS (
    INSERT INTO image_manifests (
        id,
        image_id,
        media_type,
        digest,
        manifest_content,
        platform,
        size_bytes
    ) SELECT 
        'm1234567-89ab-cdef-0123-456789abcdef'::uuid,
        sample_data.image_id,
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
        'linux/amd64',
        77594624
    FROM sample_data
    ON CONFLICT (id) DO NOTHING
    RETURNING id
),

-- Link layers to blobs (many-to-many relationship for deduplication)
layer_blob_insert AS (
    INSERT INTO image_layer_blobs (
        id,
        manifest_id,
        blob_id,
        layer_order,
        blob_digest
    ) VALUES 
    (
        'l1234567-89ab-cdef-0123-456789abcdef'::uuid,
        'm1234567-89ab-cdef-0123-456789abcdef'::uuid,
        'b1234567-89ab-cdef-0123-456789abcdef'::uuid, 
        0,
        'sha256:8be619267e30a82be32eb87a08b70da1cc01f28c36f45b1b7c6b61b2e8c7f50f'
    ),
    (
        'l2345678-9abc-def0-1234-56789abcdef0'::uuid,
        'm1234567-89ab-cdef-0123-456789abcdef'::uuid,
        'b2345678-9abc-def0-1234-56789abcdef0'::uuid,
        1,
        'sha256:6b4cf1b7e1e2e5c5a0c8f9d8e7f6g5h4i3j2k1l0m9n8o7p6q5r4s3t2u1v0w9x8'
    ) ON CONFLICT (id) DO NOTHING
    RETURNING id
),

-- Storage metrics for cost tracking
metrics_insert AS (
    INSERT INTO storage_metrics (
        id,
        storage_backend_id,
        metric_date,
        total_objects,
        total_size_bytes,
        metrics_data
    ) VALUES (
        's1234567-89ab-cdef-0123-456789abcdef'::uuid,
        'a1b2c3d4-e5f6-7890-1234-567890abcdef'::uuid,
        CURRENT_DATE,
        2,
        77594624, -- Total size of our 2 blobs
        '{
            "storage_class_breakdown": {
                "standard": 2,
                "intelligent_tiering": 0,
                "glacier": 0
            },
            "cost_breakdown": {
                "storage": 1.79,
                "requests": 0.004,
                "data_transfer": 0.02
            },
            "bandwidth_usage": {
                "ingress_bytes": 77594624,
                "egress_bytes": 155189248
            }
        }'::jsonb
    ) ON CONFLICT (id) DO NOTHING
    RETURNING id
),

-- Update catalog image metadata to link to the manifest and storage backend
image_update AS (
    UPDATE catalog_images
    SET 
        size_bytes = 77594624,
        metadata = COALESCE(catalog_images.metadata, '{}'::jsonb)
            || jsonb_build_object(
                'primary_manifest_id', 'm1234567-89ab-cdef-0123-456789abcdef',
                'total_blob_size', 77594624,
                'unique_blob_size', 77594624,
                'blob_count', 2,
                'storage_backend_id', 'a1b2c3d4-e5f6-7890-1234-567890abcdef',
                'storage_cost_estimate', 1.79,
                'transfer_cost_estimate', 0.02
            )
    FROM sample_data
    WHERE catalog_images.id = sample_data.image_id
    RETURNING catalog_images.id
),

-- Update container_registries to use S3 storage backend
registry_update AS (
    UPDATE container_registries 
    SET 
        storage_backend_id = 'a1b2c3d4-e5f6-7890-1234-567890abcdef'::uuid,
        storage_prefix = 'registry-prod/',
        blob_mount_enabled = true,
        cross_repository_blob_mount = true,
        gc_policy = '{
            "enabled": true,
            "retention_days": 90,
            "keep_minimum_tags": 5,
            "delete_untagged_after_days": 7
        }'::jsonb
    FROM sample_data
    WHERE container_registries.id = sample_data.registry_id
    RETURNING container_registries.id
)

SELECT 'S3 Storage Demo Data Loaded Successfully!' as status;
