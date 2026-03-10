-- ============================================================================
-- Demo Data Integrity Validation
-- ============================================================================
-- Validates demo datasets required for functional UI/system checks.
-- Run after demo seeding.
-- ============================================================================

DO $$
DECLARE
    v_sysadmin_tenant_id UUID;
    v_nginx_image_id UUID;
    v_node_image_id UUID;
    v_python_image_id UUID;
    v_missing_image_count INTEGER := 0;
BEGIN
    SELECT id
      INTO v_sysadmin_tenant_id
      FROM tenants
     WHERE tenant_code = 'sysadmin'
     LIMIT 1;

    IF v_sysadmin_tenant_id IS NULL THEN
        RAISE EXCEPTION 'Demo integrity failure: sysadmin tenant missing';
    END IF;

    -- ------------------------------------------------------------------------
    -- Demo seed must NOT provision users
    -- ------------------------------------------------------------------------
    IF EXISTS (
        SELECT 1
          FROM users
         WHERE email IN (
            'michael.richardson@imagefactory.local',
            'alice.johnson@imagefactory.local',
            'david.wilson@imagefactory.local',
            'eve.martinez@imagefactory.local',
            'frank.thompson@imagefactory.local',
            'grace.lee@imagefactory.local',
            'carol.davis@imagefactory.local',
            'bob.smith@imagefactory.local',
            'sarah.mitchell@imagefactory.local',
            'mark.anderson@imagefactory.local',
            'jennifer.chang@imagefactory.local',
            'lisa.taylor@imagefactory.local',
            'thomas.brown@imagefactory.local'
         )
    ) THEN
        RAISE EXCEPTION 'Demo integrity failure: demo user provisioning detected in users table';
    END IF;

    -- ------------------------------------------------------------------------
    -- Infrastructure provider demo checks
    -- ------------------------------------------------------------------------
    IF NOT EXISTS (
        SELECT 1
          FROM infrastructure_providers ip
         WHERE ip.name = 'rancher-desktop'
           AND ip.tenant_id = v_sysadmin_tenant_id
           AND ip.provider_type = 'kubernetes'
    ) THEN
        RAISE EXCEPTION 'Demo integrity failure: rancher-desktop provider missing for sysadmin tenant';
    END IF;

    IF NOT EXISTS (
        SELECT 1
          FROM provider_permissions pp
          JOIN infrastructure_providers ip ON ip.id = pp.provider_id
         WHERE ip.name = 'rancher-desktop'
           AND pp.permission = 'infrastructure:configure'
           AND pp.tenant_id = v_sysadmin_tenant_id
    ) THEN
        RAISE EXCEPTION 'Demo integrity failure: missing infrastructure:configure permission on rancher-desktop for sysadmin tenant';
    END IF;

    IF NOT EXISTS (
        SELECT 1
          FROM provider_permissions pp
          JOIN infrastructure_providers ip ON ip.id = pp.provider_id
         WHERE ip.name = 'rancher-desktop'
           AND pp.permission = 'infrastructure:select'
    ) THEN
        RAISE EXCEPTION 'Demo integrity failure: missing infrastructure:select permissions on rancher-desktop';
    END IF;

    IF (SELECT COUNT(*) FROM infrastructure_nodes WHERE tenant_id = v_sysadmin_tenant_id) < 3 THEN
        RAISE EXCEPTION 'Demo integrity failure: expected at least 3 infrastructure_nodes for sysadmin tenant';
    END IF;

    -- ------------------------------------------------------------------------
    -- Catalog image demo checks
    -- ------------------------------------------------------------------------
    SELECT id INTO v_nginx_image_id
      FROM catalog_images
     WHERE tenant_id = v_sysadmin_tenant_id
       AND name = 'nginx-runtime'
     LIMIT 1;
    IF v_nginx_image_id IS NULL THEN
        v_missing_image_count := v_missing_image_count + 1;
    END IF;

    SELECT id INTO v_node_image_id
      FROM catalog_images
     WHERE tenant_id = v_sysadmin_tenant_id
       AND name = 'nodejs-builder'
     LIMIT 1;
    IF v_node_image_id IS NULL THEN
        v_missing_image_count := v_missing_image_count + 1;
    END IF;

    SELECT id INTO v_python_image_id
      FROM catalog_images
     WHERE tenant_id = v_sysadmin_tenant_id
       AND name = 'python-ml-runtime'
     LIMIT 1;
    IF v_python_image_id IS NULL THEN
        v_missing_image_count := v_missing_image_count + 1;
    END IF;

    IF v_missing_image_count > 0 THEN
        RAISE EXCEPTION 'Demo integrity failure: missing % required demo catalog_images', v_missing_image_count;
    END IF;

    IF (SELECT COUNT(*) FROM catalog_image_versions WHERE catalog_image_id IN (v_nginx_image_id, v_node_image_id, v_python_image_id)) < 3 THEN
        RAISE EXCEPTION 'Demo integrity failure: expected at least 3 catalog_image_versions across demo images';
    END IF;

    IF (SELECT COUNT(*) FROM catalog_image_tags WHERE catalog_image_id IN (v_nginx_image_id, v_node_image_id, v_python_image_id)) < 6 THEN
        RAISE EXCEPTION 'Demo integrity failure: expected at least 6 catalog_image_tags across demo images';
    END IF;

    IF (SELECT COUNT(*) FROM catalog_image_layers WHERE image_id IN (v_nginx_image_id, v_node_image_id, v_python_image_id)) < 9 THEN
        RAISE EXCEPTION 'Demo integrity failure: expected at least 9 catalog_image_layers across demo images';
    END IF;

    IF (SELECT COUNT(*) FROM catalog_image_sbom WHERE image_id IN (v_nginx_image_id, v_node_image_id, v_python_image_id)) < 3 THEN
        RAISE EXCEPTION 'Demo integrity failure: expected SBOM rows for all 3 demo images';
    END IF;

    IF (SELECT COUNT(*) FROM sbom_packages WHERE image_id IN (v_nginx_image_id, v_node_image_id, v_python_image_id)) < 8 THEN
        RAISE EXCEPTION 'Demo integrity failure: expected at least 8 sbom_packages across demo images';
    END IF;

    IF (SELECT COUNT(*) FROM catalog_image_vulnerability_scans WHERE image_id IN (v_nginx_image_id, v_node_image_id, v_python_image_id)) < 3 THEN
        RAISE EXCEPTION 'Demo integrity failure: expected vulnerability scans for all 3 demo images';
    END IF;

    IF NOT EXISTS (SELECT 1 FROM cve_database WHERE cve_id = 'CVE-2024-33333') THEN
        RAISE EXCEPTION 'Demo integrity failure: expected CVE-2024-33333 in cve_database';
    END IF;

    IF NOT EXISTS (
        SELECT 1
          FROM package_vulnerabilities
         WHERE cve_id = 'CVE-2024-33333'
           AND package_name = 'urllib3'
    ) THEN
        RAISE EXCEPTION 'Demo integrity failure: expected package_vulnerabilities mapping for urllib3 -> CVE-2024-33333';
    END IF;

    IF NOT EXISTS (
        SELECT 1
          FROM vulnerability_suppressions
         WHERE cve_id = 'CVE-2024-33333'
           AND image_id = v_python_image_id
           AND status = 'active'
    ) THEN
        RAISE EXCEPTION 'Demo integrity failure: expected active vulnerability suppression for python demo image';
    END IF;

    -- ------------------------------------------------------------------------
    -- Cross-table referential sanity
    -- ------------------------------------------------------------------------
    IF EXISTS (
        SELECT 1
          FROM catalog_image_versions v
          LEFT JOIN catalog_images i ON i.id = v.catalog_image_id
         WHERE i.id IS NULL
    ) THEN
        RAISE EXCEPTION 'Demo integrity failure: orphan rows detected in catalog_image_versions';
    END IF;

    IF EXISTS (
        SELECT 1
          FROM catalog_image_layers l
          LEFT JOIN catalog_images i ON i.id = l.image_id
         WHERE i.id IS NULL
    ) THEN
        RAISE EXCEPTION 'Demo integrity failure: orphan rows detected in catalog_image_layers';
    END IF;

    IF EXISTS (
        SELECT 1
          FROM sbom_packages p
          LEFT JOIN catalog_image_sbom s ON s.id = p.image_sbom_id
         WHERE s.id IS NULL
    ) THEN
        RAISE EXCEPTION 'Demo integrity failure: orphan rows detected in sbom_packages';
    END IF;

    RAISE NOTICE 'Demo integrity validation passed.';
END $$;
