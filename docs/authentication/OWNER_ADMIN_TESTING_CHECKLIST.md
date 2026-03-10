# Owner/Admin Role - Testing Checklist

Use this checklist to systematically test the Owner/Admin role capabilities.

---

## Pre-Test Setup

- [ ] Backend running: `go run cmd/server/main.go --env .env.development`
- [ ] Frontend running: `npm run dev --prefix frontend`
- [ ] Browser console open (F12) for debugging
- [ ] Backend logs accessible: `tail -f logs/backend.log`
- [ ] Test user credentials ready (Owner/Admin role in at least one tenant)

**Test User**: ________________  
**Tenant Name**: ________________  
**Test Date**: ________________  
**Tester Name**: ________________

---

## Section 1: Login & Navigation

### Step 1.1: Login
- [ ] Navigate to `http://localhost:5173/login`
- [ ] Enter Owner/Admin user credentials
- [ ] Click "Login" button
- [ ] Successfully redirected to `/dashboard`
- [ ] No console errors

**Notes**: _________________________________________________________

### Step 1.2: Check Header
- [ ] Header shows "Image Factory" logo/title
- [ ] Context switcher displays current tenant name
- [ ] Role indicator shows "Owner" or "Administrator"
- [ ] Theme toggle (sun/moon) icon visible
- [ ] Refresh button visible
- [ ] User avatar/profile button visible

**Notes**: _________________________________________________________

### Step 1.3: Check Sidebar Navigation
- [ ] Dashboard nav item visible ✅
- [ ] Projects nav item visible ✅
- [ ] **Builds nav item visible** ✅ (Owner/Admin only)
- [ ] Images nav item visible ✅
- [ ] **Tenants nav item visible** ✅ (Owner/Admin only - KEY TEST)
- [ ] Profile nav item visible ✅
- [ ] Settings nav item visible ✅
- [ ] **Admin Dashboard NOT visible** ❌ (unless system admin)

**Critical Check**: Tenants and Builds should be visible for Owner/Admin  
**Pass/Fail**: _____ (must be PASS)

---

## Section 2: Project Management

### Step 2.1: Navigate to Projects
- [ ] Click "Projects" in sidebar
- [ ] Projects page loads
- [ ] Shows list of projects (or empty state)
- [ ] "+ New Project" button visible
- [ ] No console errors

**Notes**: _________________________________________________________

### Step 2.2: View Project Details
- [ ] Click on a project name
- [ ] Project detail page loads
- [ ] Show breadcrumb navigation (Dashboard > Projects > Project Name)
- [ ] Show two tabs: "Overview" and "Members"
- [ ] Overview tab shows project properties

**Notes**: _________________________________________________________

### Step 2.3: Edit Project
- [ ] Click "Edit" button on Overview tab
- [ ] Form becomes editable
- [ ] Fields can be modified:
  - [ ] Project name
  - [ ] Description
  - [ ] Repository URL
  - [ ] Branch
- [ ] Click "Save" button
- [ ] Success message appears
- [ ] Changes are saved and displayed
- [ ] Can verify with page refresh

**Notes**: _________________________________________________________

---

## Section 3: Project Member Management (🎯 CORE NEW FEATURE)

### Step 3.1: View Members Tab
- [ ] Click "Members" tab
- [ ] Members list loads
- [ ] Shows all current members in table with columns:
  - [ ] User ID
  - [ ] Email
  - [ ] Role
  - [ ] Actions
- [ ] "+ Add Member" button visible
- [ ] Table is empty or shows members from previous tests

**Notes**: _________________________________________________________

### Step 3.2: Add Project Member
- [ ] Click "+ Add Member" button
- [ ] Modal opens with:
  - [ ] Search/input field for users
  - [ ] "Add" button (starts disabled)
  - [ ] "Cancel" button
- [ ] Type user name in search (e.g., "alice" or "bob")
- [ ] Dropdown appears with matching users
- [ ] Select a user from dropdown
- [ ] Selected user is highlighted
- [ ] "Add" button becomes enabled
- [ ] Click "Add" button
- [ ] Modal closes
- [ ] Success toast notification appears: "Member added successfully"
- [ ] New member appears in table
- [ ] Member shows with no role initially

**Critical Check**: This is the main new feature, must work perfectly  
**Pass/Fail**: _____ (must be PASS)

**Notes**: _________________________________________________________

### Step 3.3: Edit Member Role
- [ ] In Members table, find the member you just added
- [ ] Click "Edit Role" button for that member
- [ ] Role edit modal opens with:
  - [ ] Role dropdown selector
  - [ ] "Save" and "Cancel" buttons
- [ ] Click role dropdown
- [ ] See available roles:
  - [ ] Owner
  - [ ] Administrator
  - [ ] Developer ← select this for testing
  - [ ] Operator
  - [ ] Viewer
  - [ ] (empty/None)
- [ ] Select "Developer"
- [ ] Click "Save"
- [ ] Modal closes
- [ ] Success toast appears: "Member role updated"
- [ ] Member row now shows "Developer" in Role column
- [ ] Refresh page
- [ ] Role change persists (still shows "Developer")

**Critical Check**: Role persistence is essential  
**Pass/Fail**: _____ (must be PASS)

**Notes**: _________________________________________________________

### Step 3.4: Remove Member
- [ ] Click "Remove" button for the member
- [ ] Confirmation dialog appears asking:
  - [ ] "Are you sure you want to remove this member?"
  - [ ] "Remove" button (red/danger)
  - [ ] "Cancel" button
- [ ] Click "Remove" button
- [ ] Dialog closes
- [ ] Success toast appears: "Member removed successfully"
- [ ] Member disappears from table
- [ ] Refresh page
- [ ] Member is still gone (deletion persisted)

**Critical Check**: Member removal must be persistent  
**Pass/Fail**: _____ (must be PASS)

**Notes**: _________________________________________________________

### Step 3.5: Member Management - Error Cases
- [ ] Try adding same user twice
  - [ ] Should see error: "User already a member"
  - [ ] Member not duplicated
- [ ] Try invalid role assignment
  - [ ] Should handle gracefully
- [ ] Try removing non-existent member
  - [ ] Should handle gracefully

**Pass/Fail**: _____ (pass if errors handled)

**Notes**: _________________________________________________________

---

## Section 4: Build Management

### Step 4.1: Navigate to Builds
- [ ] Click "Builds" in sidebar
- [ ] Builds page loads
- [ ] Shows list of builds (or empty state if none)
- [ ] "+ New Build" button visible

**Notes**: _________________________________________________________

### Step 4.2: Create Build
- [ ] Click "+ New Build" button OR navigate to `/builds/new`
- [ ] Build form loads
- [ ] Select a project from dropdown
- [ ] Enter build name/settings
- [ ] Click "Create Build"
- [ ] Build created and appears in list
- [ ] Can view build details

**Pass/Fail**: _____ 

**Notes**: _________________________________________________________

### Step 4.3: Cancel Build (If Available)
- [ ] Find a running build (may need to create one)
- [ ] Open build details
- [ ] If status is "running" or "pending":
  - [ ] Click "Cancel Build" button
  - [ ] Confirmation dialog appears
  - [ ] Confirm cancellation
  - [ ] Build status changes to "cancelled"

**Pass/Fail**: _____ 

**Notes**: _________________________________________________________

---

## Section 5: Tenant Management (Owner/Admin ONLY)

### Step 5.1: Navigate to Tenants
- [ ] Click "Tenants" in sidebar
- [ ] Tenants page loads
- [ ] Shows list of tenants in table with columns:
  - [ ] Name
  - [ ] Slug
  - [ ] Status (active/suspended/pending)
  - [ ] Quota (max builds, storage)
  - [ ] Created date
  - [ ] Actions (View, Edit, Delete)
- [ ] "+ New Tenant" button visible
- [ ] Breadcrumb shows: Dashboard > Tenants

**Critical Check**: Tenants page is Owner/Admin only feature  
**Pass/Fail**: _____ (must be PASS)

**Notes**: _________________________________________________________

### Step 5.2: View Tenants Table
- [ ] Verify all tenant information displays:
  - [ ] Tenant names are readable
  - [ ] Status badges have correct colors:
    - [ ] Green = active
    - [ ] Red = suspended
    - [ ] Yellow = pending
  - [ ] Quotas show (builds, storage GB)
  - [ ] Creation dates are formatted
- [ ] Table is sortable (if implemented)
- [ ] Responsive on mobile (scroll if needed)

**Notes**: _________________________________________________________

### Step 5.3: Delete Tenant
- [ ] Click "Delete" button for a tenant
- [ ] Confirmation dialog appears:
  - [ ] "Delete Tenant" title
  - [ ] "Are you sure you want to delete this tenant?"
  - [ ] "Delete" button (red)
  - [ ] "Cancel" button
- [ ] Click "Delete"
- [ ] Dialog closes
- [ ] Success toast: "Tenant deleted successfully"
- [ ] Tenant disappears from list
- [ ] Refresh page
- [ ] Tenant is still gone (deletion persisted)

**Pass/Fail**: _____ 

**Notes**: _________________________________________________________

### Step 5.4: Tenant Creation (Future Feature)
- [ ] Click "+ New Tenant" button
- [ ] Form/modal opens (may be NYI - not yet implemented)
- [ ] Try to create tenant if available
  - [ ] Enter tenant name
  - [ ] Enter slug
  - [ ] Configure quotas
  - [ ] Submit form
  - [ ] Verify new tenant appears in list

**Status**: [ ] Implemented [ ] Not Yet Implemented  
**Pass/Fail**: _____ 

**Notes**: _________________________________________________________

---

## Section 6: Role-Based Access Control

### Step 6.1: Test Role Switching
- [ ] Open context switcher (dropdown near tenant name)
- [ ] Switch to a different tenant
- [ ] Verify new tenant is selected
- [ ] Check header shows new tenant name
- [ ] Check header shows role for new tenant
- [ ] Verify nav items change based on new role

**Notes**: _________________________________________________________

### Step 6.2: Viewer Role Restrictions
- [ ] Switch to a tenant where you have "Viewer" role (if available)
- [ ] Verify:
  - [ ] "Tenants" nav item disappears
  - [ ] "Builds" nav item disappears (Viewer can't create builds)
  - [ ] Header shows "Viewer" role
- [ ] Try to access `/tenants` directly via URL
  - [ ] Should redirect to `/dashboard`
  - [ ] Should NOT show error 403
- [ ] Try to access `/builds/new` directly
  - [ ] Should redirect or show access denied

**Critical Check**: Role-based navigation must work correctly  
**Pass/Fail**: _____ 

**Notes**: _________________________________________________________

### Step 6.3: Switch Back to Owner/Admin
- [ ] Use context switcher
- [ ] Switch back to Owner/Admin role tenant
- [ ] Verify:
  - [ ] "Tenants" nav item reappears
  - [ ] "Builds" nav item reappears
  - [ ] Header shows "Owner" or "Administrator"
  - [ ] Can access `/tenants` again

**Pass/Fail**: _____ 

**Notes**: _________________________________________________________

---

## Section 7: Empty States

### Step 7.1: Empty Project Members
- [ ] Create or find a project with no members
- [ ] Go to Project > Members tab
- [ ] Should see empty state or empty table
- [ ] "+ Add Member" button should still be available
- [ ] Should show message about no members
- [ ] Click "+ Add Member"
- [ ] Add a member
- [ ] Verify member appears and empty state is gone

**Pass/Fail**: _____ 

**Notes**: _________________________________________________________

### Step 7.2: Empty Projects (Future Test)
- [ ] Create a new empty tenant (if available)
- [ ] Go to Projects page
- [ ] Should see empty state:
  - [ ] Icon (📁)
  - [ ] Message: "No projects yet"
  - [ ] Button: "Create Project"
- [ ] Click button
- [ ] Should navigate to create project form

**Pass/Fail**: _____ 

**Notes**: _________________________________________________________

---

## Section 8: Dark Mode

### Step 8.1: Toggle Dark Mode
- [ ] Click theme toggle button (☀️/🌙) in header
- [ ] Page switches to dark theme
- [ ] Test on various pages:
  - [ ] Projects page - readable
  - [ ] Project detail - readable
  - [ ] Members tab - readable
  - [ ] Tenants page - readable
- [ ] Toggle back to light mode
- [ ] Page switches correctly
- [ ] Theme persists on refresh

**Pass/Fail**: _____ 

**Notes**: _________________________________________________________

---

## Section 9: Responsive Design

### Step 9.1: Test on Desktop (1920px)
- [ ] Sidebar visible
- [ ] All nav items visible and readable
- [ ] Tables display properly
- [ ] Buttons accessible
- [ ] Modals appear centered

**Pass/Fail**: _____ 

### Step 9.2: Test on Tablet (768px)
- [ ] Sidebar collapses to hamburger menu
- [ ] Can open/close menu
- [ ] Tables are scrollable or responsive
- [ ] Modals resize to fit screen
- [ ] Touch-friendly button sizes

**Pass/Fail**: _____ 

### Step 9.3: Test on Mobile (375px)
- [ ] Full responsive layout
- [ ] All critical functions accessible
- [ ] Readable text (no tiny fonts)
- [ ] Buttons are touch-friendly (at least 44px)
- [ ] Forms are usable on small screen

**Pass/Fail**: _____ 

**Notes**: _________________________________________________________

---

## Section 10: Error Handling & Edge Cases

### Step 10.1: Network Error
- [ ] Stop backend server
- [ ] Try to perform an action (add member, etc.)
- [ ] Should show error message
- [ ] No console crashes
- [ ] Can recover when backend restarted

**Pass/Fail**: _____ 

**Notes**: _________________________________________________________

### Step 10.2: Invalid Data
- [ ] Try to add empty user (no selection)
  - [ ] Button should be disabled
  - [ ] Cannot submit
- [ ] Try to assign invalid role
  - [ ] Should show error
- [ ] Try to remove member twice
  - [ ] Second attempt should fail gracefully

**Pass/Fail**: _____ 

**Notes**: _________________________________________________________

### Step 10.3: Permission Denied
- [ ] As Viewer, try to add member directly via URL
  - [ ] Should deny access
  - [ ] Should show permission error or redirect
- [ ] Try to access admin endpoints
  - [ ] Should deny access
  - [ ] Should redirect to dashboard

**Pass/Fail**: _____ 

**Notes**: _________________________________________________________

---

## Section 11: Audit & Logging

### Step 11.1: Check Backend Logs
- [ ] Watch backend logs during testing
- [ ] Should see entries for:
  - [ ] Member additions
  - [ ] Member role changes
  - [ ] Member removals
  - [ ] Tenant operations
  - [ ] Project updates

**Notes**: _________________________________________________________

### Step 11.2: Verify Audit Trail
- [ ] (If audit UI is available) Check audit logs
- [ ] Should show entries for actions taken
- [ ] Include timestamp, user, action, resource
- [ ] Entries should be immutable (can't edit history)

**Pass/Fail**: _____ 

**Notes**: _________________________________________________________

---

## Section 12: Performance

### Step 12.1: Page Load Times
- [ ] Projects page loads: < 2 seconds
- [ ] Tenants page loads: < 2 seconds
- [ ] Project detail loads: < 2 seconds
- [ ] Members modal opens: < 1 second

**Notes**: _________________________________________________________

### Step 12.2: Data Loading
- [ ] Large member lists load without freezing
- [ ] Pagination works (if implemented)
- [ ] Search is responsive
- [ ] Filters apply quickly

**Pass/Fail**: _____ 

**Notes**: _________________________________________________________

---

## Final Summary

### Critical Tests (Must All Pass ✅)
- [ ] Owner/Admin can see Tenants nav item
- [ ] Owner/Admin can see Builds nav item
- [ ] Owner/Admin can add members to project
- [ ] Owner/Admin can edit member role
- [ ] Owner/Admin can remove member
- [ ] Member changes persist after refresh
- [ ] Role-based nav works correctly
- [ ] Tenants page shows list with delete

### Important Tests (Should Pass ✅)
- [ ] Build management works
- [ ] Project editing works
- [ ] Empty states display
- [ ] Error messages helpful
- [ ] Dark mode works
- [ ] Responsive on all devices

### Nice-to-Have Tests (Would Be Good ✅)
- [ ] Tenant creation form works
- [ ] Audit logs visible
- [ ] Performance is good
- [ ] Advanced filtering works

### Test Results

| Area | Pass/Fail | Notes |
|------|-----------|-------|
| Navigation | [ ] | |
| Project Management | [ ] | |
| Member Management | [ ] | |
| Build Management | [ ] | |
| Tenant Management | [ ] | |
| Role-Based Access | [ ] | |
| Error Handling | [ ] | |
| UI/UX | [ ] | |
| Performance | [ ] | |
| **Overall** | [ ] | |

---

## Sign-Off

**Tester**: ________________________  
**Date**: ________________________  
**Overall Result**: ⭕ PASS ⭕ FAIL ⭕ PARTIAL  

**Issues Found**:
```
1. _______________________________________________
2. _______________________________________________
3. _______________________________________________
```

**Recommendations**:
```
1. _______________________________________________
2. _______________________________________________
3. _______________________________________________
```

**Next Steps**:
```
1. _______________________________________________
2. _______________________________________________
```

