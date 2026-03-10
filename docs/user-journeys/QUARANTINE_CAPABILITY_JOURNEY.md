# Quarantine Capability Journey

This document describes the intended user experience for capability-gated quarantine workflows. It is forward-looking reference material for a capability-first quarantine model.

## 1. Goal

Ensure the platform is capability-first:

- Admins choose tenant entitlements at onboarding.
- Tenant users only see capabilities they are entitled to after login.
- Quarantine workflows are available only when entitlement + prerequisite checks pass.

---

## 2. Capability Model (v1)

The following capabilities are tenant-scoped and independently toggleable:

- `build` (image build)
- `quarantine_request` (tenant trigger for quarantine request -> import/scan pipeline)
- `ondemand_image_scanning` (manual scan trigger for an image)

Optional related capability for later phases:

- `quarantine_release` (admin capability to release quarantined image to approved channel)

Default recommendation:

- `build`: enabled by policy during onboarding as needed
- `quarantine_request`: disabled by default unless explicitly approved
- `ondemand_image_scanning`: disabled by default unless explicitly approved

---

## 3. Primary UX Rules

1. Visibility follows entitlement.
2. API remains authoritative (UI hide + backend enforce).
3. Denials are actionable, not generic.
4. No side effects on denied requests (for example no approval workflow creation).
5. Login/home experience is deterministic per tenant context.

---

## 4. Journey A: Admin Onboards Tenant With Entitlements

### Trigger
- Admin creates a new tenant or updates a tenant profile.

### Steps
1. Admin opens tenant onboarding/edit wizard.
2. Admin sees a capability entitlement section:
   - Build
   - Quarantine Request
   - On-Demand Image Scanning
3. Admin selects entitlements for that tenant.
4. Admin saves with change reason.
5. System persists tenant capability policy and emits audit event.

### UX Requirements
- Clear labels and business descriptions for each capability.
- Explicit warning for high-control capabilities (`quarantine_request`).
- Confirmation summary before save.

### Success Criteria
- Tenant capability set is persisted and visible in admin tenant details.
- Audit trail includes actor, tenant, before/after values, timestamp.

---

## 5. Journey B: Tenant Group Login and Capability-Scoped Experience

### Trigger
- A tenant user logs in with a selected tenant context.

### Steps
1. User authenticates and selects tenant context.
2. Frontend requests effective tenant capabilities.
3. Navigation and dashboards render only entitled features.
4. Hidden capabilities are not shown in nav, quick actions, or CTA cards.

### Example Outcomes
- Tenant with `build=true`, `quarantine_request=false`, `ondemand_image_scanning=false`:
  - Build pages visible.
  - Quarantine request UI hidden/disabled with “Contact administrator” guidance.
  - On-demand scan action not available.
- Tenant with all 3 enabled:
  - Build, quarantine request, and manual scan actions visible.

### UX Requirements
- No dead links to disabled capabilities.
- Optional “Feature not enabled” informational state in settings/help panel.

---

## 6. Journey C: Quarantine Request (Entitled Path)

### Preconditions
- `quarantine_request=true` for tenant.
- EPR registration prerequisite satisfied.

### Steps
1. User opens “Import/Quarantine Image” flow.
2. User enters source registry/image and SOR record ID.
3. System validates SOR record.
4. Request is created in `pending`.
5. Approval workflow starts.
6. On approval, import pipeline runs (pull -> scan -> SBOM -> policy evaluate).
7. Result surfaces as `success`, `quarantined`, or `failed`.

### UX States
- `pending_approval`
- `importing`
- `catalog_sync_pending` (retryable)
- `completed` (success/quarantined)
- `failed`

### Success Criteria
- Tenant can track request status with clear state labels and timestamps.
- Evidence and logs are discoverable from image/build context.

---

## 7. Journey D: Denied Paths

### D1. Capability Denied
- Condition: `quarantine_request=false`
- API response: `403 tenant_capability_not_entitled`
- UX message:
  - “Quarantine request is not enabled for this tenant.”
  - “Contact your administrator to request access.”

### D2. SOR Prerequisite Denied
- Condition: missing/invalid EPR registration
- API response: `412 sor_registration_required`
- UX message:
  - “Enterprise EPR registration is required before requesting quarantine.”
  - Include failed `sor_record_id` and remediation guidance.

### Denied Flow Guarantees
- No approval ticket created.
- No runtime dispatch created.
- Denial event auditable and metered.

---

## 8. Navigation and Page-Level Behavior

### Tenant Navigation
- Show menu item only when capability is enabled:
  - Build: show/hide build menu.
  - Quarantine Request: show/hide import/quarantine menu and CTAs.
  - On-Demand Scanning: show/hide manual scan actions in image detail pages.

### Page Guarding
- Direct URL access to disabled capability pages returns deterministic empty/denied state and links back to available features.

---

## 9. Telemetry and Audit

Track at minimum:

- Capability assignment change events (admin action).
- Capability-denied attempt counts by tenant and capability key.
- SOR-denied attempt counts.
- Quarantine request funnel: submitted -> approved -> importing -> terminal.

---

## 10. Implementation Checklist (UX-Focused)

- [ ] Admin tenant onboarding/edit: capability entitlement controls.
- [ ] Tenant capability read endpoint wired to session bootstrap.
- [ ] Capability-aware nav rendering and route guards.
- [ ] Quarantine request form with SOR validation UX.
- [ ] Denial error mapping for `tenant_capability_not_entitled` and `sor_registration_required`.
- [ ] On-demand scanning action guarded by `ondemand_image_scanning`.
- [ ] Audit + metrics instrumentation for denied and successful journeys.

---

## 11. Open Decisions

1. Should `build` be enabled by default for all new tenants or only selected tenant groups?
2. Should disabled capabilities be fully hidden or shown as locked with explanatory text?
3. Should `ondemand_image_scanning` be project-level override in addition to tenant-level entitlement?
