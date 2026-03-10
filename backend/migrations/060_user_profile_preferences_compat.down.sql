-- Compatibility migration down intentionally does not drop user_profile_preferences,
-- because it may already be the active persistence store.
SELECT 1;
