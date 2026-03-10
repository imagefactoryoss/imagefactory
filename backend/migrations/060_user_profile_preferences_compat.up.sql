CREATE TABLE IF NOT EXISTS user_profile_preferences (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    preferences JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_user_profile_preferences_updated_at
    ON user_profile_preferences(updated_at DESC);

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = current_schema()
          AND table_name = 'users'
          AND column_name = 'profile_preferences'
    ) THEN
        EXECUTE '
            INSERT INTO user_profile_preferences (user_id, preferences, created_at, updated_at)
            SELECT id, COALESCE(profile_preferences, ''{}''::jsonb), NOW(), NOW()
            FROM users
            ON CONFLICT (user_id) DO NOTHING
        ';
    END IF;
END
$$;
