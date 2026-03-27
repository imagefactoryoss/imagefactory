-- LDAP users should never persist local password credentials.
UPDATE users
SET
    password_hash = NULL,
    must_change_password = FALSE,
    updated_at = CURRENT_TIMESTAMP
WHERE auth_method = 'ldap'
  AND password_hash IS NOT NULL;
