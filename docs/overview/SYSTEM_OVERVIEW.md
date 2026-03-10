# System Overview

This document summarizes the major runtime components, data flow, and integration surface of Image Factory at a system level.

## Core Services

- **Backend API**: Multi-tenant core services, RBAC, build orchestration.
- **Frontend UI**: Tenant and admin experience for projects, builds, and configuration.
- **Dispatcher**: Status-based queue processor that dispatches builds to executors.
- **Executors**: Local executors and Tekton-based execution for Kubernetes.

## Operational Capabilities

- Multi-tenant project and build management
- Kubernetes and Tekton-backed execution through infrastructure providers
- Quarantine request and release workflows
- On-demand image scanning and asynchronous result tracking
- Notifications, audit-oriented workflows, and execution visibility

## UI Snapshot

Tenant dashboard:

![Image Factory tenant dashboard](../assets/readme/if-tenant-dashboard.png)

---

## Runtime Architecture

```mermaid
flowchart LR
    U[Users] --> F[Frontend UI]
    F --> B[Backend API]
    B --> P[(PostgreSQL)]
    B --> R[(Redis)]
    B --> D[Dispatcher]
    D --> E[Executors]
    E --> T[Tekton And Kubernetes]
    B --> N[Notifications And Audit Flows]
```

---

## Core Data Flow

1. A user creates a build.
2. Build is stored with `status = queued`.
3. Dispatcher claims queued builds and starts execution.
4. Execution emits logs, status updates, and artifacts.

```mermaid
sequenceDiagram
    participant User
    participant UI as Frontend UI
    participant API as Backend API
    participant DB as PostgreSQL
    participant Dispatcher
    participant Executor as Executor or Tekton

    User->>UI: Create build
    UI->>API: Submit build request
    API->>DB: Store build with queued status
    Dispatcher->>DB: Claim queued build
    Dispatcher->>Executor: Start execution
    Executor-->>API: Send status, logs, artifacts
    API->>DB: Persist execution updates
    API-->>UI: Return progress and results
```

---

## Key Data Stores

- **PostgreSQL**: Source of truth for builds, configs, executions, and metadata.
- **Redis**: Caching/session (if enabled).

---

## Integration Surface

- REST APIs for admin and tenant workflows.
- Eventing hooks for notifications and audit trails.
- Infrastructure-provider connectivity and readiness checks for Kubernetes execution.

---

## Security & Access

- Tenant isolation with RBAC.
- Admin vs tenant-level access boundaries.
- Capability-gated workflows for build, quarantine, release, and scanning surfaces.
