# Image Factory

Image Factory is a multi-tenant platform for building and distributing container and VM images with security, compliance, and automation built in. It standardizes build workflows, centralizes policy enforcement, and makes image delivery repeatable across teams.

## What It Does

- Manages image build workflows across tenants and projects
- Supports multiple build methods for container and VM image creation
- Routes execution through local or Kubernetes-backed executors
- Prepares Kubernetes infrastructure providers for Tekton-backed execution and readiness validation
- Supports quarantine request and release workflows for controlled image intake
- Supports on-demand image scanning for asynchronous SBOM and vulnerability analysis
- Tracks build status, logs, artifacts, and execution metadata
- Applies tenant-aware access control and admin boundaries
- Integrates notification and audit-oriented operational workflows

## Who It’s For

- Platform teams standardizing image creation
- Security and compliance teams enforcing scanning and SBOM requirements
- Engineering teams needing reliable, repeatable build pipelines

## System Overview

Core runtime components:

- `backend/`: multi-tenant APIs, RBAC, and build orchestration
- `frontend/`: tenant and admin UI for projects, builds, and configuration
- `dispatcher`: status-based queue processor that claims and dispatches builds
- `executors`: local execution paths and Tekton-based Kubernetes execution

Core flow:

1. A user creates a build.
2. The build is stored with `status = queued`.
3. The dispatcher claims queued builds and starts execution.
4. Execution emits logs, status updates, and artifacts.

Primary data stores:

- PostgreSQL for builds, configs, executions, and metadata
- Redis for cache and session-oriented workloads when enabled

## Kubernetes And Tekton

Image Factory can execute builds on Kubernetes via Tekton using infrastructure-provider selection. This lets operators configure provider connectivity, validate readiness, and route builds to Kubernetes-backed execution without hardwiring a single cluster into the application.

See:

- [Kubernetes Tekton Integration](docs/kubernetes-integration/KUBERNETES_TEKTON_INTEGRATION.md)
- [Tekton Readiness Troubleshooting](docs/kubernetes-integration/TEKTON_READINESS_TROUBLESHOOTING.md)
- [Kubernetes Cluster Connectivity Guide](docs/kubernetes-integration/KUBERNETES_CLUSTER_CONNECTIVITY_GUIDE.md)

## Screenshots

### Tenant Dashboard

![Image Factory tenant dashboard](docs/assets/readme/if-tenant-dashboard.png)

### Highlighted Workflows

Tekton provider preparation and readiness:

![Tekton infrastructure provider preparation](docs/assets/readme/if-tekton-provider.png)

Build details and execution trace:

![Build details and execution trace](docs/assets/readme/if-tenant-build-details.png)

### Security Workflows

Quarantine requests and release-oriented intake:

![Quarantine requests workflow](docs/assets/readme/if-quarantine.png)

On-demand image scanning:

![On-demand image scanning workflow](docs/assets/readme/if-ondemand-scan.png)

## Documentation

Start with the public docs set in [docs/README.md](docs/README.md). For a broader map of the published material, see [docs/DOCUMENTATION_INDEX.md](docs/DOCUMENTATION_INDEX.md).

## Repository Layout

- `backend/` Go services, migrations, workers, and Tekton assets
- `frontend/` Vite/React application
- `deploy/` Helm chart and deployment assets
- `docs/` curated public documentation
- `scripts/` local development and maintenance helpers

## Quick Start

1. Copy `.env.example` to a local env file such as `.env.development`.
2. Start the required local dependencies.
3. Run the backend and frontend from their respective directories.

See [docs/getting-started/LOCAL_DEV_SETUP.md](docs/getting-started/LOCAL_DEV_SETUP.md) for the current setup flow.

## License

This repository is licensed under Apache 2.0. See [LICENSE](LICENSE).
