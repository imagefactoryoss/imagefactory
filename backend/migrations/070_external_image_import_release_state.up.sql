ALTER TABLE external_image_imports
ADD COLUMN IF NOT EXISTS release_state VARCHAR(50),
ADD COLUMN IF NOT EXISTS release_blocker_reason TEXT,
ADD COLUMN IF NOT EXISTS release_actor_user_id UUID REFERENCES users(id),
ADD COLUMN IF NOT EXISTS release_reason TEXT,
ADD COLUMN IF NOT EXISTS release_requested_at TIMESTAMPTZ,
ADD COLUMN IF NOT EXISTS released_at TIMESTAMPTZ;

DO $$
BEGIN
	IF NOT EXISTS (
		SELECT 1
		FROM pg_constraint
		WHERE conname = 'chk_external_image_import_release_state'
	) THEN
		ALTER TABLE external_image_imports
		ADD CONSTRAINT chk_external_image_import_release_state
		CHECK (release_state IS NULL OR release_state IN ('not_ready', 'ready_for_release', 'release_approved', 'released', 'release_blocked', 'unknown'));
	END IF;
END $$;

UPDATE external_image_imports
SET release_state = CASE
	WHEN status = 'success' THEN 'ready_for_release'
	WHEN status = 'quarantined' THEN 'release_blocked'
	WHEN status = 'failed' THEN 'release_blocked'
	WHEN status IN ('pending', 'approved', 'importing') THEN 'not_ready'
	ELSE 'unknown'
END,
release_blocker_reason = CASE
	WHEN status = 'quarantined' THEN 'policy_quarantined'
	WHEN status = 'failed' THEN 'import_failed'
	WHEN status IN ('pending', 'approved', 'importing') THEN 'import_not_terminal'
	ELSE NULL
END
WHERE release_state IS NULL;
