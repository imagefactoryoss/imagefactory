# Build Management Database Schema Quick Reference

**Current design:** Builds are queued via `builds.status = 'queued'` and dispatched by the build dispatcher. No queue tables are used.

---

## Schema Dependency Graph

```
tenants
  ├─→ builds (tenant_id FK)
  │    ├─→ build_configs (FK) [NEW: method-specific config]
  │    ├─→ build_steps (FK)
  │    ├─→ build_logs (FK)
  │    ├─→ build_metrics (FK)
  │    ├─→ build_artifacts (FK)
  │    ├─→ build_results (FK) [NEW: execution summary]
  │    ├─→ build_status_history (FK) [NEW: audit trail]
  │    └─→ build_triggers (FK) [NEW: how it was triggered]
  │
  ├─→ projects (tenant_id FK)
  │    ├─→ images (project_id FK)
  │    │    ├─→ image_metadata (image_id FK)
  │    │    └─→ image_layers (image_id FK)
  │    │
  │    ├─→ webhook_configs (project_id FK) [NEW: Git webhooks]
  │    └─→ build_schedules (project_id FK) [NEW: scheduled builds]
  │
  ├─→ build_workers (tenant_id FK) [NEW: execution infrastructure]
  ├─→ build_concurrency_policies (tenant_id FK) [NEW: limits]
  └─→ build_performance_daily (tenant_id FK) [NEW: analytics]
```

---

## Table Reference Matrix

### 1. CORE BUILD TABLES

#### `builds` (Migration 001)
| Column | Type | Purpose |
|--------|------|---------|
| id | UUID PK | Build identity |
| tenant_id | UUID FK | Tenant scoping |
| project_id | UUID FK | Project scoping |
| image_id | UUID FK | Output image reference |
| build_number | INT | Sequential per-project number |
| triggered_by_user_id | UUID FK | Audit trail |
| triggered_by_git_event | VARCHAR | How build was triggered |
| git_commit, branch, author, message | Various | Git context |
| started_at, completed_at | TIMESTAMP | Execution window |
| status | VARCHAR ENUM | `queued, in_progress, success, failed, cancelled` |
| error_message | TEXT | Failure details |
| cleanup_at | TIMESTAMP | When build artifacts were cleaned |

**Notes:** Core structure is solid, with some summary-oriented details separated into related tables.  
**Related:** Link to `build_results` for summary queries.

---

#### `build_configs` (NEW - Migration 012)
| Column | Type | Purpose |
|--------|------|---------|
| id | UUID PK | Config identity |
| build_id | UUID FK UNIQUE | 1:1 with build |
| build_method | VARCHAR | `kaniko, buildx, container, paketo, packer` |
| dockerfile | TEXT | For: kaniko, buildx |
| build_context | VARCHAR | For: kaniko, buildx (usually ".") |
| cache_enabled | BOOLEAN | For: kaniko, buildx |
| cache_repo | VARCHAR | For: kaniko layer cache |
| metadata | JSONB | Method-specific values not covered by schema (e.g., `registry_repo` for Kaniko) |
| platforms | JSONB | For: buildx (["linux/amd64", "linux/arm64"]) |
| cache_from, cache_to | VARCHAR/JSONB | For: buildx |
| target_stage | VARCHAR | For: buildx, container (multi-stage) |
| builder | VARCHAR | For: paketo ("paketobuildpacks/builder:base") |
| buildpacks | JSONB | For: paketo (array of buildpack images) |
| packer_template | TEXT | For: packer (HCL template) |
| build_args, environment, secrets | JSONB | Shared across all methods |

**Notes:** This model supports method-specific validation and configuration more cleanly.  
**Replaces:** JSONB-heavy configuration in `build_manifest`.

---

#### `build_triggers` (NEW - Migration 013)
| Column | Type | Purpose |
|--------|------|---------|
| id | UUID PK | Trigger event identity |
| build_id | UUID FK | Which build was triggered |
| trigger_type | VARCHAR | `manual, webhook, schedule, git_event` |
| trigger_source | VARCHAR | Git branch, cron expr, webhook ID, etc. |
| created_by_user_id | UUID FK | Who/what initiated |
| created_at | TIMESTAMP | When triggered |

**Notes:** Adds an audit trail for how builds start.  
**Why it matters:** Tracks whether a build came from a webhook, schedule, or manual action.

---

### 2. EXECUTION TRACKING

#### `build_steps` (Migration 005)
| Column | Type | Purpose |
|--------|------|---------|
| id | UUID PK | Step identity |
| build_id | UUID FK | Which build |
| step_number | INT | Sequential within build |
| step_name | VARCHAR | "FROM", "RUN apt-get", etc. |
| instruction_type | VARCHAR | `FROM, RUN, COPY, ADD, ENV, EXPOSE, WORKDIR` |
| status | VARCHAR | `pending, running, success, failed, skipped` |
| layer_digest | VARCHAR | SHA256 of generated layer |
| layer_size_bytes | BIGINT | Layer size |
| cached | BOOLEAN | Was this layer reused? |
| stdout, stderr | TEXT | Step output |
| error_message, error_code | TEXT | Why failed? |
| duration_seconds | INT | How long step took |

**Notes:** Tracks Dockerfile execution line-by-line.  
**Use:** `SELECT * FROM build_steps WHERE build_id = ? ORDER BY step_number` for a full build trace.

---

#### `build_logs` (Migration 001)
| Column | Type | Purpose |
|--------|------|---------|
| id | UUID PK | Log line identity |
| build_id | UUID FK | Which build |
| log_content | TEXT | The actual log message |
| log_level | VARCHAR | `INFO, WARN, ERROR, DEBUG` |
| logged_at | TIMESTAMP | When logged |

**Notes:** Works well for batch inserts.  
**Limitation:** No streaming support, so sequential inserts may be slow for high-volume logs.

---

#### `build_metrics` (Migration 005)
| Column | Type | Purpose |
|--------|------|---------|
| id | UUID PK | Metrics identity |
| build_id | UUID FK | Which build |
| total_duration_seconds | INT | Wall clock time |
| docker_build_duration_seconds | INT | Just build phase |
| docker_push_duration_seconds | INT | Just push phase |
| peak_memory_usage_mb | INT | Max memory during build |
| cpu_usage_percent | DECIMAL | Average CPU |
| disk_read_bytes, disk_write_bytes | BIGINT | I/O metrics |
| total_layers, reused_layers, new_layers | INT | Layer reuse stats |
| final_image_size_bytes | BIGINT | Final image size |
| compression_ratio | DECIMAL | Compressed vs uncompressed |

**Notes:** Supports comprehensive performance tracking.  
**Use:** Identify slow builds and optimization opportunities.

---

#### `build_artifacts` (Migration 005)
| Column | Type | Purpose |
|--------|------|---------|
| id | UUID PK | Artifact identity |
| build_id | UUID FK | Which build produced it |
| artifact_type | VARCHAR | `docker_image, sbom, test_report, build_log, security_scan` |
| artifact_name | VARCHAR | "myapp:1.0", "sbom.json", etc. |
| artifact_location | VARCHAR | URL or S3 path |
| artifact_size_bytes | BIGINT | Size in bytes |
| sha256_digest | VARCHAR | Integrity hash |
| is_available | BOOLEAN | Still accessible? |
| retention_policy | VARCHAR | `permanent, days_30, days_90, days_365, delete_on_success` |
| expires_at | TIMESTAMP | When to delete |
| image_id | UUID FK | Links to images table |

**Notes:** Supports full artifact lifecycle management.  
**Use:** Track all build outputs such as images, SBOMs, reports, and logs.

---

### 3. IMAGE & LAYER TRACKING

#### `image_metadata` (Migration 005)
| Column | Type | Purpose |
|--------|------|---------|
| id | UUID PK | Metadata identity |
| image_id | UUID FK UNIQUE | 1:1 with image |
| docker_config_digest | VARCHAR | Docker config hash |
| docker_manifest_digest | VARCHAR | Manifest hash |
| total_layer_count | INT | # of layers |
| compressed_size_bytes | BIGINT | On disk |
| uncompressed_size_bytes | BIGINT | In memory |
| packages_count | INT | Installed packages |
| vulnerabilities_high/med/low | INT | Scan results |
| entrypoint, cmd, env_vars, working_dir | JSON | Runtime config |
| last_scanned_at | TIMESTAMP | When last scanned |
| scan_tool | VARCHAR | Tool used |

**Notes:** Captures rich image metadata.  
**Use:** Support image comparison, vulnerability tracking, and compliance reporting.

---

#### `image_layers` (Migration 005)
| Column | Type | Purpose |
|--------|------|---------|
| id | UUID PK | Layer identity |
| image_id | UUID FK | Which image |
| layer_number | INT | Position in image |
| layer_digest | VARCHAR | SHA256 (content addressed) |
| layer_size_bytes | BIGINT | Size |
| is_base_layer | BOOLEAN | Is this from base image? |
| base_image_name, base_image_tag | VARCHAR | Which base? |
| used_in_builds_count | INT | Reuse counter (optimization!) |
| last_used_in_build_at | TIMESTAMP | When reused |

**Notes:** Useful for layer reuse and optimization analysis.  
**Use:** Identify reusable layers and improve caching strategy.

---

### 4. TRIGGER & WEBHOOK MANAGEMENT

#### `webhook_configs` (NEW - Migration 013)
| Column | Type | Purpose |
|--------|------|---------|
| id | UUID PK | Webhook identity |
| tenant_id, project_id | UUID FK | Scoping |
| webhook_name | VARCHAR | "Deploy on main push" |
| webhook_url | VARCHAR | Where to send events |
| webhook_secret | VARCHAR | HMAC secret for verification |
| event_types | JSONB | ["push", "pull_request", "release"] |
| branch_patterns | JSONB | ["main", "release/*"] |
| auto_build_enabled | BOOLEAN | Should trigger build? |
| build_method | VARCHAR | Which method to use |
| build_config_preset | JSONB | Template for build config |
| is_active | BOOLEAN | Enabled? |
| is_verified | BOOLEAN | HMAC test passed? |
| last_delivery_at, last_delivery_status | TIMESTAMP/VARCHAR | Monitoring |

**Notes:** Adds Git webhook management.  
**Enables:** Automatic builds on Git push events.

---

#### `webhook_deliveries` (NEW - Migration 013)
| Column | Type | Purpose |
|--------|------|---------|
| id | UUID PK | Delivery identity |
| webhook_config_id | UUID FK | Which webhook |
| build_id | UUID FK | Build it triggered |
| event_type | VARCHAR | "push", "pull_request", etc. |
| payload | JSONB | Full git event data |
| response_status | INT | HTTP response code |
| error_message | TEXT | Why failed? |
| attempt_number | INT | Retry count |
| next_retry_at | TIMESTAMP | When to retry |

**Notes:** Adds a webhook delivery audit trail.  
**Use:** Debug webhook failures and trace events to builds.

---

#### `build_schedules` (NEW - Migration 013)
| Column | Type | Purpose |
|--------|------|---------|
| id | UUID PK | Schedule identity |
| tenant_id, project_id | UUID FK | Scoping |
| schedule_name | VARCHAR | "Nightly rebuild" |
| cron_expression | VARCHAR | "0 2 * * *" (2 AM daily) |
| timezone | VARCHAR | Timezone for cron |
| build_method | VARCHAR | Which method to use |
| build_config_preset | JSONB | Build template |
| git_branch | VARCHAR | Which branch to build |
| is_active | BOOLEAN | Enabled? |
| last_triggered_at | TIMESTAMP | When last executed |
| next_trigger_at | TIMESTAMP | When next execution |

**Notes:** Adds scheduled rebuild support.  
**Enables:** Nightly builds, recurring rescans, and other scheduled automation.

---

### 5. DISPATCHING & WORKER MANAGEMENT

#### `build_workers` (NEW - Migration 014)
| Column | Type | Purpose |
|--------|------|---------|
| id | UUID PK | Worker identity |
| tenant_id | UUID FK | Which tenant |
| worker_name | VARCHAR UNIQUE | "k8s-node-1", "docker-01" |
| worker_type | VARCHAR | `docker, kubernetes, local` |
| concurrent_builds_limit | INT | Max parallel builds |
| current_builds_count | INT | Currently running |
| status | VARCHAR | `healthy, busy, degraded, offline` |
| zone | VARCHAR | Geographic zone |
| labels | JSONB | `{"gpu": true, "memory": "32gb"}` |
| last_heartbeat | TIMESTAMP | Health check |
| uptime_percent | DECIMAL | Reliability metric |

**Notes:** Adds worker pool management.  
**Enables:** Multi-worker builds, load balancing, and affinity rules.

---

#### `build_concurrency_policies` (NEW - Migration 014)
| Column | Type | Purpose |
|--------|------|---------|
| id | UUID PK | Policy identity |
| tenant_id | UUID FK UNIQUE | One policy per tenant |
| max_concurrent_builds | INT | Tenant-wide limit |
| max_concurrent_per_project | INT | Per-project limit |
| kaniko_max_concurrent | INT | Method-specific limit |
| buildx_max_concurrent | INT | Method-specific limit |
| packer_max_concurrent | INT | Method-specific limit |
| paketo_max_concurrent | INT | Method-specific limit |
| priority_dispatch_enabled | BOOLEAN | Enable dispatcher priority routing |
| max_queued_wait_minutes | INT | Timeout for queued builds |

**Notes:** Adds concurrency enforcement controls.  
**Enables:** Resource control, fair scheduling, and method-specific limits.

---

### 6. RESULTS & ANALYTICS

#### `build_results` (NEW - Migration 015)
| Column | Type | Purpose |
|--------|------|---------|
| id | UUID PK | Result identity |
| build_id | UUID FK UNIQUE | 1:1 with build |
| status | VARCHAR | `success, failed, cancelled, timeout` |
| exit_code | INT | Process exit code |
| output_image_id | UUID FK | Resulting image |
| output_image_url | VARCHAR | Registry URL |
| output_image_digest | VARCHAR | Image hash |
| failure_reason | VARCHAR | Why failed? |
| failure_stage | VARCHAR | `compile, build, push, scan, verification` |
| total_duration_milliseconds | BIGINT | End-to-end time |
| queue_wait_milliseconds | BIGINT | Time in queued status |
| build_execution_milliseconds | BIGINT | Actual build time |
| artifact_push_milliseconds | BIGINT | Push time |
| retry_count | INT | How many retries? |
| is_retry | BOOLEAN | Is this a retry? |
| original_build_id | UUID FK | What it's retrying |
| cleanup_status | VARCHAR | `pending, completed, failed` |

**Notes:** Adds an execution summary model.  
**Enables:** Quick result queries, analytics, and retry logic.

---

#### `build_status_history` (NEW - Migration 015)
| Column | Type | Purpose |
|--------|------|---------|
| id | UUID PK | History record identity |
| build_id | UUID FK | Which build |
| from_status | VARCHAR | Previous status |
| to_status | VARCHAR | New status |
| reason | TEXT | Why the change? |
| changed_by_user_id | UUID FK | Who initiated? |
| changed_by_system | BOOLEAN | Or was it automatic? |
| changed_at | TIMESTAMP | When changed |

**Notes:** Adds a status change audit trail.  
**Enables:** Full lifecycle visibility, debugging, and compliance auditing.

---

#### `build_performance_daily` (NEW - Migration 015)
| Column | Type | Purpose |
|--------|------|---------|
| id | UUID PK | Record identity |
| tenant_id, date | UUID, DATE PK | Time-series data |
| total_builds, successful, failed, cancelled | INT | Build counts |
| average_duration_seconds | INT | Performance metric |
| median_queue_wait_seconds | INT | Queued wait performance |
| failure_rate_percent | DECIMAL | Health metric |
| most_common_failure | VARCHAR | Common issues |
| kaniko_count, buildx_count, packer_count, paketo_count | INT | Method breakdown |
| max_concurrent_at_peak | INT | Peak load |

**Notes:** Adds analytics aggregation.  
**Enables:** Trending, capacity planning, and performance optimization.

---

## Summary: What's Missing vs. What Needs Fixing

| Area | Current State | Action |
|------|---------------|--------|
| **Build Core** | ✅ Good structure | Minor: add result summary table |
| **Execution Tracking** | ✅ Steps, logs, metrics all good | Optional: improve log streaming |
| **Dispatching** | ⚠️ Status-based only | ✅ **Implement** dispatcher service |
| **Config Storage** | ❌ JSONB only | ✅ **Add** dedicated table with method columns |
| **Trigger Tracking** | ❌ No persistence | ✅ **Add** webhooks, schedules, history |
| **Worker Pool** | ❌ Not managed | ✅ **Add** worker registry, health checks |
| **Queue Positions** | ❌ Not in schema | Optional future feature |
| **Analytics** | ❌ No aggregation | ✅ **Add** daily performance rollups |

---

## Quick Decision Table

**If you want to support...**

| Feature | Tables Needed |
|---------|---------------|
| Automatic builds on git push | `webhook_configs`, `webhook_deliveries` |
| Nightly/scheduled builds | `build_schedules` |
| Dispatching and worker routing | `builds`, `build_workers`, `build_concurrency_policies` |
| Multiple concurrent builders | `build_workers`, `build_concurrency_policies` |
| Build method validation | `build_configs` (method-specific columns) |
| Full audit trail | `build_triggers`, `build_status_history`, `webhook_deliveries` |
| Performance analytics | `build_results`, `build_performance_daily` |
