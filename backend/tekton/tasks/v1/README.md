# Tekton task assets

This folder contains Image Factory managed Tekton Task assets used by the `v1` pipelines:
- `git-clone`
- `docker-build`
- `buildx`
- `kaniko`
- `scan-image`
- `generate-sbom`
- `push-image`
- `sign-image`
- `packer`

These tasks are included by `backend/tekton/kustomization.yaml`, so an installer/apply flow can bootstrap both tasks and pipelines together.

## In-Cluster Registry

The Tekton kustomization also includes an internal registry stack for prototype/local clusters:

- `jobs/v1/internal-registry-pvc.yaml`
- `jobs/v1/internal-registry-deployment.yaml`
- `jobs/v1/internal-registry-service.yaml`

Default in-cluster service endpoint:

- `image-factory-registry:5000` (same namespace resolution)

Use that service in build/quarantine image references when external registries are not available, for example:

- `image-factory-registry:5000/published/<tenant>/<image>:<tag>`

Note:

- Build methods still require a `registry_auth_id`. For anonymous/internal registry usage, provide a tenant/project registry auth entry with `auth_type=dockerconfigjson` and payload `{"auths":{}}`.

## Trivy DB Mirror Warmup

`backend/tekton/jobs/v1/trivy-db-warmup-cronjob.yaml` mirrors Trivy DB OCI artifacts
into the internal registry:

- `image-factory-registry:5000/security/trivy-db:2`
- `image-factory-registry:5000/security/trivy-java-db:1`

Scan tasks default to these internal references first, with public mirrors as fallback.
