# Owner/Admin Role - Feature Matrix

## Complete Feature Matrix by Role

```
╔════════════════════════════════════════════════════════════════════════════╗
║                          ROLE-BASED FEATURE MATRIX                         ║
╠════════════════════════════════════════════════════════════════════════════╣
║ FEATURE                    │ Owner │ Admin │ Dev │ Op │ View │ SysAdmin │
║────────────────────────────┼───────┼───────┼─────┼────┼──────┼──────────┤
║ NAVIGATION                 │       │       │     │    │      │          │
║ - Dashboard                │  ✅   │  ✅   │ ✅  │ ✅ │  ✅  │    ✅     │
║ - Projects                 │  ✅   │  ✅   │ ✅  │ ✅ │  ✅  │    ✅     │
║ - Builds                   │  ✅   │  ✅   │ ✅  │ ✅ │  ❌  │    ✅     │
║ - Images                   │  ✅   │  ✅   │ ✅  │ ✅ │  ✅  │    ✅     │
║ - Tenants                  │  ✅   │  ✅   │ ❌  │ ❌ │  ❌  │    ✅     │
║ - Settings                 │  ✅   │  ✅   │ ✅  │ ✅ │  ✅  │    ✅     │
║ - Profile                  │  ✅   │  ✅   │ ✅  │ ✅ │  ✅  │    ✅     │
║ - Admin Panel              │  ❌   │  ❌   │ ❌  │ ❌ │  ❌  │    ✅     │
║────────────────────────────┼───────┼───────┼─────┼────┼──────┼──────────┤
║ PROJECT MANAGEMENT         │       │       │     │    │      │          │
║ - List projects            │  ✅   │  ✅   │ ✅  │ ✅ │  ✅  │    ✅     │
║ - Create project           │  ✅   │  ✅   │ ✅  │ ✅ │  ❌  │    ✅     │
║ - Edit project             │  ✅   │  ✅   │ ✅  │ ✅ │  ❌  │    ✅     │
║ - Delete project           │  ✅   │  ✅   │ ❌  │ ❌ │  ❌  │    ✅     │
║────────────────────────────┼───────┼───────┼─────┼────┼──────┼──────────┤
║ MEMBER MANAGEMENT (NEW)    │       │       │     │    │      │          │
║ - View members             │  ✅   │  ✅   │ ✅  │ ✅ │  ✅  │    ✅     │
║ - Add member               │  ✅   │  ✅   │ ❌  │ ❌ │  ❌  │    ✅     │
║ - Edit member role         │  ✅   │  ✅   │ ❌  │ ❌ │  ❌  │    ✅     │
║ - Remove member            │  ✅   │  ✅   │ ❌  │ ❌ │  ❌  │    ✅     │
║ - Assign RBAC roles        │  ✅   │  ✅   │ ❌  │ ❌ │  ❌  │    ✅     │
║────────────────────────────┼───────┼───────┼─────┼────┼──────┼──────────┤
║ BUILD MANAGEMENT           │       │       │     │    │      │          │
║ - List builds              │  ✅   │  ✅   │ ✅  │ ✅ │  ✅  │    ✅     │
║ - Create build             │  ✅   │  ✅   │ ✅  │ ✅ │  ❌  │    ✅     │
║ - View build details       │  ✅   │  ✅   │ ✅  │ ✅ │  ✅  │    ✅     │
║ - Cancel build             │  ✅   │  ✅   │ ✅  │ ✅ │  ❌  │    ✅     │
║ - Restart build            │  ✅   │  ✅   │ ✅  │ ✅ │  ❌  │    ✅     │
║────────────────────────────┼───────┼───────┼─────┼────┼──────┼──────────┤
║ IMAGE MANAGEMENT           │       │       │     │    │      │          │
║ - List images              │  ✅   │  ✅   │ ✅  │ ✅ │  ✅  │    ✅     │
║ - Search images            │  ✅   │  ✅   │ ✅  │ ✅ │  ✅  │    ✅     │
║ - View image details       │  ✅   │  ✅   │ ✅  │ ✅ │  ✅  │    ✅     │
║ - Push image               │  ✅   │  ✅   │ ✅  │ ✅ │  ❌  │    ✅     │
║────────────────────────────┼───────┼───────┼─────┼────┼──────┼──────────┤
║ TENANT MANAGEMENT          │       │       │     │    │      │          │
║ - List tenants             │  ✅   │  ✅   │ ❌  │ ❌ │  ❌  │    ✅     │
║ - Create tenant            │  ✅   │  ✅   │ ❌  │ ❌ │  ❌  │    ✅     │
║ - Edit tenant              │  ✅   │  ✅   │ ❌  │ ❌ │  ❌  │    ✅     │
║ - Delete tenant            │  ✅   │  ✅   │ ❌  │ ❌ │  ❌  │    ✅     │
║ - Set quotas               │  ✅   │  ✅   │ ❌  │ ❌ │  ❌  │    ✅     │
║────────────────────────────┼───────┼───────┼─────┼────┼──────┼──────────┤
║ PERSONAL                   │       │       │     │    │      │          │
║ - Edit own profile         │  ✅   │  ✅   │ ✅  │ ✅ │  ✅  │    ✅     │
║ - Edit own settings        │  ✅   │  ✅   │ ✅  │ ✅ │  ✅  │    ✅     │
║ - Manage own 2FA           │  ✅   │  ✅   │ ✅  │ ✅ │  ✅  │    ✅     │
║ - Change own password      │  ✅   │  ✅   │ ✅  │ ✅ │  ✅  │    ✅     │
║────────────────────────────┼───────┼───────┼─────┼────┼──────┼──────────┤
║ ADMINISTRATION             │       │       │     │    │      │          │
║ - System config            │  ❌   │  ❌   │ ❌  │ ❌ │  ❌  │    ✅     │
║ - User management          │  ❌   │  ❌   │ ❌  │ ❌ │  ❌  │    ✅     │
║ - View audit logs (global) │  ❌   │  ❌   │ ❌  │ ❌ │  ❌  │    ✅     │
║ - System stats             │  ❌   │  ❌   │ ❌  │ ❌ │  ❌  │    ✅     │
╚════════════════════════════════════════════════════════════════════════════╝

Legend:
  ✅ = Full access/capability
  ❌ = No access
  Owner/Admin: Owner and Administrator roles (same permissions)
  Dev = Developer role
  Op = Operator role
  View = Viewer role
  SysAdmin = System Administrator (global)
```

---

## Feature Breakdown

### Owner & Administrator Roles (Same Permissions)

#### Tenant-Scoped Permissions
- **Owner role** in Tenant A = Can manage Tenant A and all its projects/builds
- **Admin role** in Tenant B = Can manage Tenant B and all its projects/builds
- Can only manage resources within assigned tenant(s)
- Cannot access other tenants (unless explicitly assigned)

#### Global Permissions (If System Admin)
- System Administrator = Can manage all tenants and system-wide resources
- Cannot accidentally manage other tenants

---

## Page-by-Page Feature List

### 1️⃣ Dashboard Page
| Feature | Owner | Admin | Dev | Op | View |
|---------|-------|-------|-----|----|----|
| View overview | ✅ | ✅ | ✅ | ✅ | ✅ |
| View analytics | ✅ | ✅ | ✅ | ✅ | ✅ |
| View stats | ✅ | ✅ | ✅ | ✅ | ✅ |

### 2️⃣ Projects Page
| Feature | Owner | Admin | Dev | Op | View |
|---------|-------|-------|-----|----|----|
| List projects | ✅ | ✅ | ✅ | ✅ | ✅ |
| Create project | ✅ | ✅ | ✅ | ✅ | ❌ |
| Search projects | ✅ | ✅ | ✅ | ✅ | ✅ |
| Filter projects | ✅ | ✅ | ✅ | ✅ | ✅ |

### 3️⃣ Project Detail Page
| Feature | Owner | Admin | Dev | Op | View |
|---------|-------|-------|-----|----|----|
| View details | ✅ | ✅ | ✅ | ✅ | ✅ |
| Edit project | ✅ | ✅ | ✅ | ✅ | ❌ |
| Delete project | ✅ | ✅ | ❌ | ❌ | ❌ |
| **View members** | ✅ | ✅ | ✅ | ✅ | ✅ |
| **Add member** | ✅ | ✅ | ❌ | ❌ | ❌ |
| **Edit member role** | ✅ | ✅ | ❌ | ❌ | ❌ |
| **Remove member** | ✅ | ✅ | ❌ | ❌ | ❌ |
| View builds | ✅ | ✅ | ✅ | ✅ | ✅ |

### 4️⃣ Builds Page
| Feature | Owner | Admin | Dev | Op | View |
|---------|-------|-------|-----|----|----|
| List builds | ✅ | ✅ | ✅ | ✅ | ✅ |
| Create build | ✅ | ✅ | ✅ | ✅ | ❌ |
| Filter builds | ✅ | ✅ | ✅ | ✅ | ✅ |
| Search builds | ✅ | ✅ | ✅ | ✅ | ✅ |

### 5️⃣ Build Detail Page
| Feature | Owner | Admin | Dev | Op | View |
|---------|-------|-------|-----|----|----|
| View details | ✅ | ✅ | ✅ | ✅ | ✅ |
| View logs | ✅ | ✅ | ✅ | ✅ | ✅ |
| Cancel build | ✅ | ✅ | ✅ | ✅ | ❌ |
| Restart build | ✅ | ✅ | ✅ | ✅ | ❌ |

### 6️⃣ Images Page
| Feature | Owner | Admin | Dev | Op | View |
|---------|-------|-------|-----|----|----|
| List images | ✅ | ✅ | ✅ | ✅ | ✅ |
| Search images | ✅ | ✅ | ✅ | ✅ | ✅ |
| Filter images | ✅ | ✅ | ✅ | ✅ | ✅ |
| Push image | ✅ | ✅ | ✅ | ✅ | ❌ |

### 7️⃣ Image Detail Page
| Feature | Owner | Admin | Dev | Op | View |
|---------|-------|-------|-----|----|----|
| View details | ✅ | ✅ | ✅ | ✅ | ✅ |
| View versions | ✅ | ✅ | ✅ | ✅ | ✅ |
| View tags | ✅ | ✅ | ✅ | ✅ | ✅ |
| Edit metadata | ✅ | ✅ | ✅ | ✅ | ❌ |

### 8️⃣ Tenants Page (NEW)
| Feature | Owner | Admin | Dev | Op | View |
|---------|-------|-------|-----|----|----|
| List tenants | ✅ | ✅ | ❌ | ❌ | ❌ |
| Create tenant | ✅ | ✅ | ❌ | ❌ | ❌ |
| View tenant | ✅ | ✅ | ❌ | ❌ | ❌ |
| Edit tenant | ✅ | ✅ | ❌ | ❌ | ❌ |
| Delete tenant | ✅ | ✅ | ❌ | ❌ | ❌ |
| Set quotas | ✅ | ✅ | ❌ | ❌ | ❌ |

### 9️⃣ Settings Page
| Feature | Owner | Admin | Dev | Op | View |
|---------|-------|-------|-----|----|----|
| Edit email | ✅ | ✅ | ✅ | ✅ | ✅ |
| Edit theme | ✅ | ✅ | ✅ | ✅ | ✅ |
| Edit timezone | ✅ | ✅ | ✅ | ✅ | ✅ |
| Change password | ✅ | ✅ | ✅ | ✅ | ✅ |
| Enable 2FA | ✅ | ✅ | ✅ | ✅ | ✅ |
| View sessions | ✅ | ✅ | ✅ | ✅ | ✅ |

### 🔟 Profile Page
| Feature | Owner | Admin | Dev | Op | View |
|---------|-------|-------|-----|----|----|
| View profile | ✅ | ✅ | ✅ | ✅ | ✅ |
| Edit name | ✅ | ✅ | ✅ | ✅ | ✅ |
| Upload avatar | ✅ | ✅ | ✅ | ✅ | ✅ |
| View roles | ✅ | ✅ | ✅ | ✅ | ✅ |

---

## Permission Levels Explained

### 🔴 No Access (❌)
User cannot see the page, nav item, or feature
- Redirected if trying to access directly
- UI elements hidden or disabled

### 🟡 Read-Only (👁️)
User can view but not modify
- View project details
- View build logs
- Search/filter data

### 🟢 Full Access (✅)
User can perform all operations
- Create, read, update, delete
- Manage subresources (members, settings)
- Configure properties

---

## Summary for Testing

**For Owner/Admin Role Testing, Focus On**:

| Area | Test | Priority |
|------|------|----------|
| Navigation | See Tenants + Builds items | 🥇 Must |
| Project Members | Add/Edit/Remove members | 🥇 Must |
| Tenant Management | List/Delete tenants | 🥇 Must |
| Role-Based Nav | Verify nav changes with role | 🥇 Must |
| Build Mgmt | Create/cancel builds | 🥈 Should |
| Image Mgmt | Browse/search images | 🥈 Should |
| Personal Settings | Change theme/password | 🥉 Nice |
| Error Handling | Invalid actions show errors | 🥉 Nice |

