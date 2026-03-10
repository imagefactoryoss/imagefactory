-- +migrate Down
DROP TRIGGER IF EXISTS update_build_policies_updated_at ON build_policies;
DROP TABLE IF EXISTS build_policies;