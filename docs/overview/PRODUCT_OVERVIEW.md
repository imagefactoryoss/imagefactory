# Image Factory Product Overview

This document provides a concise overview of what Image Factory is, who it is for, and the core product capabilities exposed in the published repository.

## What It Is

Image Factory is a multi-tenant system for building and distributing container and VM images with security, compliance, and automation built in. It standardizes build workflows, centralizes policies, and makes image delivery repeatable across teams.

## Product Capability Map

```mermaid
flowchart TD
    A[Image Factory] --> B[Projects And Tenants]
    A --> C[Build Execution]
    A --> D[Kubernetes And Tekton]
    A --> E[Security Workflows]
    A --> F[Audit And Notifications]
    C --> G[Artifacts And Logs]
    D --> H[Provider Preparation]
    E --> I[Quarantine]
    E --> J[On Demand Scanning]
```

---

## Who It’s For

- Platform teams standardizing image creation.
- Security and compliance teams enforcing scanning and SBOM requirements.
- Engineering teams needing reliable, repeatable build pipelines.

---

## Core Capabilities

- Multi-tenant projects with role-based access control.
- Build orchestration with a dispatcher (status-based queue).
- Build methods for containers and VMs.
- Build metadata, artifacts, and execution tracking.
- Infrastructure providers for execution selection (local or Kubernetes).
- Tekton-backed provider preparation and readiness validation for Kubernetes execution.
- Quarantine request and release workflows for controlled image intake.
- On-demand image scanning for asynchronous SBOM and vulnerability analysis.
- Notifications and audit trail for user actions.

## Typical Product Flow

```mermaid
sequenceDiagram
    participant Team
    participant Platform as Image Factory
    participant Runtime as Executor or Tekton

    Team->>Platform: Configure project and build
    Team->>Platform: Submit build
    Platform->>Runtime: Start execution
    Runtime-->>Platform: Return logs, status, artifacts
    Platform-->>Team: Show results and follow-up actions
```

## Product Snapshots

Tenant dashboard:

![Image Factory tenant dashboard](../assets/readme/if-tenant-dashboard.png)

Tekton provider preparation:

![Tekton infrastructure provider preparation](../assets/readme/if-tekton-provider.png)

Build execution details:

![Build details and execution trace](../assets/readme/if-tenant-build-details.png)

Security workflows:

![Quarantine requests workflow](../assets/readme/if-quarantine.png)

![On-demand image scanning workflow](../assets/readme/if-ondemand-scan.png)

---

## Current Scope Boundaries

- Multi-stage release workflows across environments.
- Advanced capacity scheduling and predictive ETA.
- Full enterprise SSO and MFA hardening (beyond current integrations).

---

## What Success Looks Like

- Tenants can create projects, configure builds, and run executions end-to-end.
- Dispatcher reliably moves builds from `queued` to `running` with metrics visibility.
- Basic audit and notification workflows function as expected.
