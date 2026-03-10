# Unified Messaging Event Catalog

This catalog defines the canonical event names, payload expectations, and versioning approach for the unified messaging bus.

## Overview

This catalog defines the canonical event names and payload expectations for the unified messaging bus. It is intended to:

- Prevent breaking changes across consumers.
- Standardize naming and payload structure.
- Enable validation and schema evolution.

**Envelope fields** (required for all events):
- `id` (string)
- `type` (string)
- `occurred_at` (RFC3339)
- `schema_version` (string)
- `tenant_id` (string, optional)
- `actor_id` (string, optional)
- `source` (string, optional)
- `correlation_id` (string, optional)
- `request_id` (string, optional)
- `trace_id` (string, optional)

---

## Build Events

### `build.created` (v1)
**Producer:** Build Service  
**Consumers:** Audit, Notifications, Analytics  

Payload:
- `build_id` (string, UUID)
- `build_name` (string)
- `build_type` (string)

### `build.started` (v1)
**Producer:** Build Service  
**Consumers:** Audit, Notifications  

Payload:
- `build_id` (string, UUID)

### `build.completed` (v1)
**Producer:** Build Service  
**Consumers:** Audit, Notifications  

Payload:
- `build_id` (string, UUID)
- `image_id` (string)
- `image_size` (number)
- `duration` (number, seconds)

### `build.failed` (v1)
**Producer:** Build Execution Service  
**Consumers:** Audit, Notifications  

Payload:
- `build_id` (string, UUID)
- `status` (string)
- `message` (string)

### `build.execution.completed` (v1)
**Producer:** Build Execution Service  
**Consumers:** WebSocket  

Payload:
- `build_id` (string, UUID)
- `status` (string)
- `message` (string)
- `duration` (number, seconds)
- `metadata` (object, optional)

### `build.execution.failed` (v1)
**Producer:** Build Execution Service  
**Consumers:** WebSocket  

Payload:
- `build_id` (string, UUID)
- `status` (string)
- `message` (string)
- `duration` (number, seconds)
- `metadata` (object, optional)

### `build.execution.status.updated` (v1)
**Producer:** Build Execution Service  
**Consumers:** WebSocket  

Payload:
- `build_id` (string, UUID)
- `status` (string)
- `message` (string)
- `duration` (number, seconds)
- `metadata` (object, optional)

---

## Tenant Events

### `tenant.created` (v1)
**Producer:** Tenant Service  
**Consumers:** Audit, Notifications  

Payload:
- `tenant_id` (string, UUID)
- `tenant_name` (string)

### `tenant.activated` (v1)
**Producer:** Tenant Service  
**Consumers:** Audit, Notifications  

Payload:
- `tenant_id` (string, UUID)

---

## Infrastructure Provider Events

### `infra.provider.created` (v1)
**Producer:** Infrastructure Service  
**Consumers:** Audit  

Payload:
- `provider_id` (string, UUID)
- `provider_type` (string)
- `name` (string)
- `created_by` (string, UUID)

### `infra.provider.updated` (v1)
**Producer:** Infrastructure Service  
**Consumers:** Audit  

Payload:
- `provider_id` (string, UUID)
- `updated_by` (string, UUID)

### `infra.provider.deleted` (v1)
**Producer:** Infrastructure Service  
**Consumers:** Audit  

Payload:
- `provider_id` (string, UUID)
- `deleted_by` (string, UUID)

---

## Project Events

### `project.created` (v1)
**Producer:** Project Service  
**Consumers:** Audit  

Payload:
- `project_id` (string, UUID)
- `project_name` (string)
- `visibility` (string)
- `git_repo` (string, optional)
- `git_branch` (string, optional)
- `git_provider` (string, optional)

### `project.updated` (v1)
**Producer:** Project Service  
**Consumers:** Audit  

Payload:
- `project_id` (string, UUID)
- `project_name` (string)
- `visibility` (string)
- `git_repo` (string, optional)
- `git_branch` (string, optional)
- `git_provider` (string, optional)

### `project.deleted` (v1)
**Producer:** Project Service  
**Consumers:** Audit  

Payload:
- `project_id` (string, UUID)

---

## Notification Events (Planned)

### `notification.requested` (v1)
**Producer:** API / Domain Services  
**Consumers:** Notification Worker  

Payload:
- `notification_type` (string)
- `channel` (string: `email`, `webhook`)
- `tenant_id` (string, UUID)
- `to` (string)
- `cc` (string, optional)
- `from` (string)
- `subject` (string)
- `body_text` (string)
- `body_html` (string, optional)
- `email_type` (string)
- `priority` (number)
- `metadata` (object, optional)

### `notification.sent` (v1)
**Producer:** Notification Worker  
**Consumers:** Audit, Analytics  

Payload:
- `notification_type` (string)
- `channel` (string)
- `tenant_id` (string, UUID)
- `recipient` (string)
- `duration_ms` (number)

### `notification.failed` (v1)
**Producer:** Notification Worker  
**Consumers:** Audit, Analytics  

Payload:
- `notification_type` (string)
- `channel` (string)
- `tenant_id` (string, UUID)
- `recipient` (string)
- `error` (string)
- `retryable` (boolean)

---

## Versioning Rules

- **Additive changes** (new fields) are allowed without bumping version.
- **Breaking changes** require a new version and parallel consumption support.
- Consumers should tolerate unknown fields.

---

## Next Steps

- Implement notification event producers.
- Build notification worker with retry/backoff.
