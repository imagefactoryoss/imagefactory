# Tekton Readiness Troubleshooting

This guide helps diagnose failures from:

`GET /api/v1/admin/infrastructure/providers/{id}/readiness`

## Quick usage

- Baseline checks:
  - `/readiness`
- Include PVC storage probe:
  - `/readiness?probe_pvc=true`
- Include namespace RBAC probe:
  - `/readiness?probe_rbac=true`
- Scope to a tenant namespace:
  - `/readiness?tenant_id=<tenant-uuid>`

## Common failing checks

- `provider_config.tekton_enabled`
  - Meaning: Provider config does not have `tekton_enabled=true`.
  - Fix: Update provider config and re-run readiness.

- `provider_kubeconfig`
  - Meaning: Provider kube config/credentials are invalid.
  - Fix: Re-save API endpoint/credentials and verify connectivity.

- `kubernetes_api` / `kubernetes_api_latency`
  - Meaning: K8s API unreachable or too slow.
  - Fix: Check control-plane endpoint routing, TLS, and network policy.

- `tekton_api` / `tekton_api_latency`
  - Meaning: Tekton CRDs/API not reachable in target namespace.
  - Fix: Confirm Tekton installation and access to `PipelineRun` resources.

- `task.<name>`
  - Meaning: Required task missing in tenant namespace.
  - Fix: Install required tasks (`git-clone`, `docker-build`, `buildx`, `kaniko`, `packer`).

- `pipeline.<name>`
  - Meaning: Required pipeline missing (pipelineRef mode).
  - Fix: Apply Image Factory Tekton assets:
    - `kubectl apply -k backend/tekton`

- `secret.docker-config`
  - Meaning: Registry auth secret not present in tenant namespace.
  - Fix: Materialize/create `docker-config` in tenant namespace.

- `storage.pvc_probe`
  - Meaning: PVC cannot be provisioned.
  - Fix: Verify default or configured storage class and PVC quota limits.

- `namespace_rbac_probe`
  - Meaning: Service account cannot create/delete ConfigMaps in tenant namespace.
  - Fix: Grant namespace-scoped RBAC for required resource verbs.

- `cluster_capacity`
  - Meaning: No nodes or no Ready nodes detected.
  - Fix: Recover node readiness, autoscaler, or underlying cluster health.

## Minimum namespace permissions for runtime

At minimum, Image Factory runtime identity should be able to:

- Get/List/Create/Delete `pipelineruns.tekton.dev`
- Get `pipelines.tekton.dev` and `tasks.tekton.dev`
- Get `secrets`
- Create/Delete `persistentvolumeclaims` (if using PVC workspaces)
- (optional readiness) Create/Delete `configmaps` (for RBAC probe)

## Recommended workflow

1. Run baseline readiness.
2. Resolve `missing_prereqs` in order (credentials, APIs, tasks/pipelines, secret).
3. Run with probes enabled:
   - `probe_pvc=true`
   - `probe_rbac=true`
4. Re-run a Tekton build and confirm scheduling + completion.
