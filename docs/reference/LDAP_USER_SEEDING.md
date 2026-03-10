# LDAP User Seeding

## Overview
This document describes the database migration used to seed users from the LDAP configuration into the `users` table for development and testing scenarios.

## Migration Files Created

### 1. `backend/migrations/013_seed_ldap_users.up.sql`
- **Purpose**: Populate users table with LDAP users from ldap-config.cfg
- **Method**: INSERT with ON CONFLICT DO NOTHING to prevent duplicates
- **Scope**: 14 LDAP users + 1 system user = 15 total users
- **User Fields Populated**:
  - `email` - Primary identifier
  - `first_name` - From LDAP givenName
  - `last_name` - From LDAP sn
  - `is_ldap_user` - Set to `true` for all LDAP users
  - `status` - Set to `active`
  - `email_verified` - Set to `true` (LDAP users are trusted)

### 2. `backend/migrations/013_seed_ldap_users.down.sql`
- **Purpose**: Rollback migration to remove seeded LDAP users
- **Method**: Delete only LDAP users marked with `is_ldap_user = true`
- **Safety**: Preserves system users and any manually created users

## Seeded Users (14 LDAP Users)

### Admin & Leadership
1. **admin@imagefactory.local** - Michael Rodriguez (Central Governance Administrator)
2. **michael.richardson@imagefactory.local** - Michael Richardson (Practice Owner - Compliance)
3. **bob.smith@imagefactory.local** - Bob Smith (Operations Lead)

### Security Department
4. **alice.johnson@imagefactory.local** - Alice Johnson (Security Practice Author)
5. **david.wilson@imagefactory.local** - David Wilson (Security Practice Reviewer)
6. **eve.martinez@imagefactory.local** - Eve Martinez (Security Practice Approver)

### Compliance Department
7. **frank.thompson@imagefactory.local** - Frank Thompson (CGA Control Reviewer)
8. **grace.lee@imagefactory.local** - Grace Lee (CGA Control Approver)
9. **carol.davis@imagefactory.local** - Carol Davis (Control Governance Manager)

### Cloud Infrastructure Department
10. **sarah.mitchell@imagefactory.local** - Sarah Mitchell (Cloud Practice Author)
11. **mark.anderson@imagefactory.local** - Mark Anderson (Cloud Practice Reviewer)
12. **jennifer.chang@imagefactory.local** - Jennifer Chang (Cloud Practice Owner)

### Data Privacy Department
13. **lisa.taylor@imagefactory.local** - Lisa Taylor (Data Privacy Practice Author)
14. **thomas.brown@imagefactory.local** - Thomas Brown (Data Privacy Practice Reviewer)

## Execution Steps

1. **Created Migration Files**
   - `013_seed_ldap_users.up.sql` - Inserts LDAP users
   - `013_seed_ldap_users.down.sql` - Rollback script

2. **Ran Migration**
   ```bash
   cd backend && \
   IF_AUTH_JWT_SECRET="__SET_BEFORE_DEPLOYMENT__" \
   IF_DATABASE_HOST=localhost \
   IF_DATABASE_PORT=5432 \
   IF_DATABASE_NAME=image_factory_dev \
   IF_DATABASE_USER=postgres \
   IF_DATABASE_PASSWORD=__SET_BEFORE_DEPLOYMENT__ \
   IF_DATABASE_SSL_MODE=disable \
   go run cmd/migrate/main.go up --env ../.env.development
   ```

3. **Verified Seeding**
   ```sql
   SELECT COUNT(*) as total_users, 
          SUM(CASE WHEN is_ldap_user = true THEN 1 ELSE 0 END) as ldap_users 
   FROM users;
   
   -- Result: 15 total users, 14 LDAP users
   ```

## Testing UserManagementPage

With the LDAP users seeded, you can now:

1. **Test User Listing**
   - Navigate to `/admin/users`
   - All 14 LDAP users should be visible in the paginated table
   - Test filtering, sorting, and pagination

2. **Test User Operations**
   - **Search**: Type names to filter users (e.g., "Alice", "Cloud")
   - **Filter by Status**: All users are "active"
   - **Edit User**: Click "Edit" to modify user details
   - **Suspend/Activate**: Test user status transitions
   - **Delete**: Test user removal (with confirmation)
   - **View Details**: Each row is clickable to view full user info

3. **Test Pagination**
   - Set items per page to 5 and navigate through pages
   - Verify prev/next buttons work correctly

4. **Test Dark Mode**
   - Toggle dark mode to verify styling
   - Check all user details render correctly

## Database Verification

Current user table status:
- **Total Users**: 15
- **LDAP Users**: 14 (is_ldap_user = true)
- **System Users**: 1 (system@imagefactory.local)

All LDAP users have:
- ✅ Active status
- ✅ Email verified
- ✅ First and last names populated
- ✅ LDAP flag set to true

## Integration with LDAP Authentication

These seeded users match the GLAuth LDAP configuration, so users can:
1. Log in via LDAP with their ldap-config credentials
2. Have pre-populated user records in the database
3. Be managed through the UserManagementPage admin interface

## Rollback Instructions

If needed to rollback the migration:

```bash
cd backend && \
IF_AUTH_JWT_SECRET="__SET_BEFORE_DEPLOYMENT__" \
IF_DATABASE_HOST=localhost \
IF_DATABASE_PORT=5432 \
IF_DATABASE_NAME=image_factory_dev \
IF_DATABASE_USER=postgres \
IF_DATABASE_PASSWORD=__SET_BEFORE_DEPLOYMENT__ \
IF_DATABASE_SSL_MODE=disable \
go run cmd/migrate/main.go down
```

This will:
- Remove all 14 LDAP users
- Preserve system users (system@imagefactory.local)
- Leave database schema intact

---

**Status**: ✅ Migration Complete - 14 LDAP Users Seeded Successfully
