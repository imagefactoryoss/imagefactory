# Role Consolidation Architecture

This document describes the current role model and how access is resolved in Image Factory.

## Core Model

- **System-level roles** are the source of truth.
- **Tenant groups** map users to roles within a tenant context.
- **Permissions** are attached to roles and evaluated at request time.

---

## Role Resolution Flow

1. User logs in.
2. Tenant context is selected.
3. System role is resolved via tenant group membership.
4. Permissions are evaluated for the requested resource/action.

---

## Data Tables

- `rbac_roles`
- `tenant_groups`
- `group_members`
- `permissions`
- `role_permissions`
- `user_role_assignments` (not used in current flow)
