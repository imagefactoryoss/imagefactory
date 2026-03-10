-- Create config_templates table for storing template configurations
CREATE TABLE IF NOT EXISTS config_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    created_by_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    name VARCHAR(255) NOT NULL,
    description TEXT,
    
    method VARCHAR(50) NOT NULL,
    template_data JSONB NOT NULL,
    
    is_shared BOOLEAN DEFAULT FALSE,
    is_public BOOLEAN DEFAULT FALSE,
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    -- Ensure unique template names within a project
    UNIQUE(project_id, name),
    CHECK(method IN ('packer', 'buildx', 'kaniko', 'docker', 'nix'))
);

-- Create indexes for efficient queries
CREATE INDEX IF NOT EXISTS idx_config_templates_project_id ON config_templates(project_id);
CREATE INDEX IF NOT EXISTS idx_config_templates_created_by ON config_templates(created_by_user_id);
CREATE INDEX IF NOT EXISTS idx_config_templates_method ON config_templates(method);
CREATE INDEX IF NOT EXISTS idx_config_templates_created_at ON config_templates(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_config_templates_is_shared ON config_templates(is_shared) WHERE is_shared = TRUE;
CREATE INDEX IF NOT EXISTS idx_config_templates_is_public ON config_templates(is_public) WHERE is_public = TRUE;

-- Create config_template_shares table for managing template sharing between users
CREATE TABLE IF NOT EXISTS config_template_shares (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    template_id UUID NOT NULL REFERENCES config_templates(id) ON DELETE CASCADE,
    shared_with_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    can_use BOOLEAN DEFAULT TRUE,
    can_edit BOOLEAN DEFAULT FALSE,
    can_delete BOOLEAN DEFAULT FALSE,
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    -- Prevent duplicate shares
    UNIQUE(template_id, shared_with_user_id)
);

-- Create indexes for efficient share queries
CREATE INDEX IF NOT EXISTS idx_config_template_shares_template_id ON config_template_shares(template_id);
CREATE INDEX IF NOT EXISTS idx_config_template_shares_shared_with ON config_template_shares(shared_with_user_id);
CREATE INDEX IF NOT EXISTS idx_config_template_shares_permissions ON config_template_shares(template_id, can_use, can_edit);

-- Create function to auto-update updated_at timestamp
CREATE OR REPLACE FUNCTION update_config_templates_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger for config_templates
DROP TRIGGER IF EXISTS trigger_update_config_templates_timestamp ON config_templates;
CREATE TRIGGER trigger_update_config_templates_timestamp
BEFORE UPDATE ON config_templates
FOR EACH ROW
EXECUTE FUNCTION update_config_templates_timestamp();

-- Create trigger for config_template_shares
DROP TRIGGER IF EXISTS trigger_update_config_template_shares_timestamp ON config_template_shares;
CREATE TRIGGER trigger_update_config_template_shares_timestamp
BEFORE UPDATE ON config_template_shares
FOR EACH ROW
EXECUTE FUNCTION update_config_templates_timestamp();
