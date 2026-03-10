# Roadmap

This roadmap reflects concrete gaps and planned improvements visible in the current open source codebase. It is intentionally conservative: items listed here are based on existing partial implementations, documented planned features, or clear platform extension points.

## Near Term

- Admin operations endpoints
  - Add backend support for system maintenance, system logs, and system health APIs that are already represented in the admin frontend.
- Scheduled builds and automation
  - Complete scheduled build workflows so recurring build execution becomes a first-class platform capability.
- Account security settings
  - Expand owner and administrator account controls with user-facing security settings, including MFA and related account-hardening flows.
- Notification preferences
  - Add user-configurable notification preferences for operational events and workflow outcomes.

## In Progress Areas

- Build control-plane completion
  - Complete the phase-two build control-plane handlers and replace scaffolded workflow paths with full execution handling.
- Provider-backed operational visibility
  - Improve provider and cluster operational visibility so readiness, health, and administrative operations reflect live backend state rather than partial or placeholder responses.
- Tenant-aware outbound email configuration
  - Replace default or placeholder outbound email assumptions with tenant-aware and environment-driven configuration.

## Future Direction

- Expanded non-Kubernetes runtime guidance
  - Improve support and documentation for running Image Factory outside Kubernetes using published binaries and dependency services.
- Stronger operator observability
  - Extend diagnostics, logs, and platform health reporting for administrators and operators.
- Policy-driven automation
  - Continue deepening quarantine, scanning, and policy evaluation workflows so security controls can be enforced more consistently across build and intake paths.

## Notes

- Deployment placeholders such as `__SET_BEFORE_DEPLOYMENT__` are not roadmap items. They are intentional publish-safe defaults.
- Demo seed data and development scripts are also not roadmap items. They exist to support local setup and testing.
- This roadmap is directional, not a delivery schedule.
