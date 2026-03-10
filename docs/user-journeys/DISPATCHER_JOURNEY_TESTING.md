# Build Dispatcher - User Journey Testing

This document validates dispatcher-based build execution end-to-end, using `builds.status = queued` as the queue representation.

## Scope

This document tests the dispatcher flow that:
1. Queues builds by status.
2. Claims queued builds.
3. Dispatches executions.
4. Exposes metrics via `/api/v1/admin/dispatcher/metrics`.

---

## Prerequisites

1. Backend running with dispatcher enabled (`dispatcher.enabled = true` in config).
2. Database migrated and seeded.
3. You have a user with system read permissions to access dispatcher metrics.
4. Build method configs are valid for the selected build type.
5. At least one infrastructure provider exists if using Tekton execution.

---

## Journey A: Dispatcher Health + Metrics

**Goal:** Verify dispatcher is running and metrics endpoint is reachable.

1. Start backend server.
2. Confirm logs contain “Build dispatcher started”.
3. Call `GET /api/v1/admin/dispatcher/metrics`.
4. Verify JSON shape includes:
   - `claims`, `dispatches`, `claim_errors`, `dispatch_errors`, `requeues`, `skipped_for_limit`
   - `claim_*` and `dispatch_*` latency fields

**Expected:** Metrics endpoint responds `200` and values are numeric.

---

## Journey B: Queued Build → Running Execution

**Goal:** Create a build and confirm dispatcher claims it and dispatches execution.

1. Create a build (UI or API) with a valid build config.
2. Verify initial build status is `queued`.
3. Wait one dispatcher tick (default poll interval is 3s).
4. Verify build status transitions to `running`.
5. Verify a build execution exists.

**DB checks (examples):**
```sql
SELECT id, status, created_at FROM builds ORDER BY created_at DESC LIMIT 5;
SELECT id, build_id, status, created_at FROM build_executions ORDER BY created_at DESC LIMIT 5;
```

**Expected:** The latest build is `running` and has a corresponding execution row.

---

## Journey C: Concurrency Limit Enforcement

**Goal:** Confirm the dispatcher respects tenant concurrency limits.

1. Set the tenant build config `MaxConcurrentJobs = 1` via System Configuration.
2. Create 2 builds quickly for the same tenant.
3. Verify only one build transitions to `running`.
4. Confirm the other remains `queued`.
5. Check dispatcher metrics for `skipped_for_limit`.

**Expected:** Only one running build at a time for the tenant, and `skipped_for_limit` increments.

---

## Journey D: Dispatch Failure → Requeue

**Goal:** Ensure dispatcher requeues builds when dispatch fails.

1. Create a build with a valid config.
2. Temporarily simulate a dispatch failure (e.g., remove Tekton credentials if using Tekton).
3. Wait one dispatcher tick.
4. Verify build status returns to `queued`.
5. Check dispatcher metrics for `dispatch_errors` and `requeues`.

**Expected:** Build is requeued and error counters increment.

---

## UX Validation (Admin Panel)

1. Visit Admin Dashboard.
2. Confirm the Dispatcher Metrics panel shows values.
3. Refresh after running Journeys B–D.

**Expected:** Metrics panel reflects recent dispatcher activity.

---

## Troubleshooting

1. If builds stay `queued`, confirm dispatcher is running and `dispatcher.enabled = true`.
2. If metrics are empty, check `/api/v1/admin/dispatcher/metrics` permissions.
3. If dispatch fails, check build config validity and executor availability.

---

## Success Criteria

1. Metrics endpoint responds with valid data.
2. Queued builds transition to `running`.
3. Executions are created for dispatched builds.
4. Concurrency limits are enforced.
5. Dispatch failures requeue builds and increment metrics.
