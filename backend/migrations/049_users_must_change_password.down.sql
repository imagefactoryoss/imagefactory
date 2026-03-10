DROP INDEX IF EXISTS idx_users_must_change_password;
ALTER TABLE users DROP COLUMN IF EXISTS must_change_password;
