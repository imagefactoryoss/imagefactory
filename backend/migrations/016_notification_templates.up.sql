-- Create notification_templates table to store email templates
-- These templates are used by the email notification service
-- Note: Migration 008 creates this table with NOT NULL company_id, so we drop and recreate with nullable company_id
DROP TABLE IF EXISTS notification_templates CASCADE;

CREATE TABLE IF NOT EXISTS notification_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID,
    template_type VARCHAR(50) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    subject_template TEXT NOT NULL,
    body_template TEXT,
    html_template TEXT,
    is_default BOOLEAN DEFAULT false,
    enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT notification_templates_type_unique UNIQUE(template_type)
);

-- Create indexes for faster lookups
CREATE INDEX IF NOT EXISTS idx_notification_templates_type ON notification_templates(template_type);
CREATE INDEX IF NOT EXISTS idx_notification_templates_enabled ON notification_templates(enabled);
CREATE INDEX IF NOT EXISTS idx_notification_templates_company ON notification_templates(company_id);
