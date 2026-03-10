# Image Factory RBAC Quick Reference Matrix

## System Roles Overview

| Role | Scope | Primary Purpose | Users |
|------|-------|----------------|-------|
| **system-admin** | Global | Platform administration | IT Operations |
| **security-admin** | Global | Security policy & compliance | Security Team |
| **platform-operator** | Global | Day-to-day operations | Operations Team |
| **global-auditor** | Global (Read-only) | Compliance oversight | Auditors |
| **tenant-admin** | Tenant | Tenant administration | Team Leads |
| **tenant-operator** | Tenant | Tenant operations | DevOps Engineers |
| **project-manager** | Project | Project management | Project Leads |
| **build-engineer** | Project | Advanced builds | Senior Developers |
| **developer** | Project | Basic development | Developers |
| **security-reviewer** | Tenant | Security approval | Security Analysts |
| **viewer** | Project | Read-only access | Stakeholders |

## Key Permissions Matrix

### Core Operations
| Operation | System Admin | Security Admin | Tenant Admin | Build Engineer | Developer | Viewer |
|-----------|:------------:|:--------------:|:------------:|:--------------:|:---------:|:------:|
| Create Tenant | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ |
| Manage Users | ✅ | ❌ | ✅* | ❌ | ❌ | ❌ |
| Create Project | ✅ | ❌ | ✅ | ❌ | ❌ | ❌ |
| Submit Build | ✅ | ❌ | ✅ | ✅ | ✅ | ❌ |
| Approve Security | ✅ | ✅ | ✅ | ❌ | ❌ | ❌ |
| View Builds | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Delete Images | ✅ | ✅ | ✅ | ❌ | ❌ | ❌ |

*✅ = Full Access, ❌ = No Access, * = Scoped to tenant*

### Resource Access Levels

#### Builds & Images
| Resource | Create | Read | Update | Delete | Special |
|----------|:------:|:----:|:------:|:------:|:-------:|
| **Builds** | developer+ | project-member+ | build-engineer+ | project-manager+ | cancel: build-owner+ |
| **Images** | system | project-member+ | build-engineer+ | project-manager+ | sign: security-reviewer+ |
| **Artifacts** | system | project-member+ | build-engineer+ | project-manager+ | promote: build-engineer+ |

#### Administration
| Resource | Create | Read | Update | Delete | Special |
|----------|:------:|:----:|:------:|:------:|:-------:|
| **Tenants** | system-admin | all-auth | system-admin | system-admin | suspend: system-admin |
| **Users** | tenant-admin+ | tenant-admin+ | tenant-admin+ | tenant-admin+ | roles: tenant-admin+ |
| **Projects** | tenant-operator+ | project-member+ | project-manager+ | tenant-operator+ | archive: tenant-admin+ |

## Bootstrap Data Summary

### Required System Roles
- **system-admin**: Complete platform control
- **security-admin**: Security policy management  
- **platform-operator**: Operations and monitoring
- **global-auditor**: Cross-tenant compliance

### Required Tenant Roles
- **tenant-admin**: Tenant-level administration
- **tenant-operator**: Tenant operations management
- **build-engineer**: Advanced build capabilities
- **developer**: Basic development tasks
- **viewer**: Read-only access

### Initial Users
- **System Admin**: `admin@imagefactory.local` (system-admin)
- **Default Tenant Admin**: `admin@imagefactory.local` (tenant-admin for default tenant)

### Default Policies
- **Default Scan Policy**: Basic security scanning requirements
- **Production Promotion Policy**: Requirements for production releases

## Security Model

### Tenant Isolation
- **Data**: Row-Level Security (RLS) by tenant_id
- **Network**: Kubernetes namespaces + network policies
- **Compute**: Resource quotas and worker isolation
- **Storage**: Tenant-specific buckets/paths

### Permission Inheritance
```
System Roles > Tenant Roles > Project Roles > Resource Permissions
```

### Cross-Tenant Access
- **Allowed**: system-admin, security-admin, global-auditor
- **Prohibited**: All tenant-scoped users

## Implementation Notes

### Database Requirements
- Users table with tenant association
- Roles table with permission arrays
- User-role assignments with scope (system/tenant/project)
- Row-Level Security policies for tenant isolation

### API Security
- JWT tokens with embedded roles and permissions
- Middleware for permission checking
- Tenant context injection from request headers
- Resource-level authorization guards

### Bootstrap Sequence
1. Create system roles and permissions
2. Create initial system administrator
3. Create default security policies  
4. Create default tenant with tenant admin
5. Set up LDAP integration and sync users