# Image Factory Helm Chart

This chart deploys Image Factory and required runtime dependencies:

- Backend API
- Frontend UI
- User-facing docs server
- Dispatcher worker
- Notification worker
- Email worker
- Internal registry GC worker
- PostgreSQL
- Redis
- NATS (JetStream)
- MinIO
- Docker Registry
- Mailpit
- GLAuth (LDAP simulation)
- Optional Ollama runtime for local/in-cluster SRE Smart Bot interpretation

## Prerequisites

- Kubernetes 1.25+
- Helm 3.12+
- Published images for backend/frontend (and optional per-worker overrides)

## Quick Start

```bash
kubectl create ns image-factory

# If images are private, create pull secret first
kubectl -n image-factory create secret generic gitlab-registry \
  --from-file=.dockerconfigjson=$HOME/.config/containers/auth.json \
  --type=kubernetes.io/dockerconfigjson

helm upgrade --install image-factory ./deploy/helm/image-factory \
  -n image-factory \
  --set imagePullSecrets[0].name=gitlab-registry
```

To expose the docs server on a separate hostname, enable ingress and set `ingress.docsHost`.

## OKE Ingress + TLS (NLB, Static IP, cert-manager)

For a shared ingress model (multiple apps behind one ingress controller), use:

- `ingress-nginx` controller Service type `LoadBalancer` (OCI NLB)
- reserved public IP (optional but recommended)
- `image-factory` chart ingress (`ingress.enabled=true`)
- cert-manager with Let's Encrypt HTTP-01

Cloudflare API credentials are **not required** for HTTP-01. They are only required if you choose DNS-01 challenges.

Minimum OCI network requirements:

- Service LB subnet security list ingress:
  - `0.0.0.0/0` -> TCP `80`
  - `0.0.0.0/0` -> TCP `443`
- Worker/node subnet security list ingress:
  - LB subnet CIDR -> TCP NodePort range (`30000-32767`) or at least the two allocated NodePorts for ingress controller
- Service LB subnet route table:
  - `0.0.0.0/0` -> Internet Gateway

If HTTP-01 fails with connection timeout, validate public reachability:

```bash
curl -I http://<your-domain>/.well-known/acme-challenge/ping
```

You should get an HTTP response from nginx (often `308` redirect to HTTPS), not a connection timeout.

Example ingress install values for OCI NLB with reserved IP:

```bash
helm upgrade --install ingress-nginx ingress-nginx \
  -n ingress-nginx --create-namespace \
  --repo https://kubernetes.github.io/ingress-nginx \
  --set controller.service.type=LoadBalancer \
  --set controller.service.externalTrafficPolicy=Cluster \
  --set controller.service.annotations.\"oci\\.oraclecloud\\.com/load-balancer-type\"=nlb \
  --set controller.service.annotations.\"oci\\.oraclecloud\\.com/reserved-ips\"=\"161.153.65.85\" \
  --set controller.service.annotations.\"oci-network-load-balancer\\.oraclecloud\\.com/external-ip-only\"=\"true\"
```

## Recommended Image Overrides

```bash
export IMAGE_TAG=v0.1.0-$(git rev-parse --short HEAD)

helm upgrade --install image-factory ./deploy/helm/image-factory -n image-factory \
  --set imagePullSecrets[0].name=gitlab-registry \
  --set backend.image.repository=registry.gitlab.com/s4cna/image-factory/image-factory-backend \
  --set backend.image.tag=$IMAGE_TAG \
  --set backend.image.pullPolicy=Always \
  --set frontend.image.repository=registry.gitlab.com/s4cna/image-factory/image-factory-frontend \
  --set frontend.image.tag=$IMAGE_TAG \
  --set frontend.image.pullPolicy=Always \
  --set docs.image.repository=registry.gitlab.com/s4cna/image-factory/image-factory-docs \
  --set docs.image.tag=$IMAGE_TAG \
  --set docs.image.pullPolicy=Always \
  --set frontend.service.type=LoadBalancer \
  --set workers.dispatcher.image.repository=registry.gitlab.com/s4cna/image-factory/image-factory-dispatcher \
  --set workers.dispatcher.image.tag=$IMAGE_TAG \
  --set workers.notification.image.repository=registry.gitlab.com/s4cna/image-factory/image-factory-notification-worker \
  --set workers.notification.image.tag=$IMAGE_TAG \
  --set workers.email.image.repository=registry.gitlab.com/s4cna/image-factory/image-factory-email-worker \
  --set workers.email.image.tag=$IMAGE_TAG \
  --set workers.internalRegistryGc.image.repository=registry.gitlab.com/s4cna/image-factory/image-factory-internal-registry-gc-worker \
  --set workers.internalRegistryGc.image.tag=$IMAGE_TAG
```

The chart defaults are intended to pull both application images and bundled runtime dependencies from GitLab. If you mirror runtime dependencies into a different registry, override `postgres.image.repository`, `redis.image.repository`, `nats.image.repository`, `minio.image.repository`, `registry.image.repository`, `mailpit.image.repository`, and `glauth.image.repository` explicitly.

Before deploying with GitLab as the source of truth, verify that the mirrored runtime images really exist in the registry:

```bash
GITLAB_USER=<registry-user> \
GITLAB_PASSWORD=<registry-password-or-token> \
./scripts/verify-gitlab-runtime-images.sh
```

If you mirror runtime images with `./scripts/mirror-runtime-images.sh`, set the same `GITLAB_USER` and `GITLAB_PASSWORD` so the script verifies each pushed manifest after upload.

## External Postgres / Supabase

When using an external Postgres instance (for example Supabase), set `database.mode=external`, disable bundled Postgres, and set DB values explicitly.

Important: the chart default schema is `image_factory`. If your application objects live in a different schema, set `database.schema` explicitly to match.

```bash
helm upgrade --install image-factory ./deploy/helm/image-factory -n image-factory \
  --set database.mode=external \
  --set postgres.enabled=false \
  --set database.host=<db-host> \
  --set database.port=5432 \
  --set database.name=postgres \
  --set database.user=postgres \
  --set database.password=<db-password> \
  --set database.sslMode=require \
  --set database.schema=image_factory
```

You can also start from the example override file:

```bash
cp deploy/helm/image-factory/values.external.example.yaml deploy/helm/image-factory/values.external.yaml
# edit deploy/helm/image-factory/values.external.yaml
helm upgrade --install image-factory ./deploy/helm/image-factory -n image-factory \
  -f deploy/helm/image-factory/values.yaml \
  -f deploy/helm/image-factory/values.external.yaml
```

## Docs Service

The chart can deploy the user-facing OSS documentation server separately from the application UI.

- `docs.enabled=true` deploys the docs workload and service
- `docs.image.*` controls the docs image
- `ingress.docsHost` exposes the docs site on a dedicated host such as `docs.imagefactory.dev`

Example:

```bash
helm upgrade --install image-factory ./deploy/helm/image-factory -n image-factory \
  --set ingress.enabled=true \
  --set ingress.docsHost=docs.imagefactory.dev \
  --set docs.image.repository=registry.gitlab.com/s4cna/image-factory/image-factory-docs \
  --set docs.image.tag=$IMAGE_TAG
```

## Database Mode Guardrails

The chart now fails fast on ambiguous DB config. No implicit DB fallback is used.

- `database.mode=incluster` requires `postgres.enabled=true`.
- `database.mode=external` requires `postgres.enabled=false`.
- For `external`, `database.host/name/user/password` are required.
- For `incluster`, `database.host/name/user/password` must be empty (derived from `postgres.*` values).

## Strict Config (No Silent Fallbacks)

The chart intentionally rejects ambiguous component config at render time.

- Worker images no longer inherit backend image values.
- Storage types (`postgres/redis/nats/minio/registry`) must be explicitly set to one of:
  - `emptyDir`
  - `pvc`
  - `hostPath`
- Invalid or incomplete storage/image configuration now fails `helm template/upgrade` with explicit errors.
- `ollama.storage.mode` must be explicitly set to one of:
  - `baked`
  - `emptyDir`
  - `pvc`
  - `hostPath`
- `loki.storage.mode` must be explicitly set to one of:
  - `emptyDir`
  - `pvc`
  - `hostPath`

## Optional Ollama Runtime

The chart can also deploy an internal Ollama service for SRE Smart Bot's optional local interpretation layer.

- `ollama.enabled=true` deploys the Ollama runtime and service
- `ollama.storage.mode=baked` means no PVC or runtime volume is created; the model is expected to already be inside the image
- `ollama.storage.mode=pvc` uses a PVC-backed model store
- `ollama.storage.mode=hostPath` or `emptyDir` are also supported for controlled environments

Recommended air-gapped demo profile:

```bash
make docker-build-ollama-baked \
  OLLAMA_MODEL_STORE=$HOME/.ollama \
  OLLAMA_MODEL_NAME=llama3.2:3b \
  OLLAMA_IMAGE_TAG=registry.gitlab.com/s4cna/image-factory/image-factory-ollama:llama3.2-3b

helm upgrade --install image-factory ./deploy/helm/image-factory -n image-factory \
  --set ollama.enabled=true \
  --set ollama.image.repository=registry.gitlab.com/s4cna/image-factory/image-factory-ollama \
  --set ollama.image.tag=llama3.2-3b \
  --set ollama.storage.mode=baked
```

To publish that baked image into your normal registry workflow:

```bash
make docker-build-ollama-baked-push \
  IMAGE_REGISTRY=registry.gitlab.com/s4cna/image-factory \
  IMAGE_TAG=v0.1.0-abc123 \
  OLLAMA_MODEL_STORE=$HOME/.ollama \
  OLLAMA_MODEL_NAME=llama3.2:3b
```

Or include it as part of `release-deploy`:

```bash
make release-deploy \
  IMAGE_REGISTRY=registry.gitlab.com/s4cna/image-factory \
  OLLAMA_ENABLED=true \
  OLLAMA_STORAGE_MODE=baked \
  OLLAMA_MODEL_STORE=$HOME/.ollama \
  OLLAMA_MODEL_NAME=llama3.2:3b
```

Note: the baked image path is intentionally treated as a controlled release artifact rather than part of the generic multi-arch image set, because it depends on a pre-seeded local model store.

Recommended larger-enterprise/runtime-managed profile:

```bash
helm upgrade --install image-factory ./deploy/helm/image-factory -n image-factory \
  --set ollama.enabled=true \
  --set ollama.storage.mode=pvc \
  --set ollama.persistence.enabled=true \
  --set ollama.persistence.size=20Gi
```

When using the in-cluster runtime, point SRE Smart Bot at:

```text
http://image-factory-ollama.image-factory.svc.cluster.local:11434
```

Adjust the release name/namespace as needed. If you want first-run installs to persist the deployment-aware SRE Smart Bot runtime into the saved `robot_sre_policy`, enable:

```bash
--set bootstrap.seedRobotSREPolicyDefaults=true
```

This only seeds the saved policy during bootstrap/reset when `ollama.enabled=true` and the config does not already exist. It does not overwrite an operator-customized policy on later upgrades.

## Optional Loki + Alloy Runtime

The chart can also deploy a lightweight in-cluster log path for SRE Smart Bot:

- `loki.enabled=true` deploys a single-binary Loki instance for detector queries and MCP log evidence
- `alloy.enabled=true` deploys an Alloy `DaemonSet` that tails node pod logs and pushes them into Loki
- `sre.logDetector.enabled=true` enables the backend log-detector runner that turns Loki matches into SRE incidents
- `IF_SRE_LOG_DETECTOR_LOKI_BASE_URL` is automatically pointed at the in-cluster Loki service when `loki.enabled=true`
- if you want Alloy to push to an external Loki instead, leave `loki.enabled=false` and set `alloy.loki.pushUrl`

Recommended external-cluster profile:

Use `deploy/helm/image-factory/values.external-cluster.example.yaml` as the canonical OSS starter values file.

```bash
helm upgrade --install image-factory ./deploy/helm/image-factory -n image-factory \
  -f deploy/helm/image-factory/values.external-cluster.example.yaml
```

Minimal direct example:

```bash
helm upgrade --install image-factory ./deploy/helm/image-factory -n image-factory \
  --set loki.enabled=true \
  --set loki.storage.mode=pvc \
  --set loki.persistence.enabled=true \
  --set loki.persistence.size=20Gi \
  --set alloy.enabled=true \
  --set alloy.clusterName=image-factory-external \
  --set sre.logDetector.enabled=true
```

When Loki is deployed in-cluster, the detector/MCP path uses:

```text
http://image-factory-loki.image-factory.svc.cluster.local:3100
```

Operational recommendation:

- use Loki in monolithic mode first
- keep retention short initially, such as `7d`
- keep labels low-cardinality
- use Alloy for pod log tailing instead of pushing raw logs through NATS
- the chart defaults to GitLab-mirrored `image-factory-loki` and `image-factory-alloy` images for controlled cluster installs

If you have not mirrored those runtime images yet:

```bash
GITLAB_USER=<user> GITLAB_PASSWORD=<token> ./scripts/mirror-runtime-images.sh
GITLAB_USER=<user> GITLAB_PASSWORD=<token> ./scripts/verify-gitlab-runtime-images.sh
```

To build/push the changed app images and deploy the external-cluster profile in one workflow:

```bash
make release-deploy-external-cluster \
  IMAGE_REGISTRY=registry.gitlab.com/s4cna/image-factory \
  HELM_NAMESPACE=image-factory
```

## Quarantine Reviewer Bootstrap Check

For central reviewer workflow (`/admin/quarantine/review`), ensure bootstrap seeds are present:

- tenant group `security-reviewers` (`role_type=security_reviewer`) under `sysadmin`
- role `Security Reviewer`
- permissions `quarantine:read`, `quarantine:approve`, `quarantine:reject`
- role-permission mappings for `Security Reviewer`

The chart bootstrap job runs `seed-essential-data.sql` and `essential-config-seeder --action seed`.
If your environment was provisioned before these rows existed, re-run bootstrap/seed jobs or apply the seed SQL once.

When `bootstrap.seedRobotSREPolicyDefaults=true` and `ollama.enabled=true`, the same bootstrap flow also seeds the global `robot_sre_policy` with the deployment-aware agent runtime defaults (provider/base URL/model) while leaving the AI runtime disabled until an operator enables it.

## Current Bootstrap Flow

The chart uses a **post-install/post-upgrade bootstrap hook job** (`image-factory-bootstrap`) when `bootstrap.enabled=true`.

The bootstrap job runs:

1. `migrate up`
2. SQL essential seed (`/app/bootstrap/seed-essential-data.sql`)
3. Optional SQL demo seed (`/app/bootstrap/seed-demo-data.sql`) when `bootstrap.seedDemoData=true`
4. `essential-config-seeder --action seed`
5. `email-template-seeder --action seed`
6. `external-service-seeder --action seed`

This job is idempotent and designed for first-run + safe re-runs.

## Reset + Bootstrap (Parity With Local Reset Script)

`bootstrap.enabled=true` does **not** reset existing data. It only runs migrate + seed.

For clean reset parity, use the dedicated `dbReset` hook job. This job:

1. Drops and recreates the configured schema (`IF_DATABASE_SCHEMA`)
2. Runs `migrate up`
3. Runs SQL seeds (`seed-essential-data.sql`, optional demo seed)
4. Runs seeders (`essential-config-seeder`, `email-template-seeder`, `external-service-seeder`)
5. Runs post-reset sanity checks (optional)

One-time reset example:

```bash
helm upgrade --install image-factory ./deploy/helm/image-factory -n image-factory --reuse-values \
  --set dbReset.enabled=true \
  --set dbReset.runOnUpgrade=true \
  --set dbReset.confirmation=RESET_IMAGE_FACTORY \
  --set dbReset.allowPublicSchemaReset=false \
  --set dbReset.seedDemoData=false
```

Immediately disable after successful run:

```bash
helm upgrade --install image-factory ./deploy/helm/image-factory -n image-factory --reuse-values \
  --set dbReset.enabled=false \
  --set dbReset.runOnUpgrade=false \
  --set dbReset.confirmation=
```

Safety behavior:

- Reset is blocked unless `dbReset.confirmation=RESET_IMAGE_FACTORY`.
- Reset is blocked for schema `public` unless `dbReset.allowPublicSchemaReset=true`.
- Schema name must be alphanumeric/underscore.

## Key Values

- `app.jwtSecret`: JWT signing secret
- `app.encryptionKey`: **base64-encoded 32-byte key** for AES-GCM (required)
- `backend.image.*`: backend image settings
- `frontend.image.*`: frontend image settings
- `frontend.apiBaseUrl`: optional external API origin/path root (for separate API host). Example: `https://api.example.com` or `https://api.example.com/api`.
- `frontend.extraEnv`: additional frontend container env vars
- `build.tektonEnabled`: enables Tekton executor wiring in backend/dispatcher (default `true` for chart deployments)
- `build.tektonKubeconfig`: optional kubeconfig path override (usually empty for in-cluster config)
- `database.mode`: required DB wiring mode (`incluster` or `external`)
- `database.maxOpenConns` / `database.maxIdleConns` / `database.connMaxLifetime`: app DB pool tuning (applies to all services)
- `workers.*.enabled`: enable/disable workers
- `workers.*.image.*`: per-worker image override
- `bootstrap.enabled`: enable bootstrap hook job
- `bootstrap.seedDemoData`: run demo SQL seed during bootstrap (default `false`)
- `bootstrap.seedRobotSREPolicyDefaults`: seed global `robot_sre_policy` on first bootstrap/reset when `ollama.enabled=true` (default `false`)
- `dbReset.enabled`: enable destructive reset hook job (default `false`)
- `dbReset.runOnUpgrade`: allow running reset job on `helm upgrade`
- `dbReset.confirmation`: must be `RESET_IMAGE_FACTORY` for reset to execute
- `dbReset.allowPublicSchemaReset`: extra safety gate for `public` schema resets
- `dbReset.seedDemoData`: include demo SQL during reset
- `dbReset.runValidationChecks`: run post-reset sanity queries
- `registry.storage.type`: `pvc` (default), `emptyDir`, or `hostPath`
- `registry.storage.hostPath.path`: node-local path when `registry.storage.type=hostPath`
- `migrations.enabled`: legacy migration hook job (defaults to `false`)
- `postgres/redis/nats/minio/registry/mailpit/glauth.enabled`: bundled dependency toggles
- `ingress.enabled`: ingress-based frontend exposure

### Separate API Host (runtime, no rebuild)

Frontend reads API endpoint from runtime-generated `/config.js` using env `IF_FRONTEND_API_BASE_URL`.

- if empty, frontend defaults to same-origin `/api/v1`.
- if set to `https://api.example.com`, frontend uses `https://api.example.com/api/v1`.
- if set to `https://api.example.com/api`, frontend uses `https://api.example.com/api/v1`.

Example:

```bash
helm upgrade --install image-factory ./deploy/helm/image-factory -n image-factory \
  --set frontend.apiBaseUrl=https://api.example.com
```

### Local Secure Overrides

Keep sensitive values out of `values.yaml` by using a local override file:

```bash
cp deploy/helm/image-factory/values.local.example.yaml deploy/helm/image-factory/values.local.yaml
```

Then set secure values in `values.local.yaml` and deploy with:

```bash
helm upgrade --install image-factory ./deploy/helm/image-factory -n image-factory \
  -f deploy/helm/image-factory/values.yaml \
  -f deploy/helm/image-factory/values.local.yaml
```

## GLAuth

GLAuth is deployed by default via:

- ConfigMap: `<release>-glauth-config`
- Deployment: `<release>-glauth`
- Service: `<release>-glauth` on port `3893`

Default config lives in `values.yaml` under `glauth.config`.

## Build/Push Images

```bash
make build-all-images
make docker-build-all-multiarch IMAGE_REGISTRY=<registry> IMAGE_TAG=<tag>
make release-deploy IMAGE_REGISTRY=<registry>

# Podman
make build-all-images CONTAINER_ENGINE=podman COMPOSE_CMD="podman compose"
make docker-build-all-multiarch CONTAINER_ENGINE=podman IMAGE_REGISTRY=<registry> IMAGE_TAG=<tag>
make release-deploy CONTAINER_ENGINE=podman IMAGE_REGISTRY=<registry>
```

## Quirks And Troubleshooting

- If reusing mutable tags (`backend`, `frontend`, etc.), set `imagePullPolicy=Always`.
- If builds fail with `docker not installed (required for kaniko)`, verify `build.tektonEnabled=true` and redeploy backend + dispatcher.
- `app.encryptionKey` must be base64 for exactly 32 bytes. Example:
  - `openssl rand -base64 32 | tr -d '\n'`
- If backend fails with bootstrap/admin errors, verify `admin@imgfactory.com` exists in `users` and bootstrap job completed.
- If using Supabase pooler (session mode on `:5432`) and you see `MaxClientsInSessionMode`, reduce `database.maxOpenConns` and `database.maxIdleConns` (for example `5`/`2`) or move to transaction pooler endpoint/port.
- If PostgreSQL fails with `lost+found` errors, ensure PGDATA points to a subdirectory (`/var/lib/postgresql/data/pgdata`) as configured by this chart.
- Verify bootstrap hook status with:
  - `kubectl -n image-factory get events --sort-by=.lastTimestamp | grep image-factory-bootstrap`
- If GLAuth is crash-looping, ensure chart command uses `/app/glauth` and config file path is `/etc/glauth/glauth.cfg`.
- OCI NLB reserved IPs cannot be updated in place on an existing Service. If you change `oci.oraclecloud.com/reserved-ips`, recreate the ingress Service (or reinstall controller) so OCI can re-provision the LB with the new IP.

## Switching Domain And Reissuing Certs

When changing from one hostname to another, update ingress host + TLS config and trigger a fresh certificate order.

1. Update DNS:
- Point new domain (`A` record) to ingress public IP (for example `161.153.65.85`).
- Disable CDN proxy during validation (DNS-only mode) when using HTTP-01.

2. Upgrade chart with new hosts:

```bash
helm upgrade --install image-factory ./deploy/helm/image-factory -n image-factory \
  --set ingress.enabled=true \
  --set ingress.className=nginx \
  --set ingress.hosts[0].host=www.imagefactory.dev \
  --set ingress.appHost=app.imagefactory.dev \
  --set ingress.tls[0].hosts[0]=www.imagefactory.dev \
  --set ingress.tls[0].hosts[1]=app.imagefactory.dev \
  --set ingress.tls[0].secretName=imagefactory-dev-tls \
  --set ingress.certManager.enabled=true \
  --set ingress.certManager.clusterIssuer.name=letsencrypt-staging
```

Notes:
- `www.imagefactory.dev` serves the public landing experience.
- `app.imagefactory.dev` uses an app ingress with `nginx.ingress.kubernetes.io/app-root: /login` so `/` lands on the login page.
- Headlamp hostname is managed in the Headlamp chart/release, not this chart.

3. (If previous order is stuck/invalid) reset cert-manager resources:

```bash
kubectl -n image-factory delete order --all
kubectl -n image-factory delete challenge --all
kubectl -n image-factory delete certificaterequest --all
kubectl -n image-factory delete certificate <new-domain-tls-secret>
```

4. Verify issuance:

```bash
kubectl -n image-factory get certificate,certificaterequest,order,challenge
```

Expected end state:
- `challenge` -> `valid`
- `order` -> `valid`
- `certificate` -> `READY=True`

5. Move to production issuer after staging success:
- set `ingress.certManager.clusterIssuer.name=letsencrypt-prod`
- set server to `https://acme-v02.api.letsencrypt.org/directory`
- re-run helm upgrade.

## Production Notes

- Prefer managed Postgres/Redis/NATS/object storage/registry for production.
- Disable bundled dependencies when using managed services.
- Replace default secrets and tune resource requests/limits before production rollout.
