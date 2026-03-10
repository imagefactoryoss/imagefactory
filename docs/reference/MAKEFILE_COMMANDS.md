# Makefile Commands Reference

This guide documents the repository `Makefile` commands used for local development, image builds, and container runtime workflows.

## Container Engine Support

The Makefile supports both Docker and Podman.

- `CONTAINER_ENGINE` (default: `podman`)
- `COMPOSE_CMD` (default: `podman compose`)
- `FRONTEND_USE_LOCAL_DIST` (default: `false`; when `true`, builds local `frontend/dist` and copies it into nginx image)

Examples:

```bash
# Podman (default)
make build-all-images

# Docker override
make build-all-images CONTAINER_ENGINE=docker COMPOSE_CMD="docker-compose"

# Use local frontend dist in image builds
make build-all-images FRONTEND_USE_LOCAL_DIST=true
```

## Most Common Workflows

### 1) Local development

```bash
make dev
make dev-logs
make dev-stop
```

### 2) Build runtime release tarballs

```bash
make release-binaries IMAGE_VERSION=v0.1.0
```

This creates `.tar.gz` artifacts and `checksums.txt` under `release/dist/` for:
- `image-factory-server`
- `image-factory-dispatcher`
- `image-factory-notification-worker`
- `image-factory-email-worker`
- `image-factory-internal-registry-gc-worker`
- `image-factory-docs-server`
- `image-factory-migrate`
- `image-factory-essential-config-seeder`
- `image-factory-external-tenant-service`

Default targets:
- `linux/amd64`
- `linux/arm64`
- `darwin/amd64`
- `darwin/arm64`

Optional overrides:

```bash
make release-binaries IMAGE_VERSION=v0.1.0 RELEASE_TARGETS="linux/amd64 linux/arm64"
```

To upload assets to a GitHub release:

```bash
make release-upload-assets TAG=v0.1.0
```

### 3) Build all Image Factory images

```bash
make build-all-images
```

This builds:
- `image-factory-backend:latest`
- `image-factory-frontend:latest`
- `image-factory-docs:latest`
- `image-factory-dispatcher:latest`
- `image-factory-notification-worker:latest`
- `image-factory-email-worker:latest`
- `image-factory-internal-registry-gc-worker:latest`

### 4) Pre-pull runtime service images for Helm

```bash
make docker-pull-runtime
```

This pulls runtime dependency images used by the Helm chart:
- `postgres:15-alpine`
- `redis:7-alpine`
- `nats:2.10-alpine`
- `minio/minio:latest`
- `registry:2`
- `axllent/mailpit:latest`
- `glauth/glauth:latest`

### 5) Build and push multi-platform images (amd64/arm64)

```bash
make docker-build-all-multiarch IMAGE_REGISTRY=<registry>
# optional overrides:
# make docker-build-all-multiarch IMAGE_REGISTRY=<registry> IMAGE_VERSION=v0.1.0 IMAGE_ID=$(git rev-parse --short HEAD)
# make docker-build-all-multiarch IMAGE_REGISTRY=<registry> IMAGE_TAG=v0.1.0-a1b2c3d
```

Default pushed tag format:
- `IMAGE_TAG=$(IMAGE_VERSION)-$(IMAGE_ID)`
- default `IMAGE_VERSION=v0.1.0`
- default `IMAGE_ID=$(git rev-parse --short HEAD)`

This publishes manifest lists for:
- backend
- frontend
- docs
- dispatcher worker
- notification worker
- email worker
- internal registry gc worker

### 6) Podman equivalents

```bash
make build-all-images CONTAINER_ENGINE=podman COMPOSE_CMD="podman compose"
make docker-pull-runtime CONTAINER_ENGINE=podman
make docker-build-all-multiarch CONTAINER_ENGINE=podman IMAGE_REGISTRY=<registry>
make docker-build-all-multiarch CONTAINER_ENGINE=podman IMAGE_REGISTRY=<registry> FRONTEND_USE_LOCAL_DIST=true
```

### 7) Build + Push + Helm Deploy in one command

```bash
make release-deploy \
  IMAGE_REGISTRY=docker.io/imagefactoryoss \
  CONTAINER_ENGINE=podman

# Optional: package frontend using local dist output
make release-deploy \
  IMAGE_REGISTRY=docker.io/imagefactoryoss \
  CONTAINER_ENGINE=podman \
  FRONTEND_USE_LOCAL_DIST=true
```

What it does:
- builds and pushes all app/worker multi-platform images with `IMAGE_TAG`
- runs `helm upgrade --install` with `--reuse-values`
- updates backend/frontend/docs/worker image repositories and tags
- waits for backend/frontend/docs rollout success

Optional overrides:
- `IMAGE_VERSION`, `IMAGE_ID`, or explicit `IMAGE_TAG`
- `HELM_RELEASE` (default `image-factory`)
- `HELM_NAMESPACE` (default `image-factory`)
- `HELM_CHART` (default `./deploy/helm/image-factory`)

## Docker/Image Targets

 - `make docker-build` builds backend, frontend, and docs images.
- `make docker-build-workers` builds worker images.
- `make build-all-images` builds app + workers.
- `make docker-build-all-multiarch` builds amd64/arm64 and pushes manifests.
- `make release-deploy` builds, pushes, and deploys Helm release in one workflow.
- `make docker-pull-runtime` pulls runtime dependency images.
- `make docker-push` provides placeholder push guidance.
- `make docker-clean` prunes local container/volume resources.

## Compose Targets

- `make dev` starts development stack (`docker-compose.yml` + `docker-compose.dev.yml`).
- `make prod` starts production compose stack.
- `make dev-clean` removes dev stack resources and prunes system artifacts.

If you want Docker Compose instead, set:

```bash
COMPOSE_CMD="docker-compose"
```

## Database Targets

- `make db-shell`
- `make db-backup`
- `make db-restore BACKUP=<file.sql>`

These use `$(CONTAINER_ENGINE) exec`, so they work with Docker or Podman as long as container names are consistent.

## Testing and Quality

- `make test`
- `make test-coverage`
- `make lint`
- `make quality-check`

## Tool Check

```bash
make check-tools
```

Checks for:
- `go`
- `node`
- selected `CONTAINER_ENGINE`
- first binary in `COMPOSE_CMD`

## Notes for Helm Deployments

Helm chart path:

- `deploy/helm/image-factory`

Typical sequence before Helm install/upgrade:

```bash
make build-all-images
make docker-pull-runtime
# tag/push images to your registry, then set helm values for repositories/tags
```

For worker-specific image overrides in Helm values, see:
- `deploy/helm/image-factory/README.md`
