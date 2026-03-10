# Complete Permission Matrix - All Roles x All Operations

**Purpose**: Quick lookup for all permissions  
**Format**: Read as "Can [Role] [Action] on [Resource]?"  
**Date**: January 29, 2026

---

## SCOPE HIERARCHY

```
System Level
├─ Tenant (system admin assigns users to tenants)
│  ├─ Project (tenant admin creates projects)
│  │  ├─ Build Config (engineer defines configs)
│  │  │  └─ Build Pipeline (runs builds)
```

**Permission Flow**:
- System Admin: Access ALL tenants, projects, builds
- Tenant Admin: Access ASSIGNED tenants, their projects, all builds
- Tenant Op/Build Eng/Dev: Access ASSIGNED tenant + ASSIGNED projects within that tenant
- Developer: Can only create/edit builds for Dockerfile/Buildx, only in assigned projects
- Viewer: Read-only access to assigned projects and builds

---

## PROJECT SCOPE OPERATIONS MATRIX

| Operation | Scope | Sys Admin | Tenant Admin | Tenant Op | Build Eng | Developer | Sec Rev | Viewer |
|-----------|-------|:---:|:---:|:---:|:---:|:---:|:---:|:---:|
| **View Projects** | All | ✅ | ✅ Tenant | ✅ Tenant | ✅ Assigned | ✅ Assigned | ❌ | ✅ Assigned |
| **Create Project** | Tenant | ✅ | ✅ | ✅ | ❌ | ❌ | ❌ | ❌ |
| **Edit Project** | Project | ✅ | ✅ | ✅ | ❌ | ❌ | ❌ | ❌ |
| **Delete Project** | Tenant | ✅ | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ |
| **Manage Project Members** | Project | ✅ | ✅ | ✅ | ✅ | ❌ | ❌ | ❌ |
| **Set Project Quota** | Project | ✅ | ✅ | ✅ | ❌ | ❌ | ❌ | ❌ |
| **Archive Project** | Tenant | ✅ | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ |

---

## BUILD CONFIG OPERATIONS MATRIX

| Operation | Scope | Sys Admin | Tenant Admin | Tenant Op | Build Eng | Developer | Sec Rev | Viewer |
|-----------|-------|:---:|:---:|:---:|:---:|:---:|:---:|:---:|
| **Create Build Config** | Project | ✅ | ✅ | ✅ | ✅ | ⚠️* | ❌ | ❌ |
| **Edit Build Config** | Config | ✅ | ✅ | ✅ Own | ✅ Own | ⚠️ Own* | ❌ | ❌ |
| **Delete Build Config** | Project | ✅ | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ |
| **View Build Configs** | Project | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| **Clone Build Config** | Project | ✅ | ✅ | ✅ | ✅ | ⚠️* | ❌ | ❌ |

`*` Developer: Only Dockerfile/Buildx methods

---

## BUILD OPERATIONS MATRIX

| Operation | Scope | Sys Admin | Sec Admin | Op Admin | Auditor | Ten Admin | Ten Op | Build Eng | Dev | Sec Rev | Viewer |
|-----------|-------|:---:|:---:|:---:|:---:|:---:|:---:|:---:|:---:|:---:|:---:|
| **View Builds** | Config | ✅ All | ✅ All | ✅ All | ✅ All | ✅ Ten | ✅ Ten | ✅ Ten | ✅ Ten | ✅ Ten | ✅ Ten |
| **Create Build** | Config | ✅ | ❌ | ✅ | ❌ | ✅ | ✅ | ✅ | ⚠️* | ❌ | ❌ |
| **Edit Build** | Build | ✅ Any | ❌ | ⚠️ Own | ❌ | ✅ Any | ⚠️ Own | ✅ Own | ⚠️ Own | ❌ | ❌ |
| **Cancel Build** | Build | ✅ Any | ❌ | ⚠️ Own | ❌ | ✅ Any | ⚠️ Own | ✅ Own | ⚠️ Own | ❌ | ❌ |
| **Retry Build** | Build | ✅ Any | ❌ | ⚠️ Own | ❌ | ✅ Any | ⚠️ Own | ✅ Own | ⚠️ Own | ❌ | ❌ |
| **Delete Build** | Build | ✅ | ❌ | ❌ | ❌ | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ |
| **View Logs** | Build | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| **Approve Build** | Build | ✅ | ✅ | ❌ | ❌ | ✅ | ❌ | ❌ | ❌ | ✅ | ❌ |

**Legend**:
- ✅ = Can do
- ⚠️ = Conditional (own only, own tenant only, etc.)
- ❌ = Cannot do
- `*` = Developer: Dockerfile & Buildx only

---

## PROJECT OPERATIONS MATRIX

| Operation | Sys Admin | Sec Admin | Op Admin | Auditor | Ten Admin | Ten Op | Build Eng | Dev | Sec Rev | Viewer |
|-----------|:---:|:---:|:---:|:---:|:---:|:---:|:---:|:---:|:---:|:---:|
| **View Projects** | ✅ All | ❌ | ✅ All | ❌ | ✅ Ten | ✅ Ten | ✅ Ten | ✅ Ten | ❌ | ✅ Ten |
| **Create Project** | ✅ | ❌ | ✅ | ❌ | ✅ | ✅ | ❌ | ❌ | ❌ | ❌ |
| **Edit Project** | ✅ | ❌ | ✅ | ❌ | ✅ | ✅ | ❌ | ❌ | ❌ | ❌ |
| **Delete Project** | ✅ | ❌ | ✅ | ❌ | ✅ | ✅ | ❌ | ❌ | ❌ | ❌ |
| **View Project Stats** | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |

---

## USER MANAGEMENT MATRIX

| Operation | Sys Admin | Sec Admin | Op Admin | Auditor | Ten Admin | Ten Op | Build Eng | Dev | Sec Rev | Viewer |
|-----------|:---:|:---:|:---:|:---:|:---:|:---:|:---:|:---:|:---:|:---:|
| **View Users (All)** | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ |
| **View Users (Tenant)** | ✅ | ❌ | ❌ | ❌ | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ |
| **Create User** | ✅ | ❌ | ❌ | ❌ | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ |
| **Edit User** | ✅ | ❌ | ❌ | ❌ | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ |
| **Delete User** | ✅ | ❌ | ❌ | ❌ | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ |
| **Suspend User** | ✅ | ❌ | ❌ | ❌ | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ |
| **Manage Roles** | ✅ | ❌ | ❌ | ❌ | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ |

---

## ROLE MANAGEMENT MATRIX

| Operation | Sys Admin | Sec Admin | Op Admin | Auditor | Ten Admin | Ten Op | Build Eng | Dev | Sec Rev | Viewer |
|-----------|:---:|:---:|:---:|:---:|:---:|:---:|:---:|:---:|:---:|:---:|
| **View Roles (All)** | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ |
| **View Roles (Tenant)** | ✅ | ❌ | ❌ | ❌ | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ |
| **Create Role** | ✅ | ❌ | ❌ | ❌ | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ |
| **Edit Role** | ✅ | ❌ | ❌ | ❌ | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ |
| **Delete Role** | ✅ | ❌ | ❌ | ❌ | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ |
| **Assign Roles** | ✅ | ❌ | ❌ | ❌ | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ |

---

## TENANT MANAGEMENT MATRIX

| Operation | Sys Admin | Sec Admin | Op Admin | Auditor | Ten Admin | Ten Op | Build Eng | Dev | Sec Rev | Viewer |
|-----------|:---:|:---:|:---:|:---:|:---:|:---:|:---:|:---:|:---:|:---:|
| **View All Tenants** | ✅ | ❌ | ❌ | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ |
| **View Tenant Details** | ✅ | ❌ | ❌ | ❌ | ✅ | ✅ | ✅ | ✅ | ✅ | ❌ |
| **Create Tenant** | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ |
| **Edit Tenant** | ✅ | ❌ | ❌ | ❌ | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ |
| **Suspend Tenant** | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ |
| **Delete Tenant** | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ |
| **Manage Quotas** | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ |

---

## SECURITY OPERATIONS MATRIX

| Operation | Sys Admin | Sec Admin | Op Admin | Auditor | Ten Admin | Ten Op | Build Eng | Dev | Sec Rev | Viewer |
|-----------|:---:|:---:|:---:|:---:|:---:|:---:|:---:|:---:|:---:|:---:|
| **View Scan Results** | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| **View Vulnerabilities** | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| **Quarantine Image** | ✅ | ✅ | ✅ | ❌ | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ |
| **Approve Image** | ✅ | ✅ | ❌ | ❌ | ✅ | ❌ | ❌ | ❌ | ✅ | ❌ |
| **Reject Image** | ✅ | ✅ | ❌ | ❌ | ✅ | ❌ | ❌ | ❌ | ✅ | ❌ |
| **Set Security Policy** | ✅ | ✅ | ❌ | ❌ | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ |

---

## ADMIN & SETTINGS MATRIX

| Operation | Sys Admin | Sec Admin | Op Admin | Auditor | Ten Admin | Ten Op | Build Eng | Dev | Sec Rev | Viewer |
|-----------|:---:|:---:|:---:|:---:|:---:|:---:|:---:|:---:|:---:|:---:|
| **Tool Availability (All)** | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ |
| **Tool Availability (Tenant)** | ✅ | ❌ | ❌ | ❌ | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ |
| **System Settings** | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ |
| **Registry Credentials** | ✅ | ❌ | ✅ | ❌ | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ |
| **Secret Management** | ✅ | ❌ | ❌ | ❌ | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ |

---

## AUDIT & COMPLIANCE MATRIX

| Operation | Sys Admin | Sec Admin | Op Admin | Auditor | Ten Admin | Ten Op | Build Eng | Dev | Sec Rev | Viewer |
|-----------|:---:|:---:|:---:|:---:|:---:|:---:|:---:|:---:|:---:|:---:|
| **View Audit Logs (All)** | ✅ | ✅ | ✅ | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ |
| **View Audit Logs (Tenant)** | ✅ | ✅ | ❌ | ❌ | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ |
| **Export Audit Logs** | ✅ | ✅ | ❌ | ✅ | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ |
| **View Compliance Reports** | ✅ | ✅ | ❌ | ✅ | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ |
| **Generate Report** | ✅ | ✅ | ❌ | ✅ | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ |

---

## BUILD METHOD AVAILABILITY BY ROLE

| Build Method | Sys Admin | Build Eng | Tenant Op | Developer | Viewer |
|--------------|:---:|:---:|:---:|:---:|:---:|
| **Packer** (Infrastructure) | ✅ | ✅ | ✅ | ❌ | ❌ |
| **Paketo** (Buildpacks) | ✅ | ✅ | ✅ | ❌ | ❌ |
| **Kaniko** (K8s-native) | ✅ | ✅ | ✅ | ❌ | ❌ |
| **Dockerfile/Buildx** | ✅ | ✅ | ✅ | ✅ | ❌ |

---

## TOOL CONFIGURATION BY ROLE

| Tool Configuration | Sys Admin | Tenant Admin | Build Engineer | Developer |
|-------------------|:---:|:---:|:---:|:---:|
| **SBOM Tool Selection** | ✅ System-wide | ✅ Tenant-wide | ⚠️* | ⚠️* |
| **Security Scanner** | ✅ System-wide | ✅ Tenant-wide | ⚠️* | ⚠️* |
| **Registry Selection** | ✅ All | ✅ Tenant | ✅ Own builds | ❌ |
| **Secret Manager** | ✅ All | ✅ Tenant | ✅ Own builds | ❌ |
| **Advanced Options** | ✅ | ✅ | ✅ | ❌ |

`*` Limited to available/enabled tools

---

## PAGE ACCESS MATRIX

| Page | Sys Admin | Sec Admin | Op Admin | Auditor | Ten Admin | Ten Op | Build Eng | Dev | Sec Rev | Viewer |
|------|:---:|:---:|:---:|:---:|:---:|:---:|:---:|:---:|:---:|:---:|
| `/dashboard` | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| `/builds` | ✅ All | ✅ All | ✅ All | ✅ All | ✅ Ten | ✅ Ten | ✅ Ten | ✅ Ten | ✅ Ten | ✅ Ten |
| `/builds/new` | ✅ | ❌ | ✅ | ❌ | ✅ | ✅ | ✅ | ⚠️* | ❌ | ❌ |
| `/builds/{id}` | ✅ All | ✅ All | ✅ All | ✅ All | ✅ Ten | ✅ Ten | ✅ Ten | ✅ Ten | ✅ Ten | ✅ Ten |
| `/projects` | ✅ All | ❌ | ✅ All | ❌ | ✅ Ten | ✅ Ten | ✅ Ten | ✅ Ten | ❌ | ✅ Ten |
| `/projects/new` | ✅ | ❌ | ✅ | ❌ | ✅ | ✅ | ❌ | ❌ | ❌ | ❌ |
| `/projects/{id}` | ✅ All | ❌ | ✅ All | ❌ | ✅ Ten | ✅ Ten | ✅ Ten | ✅ Ten | ❌ | ✅ Ten |
| `/admin/users` | ✅ All | ❌ | ❌ | ❌ | ✅ Ten | ❌ | ❌ | ❌ | ❌ | ❌ |
| `/admin/roles` | ✅ All | ❌ | ❌ | ❌ | ✅ Ten | ❌ | ❌ | ❌ | ❌ | ❌ |
| `/admin/tenants` | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ |
| `/admin/tools` | ✅ | ❌ | ❌ | ❌ | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ |
| `/admin/settings` | ✅ | ❌ | ❌ | ❌ | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ |
| `/admin/audit` | ✅ | ✅ | ✅ | ✅ | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ |
| `/security/policies` | ✅ | ✅ | ❌ | ❌ | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ |

---

## Summary Statistics

### by Role

| Role | Total Permissions | Can Create | Can Edit | Can Delete | Can Approve |
|------|:---:|:---:|:---:|:---:|:---:|
| System Admin | 90+ | ✅ | ✅ | ✅ | ✅ |
| Security Admin | 40+ | ❌ | ❌ | ❌ | ✅ |
| Op Admin | 30+ | ✅ | ✅ | ✅ | ❌ |
| Global Auditor | 20+ | ❌ | ❌ | ❌ | ❌ |
| Tenant Admin | 60+ | ✅ | ✅ | ✅ | ✅ |
| Tenant Op | 40+ | ✅ Own | ✅ Own | ❌ | ❌ |
| Build Engineer | 35+ | ✅ | ✅ Own | ❌ | ❌ |
| Developer | 25+ | ⚠️ Basic | ✅ Own | ❌ | ❌ |
| Security Reviewer | 20+ | ❌ | ❌ | ❌ | ✅ |
| Viewer | 15+ | ❌ | ❌ | ❌ | ❌ |

### by Operation

| Operation | # Roles with Access | Most Restrictive |
|-----------|:---:|---|
| Create Build | 5 | Viewer, Security Reviewer |
| Edit Build | 6 (some own-only) | Viewer, Security Reviewer |
| Delete Build | 2 | Everyone except System/Tenant Admin |
| Approve Build | 3 | Most roles |
| Manage Users | 2 | Everyone except System/Tenant Admin |
| Manage Roles | 2 | Everyone except System/Tenant Admin |
| View Audit Logs | 4 | Most roles |

---

## Using This Matrix

### To Check If Role Can Do Something:
1. Find role in column
2. Find operation in row
3. ✅ = Can do, ❌ = Cannot, ⚠️ = Conditional

### To Test Implementation:
1. For each ✅: Write test that passes
2. For each ❌: Write test that fails with permission denied
3. For each ⚠️: Write test for the condition

### To Debug Permission Issues:
1. Verify row and column intersection
2. Check for conditional logic (⚠️)
3. Verify backend enforces it
4. Verify frontend hides UI element

---

## Export for Testing

```go
// Use this in tests
var permissionMatrix = map[string]map[string]bool{
  "system-admin": {
    "create_build": true,
    "delete_build": true,
    "approve_build": true,
    // ...all 90+ permissions
  },
  "developer": {
    "create_build": true,  // Dockerfile only
    "delete_build": false,
    "approve_build": false,
    // ...35 permissions
  },
  // ...all 10 roles
}

// In test:
if permissionMatrix[role][operation] != expected {
  t.Errorf("Permission mismatch for %s.%s", role, operation)
}
```

---

## Related Documents

- [ROLE_BASED_UX_DESIGN.md](ROLE_BASED_UX_DESIGN.md) - UX details for each role
- [ROLE_BASED_TDD_TEST_PLAN.md](ROLE_BASED_TDD_TEST_PLAN.md) - Test structure
- [PHASE_2_2_4_IMPLEMENTATION_ROADMAP.md](PHASE_2_2_4_IMPLEMENTATION_ROADMAP.md) - Schedule
- [RBAC-Permissions-Model.md](RBAC-Permissions-Model.md) - Full RBAC spec
