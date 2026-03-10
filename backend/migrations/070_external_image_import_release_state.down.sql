ALTER TABLE external_image_imports
DROP CONSTRAINT IF EXISTS chk_external_image_import_release_state;

ALTER TABLE external_image_imports
DROP COLUMN IF EXISTS released_at,
DROP COLUMN IF EXISTS release_requested_at,
DROP COLUMN IF EXISTS release_reason,
DROP COLUMN IF EXISTS release_actor_user_id,
DROP COLUMN IF EXISTS release_blocker_reason,
DROP COLUMN IF EXISTS release_state;
