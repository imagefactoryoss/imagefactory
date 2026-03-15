# Robot SRE Log Intelligence And Incident Detection

## Purpose

Define how logs should be ingested, analyzed, and turned into structured incident signals for the Robot SRE / Ops Persona.

This document answers three questions:

1. How should Image Factory ingest logs on a very small OKE cluster?
2. How should NATS fit into the design?
3. How should log analytics feed Robot SRE without making remediation unsafe?

## Executive Recommendation

For the current cluster footprint, the best starting design is:

- Grafana Loki in monolithic mode
- Grafana Alloy as the log collector
- NATS JetStream for structured findings, incident events, approvals, and remediation workflow messages
- Robot SRE consuming both:
  - direct cluster/runtime signals
  - structured log findings

### Recommended v1 architecture

- `Alloy` collects pod logs and Kubernetes events
- `Loki` stores and indexes logs
- `Log detector` service evaluates rules and anomalies
- `NATS JetStream` carries normalized findings and workflow events
- `Robot SRE` consumes findings and decides whether to:
  - notify
  - correlate
  - request approval
  - execute bounded remediation

### What not to do in v1

- do not stream raw logs through NATS
- do not deploy distributed Loki on this cluster
- do not let an LLM consume raw log firehoses directly and autonomously execute actions

## Current Cluster Constraints

The current remote OKE worker capacity is small:

- 2 worker nodes
- allocatable CPU per node: about `1830m`
- allocatable memory per node: about `9.6 Gi`
- allocatable ephemeral storage per node: about `34.2 GB`

This is enough for a modest observability footprint, but not enough for a heavy self-hosted logging platform.

Design implications:

- keep ingestion lightweight
- keep retention short
- avoid running distributed read/write/backend observability stacks
- prefer low-cardinality labels
- prefer structured findings over shipping raw logs into workflow systems

## Why Loki Fits Best Here

Loki is the best fit for this cluster because:

- it is designed for logs
- it can run in monolithic mode for small deployments
- it works well with Kubernetes log collection
- it avoids the heavier storage and indexing footprint of Elasticsearch/OpenSearch style stacks
- Grafana Alloy integrates cleanly with Kubernetes and Loki

## Why Monolithic Loki

Monolithic Loki is the right fit for this environment because:

- it is specifically recommended by Grafana for small deployments / meta-monitoring
- it is much simpler to operate than scalable or microservices mode
- it keeps the footprint manageable for your free-tier cluster

Simple scalable Loki is a bad fit here because:

- it is heavier
- the Helm chart defaults assume a much larger footprint
- it introduces more moving parts than this cluster can comfortably absorb

## Why NATS Still Matters

NATS is still a great fit, but for control-plane events, not bulk log transport.

Use NATS JetStream for:

- normalized incident findings
- detection events
- incident lifecycle events
- approval requests and responses
- remediation action requests
- remediation action outcomes
- operator conversation state pointers if useful

Do not use NATS JetStream in v1 for:

- raw pod logs
- high-volume full-text log fanout

Reason:

- raw logs are high-volume and bursty
- JetStream retention and consumers are excellent for workflow/event streams, but using them as the primary raw log store will create unnecessary storage and operational pressure

## Proposed Architecture

## 1. Log Collection Layer

### Collector: Grafana Alloy

Use Alloy to collect:

- Kubernetes pod logs
- Kubernetes events
- optionally selected system logs later

### Recommended collection mode for v1

Start with Kubernetes API-based pod log collection:

- `loki.source.kubernetes`
- `loki.source.kubernetes_events`

Why:

- no privileged container required
- no host filesystem mount required
- no root requirement
- no DaemonSet requirement

Tradeoff:

- more API and kubelet traffic than file tailing

For this small cluster, that tradeoff is acceptable and simpler operationally.

### Optional v2 collection mode

If later you need:

- node logs
- lower kubelet/API overhead
- more complete infrastructure log coverage

then add a DaemonSet-based Alloy profile using file tails on node log paths.

## 2. Log Storage Layer

### Store: Loki monolithic

Recommended v1 properties:

- single replica
- conservative resource requests/limits
- short retention
- object storage only if already available and justified

### Storage recommendation

For this environment, start with small local persistence or hostPath only if you accept that log history is noncritical.

Better medium-term option:

- back Loki with MinIO only if the extra footprint is acceptable

Because this cluster is resource-constrained, I would keep the first version simple:

- short retention
- low-cost local persistence
- logs treated as operational telemetry, not compliance evidence

## 3. Log Intelligence Layer

Introduce a `log-detector` component that consumes from Loki queries rather than raw container streams.

Responsibilities:

- periodic rule-based scans
- burst / rate detection
- known-pattern matching
- anomaly grouping
- emit structured findings

### Detector output model

Each finding should look like:

- `finding_id`
- `source=logs`
- `domain`
- `incident_type`
- `severity`
- `confidence`
- `summary`
- `evidence`
- `resource_scope`
- `dedupe_key`
- `observed_at`

For the current backend integration, detectors should publish normalized findings using the event type:

- `sre.detector.finding.observed`
- `sre.detector.finding.recovered`

## 4. Event Backbone

### Backbone: NATS JetStream

Recommended streams:

- `ops.findings`
- `ops.incidents`
- `ops.approvals`
- `ops.remediations`
- `ops.chatops`

### Recommended retention usage

- `ops.findings`: limits retention, short age, bounded size
- `ops.incidents`: limits retention, longer age
- `ops.approvals`: limits retention
- `ops.remediations`: limits retention, longer age for audit convenience

NATS should be the message bus for structured control-plane events, not the long-term store of raw log lines.

## 5. Robot SRE Consumption Layer

Robot SRE should consume:

- Kubernetes and OCI signals directly
- app runtime health directly
- `ops.findings` from the log detector

Then it should:

- correlate findings with other signals
- open or update incidents
- consult policy
- notify operators
- request approval
- execute bounded remediation

## Log Analytics Model

## Rule Classes

### A. Signature Rules

Known error patterns with high operational value.

Examples:

- `toomanyrequests`
- `ImagePullBackOff`
- `FreeDiskSpaceFailed`
- `EvictionThresholdMet`
- `nats: no servers available for connection`
- `LDAP Result Code 200`
- `dial tcp ... i/o timeout`
- `manifest unknown`
- `x509`

These should be the MVP.

### B. Rate / Spike Rules

Pattern counts or changes over time.

Examples:

- login failures > threshold in 5m
- backend 5xx spike
- repeated crash-loop stack traces
- sudden increase in image pull errors

### C. Correlation Rules

Combine multiple signal classes.

Examples:

- LDAP timeout logs + reachable GLAuth service + stale stored LDAP host
- NATS connection failures + NATS pod unavailable
- disk pressure + completed pod buildup + image pull retries

### D. LLM-Assisted Summaries

Use an LLM after detection to:

- summarize clustered log evidence
- explain likely root cause
- draft operator messages

Do not use LLM inference alone as the trigger for disruptive remediation.

## Recommended MVP Detection Rules

### Infrastructure

- disk pressure signature detection
- kubelet storage eviction signature detection

### Runtime Services

- Redis/NATS/MinIO/registry/GLAuth connection failures
- dependency health endpoint failures

### Application Services

- backend error spike
- frontend/API login failure spike
- dispatcher crash-loop with repeated same root cause

### Identity / Security

- LDAP bind/search timeout spike
- stale auth-provider host mismatch

### Release / Configuration

- image pull forbidden / manifest unknown
- release reconcile conflicts

## Labeling And Cardinality Guidance

Keep Loki labels intentionally small.

Good labels:

- `namespace`
- `app`
- `component`
- `container`
- `pod` only if needed for short retention troubleshooting
- `cluster`
- `environment`

Avoid high-cardinality labels for:

- request id
- user id
- incident id
- stack trace fragments

Those should stay in log payload, not labels.

## Retention Guidance

For this cluster, keep retention modest.

Suggested starting point:

- 3 to 7 days in-cluster

That is enough for:

- active troubleshooting
- incident correlation
- rule tuning

If you later need long-term retention:

- archive structured incidents and remediation records in app storage
- optionally push logs to an external Loki/Grafana Cloud later

## Resource Guidance

For this cluster size, prefer:

- Loki monolithic single replica
- Alloy single deployment initially
- small CPU/memory requests
- bounded retention

Avoid in v1:

- distributed Loki
- full-text heavy analytics stack
- multi-replica observability control plane
- shipping every raw log twice

## Recommended Ingestion Strategy

## Best v1 choice

### `Alloy -> Loki`

This should be the default ingestion path.

Why:

- simplest
- low moving-part count
- native Kubernetes support
- easy path to later Grafana dashboards

## Best use of NATS

### `Detector -> NATS JetStream -> Robot SRE`

This should be the control/event path.

Why:

- durable workflow events
- replayable findings
- decouples detection from remediation
- fits your existing platform direction

## Not recommended in v1

### `Apps -> NATS -> log processor -> store raw logs`

Why not:

- too much event volume
- more custom code
- more retention complexity
- less value than using Loki directly

## Operator-Defined Log Rules

The admin UI should eventually let operators add log-driven rules.

### Allowed customizations

- match patterns
- threshold windows
- severity mapping
- routing
- suppression windows
- correlation hints

### Disallowed customizations

- arbitrary code execution
- direct raw shell actions
- changing destructive policies into auto-remediation

### Example custom rule

- name: `LDAP timeout spike`
- source: `logs`
- selector: `namespace=image-factory, component=backend`
- match: `LDAP Result Code 200` OR `failed to connect to LDAP`
- threshold: `>= 5 in 10m`
- severity: `warning`
- emit incident type: `identity_provider_unreachable`

## Data Flow

1. Pod emits log line.
2. Alloy collects log.
3. Alloy pushes to Loki.
4. Detector queries recent Loki windows.
5. Detector emits structured finding to NATS JetStream.
6. Robot SRE consumes finding.
7. Robot correlates with runtime/Kubernetes/OCI state.
8. Policy engine determines:
   - ignore
   - notify
   - ask approval
   - execute bounded action
9. Outcome is published back to NATS and persisted in incident/action tables.

## Suggested MVP Components

### In-Cluster

- `loki`
- `alloy`
- `robot-sre-detector`
- existing `nats`
- existing backend-based Robot SRE incident engine

### Existing Systems Reused

- Image Factory notifications
- process health store
- system config
- admin APIs and UI

## Suggested Rollout Plan

## Phase 1

- deploy Loki monolithic
- deploy Alloy for pod logs + k8s events
- create 10-15 high-value signature rules
- emit findings into NATS
- display findings in admin UI

## Phase 2

- correlate findings into incidents
- add Telegram operator alerts
- add approval workflow

## Phase 3

- add anomaly/spike detection
- add operator-defined log rules
- add LLM-generated incident explanations

## Phase 4

- consider external Loki/Grafana Cloud if retention or query needs outgrow cluster
- consider DaemonSet Alloy for node/system logs

## Recommendation Summary

For your current cluster:

- use `Loki` for raw log ingestion and storage
- use `Alloy` to collect logs
- use `NATS JetStream` for structured findings and workflow events
- use Robot SRE policy engine as the only authority for remediation

This gives you:

- lightweight ingestion
- searchable logs
- event-driven automation
- a clean path to agent-assisted incident reasoning

without overloading the small free-tier cluster.
