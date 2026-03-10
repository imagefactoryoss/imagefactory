# Owner/Admin Role - Feature & Permission Checklist

## Role Types
- **RBAC Role**: "Owner" or "Administrator" (tenant-specific RBAC system)
- **Legacy Fallback**: Users with tenant group `role_type` = "owner" or "administrator"

## Navigation & Visibility

### ✅ Always Visible (All Authenticated Users)
- **Dashboard** (`/dashboard`) - Overview and analytics
- **Projects** (`/projects`) - List all projects in current tenant
- **Images** (`/images`) - Browse container image catalog
- **Profile** (`/profile`) - View user profile
- **Settings** (`/settings`) - User account settings

### ✅ Owner/Admin Only
- **Builds** (`/builds`) - Build management and creation
- **Tenants** (`/tenants`) - Manage tenant organizations (only if Owner/Admin role)

### ❌ System Admin Only
- **Admin Dashboard** (`/admin/dashboard`) - System-wide statistics and management

---

## Feature Breakdown by Page

### 📊 Dashboard
- **Available to**: All authenticated users
- **Features**:
  - View overview/analytics for current tenant
  - See aggregated stats across projects

### 📁 Projects
- **Available to**: All authenticated users (viewing)
- **Owner/Admin Capabilities**:
  - ✅ Create new project (`/projects/new`)
  - ✅ View project details
  - ✅ Edit project properties:
    - Project name
    - Description
    - Repository URL
    - Branch
  - ✅ View project members (NEW - Member Management Feature)
  - ✅ Manage project members:
    - **Add members** to project
    - **Edit member roles** (assign RBAC roles)
    - **Remove members** from project
  - ✅ View build history for project
  - ✅ Create builds from project

### 🔨 Builds
- **Available to**: Owner, Admin, Developer, Operator roles
- **Owner/Admin Capabilities**:
  - ✅ View all builds in tenant
  - ✅ Create new build (`/builds/new`)
  - ✅ View build details
  - ✅ View build logs
  - ✅ Cancel running builds (if in progress)
  - ✅ Restart/retry builds

### 🖼️ Images
- **Available to**: All authenticated users (read-only default)
- **Features**:
  - ✅ Browse image catalog
  - ✅ Search/filter images
  - ✅ View image details
  - ✅ View image versions and tags

### 🏢 Tenants (Owner/Admin Only)
- **Available to**: Owner/Admin role only
- **Features**:
  - ✅ View all tenants in system
  - ✅ Create new tenant
  - ✅ View tenant details
  - ✅ Edit tenant properties (future)
  - ✅ Delete tenants
  - ✅ View tenant quotas

### ⚙️ Settings (All Users)
- **Available to**: All authenticated users
- **Features**:
  - ✅ Account settings (email, theme, timezone)
  - ✅ Security settings (password, 2FA) - planned
  - ✅ Notification preferences - planned

### 👤 Profile (All Users)
- **Available to**: All authenticated users
- **Features**:
  - ✅ View user profile
  - ✅ Edit basic profile info
  - ✅ Manage profile picture

---

## Member Management Capabilities (NEW - Owner/Admin in Projects)

### 📋 For Project Members
The Owner/Admin can now manage who has access to projects:

#### Add Members
- ✅ Search and add users to project
- ✅ Add by selecting available users
- ✅ User must exist in system first

#### Edit Member Roles
- ✅ Change member's RBAC role
- ✅ Available roles: Owner, Administrator, Developer, Operator, Viewer
- ✅ Clear role (remove specific role)

#### Remove Members
- ✅ Delete member from project
- ✅ Confirmation dialog before removal
- ✅ Audit log entry created

#### View Member Details
- ✅ See member name and email
- ✅ See assigned role
- ✅ See join date

---

## Permission Checks in Code

### Access Control Logic
```
Is Owner/Admin? → canManageTenants = TRUE, canCreateBuilds = TRUE
Is Developer/Operator? → canCreateBuilds = TRUE, canManageTenants = FALSE
Is Viewer? → canCreateBuilds = FALSE, canManageTenants = FALSE
All Users? → canViewImages = TRUE, canManageSettings = TRUE
```

---

## Admin Dashboard (System Admin Only)
- ✅ View system-wide statistics
- ✅ Manage users globally
- ✅ View audit logs
- ✅ System configuration
- ⚠️ Currently redirects non-admins to dashboard

---

## Testing Plan for Owner/Admin Role

### Test User Profile
- **Username**: (an Owner-level user from your LDAP/database)
- **Roles**: Owner or Administrator in at least one tenant
- **Expected**: Should see Projects, Builds, Images, Tenants, Settings

### Test Cases to Execute

#### 1. Navigation Visibility
- [ ] Can see Tenants nav item
- [ ] Can see Builds nav item
- [ ] Cannot see Admin Dashboard nav item (unless system admin)

#### 2. Project Management
- [ ] View list of projects
- [ ] Create new project ✅ (already tested)
- [ ] Edit project details
- [ ] Delete project (if implemented)

#### 3. Member Management (NEW - High Priority)
- [ ] Open project detail page
- [ ] Click "Members" tab
- [ ] See list of existing members
- [ ] Click "Add Member" button
- [ ] Search for user to add
- [ ] Add member to project ✅ (already tested)
- [ ] Edit member role (change role dropdown)
- [ ] Delete member (with confirmation)
- [ ] Verify member is removed from list

#### 4. Build Management
- [ ] View list of builds
- [ ] Create build from project
- [ ] View build details and logs
- [ ] Cancel build (if running)

#### 5. Image Management
- [ ] Browse image catalog
- [ ] Search images
- [ ] View image details

#### 6. Tenant Management
- [ ] View list of tenants (Owner/Admin only)
- [ ] Create new tenant
- [ ] Edit tenant (future)
- [ ] Delete tenant (future)

#### 7. Role-Based Visibility
- [ ] Switch to a different tenant with only Viewer role
- [ ] Verify Tenants nav item is hidden
- [ ] Verify Builds nav item is hidden
- [ ] Switch back to Owner role tenant
- [ ] Verify nav items reappear

---

## Audit Trail

All Owner/Admin actions are logged:
- ✅ Member additions
- ✅ Member role changes
- ✅ Member removals
- ✅ Project updates
- ✅ Build operations
- ✅ Tenant operations

---

## Known Limitations / Future Enhancements

1. **Bulk Operations** - Cannot bulk-manage multiple members/projects yet
2. **Batch Permissions** - No role assignment templates
3. **Quota Management** - Cannot modify tenant quotas from UI yet
4. **Project Transfer** - Cannot transfer project ownership to another user
5. **Member Invitations** - Cannot invite external users (must exist in system first)
6. **Scheduled Builds** - Not yet implemented
7. **Build Webhooks** - Not yet configured
8. **Image Push Permissions** - Need to verify who can push images

---

## Related Components & Files

### Components
- `frontend/src/pages/projects/ProjectDetailPage.tsx` - Project details with tabs
- `frontend/src/components/projects/ProjectMembersUI.tsx` - Member management UI
- `frontend/src/pages/tenants/TenantsPage.tsx` - Tenant listing (new)
- `frontend/src/components/layout/Layout.tsx` - Role-based navigation

### Services
- `frontend/src/services/projectService.ts` - Project API calls
- `frontend/src/services/memberApi.ts` - Member management API
- `frontend/src/services/tenantService.ts` - Tenant API calls

### Backend Endpoints
- `GET /api/v1/projects` - List projects
- `POST /api/v1/projects` - Create project
- `GET /api/v1/projects/{id}` - Get project
- `PATCH /api/v1/projects/{id}` - Update project
- `GET /api/v1/projects/{id}/members` - List members
- `POST /api/v1/projects/{id}/members` - Add member
- `PATCH /api/v1/projects/{id}/members/{userId}` - Update member role
- `DELETE /api/v1/projects/{id}/members/{userId}` - Remove member
- `GET /api/v1/tenants` - List tenants
- `POST /api/v1/tenants` - Create tenant
- `DELETE /api/v1/tenants/{id}` - Delete tenant

---

## Quick Reference: Owner/Admin vs Other Roles

| Feature | Owner/Admin | Developer | Operator | Viewer | System Admin |
|---------|-----------|-----------|----------|--------|-------------|
| Dashboard | ✅ | ✅ | ✅ | ✅ | ✅ |
| Projects | ✅ (CRUD) | ✅ (R) | ✅ (R) | ✅ (R) | ✅ (R) |
| Builds | ✅ (CRUD) | ✅ (CRUD) | ✅ (RUN) | ✅ (R) | ✅ |
| Images | ✅ | ✅ | ✅ | ✅ | ✅ |
| Members | ✅ (MANAGE) | ✅ (VIEW) | ✅ (VIEW) | ✅ (VIEW) | ✅ |
| Tenants | ✅ (MANAGE) | ❌ | ❌ | ❌ | ✅ |
| Settings | ✅ (SELF) | ✅ (SELF) | ✅ (SELF) | ✅ (SELF) | ✅ (SELF) |
| Admin Panel | ❌ | ❌ | ❌ | ❌ | ✅ |

Legend: CRUD = Create/Read/Update/Delete, R = Read, RUN = Run/Start, MANAGE = Full control, VIEW = Read-only

