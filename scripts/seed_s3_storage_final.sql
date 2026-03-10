-- ============================================================================
-- S3 STORAGE DEMO DATA (FINAL VERSION)
-- Demonstrates the S3+Database architecture for container registry storage
-- ============================================================================

-- Get required IDs for demo data
DO $$
DECLARE
    sample_company_id UUID;
    sample_org_unit_id UUID;
    sample_user_id UUID;
    sample_registry_id UUID;
    sample_image_id UUID;
    storage_backend_id UUID := 'a1b2c3d4-e5f6-7890-1234-567890abcdef'::uuid;
    blob1_id UUID := 'b1234567-89ab-cdef-0123-456789abcdef'::uuid;
    blob2_id UUID := 'b2345678-9abc-def0-1234-56789abcdef0'::uuid;
    manifest_id UUID := 'm1234567-89ab-cdef-0123-456789abcdef'::uuid;
BEGIN
    -- Get sample IDs
    SELECT c.id, o.id, u.id INTO sample_company_id, sample_org_unit_id, sample_user_id
    FROM companies c
    CROSS JOIN org_units o  
    CROSS JOIN users u
    LIMIT 1;
    
    SELECT cr.id INTO sample_registry_id FROM container_registries cr LIMIT 1;
    SELECT i.id INTO sample_image_id FROM catalog_images i LIMIT 1;

    -- Insert S3 storage backend
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
    ) VALUES (
        storage_backend_id,
        sample_company_id,
        sample_org_unit_id,
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
        sample_user_id
    ) ON CONFLICT (id) DO NOTHING;

    -- Insert sample blobs
    INSERT INTO image_blobs (
        id,
        digest,
        algorithm,
        size_bytes,
        storage_backend,
        storage_bucket,
        storage_key,
        storage_region,
        media_type,
        content_encoding,
        reference_count,
        compression_algorithm,
        original_size_bytes,
        compression_ratio,
        storage_class,
        upload_status,
        uploaded_by
    ) VALUES 
    (
        blob1_id,
        'sha256:8be619267e30a82be32eb87a08b70da1cc01f28c36f45b1b7c6b61b2e8c7f50f',
        'sha256',
        52428800, -- 50MB
        's3',
        'image-factory-blobs-prod',
        'blobs/sha256/8b/8be619267e30a82be32eb87a08b70da1cc01f28c36f45b1b7c6b61b2e8c7f50f',
        'us-west-2',
        'application/vnd.docker.image.rootfs.diff.tar.gzip',
        'gzip',
        3, -- Referenced by 3 images
        'gzip',
        75497472, -- Original uncompressed size
        0.6944, -- 69.44% compression ratio
        'STANDARD',
        'completed',
        sample_user_id
    ),
    (
        blob2_id,
        'sha256:6b4cf1b7e1e2e5c5a0c8f9d8e7f6g5h4i3j2k1l0m9n8o7p6q5r4s3t2u1v0w9x8',
        'sha256',
        25165824, -- 24MB
        's3',
        'image-factory-blobs-prod',  
        'blobs/sha256/6b/6b4cf1b7e1e2e5c5a0c8f9d8e7f6g5h4i3j2k1l0m9n8o7p6q5r4s3t2u1v0w9x8',
        'us-west-2',
        'application/vnd.docker.image.rootfs.diff.tar.gzip',
        'gzip',
        1, -- Referenced by 1 image
        'gzip',
        33554432, -- Original uncompressed size
        0.7500, -- 75% compression ratio
        'STANDARD',
        'completed',
        sample_user_id
    ) ON CONFLICT (id) DO NOTHING;

    -- Insert sample manifest
    INSERT INTO image_manifests (
        id,
        image_id,
        media_type,
        digest,
        manifest_content,
        platform,
        size_bytes
    ) VALUES (
        manifest_id,
        sample_image_id,
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
    ) ON CONFLICT (id) DO NOTHING;

    -- Link layers to blobs
    INSERT INTO image_layer_blobs (
        id,
        manifest_id,
        blob_id,
        layer_order,
        blob_digest
    ) VALUES 
    (
        'l1234567-89ab-cdef-0123-456789abcdef'::uuid,
        manifest_id,
        blob1_id, 
        0,
        'sha256:8be619267e30a82be32eb87a08b70da1cc01f28c36f45b1b7c6b61b2e8c7f50f'
    ),
    (
        'l2345678-9abc-def0-1234-56789abcdef0'::uuid,
        manifest_id,
        blob2_id,
        1,
        'sha256:6b4cf1b7e1e2e5c5a0c8f9d8e7f6g5h4i3j2k1l0m9n8o7p6q5r4s3t2u1v0w9x8'
    ) ON CONFLICT (id) DO NOTHING;

    -- Add storage metrics
    INSERT INTO storage_metrics (
        id,
        storage_backend_id,
        metric_date,
        total_objects,
        total_size_bytes,
        metrics_data
    ) VALUES (
        's1234567-89ab-cdef-0123-456789abcdef'::uuid,
        storage_backend_id,
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
            },
            "deduplication_savings": {
                "total_logical_size": 232783872,
                "actual_storage_size": 77594624,
                "savings_bytes": 155189248,
                "savings_percent": 66.67
            }
        }'::jsonb
    ) ON CONFLICT (id) DO NOTHING;

    -- Update catalog image metadata to link to the manifest and storage backend
    UPDATE catalog_images
    SET 
        size_bytes = 77594624,
        metadata = COALESCE(metadata, '{}'::jsonb)
            || jsonb_build_object(
                'primary_manifest_id', manifest_id,
                'total_blob_size', 77594624,
                'unique_blob_size', 77594624,
                'blob_count', 2,
                'storage_backend_id', storage_backend_id,
                'storage_cost_estimate', 1.79,
                'transfer_cost_estimate', 0.02
            )
    WHERE id = sample_image_id;

    -- Update container_registries to use S3 storage backend  
    UPDATE container_registries 
    SET 
        storage_backend_id = storage_backend_id,
        storage_prefix = 'registry-prod/',
        blob_mount_enabled = true,
        cross_repository_blob_mount = true,
        gc_policy = '{
            "enabled": true,
            "retention_days": 90,
            "keep_minimum_tags": 5,
            "delete_untagged_after_days": 7
        }'::jsonb
    WHERE id = sample_registry_id;

    RAISE NOTICE 'S3 Storage Demo Data Loaded Successfully!';
    RAISE NOTICE 'Storage Backend ID: %', storage_backend_id;
    RAISE NOTICE 'Sample Blobs: % and %', blob1_id, blob2_id;
    RAISE NOTICE 'Manifest ID: %', manifest_id;
END $$;
