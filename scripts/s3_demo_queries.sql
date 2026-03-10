-- ============================================================================
-- S3+DATABASE ARCHITECTURE DEMONSTRATION QUERIES
-- Shows the practical benefits of using S3 object storage with database metadata
-- ============================================================================

-- Query 1: Show S3 Storage Backend Configuration
SELECT 
    sb.name,
    sb.backend_type,
    sb.s3_bucket,
    sb.s3_region,
    sb.default_storage_class,
    sb.is_default,
    sb.lifecycle_policies,
    sb.encryption_config
FROM storage_backends sb
WHERE sb.backend_type = 's3';

-- Query 2: Blob Storage Analysis - Shows deduplication benefits
SELECT 
    'Blob Storage Summary' as analysis_type,
    COUNT(*) as total_blobs,
    SUM(size_bytes) as total_storage_bytes,
    ROUND(AVG(size_bytes)) as avg_blob_size,
    SUM(reference_count) as total_references,
    SUM(CASE WHEN reference_count > 1 THEN size_bytes * (reference_count - 1) ELSE 0 END) as deduplication_savings_bytes
FROM image_blobs;

-- Query 3: Storage Cost Analysis by Registry
SELECT 
    cr.name as registry_name,
    cr.url as registry_url,
    COUNT(DISTINCT i.id) as image_count,
    COUNT(DISTINCT ib.id) as unique_blob_count,
    SUM(COALESCE((i.metadata->>'total_blob_size')::bigint, i.size_bytes, 0)) as total_logical_size,
    SUM(ib.size_bytes) as actual_storage_size,
    ROUND(
        (SUM(COALESCE((i.metadata->>'total_blob_size')::bigint, i.size_bytes, 0)) - SUM(ib.size_bytes))::numeric /
        NULLIF(SUM(COALESCE((i.metadata->>'total_blob_size')::bigint, i.size_bytes, 0)), 0) * 100, 2
    ) as deduplication_percentage,
    SUM(COALESCE((i.metadata->>'storage_cost_estimate')::numeric, 0)) as monthly_storage_cost,
    SUM(COALESCE((i.metadata->>'transfer_cost_estimate')::numeric, 0)) as monthly_transfer_cost
FROM container_registries cr
JOIN catalog_images i ON (i.metadata->>'storage_backend_id')::uuid = cr.storage_backend_id
LEFT JOIN image_manifests im ON im.image_id = i.id
LEFT JOIN image_layer_blobs ilb ON ilb.manifest_id = im.id  
LEFT JOIN image_blobs ib ON ib.id = ilb.blob_id
WHERE cr.storage_backend_id IS NOT NULL
GROUP BY cr.id, cr.name, cr.url;

-- Query 4: Blob Deduplication Opportunities
SELECT 
    ib.digest,
    ib.media_type,
    ib.size_bytes,
    ib.reference_count,
    ib.compression_algorithm,
    ROUND((ib.compression_ratio * 100)::numeric, 2) as compression_percent,
    ib.size_bytes * (ib.reference_count - 1) as savings_from_deduplication,
    ib.storage_bucket,
    ib.storage_key
FROM image_blobs ib
WHERE ib.reference_count > 1
ORDER BY (ib.size_bytes * (ib.reference_count - 1)) DESC;

-- Query 5: Storage Metrics and Cost Tracking
SELECT 
    sm.metric_date,
    sb.name as storage_backend,
    sm.total_objects,
    sm.total_size_bytes,
    ROUND((sm.total_size_bytes / 1024.0 / 1024.0 / 1024.0)::numeric, 3) as size_gb,
    sm.metrics_data->>'cost_breakdown' as cost_breakdown,
    sm.metrics_data->>'bandwidth_usage' as bandwidth_usage,
    sm.metrics_data->>'deduplication_savings' as deduplication_savings
FROM storage_metrics sm
JOIN storage_backends sb ON sb.id = sm.storage_backend_id
ORDER BY sm.metric_date DESC;

-- Query 6: Image-to-Blob Mapping (Shows OCI Manifest Structure)
SELECT 
    i.name || ':' || COALESCE(NULLIF(i.version, ''), 'latest') as image_tag,
    COALESCE(i.architecture, 'unknown') as platform,
    im.media_type as manifest_type,
    im.size_bytes as manifest_size,
    im.layer_count,
    json_build_object(
        'layers', json_agg(
            json_build_object(
                'order', ilb.layer_order,
                'digest', ib.digest,
                'size', ib.size_bytes,
                'media_type', ib.media_type,
                'storage_path', ib.storage_key,
                'compression', ib.compression_algorithm
            ) ORDER BY ilb.layer_order
        )
    ) as layer_details
FROM catalog_images i
JOIN image_manifests im ON im.image_id = i.id
JOIN image_layer_blobs ilb ON ilb.manifest_id = im.id
JOIN image_blobs ib ON ib.id = ilb.blob_id
GROUP BY i.id, i.name, i.version, i.architecture, im.id, im.media_type, im.size_bytes, im.layer_count;

-- Query 7: Storage Backend Health and Performance  
SELECT 
    sb.name,
    sb.s3_bucket,
    sb.s3_region,
    sb.is_active,
    COUNT(ib.id) as blob_count,
    SUM(ib.size_bytes) as total_size,
    AVG(ib.access_count) as avg_access_count,
    MAX(ib.last_accessed_at) as last_blob_access,
    COUNT(CASE WHEN ib.upload_status = 'completed' THEN 1 END) as successful_uploads,
    COUNT(CASE WHEN ib.upload_status = 'failed' THEN 1 END) as failed_uploads
FROM storage_backends sb
LEFT JOIN image_blobs ib ON ib.storage_bucket = sb.s3_bucket
WHERE sb.backend_type = 's3'
GROUP BY sb.id, sb.name, sb.s3_bucket, sb.s3_region, sb.is_active;

-- Query 8: Garbage Collection Candidates
SELECT 
    'Orphaned Blobs' as gc_type,
    COUNT(*) as candidate_count,
    SUM(ib.size_bytes) as reclaimable_bytes,
    ROUND((SUM(ib.size_bytes) / 1024.0 / 1024.0)::numeric, 2) as reclaimable_mb
FROM image_blobs ib
LEFT JOIN image_layer_blobs ilb ON ilb.blob_id = ib.id
WHERE ilb.id IS NULL

UNION ALL

SELECT 
    'Old Unreferenced Blobs' as gc_type,
    COUNT(*) as candidate_count,
    SUM(ib.size_bytes) as reclaimable_bytes,
    ROUND((SUM(ib.size_bytes) / 1024.0 / 1024.0)::numeric, 2) as reclaimable_mb
FROM image_blobs ib
WHERE ib.reference_count = 0 
AND ib.last_accessed_at < NOW() - INTERVAL '30 days';

-- Query 9: Cross-Registry Blob Sharing Analysis
SELECT 
    ib.digest,
    ib.size_bytes,
    ib.media_type,
    COUNT(DISTINCT cr.id) as registry_count,
    string_agg(DISTINCT cr.name, ', ') as registries_using_blob,
    ib.reference_count as total_references
FROM image_blobs ib
JOIN image_layer_blobs ilb ON ilb.blob_id = ib.id
JOIN image_manifests im ON im.id = ilb.manifest_id
JOIN catalog_images i ON i.id = im.image_id
JOIN container_registries cr ON (i.metadata->>'storage_backend_id')::uuid = cr.storage_backend_id
GROUP BY ib.id, ib.digest, ib.size_bytes, ib.media_type, ib.reference_count
HAVING COUNT(DISTINCT cr.id) > 1
ORDER BY ib.size_bytes DESC;

-- Query 10: Storage Efficiency Report
WITH storage_summary AS (
    SELECT 
        SUM(ib.size_bytes) as physical_storage,
        SUM(ib.size_bytes * ib.reference_count) as logical_storage,
        COUNT(*) as unique_blobs,
        SUM(ib.reference_count) as total_references
    FROM image_blobs ib
)
SELECT 
    'Storage Efficiency Analysis' as report_type,
    ROUND((physical_storage / 1024.0 / 1024.0 / 1024.0)::numeric, 3) as physical_storage_gb,
    ROUND((logical_storage / 1024.0 / 1024.0 / 1024.0)::numeric, 3) as logical_storage_gb,
    ROUND(((logical_storage - physical_storage) / 1024.0 / 1024.0 / 1024.0)::numeric, 3) as savings_gb,
    ROUND(((logical_storage - physical_storage)::numeric / NULLIF(logical_storage, 0) * 100), 2) as efficiency_percentage,
    unique_blobs,
    total_references,
    ROUND((total_references::numeric / NULLIF(unique_blobs, 0)), 2) as avg_reuse_factor
FROM storage_summary;
