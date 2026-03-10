# Image Factory Role-Based Access Control (RBAC) Model

## Document Information

| Attribute | Value |
|-----------|-------|
| **Document Title** | Image Factory - Role-Based Permissions Model |
| **Version** | 1.0 |
| **Date** | October 25, 2025 |
| **Author** | GitHub Copilot |
| **Status** | Draft |
| **Classification** | Internal |

---

## Table of Contents

1. [Executive Summary](#executive-summary)
2. [System Personas](#system-personas)
3. [Role Definitions](#role-definitions)
4. [Permission Matrix](#permission-matrix)
5. [Resource-Based Permissions](#resource-based-permissions)
6. [Multi-Tenant Security Model](#multi-tenant-security-model)
7. [Bootstrap Data Requirements](#bootstrap-data-requirements)
8. [Implementation Guidelines](#implementation-guidelines)

---

## Executive Summary

The Image Factory system requires a comprehensive Role-Based Access Control (RBAC) model to ensure secure, scalable, and compliant multi-tenant operations. This document defines:

- **5 Primary System Roles**: From System Admin to Read-Only User
- **3 Tenant-Scoped Roles**: Providing granular tenant-level control
- **Resource-Based Permissions**: Covering all system components and operations
- **Multi-Tenant Security**: Ensuring complete tenant isolation and data security

The RBAC model supports both system-wide administration and tenant-specific governance, enabling the platform to scale across enterprise environments while maintaining security and compliance requirements.

---

## System Personas

### Primary Personas

| Persona | Description | Primary Goals | System Scope |
|---------|-------------|---------------|--------------|
| **Platform Administrator** | IT operations team managing the entire Image Factory platform | System health, tenant management, compliance, cost optimization | System-wide |
| **Security Administrator** | Security team responsible for policies, scanning, and governance | Security policy enforcement, vulnerability management, compliance | System-wide |
| **Tenant Administrator** | Lead technical person within an organization/department | Manage tenant resources, users, and build policies | Tenant-scoped |
| **DevOps Engineer** | Developer/operations person building and managing images | Create builds, manage pipelines, deploy images | Tenant-scoped |
| **Developer** | Software developer creating applications and images | Submit build requests, view build status, access artifacts | Tenant-scoped |
| **Security Analyst** | Person reviewing security scans and approving releases | Review scan results, approve/reject builds, manage quarantine | Tenant-scoped |
| **Auditor** | Compliance person reviewing system operations | View audit logs, generate compliance reports, monitor activity | Read-only across tenants |

### Secondary Personas

| Persona | Description | Primary Goals | System Scope |
|---------|-------------|---------------|--------------|
| **Guest User** | External contractor or temporary team member | Limited access to specific projects or builds | Project-scoped |
| **Service Account** | Automated systems and CI/CD pipelines | Programmatic access for integrations | API-scoped |

---

## Role Definitions

### System-Level Roles

#### 1. System Administrator (`system-admin`)
**Purpose**: Complete system administration and platform management
**Scope**: Global across all tenants
**Key Responsibilities**:
- Platform installation, configuration, and maintenance
- Tenant lifecycle management (create, suspend, delete)
- System-wide resource management and quota allocation
- Platform monitoring, logging, and troubleshooting
- User role assignments and RBAC policy management
- System-wide compliance and audit management

#### 2. Security Administrator (`security-admin`) 
**Purpose**: System-wide security policy and compliance management
**Scope**: Global security oversight
**Key Responsibilities**:
- Define and enforce security scanning policies
- Manage image quarantine and approval workflows
- Configure compliance frameworks and audit requirements
- Oversee vulnerability management and remediation
- System-wide security monitoring and incident response

#### 3. Platform Operator (`platform-operator`)
**Purpose**: Day-to-day platform operations and monitoring
**Scope**: System-wide operational tasks
**Key Responsibilities**:
- Monitor system health and performance
- Manage build worker pools and resource allocation
- Handle operational issues and basic troubleshooting
- View system-wide metrics and logs
- Manage system notifications and alerts

#### 4. Global Auditor (`global-auditor`)
**Purpose**: Cross-tenant compliance and audit oversight
**Scope**: Read-only access across all tenants
**Key Responsibilities**:
- Generate compliance reports across all tenants
- Review audit logs and security events
- Monitor policy compliance and violations
- Assess system-wide risk and security posture

#### 5. Read-Only User (`global-viewer`)
**Purpose**: System-wide monitoring and observability
**Scope**: Read-only system access
**Key Responsibilities**:
- View system status and health metrics
- Access public documentation and system information
- Monitor system-wide build statistics and trends

### Tenant-Level Roles

#### 1. Tenant Administrator (`tenant-admin`)
**Purpose**: Complete administration within tenant scope
**Scope**: Full control within assigned tenant(s)
**Key Responsibilities**:
- Manage tenant users and role assignments
- Configure tenant-specific settings and policies
- Manage tenant resource quotas and billing
- Oversee tenant-wide compliance and governance
- Approve high-privilege operations within tenant

#### 2. Tenant Operator (`tenant-operator`)
**Purpose**: Operational management within tenant
**Scope**: Operational tasks within tenant
**Key Responsibilities**:
- Manage build projects and configurations
- Monitor tenant resource usage and performance
- Handle tenant-specific operational issues
- Manage image repositories and artifacts
- Configure build worker pools and environments

#### 3. Project Manager (`project-manager`)
**Purpose**: Project-level management and coordination
**Scope**: Specific projects within tenant
**Key Responsibilities**:
- Manage project lifecycle and team access
- Configure project-specific build policies
- Oversee project compliance and quality gates
- Manage project resource allocation
- Coordinate with security and operations teams

### User-Level Roles

#### 1. Build Engineer (`build-engineer`)
**Purpose**: Advanced build creation and management
**Scope**: Build operations within assigned projects
**Key Responsibilities**:
- Create and configure complex build manifests
- Manage build pipelines and automation
- Debug build failures and optimize performance
- Configure security scanning and compliance checks
- Manage image tagging and promotion workflows

#### 2. Developer (`developer`)
**Purpose**: Standard development and build tasks
**Scope**: Basic build operations and artifact access
**Key Responsibilities**:
- Submit build requests from source code
- Monitor build progress and view logs
- Access build artifacts and images
- Run basic security scans and view results
- Manage personal API keys and access tokens

#### 3. Security Reviewer (`security-reviewer`)
**Purpose**: Security-focused review and approval
**Scope**: Security assessment within assigned scope
**Key Responsibilities**:
- Review security scan results and vulnerabilities
- Approve or reject builds based on security policies
- Manage image quarantine and remediation
- Configure security policies and compliance rules
- Generate security and compliance reports

#### 4. Viewer (`viewer`)
**Purpose**: Read-only access to assigned resources
**Scope**: Read-only access within granted scope
**Key Responsibilities**:
- View build status and history
- Access build logs and artifacts (if permitted)
- View security scan results
- Monitor project and tenant metrics
- Generate read-only reports

---

## Permission Matrix

### System Operations

| Operation | system-admin | security-admin | platform-operator | global-auditor | global-viewer |
|-----------|--------------|----------------|-------------------|----------------|---------------|
| **Platform Management** |||||
| Install/Update Platform | ✅ | ❌ | ❌ | ❌ | ❌ |
| Configure System Settings | ✅ | ❌ | ❌ | ❌ | ❌ |
| Manage System Resources | ✅ | ❌ | 🔶* | ❌ | ❌ |
| **Tenant Management** |||||
| Create Tenant | ✅ | ❌ | ❌ | ❌ | ❌ |
| Suspend/Delete Tenant | ✅ | ❌ | ❌ | ❌ | ❌ |
| View All Tenants | ✅ | ✅ | ✅ | ✅ | ✅ |
| Modify Tenant Quotas | ✅ | ❌ | ❌ | ❌ | ❌ |
| **Security & Compliance** |||||
| Define Security Policies | ✅ | ✅ | ❌ | ❌ | ❌ |
| Manage Global Quarantine | ✅ | ✅ | ❌ | ❌ | ❌ |
| View Audit Logs | ✅ | ✅ | ✅ | ✅ | ✅ |
| Generate Compliance Reports | ✅ | ✅ | ❌ | ✅ | ❌ |
| **Monitoring & Operations** |||||
| View System Metrics | ✅ | ✅ | ✅ | ✅ | ✅ |
| Manage System Alerts | ✅ | ❌ | ✅ | ❌ | ❌ |
| Access System Logs | ✅ | ✅ | ✅ | ✅ | 🔶* |

*🔶 = Limited/Read-only access*

### Tenant Operations

| Operation | tenant-admin | tenant-operator | project-manager | build-engineer | developer | security-reviewer | viewer |
|-----------|--------------|-----------------|-----------------|----------------|-----------|-------------------|--------|
| **User Management** |||||||
| Manage Tenant Users | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ |
| Assign User Roles | ✅ | 🔶* | 🔶* | ❌ | ❌ | ❌ | ❌ |
| View User List | ✅ | ✅ | ✅ | ❌ | ❌ | ✅ | ❌ |
| **Project Management** |||||||
| Create Projects | ✅ | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ |
| Delete Projects | ✅ | ✅ | ✅ | ❌ | ❌ | ❌ | ❌ |
| Configure Project Settings | ✅ | ✅ | ✅ | ❌ | ❌ | ❌ | ❌ |
| View Projects | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| **Build Management** |||||||
| Create Build Configurations | ✅ | ✅ | ✅ | ✅ | ❌ | ❌ | ❌ |
| Submit Build Requests | ✅ | ✅ | ✅ | ✅ | ✅ | ❌ | ❌ |
| Cancel Builds | ✅ | ✅ | ✅ | ✅ | 🔶* | ❌ | ❌ |
| View Build Logs | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| **Image Management** |||||||
| Promote Images | ✅ | ✅ | ✅ | ✅ | ❌ | ❌ | ❌ |
| Delete Images | ✅ | ✅ | ✅ | ❌ | ❌ | ❌ | ❌ |
| Tag Images | ✅ | ✅ | ✅ | ✅ | ✅ | ❌ | ❌ |
| Download Images | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | 🔶* |
| **Security Operations** |||||||
| Configure Security Policies | ✅ | ✅ | ✅ | ❌ | ❌ | ✅ | ❌ |
| Approve Security Scans | ✅ | ❌ | ❌ | ❌ | ❌ | ✅ | ❌ |
| View Security Results | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Quarantine Management | ✅ | ✅ | ❌ | ❌ | ❌ | ✅ | ❌ |

*🔶 = Limited access (project-scoped or own resources only)*

### API and Integration Access

| Operation | system-admin | security-admin | tenant-admin | build-engineer | developer | service-account |
|-----------|--------------|----------------|--------------|----------------|-----------|----------------|
| **API Access** ||||||
| System Admin APIs | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ |
| Tenant Management APIs | ✅ | ❌ | ✅ | ❌ | ❌ | 🔶* |
| Build APIs | ✅ | ❌ | ✅ | ✅ | ✅ | ✅ |
| Image Registry APIs | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Security APIs | ✅ | ✅ | ✅ | ❌ | ❌ | ✅ |
| **Integration Access** ||||||
| Webhook Management | ✅ | ❌ | ✅ | ✅ | ❌ | ✅ |
| CI/CD Integration | ✅ | ❌ | ✅ | ✅ | ✅ | ✅ |
| Monitoring Integration | ✅ | ✅ | ✅ | ❌ | ❌ | ✅ |

*🔶 = Limited to assigned tenants/projects*

---

## Resource-Based Permissions

### Tenant Resources

| Resource Type | Create | Read | Update | Delete | Special Permissions |
|---------------|--------|------|--------|--------|-------------------|
| **Tenants** | system-admin | all-authenticated | system-admin | system-admin | suspend (system-admin) |
| **Users** | tenant-admin+ | tenant-admin+ | tenant-admin+ | tenant-admin+ | role-assignment (tenant-admin+) |
| **Projects** | tenant-operator+ | project-member+ | project-manager+ | tenant-operator+ | archive (tenant-admin+) |
| **Build Configs** | build-engineer+ | project-member+ | build-engineer+ | project-manager+ | template-creation (tenant-operator+) |

### Build Resources

| Resource Type | Create | Read | Update | Delete | Special Permissions |
|---------------|--------|------|--------|--------|-------------------|
| **Builds** | developer+ | project-member+ | build-engineer+ | project-manager+ | cancel (build-owner, build-engineer+) |
| **Build Logs** | system | project-member+ | system | system | export (build-engineer+) |
| **Artifacts** | system | project-member+ | build-engineer+ | project-manager+ | promote (build-engineer+) |
| **Images** | system | project-member+ | build-engineer+ | project-manager+ | sign (security-reviewer+) |

### Security Resources

| Resource Type | Create | Read | Update | Delete | Special Permissions |
|---------------|--------|------|--------|--------|-------------------|
| **Policies** | security-admin+ | all-authenticated | security-admin+ | security-admin+ | enforce (security-admin+) |
| **Scan Results** | system | project-member+ | security-reviewer+ | security-admin+ | approve (security-reviewer+) |
| **Quarantine** | system | security-reviewer+ | security-reviewer+ | security-admin+ | release (security-reviewer+) |
| **Compliance Reports** | system | auditor+ | security-admin+ | security-admin+ | certify (security-admin+) |

---

## Multi-Tenant Security Model

### Tenant Isolation

#### 1. Data Isolation
- **Database**: Row-Level Security (RLS) based on tenant_id
- **Storage**: Tenant-specific buckets/namespaces
- **Logs**: Tenant-scoped log streams
- **Metrics**: Tenant-labeled metrics with access controls

#### 2. Network Isolation
- **Kubernetes Namespaces**: One namespace per tenant
- **Network Policies**: Traffic isolation between tenants
- **Service Mesh**: mTLS with tenant-based certificates
- **Ingress**: Tenant-specific subdomain routing

#### 3. Compute Isolation
- **Resource Quotas**: CPU, memory, and storage limits per tenant
- **Build Workers**: Isolated worker pools or resource constraints
- **Secrets Management**: Tenant-scoped secret stores
- **Registry Access**: Tenant-specific registry permissions

### Cross-Tenant Operations

#### Allowed Cross-Tenant Access
- **System Administrators**: Full cross-tenant access for platform management
- **Global Auditors**: Read-only access for compliance and audit
- **Security Administrators**: Security policy enforcement across tenants

#### Prohibited Cross-Tenant Access
- **Tenant Users**: Cannot access other tenants' resources
- **Data Sharing**: No direct data sharing between tenants
- **Build Workers**: Cannot access cross-tenant artifacts

---

## Bootstrap Data Requirements

### System Initialization

#### 1. Default System Roles
```json
{
  "system_roles": [
    {
      "name": "system-admin",
      "description": "Complete system administration",
      "permissions": ["*"],
      "is_system_role": true
    },
    {
      "name": "security-admin", 
      "description": "System-wide security management",
      "permissions": ["security.*", "audit.*", "compliance.*"],
      "is_system_role": true
    },
    {
      "name": "platform-operator",
      "description": "Platform operations and monitoring",
      "permissions": ["monitor.*", "operate.*"],
      "is_system_role": true
    },
    {
      "name": "global-auditor",
      "description": "Cross-tenant audit and compliance",
      "permissions": ["audit.read", "compliance.read"],
      "is_system_role": true
    }
  ]
}
```

#### 2. Default Tenant Roles
```json
{
  "tenant_roles": [
    {
      "name": "tenant-admin",
      "description": "Tenant administration",
      "permissions": ["tenant.*"],
      "is_tenant_scoped": true
    },
    {
      "name": "tenant-operator",
      "description": "Tenant operations",
      "permissions": ["project.*", "build.*", "image.read"],
      "is_tenant_scoped": true
    },
    {
      "name": "project-manager",
      "description": "Project management",
      "permissions": ["project.manage", "build.read", "image.read"],
      "is_project_scoped": true
    },
    {
      "name": "build-engineer",
      "description": "Build creation and management",
      "permissions": ["build.*", "image.read", "artifact.*"],
      "is_project_scoped": true
    },
    {
      "name": "developer",
      "description": "Basic development tasks",
      "permissions": ["build.create", "build.read", "image.read"],
      "is_project_scoped": true
    },
    {
      "name": "security-reviewer",
      "description": "Security review and approval",
      "permissions": ["security.*", "quarantine.*", "approval.*"],
      "is_tenant_scoped": true
    },
    {
      "name": "viewer",
      "description": "Read-only access",
      "permissions": ["*.read"],
      "is_project_scoped": true
    }
  ]
}
```

#### 3. Initial System Administrator
```json
{
  "bootstrap_admin": {
    "username": "admin",
    "email": "admin@imagefactory.local", 
    "role": "system-admin",
    "tenant_id": null,
    "must_change_password": true,
    "created_by": "system-bootstrap"
  }
}
```

#### 4. Default Security Policies
```json
{
  "security_policies": [
    {
      "name": "default-scan-policy",
      "description": "Default security scanning requirements",
      "rules": {
        "vulnerability_scanning": true,
        "sbom_generation": true,
        "max_critical_vulnerabilities": 0,
        "max_high_vulnerabilities": 5,
        "quarantine_on_policy_violation": true
      },
      "applies_to": "all-images",
      "is_default": true
    },
    {
      "name": "production-promotion-policy", 
      "description": "Requirements for production image promotion",
      "rules": {
        "requires_security_approval": true,
        "requires_qa_approval": false,
        "min_scan_age_hours": 24,
        "requires_signed_image": true
      },
      "applies_to": "production-images",
      "is_default": true
    }
  ]
}
```

#### 5. Default Tenant Template
```json
{
  "default_tenant": {
    "name": "Default Organization",
    "slug": "default-org",
    "resource_quota": {
      "max_builds": 100,
      "max_images": 500,
      "max_storage_gb": 100.0,
      "max_concurrent_jobs": 10
    },
    "config": {
      "build_timeout": "2h",
      "allowed_image_types": ["container", "vm", "ami"],
      "security_policies": ["default-scan-policy"],
      "notification_settings": {
        "email_enabled": true,
        "webhook_enabled": false
      }
    },
    "users": [
      {
        "username": "tenant-admin",
        "email": "admin@imagefactory.local",
        "role": "tenant-admin"
      }
    ]
  }
}
```

---

## Implementation Guidelines

### Database Schema

#### 1. Users and Roles Tables
```sql
-- Users table
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(255) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255),
    tenant_id UUID REFERENCES tenants(id),
    is_system_user BOOLEAN DEFAULT FALSE,
    status VARCHAR(50) DEFAULT 'active',
    last_login TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Roles table
CREATE TABLE roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) UNIQUE NOT NULL,
    description TEXT,
    permissions JSONB NOT NULL DEFAULT '[]',
    is_system_role BOOLEAN DEFAULT FALSE,
    is_tenant_scoped BOOLEAN DEFAULT TRUE,
    is_project_scoped BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- User role assignments
CREATE TABLE user_roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    role_id UUID REFERENCES roles(id) ON DELETE CASCADE,
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    project_id UUID REFERENCES projects(id) ON DELETE CASCADE,
    granted_by UUID REFERENCES users(id),
    granted_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP,
    UNIQUE(user_id, role_id, tenant_id, project_id)
);
```

#### 2. Permissions and Policies
```sql
-- Permissions registry
CREATE TABLE permissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) UNIQUE NOT NULL,
    description TEXT,
    resource_type VARCHAR(100) NOT NULL,
    action VARCHAR(100) NOT NULL,
    scope VARCHAR(50) DEFAULT 'tenant', -- 'system', 'tenant', 'project'
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Security policies
CREATE TABLE security_policies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    rules JSONB NOT NULL DEFAULT '{}',
    applies_to VARCHAR(100) DEFAULT 'all-images',
    tenant_id UUID REFERENCES tenants(id),
    is_default BOOLEAN DEFAULT FALSE,
    is_active BOOLEAN DEFAULT TRUE,
    created_by UUID REFERENCES users(id),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### API Security Implementation

#### 1. JWT Token Structure
```json
{
  "user_id": "uuid",
  "username": "string",
  "email": "string", 
  "tenant_id": "uuid",
  "roles": [
    {
      "role": "tenant-admin",
      "scope": "tenant",
      "tenant_id": "uuid"
    },
    {
      "role": "developer", 
      "scope": "project",
      "project_id": "uuid"
    }
  ],
  "permissions": ["build.create", "image.read"],
  "iat": 1635724800,
  "exp": 1635811200
}
```

#### 2. Authorization Middleware
```go
func AuthorizePermission(permission string) gin.HandlerFunc {
    return func(c *gin.Context) {
        user := getUserFromContext(c)
        tenantID := getTenantFromRequest(c)
        
        if !user.HasPermission(permission, tenantID) {
            c.JSON(403, gin.H{"error": "Insufficient permissions"})
            c.Abort()
            return
        }
        
        c.Next()
    }
}
```

### Row-Level Security (RLS)

#### 1. Tenant Data Isolation
```sql
-- Enable RLS on tenant-scoped tables
ALTER TABLE builds ENABLE ROW LEVEL SECURITY;
ALTER TABLE images ENABLE ROW LEVEL SECURITY;
ALTER TABLE projects ENABLE ROW LEVEL SECURITY;

-- Tenant isolation policy
CREATE POLICY tenant_isolation ON builds
    FOR ALL
    TO application_role
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID);

-- System admin bypass policy  
CREATE POLICY system_admin_access ON builds
    FOR ALL 
    TO system_admin_role
    USING (true);
```

#### 2. Project-Level Access Control
```sql
-- Project member access policy
CREATE POLICY project_member_access ON builds
    FOR ALL
    TO application_role  
    USING (
        project_id IN (
            SELECT project_id 
            FROM user_project_access 
            WHERE user_id = current_setting('app.current_user_id')::UUID
        )
    );
```

This comprehensive RBAC model provides the foundation for secure, scalable multi-tenant operations in the Image Factory system. The model balances security, usability, and operational efficiency while ensuring complete tenant isolation and proper governance controls.