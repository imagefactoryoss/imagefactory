-- Drop email_queue table and related objects
DROP TRIGGER IF EXISTS email_queue_timestamp_trigger ON email_queue;
DROP FUNCTION IF EXISTS update_email_queue_timestamp();
DROP INDEX IF EXISTS idx_email_queue_next_retry;
DROP INDEX IF EXISTS idx_email_queue_created;
DROP INDEX IF EXISTS idx_email_queue_tenant;
DROP INDEX IF EXISTS idx_email_queue_status;
DROP INDEX IF EXISTS idx_email_queue_cc_email;
DROP TABLE IF EXISTS email_queue;
