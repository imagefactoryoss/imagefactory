-- Create email_queue table for asynchronous email delivery
-- The email-worker service processes this queue every 30 seconds
CREATE TABLE IF NOT EXISTS email_queue (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    to_email VARCHAR(255) NOT NULL,
    cc_email VARCHAR(255),
    from_email VARCHAR(255) NOT NULL,
    subject TEXT NOT NULL,
    body_text TEXT,
    body_html TEXT,
    email_type VARCHAR(50) NOT NULL DEFAULT 'notification',
    priority INT DEFAULT 5,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    retry_count INT DEFAULT 0,
    max_retries INT DEFAULT 3,
    last_error TEXT,
    next_retry_at TIMESTAMP,
    smtp_host VARCHAR(255),
    smtp_port INT,
    smtp_username VARCHAR(255),
    smtp_password VARCHAR(255),
    smtp_use_tls BOOLEAN DEFAULT false,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    sent_at TIMESTAMP,
    processed_at TIMESTAMP,
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);

-- Create indexes for efficient queue processing
CREATE INDEX IF NOT EXISTS idx_email_queue_status ON email_queue(status) WHERE status IN ('pending', 'processing');
CREATE INDEX IF NOT EXISTS idx_email_queue_tenant ON email_queue(tenant_id);
CREATE INDEX IF NOT EXISTS idx_email_queue_created ON email_queue(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_email_queue_next_retry ON email_queue(next_retry_at) WHERE status = 'failed';
CREATE INDEX IF NOT EXISTS idx_email_queue_cc_email ON email_queue(cc_email) WHERE cc_email IS NOT NULL;

-- Create trigger to auto-update updated_at timestamp
CREATE OR REPLACE FUNCTION update_email_queue_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER email_queue_timestamp_trigger
BEFORE UPDATE ON email_queue
FOR EACH ROW
EXECUTE FUNCTION update_email_queue_timestamp();
