# Testing Guide: Owner/Admin Role

## Pre-Test Setup

### Test Environment
- Backend running on: `http://localhost:8080`
- Frontend running on: `http://localhost:5173` or `http://localhost:3000`
- Database: PostgreSQL with test data

### Required Test Users
You need users with Owner/Admin role in your LDAP/database:

**Option 1: Use Existing LDAP Users**
- Check your LDAP server for users with owner/admin roles
- Default test users (if seeded): alice.johnson, bob.smith, etc.

**Option 2: Create Test Users**
- Access admin panel or database
- Create user with role = "owner" or "administrator"

---

## Test Flow 1: Navigation & Visibility

### Step 1: Login as Owner/Admin User
1. Go to `http://localhost:5173/login`
2. Enter credentials for an Owner/Admin user
3. Click "Login"
4. You should be redirected to `/dashboard`

### Step 2: Check Navigation Items
1. Look at the left sidebar (on desktop) or hamburger menu (mobile)
2. **Verify these items are visible**:
   - ✅ Dashboard
   - ✅ Projects
   - ✅ Builds
   - ✅ Images
   - ✅ **Tenants** (only for Owner/Admin)
   - ✅ Profile
   - ✅ Settings

3. **Verify these items are NOT visible** (unless you're also a system admin):
   - ❌ Admin Dashboard

4. **Check header**:
   - Should see current role displayed next to context switcher
   - Format: "Owner" or "Administrator"

### ✅ Expected Result
Navigation should match Owner/Admin permissions. If you see unexpected items or missing items, check role assignment.

---

## Test Flow 2: Project Management & Member Management

### Step 1: Navigate to Projects
1. Click **Projects** in sidebar
2. You should see a list of projects
3. There should be a **"+ New Project"** button in top right

### Step 2: View Project Details
1. Click on any project name to open details
2. You should see two tabs:
   - **Overview** (project info, repository URL, branch)
   - **Members** (NEW - member management)

### Step 3: Edit Project Details
1. On **Overview** tab, click **Edit** button
2. Change any field:
   - Project name
   - Description
   - Repository URL
   - Branch
3. Click **Save**
4. Verify changes are saved and page refreshes

### Step 4: Switch to Members Tab
1. Click **Members** tab
2. You should see:
   - **"+ Add Member"** button
   - Table of current project members
   - Columns: User ID, Email, Role, Actions
   - Each row has "Edit Role" and "Remove" buttons

### ✅ Expected Result
Project details and member list load correctly. Edit functionality works.

---

## Test Flow 3: Add Project Member (Core Feature)

### Step 1: Open Add Member Modal
1. Click **"+ Add Member"** button
2. Modal should open with:
   - Search field for users
   - "Add" button (disabled until user selected)
   - "Cancel" button

### Step 2: Search for User
1. Type a user's name in the search field
2. Dropdown should appear with matching users
3. Example: type "alice" → should show "alice.johnson"

### Step 3: Add Member
1. Click on a user from dropdown
2. User should be selected and shown
3. Click **"Add"** button
4. Modal should close
5. **Verify**:
   - Success toast appears
   - New member appears in table
   - Member has no role assigned initially

### ✅ Expected Result
Member added successfully. Member appears in list immediately.

**If it fails**:
- Check backend is running: `curl http://localhost:8080/health`
- Check API endpoint: `GET /api/v1/projects/{projectId}/members`
- Check browser console for errors (F12)

---

## Test Flow 4: Edit Member Role

### Step 1: Open Edit Role Modal
1. In Members table, find a member you just added
2. Click **"Edit Role"** button in that row
3. Modal should open with:
   - Dropdown showing available roles
   - "Save" and "Cancel" buttons

### Step 2: Select New Role
1. Click role dropdown
2. You should see options:
   - Owner
   - Administrator
   - Developer
   - Operator
   - Viewer
   - (empty/None)
3. Select **"Developer"**

### Step 3: Save Role Change
1. Click **"Save"** button
2. Modal should close
3. **Verify**:
   - Success toast appears
   - Member row now shows "Developer" role
   - Change is persisted (refresh page to confirm)

### ✅ Expected Result
Role is updated in table and persisted to backend.

**If it fails**:
- Check API endpoint: `PATCH /api/v1/projects/{projectId}/members/{userId}`
- Verify role exists in backend database
- Check audit logs for errors

---

## Test Flow 5: Remove Member

### Step 1: Open Remove Confirmation
1. In Members table, click **"Remove"** button for a member
2. Confirmation dialog should appear asking:
   - "Are you sure you want to remove this member?"
   - With "Remove" and "Cancel" buttons

### Step 2: Confirm Removal
1. Click **"Remove"** button
2. Dialog should close
3. **Verify**:
   - Success toast appears
   - Member disappears from table
   - If you refresh page, member is still gone

### ✅ Expected Result
Member is removed successfully and audit logged.

**If it fails**:
- Check API endpoint: `DELETE /api/v1/projects/{projectId}/members/{userId}`
- Check backend logs for errors
- Verify user permissions

---

## Test Flow 6: Tenant Management (Owner/Admin Only)

### Step 1: Navigate to Tenants
1. Click **Tenants** in sidebar
2. You should see list of all tenants in a table
3. Columns: Name, Slug, Status, Quota (builds, storage), Created Date, Actions

### Step 2: View Tenant Details
1. Click **"View"** action button for a tenant
2. Tenant detail page should load (future feature)

### Step 3: Create New Tenant (Future)
1. Click **"+ New Tenant"** button
2. Form should open (future feature)

### Step 4: Delete Tenant
1. Click **"Delete"** action button for a tenant
2. Confirmation dialog should appear
3. Confirm deletion
4. Tenant should be removed from list

### ✅ Expected Result
Tenant list loads. Delete functionality works.

---

## Test Flow 7: Build Management

### Step 1: Navigate to Builds
1. Click **Builds** in sidebar
2. You should see list of all builds (or empty state if none)

### Step 2: Create Build
1. Click **"+ New Build"** or navigate to `/builds/new`
2. Build form should appear
3. Complete the form with:
   - Select project
   - Enter build name
   - Configure build settings
4. Click **"Create Build"**

### Step 3: View Build Details
1. Click on a build in the list
2. Build detail page should show:
   - Build status (running, completed, failed, etc.)
   - Build logs
   - Build duration
   - Created time

### ✅ Expected Result
Build creation and viewing works correctly.

---

## Test Flow 8: Role-Based Access Control

### Step 1: Switch Tenant Role
1. Use the **Context Switcher** in header
2. Switch to a tenant where you have **Viewer** role (or less)
3. **Verify Tenants nav item disappears**
4. **Verify Builds nav item disappears** (if Viewer role)
5. Header should show "Viewer" role instead of "Owner"

### Step 2: Try Accessing Protected Routes
1. Try to access `/tenants` directly via URL
2. Should be redirected to `/dashboard`

### Step 3: Switch Back to Owner Role
1. Use Context Switcher
2. Switch back to Owner/Admin tenant
3. **Verify Tenants and Builds nav items reappear**

### ✅ Expected Result
Navigation and routing properly respect role-based access.

---

## Test Flow 9: Member Management Error Cases

### Test: Adding Duplicate Member
1. Try to add same user to project twice
2. **Should get error**: "User already a member of this project"
3. Member should not be duplicated in list

### Test: Removing Non-existent Member
1. Try to remove member that's already removed
2. **Should get error or be handled gracefully**

### Test: Invalid Role Assignment
1. Try to assign a role that doesn't exist
2. **Should get error**: "Invalid role"

### ✅ Expected Result
All error cases handled gracefully with user feedback.

---

## Test Flow 10: Empty States

### Test: Empty Project List
1. Create new tenant with no projects
2. Go to Projects page
3. **Should see empty state**:
   - Icon: 📁
   - Message: "No projects yet"
   - Button: "Create Project"

### Test: Empty Member List
1. Create new project with no members
2. Open project → Members tab
3. **Should show empty members list**
4. **Should be able to add member**

### Test: Empty Tenant List
1. (This shouldn't happen in production, but test for robustness)
2. **Should see empty state** with "Create Tenant" button

### ✅ Expected Result
All empty states display correctly and guide user action.

---

## Test Flow 11: Dark Mode

### Test: Enable Dark Mode
1. Click theme toggle button (☀️/🌙) in header
2. Page should switch to dark theme
3. **Verify all pages work in dark mode**:
   - Projects page
   - Project detail
   - Members UI
   - Tenants page
4. **Verify colors are legible**

### ✅ Expected Result
Dark mode works throughout application.

---

## Test Flow 12: Responsive Design

### Test: Desktop View (1920px width)
1. All navigation items visible in sidebar
2. Table displays properly
3. Buttons are accessible

### Test: Tablet View (768px width)
1. Menu collapses to hamburger
2. Tables are scrollable if needed
3. Modals resize properly

### Test: Mobile View (375px width)
1. Full responsive layout
2. Touch-friendly buttons
3. Readable text

### ✅ Expected Result
Application works on all screen sizes.

---

## Troubleshooting

### Issue: "Permission Denied" on Tenants page
**Solution**:
1. Check user role in database: `SELECT * FROM users WHERE email='user@example.com';`
2. Verify user has RBAC role = "Owner" or "Administrator"
3. Check role assignment: `SELECT * FROM user_roles WHERE user_id='...' AND tenant_id='...';`

### Issue: Members not loading
**Solution**:
1. Check backend logs: `tail -f logs/backend.log`
2. Verify project exists
3. Check API endpoint returns data: `curl -H "Authorization: Bearer $TOKEN" -H "X-Tenant-ID: <tenant-uuid>" http://localhost:8080/api/v1/projects/{projectId}/members`

### Issue: Add member modal shows no users
**Solution**:
1. Verify users exist in database
2. Check permissions on user endpoint
3. Verify current user has `canManageMembers` permission

### Issue: Dark mode not working
**Solution**:
1. Check localStorage: `localStorage.getItem('theme')`
2. Verify theme toggle button is visible
3. Check browser dev tools for CSS class issues

---

## Success Criteria

### All Tests Should Pass ✅
- [x] Can see Owner/Admin-specific nav items
- [x] Can view and edit projects
- [x] Can add members to projects
- [x] Can edit member roles
- [x] Can remove members
- [x] Can view tenant list
- [x] Member changes are persisted
- [x] Role-based navigation works
- [x] Empty states display correctly
- [x] Dark mode works
- [x] Responsive on all devices
- [x] Error messages are helpful

### Commands to Verify Everything is Working

```bash
# Check backend is running
curl http://localhost:8080/health

# Check frontend is running
curl http://localhost:5173

# Check specific API endpoints
curl -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: <tenant-uuid>" \
  http://localhost:8080/api/v1/projects

# Check database connection
psql -U postgres -d image_factory_dev -c "SELECT COUNT(*) FROM users;"
```

---

## Notes for Test Execution

1. **Create Fresh Test User**: Preferably create a new test account for clean testing
2. **Clear Browser Cache**: Ctrl+Shift+Delete to clear cache before testing
3. **Watch Network Tab**: Check network requests in Dev Tools (F12) for API calls
4. **Monitor Backend Logs**: Keep backend logs open: `tail -f logs/backend.log`
5. **Check Audit Logs**: All member operations should be logged
6. **Test Concurrency**: If possible, have multiple users testing simultaneously to catch race conditions

