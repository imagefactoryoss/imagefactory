-- Rollback config_templates and related objects

-- Drop triggers
DROP TRIGGER IF EXISTS trigger_update_config_template_shares_timestamp ON config_template_shares;
DROP TRIGGER IF EXISTS trigger_update_config_templates_timestamp ON config_templates;

-- Drop function
DROP FUNCTION IF EXISTS update_config_templates_timestamp();

-- Drop indexes
DROP INDEX IF EXISTS idx_config_template_shares_permissions;
DROP INDEX IF EXISTS idx_config_template_shares_shared_with;
DROP INDEX IF EXISTS idx_config_template_shares_template_id;
DROP INDEX IF EXISTS idx_config_templates_is_public;
DROP INDEX IF EXISTS idx_config_templates_is_shared;
DROP INDEX IF EXISTS idx_config_templates_created_at;
DROP INDEX IF EXISTS idx_config_templates_method;
DROP INDEX IF EXISTS idx_config_templates_created_by;
DROP INDEX IF EXISTS idx_config_templates_project_id;

-- Drop tables
DROP TABLE IF EXISTS config_template_shares;
DROP TABLE IF EXISTS config_templates;
