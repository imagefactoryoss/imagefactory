# Build Management Guide

This guide explains how to create, run, and monitor builds in Image Factory.

## At A Glance

Build details and execution trace:

![Build details and execution trace](../assets/readme/if-tenant-build-details.png)

## Build Creation

1. Select build type.
2. Provide configuration (dockerfile, build context, registry).
3. Choose infrastructure provider.
4. Submit to queue.

---

## Build Status Lifecycle

`pending → queued → running → completed|failed|cancelled`

```mermaid
flowchart LR
    A[Draft Build] --> B[Pending]
    B --> C[Queued]
    C --> D[Dispatcher Claims Build]
    D --> E[Running]
    E --> F[Completed]
    E --> G[Failed]
    E --> H[Cancelled]
    F --> I[Artifacts And Logs Available]
    G --> J[Troubleshooting And Retry]
```

## Build Workflow

```mermaid
sequenceDiagram
    participant User
    participant Wizard as Build Wizard
    participant API as Backend API
    participant Dispatcher
    participant Runtime as Executor or Tekton

    User->>Wizard: Configure build
    Wizard->>API: Submit build
    API-->>User: Build accepted and queued
    Dispatcher->>API: Claim next queued build
    Dispatcher->>Runtime: Start execution
    Runtime-->>API: Report logs and status
    API-->>User: Show progress and results
```

---

## Where to Look for Issues

- **Validation errors**: build config fields in the wizard.
- **Queue stalls**: check dispatcher metrics.
- **Execution issues**: build execution logs.
- **Kubernetes execution readiness**: review infrastructure provider readiness and Tekton setup.
