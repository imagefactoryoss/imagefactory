-- Enforce strict tenant build_method key presence for tool_availability.
-- Missing keys are backfilled as false for tenant-scoped records.

WITH normalized AS (
    SELECT
        id,
        jsonb_set(
            config_value,
            '{build_methods}',
            (
                COALESCE(config_value->'build_methods', '{}'::jsonb)
                || jsonb_build_object(
                    'container', COALESCE((config_value->'build_methods'->>'container')::boolean, false),
                    'packer', COALESCE((config_value->'build_methods'->>'packer')::boolean, false),
                    'paketo', COALESCE((config_value->'build_methods'->>'paketo')::boolean, false),
                    'kaniko', COALESCE((config_value->'build_methods'->>'kaniko')::boolean, false),
                    'buildx', COALESCE((config_value->'build_methods'->>'buildx')::boolean, false),
                    'nix', COALESCE((config_value->'build_methods'->>'nix')::boolean, false)
                )
            ),
            true
        ) AS new_config_value
    FROM system_configs
    WHERE tenant_id IS NOT NULL
      AND config_type = 'tool_settings'
      AND config_key = 'tool_availability'
)
UPDATE system_configs sc
SET
    config_value = n.new_config_value,
    updated_at = CURRENT_TIMESTAMP,
    version = COALESCE(sc.version, 1) + 1
FROM normalized n
WHERE sc.id = n.id
  AND sc.config_value IS DISTINCT FROM n.new_config_value;

