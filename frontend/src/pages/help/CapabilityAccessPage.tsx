import Drawer from '@/components/ui/Drawer'
import MermaidSequencePreview from '@/components/common/MermaidSequencePreview'
import { useCapabilitySurfacesStore } from '@/store/capabilitySurfaces'
import { ChevronRight } from 'lucide-react'
import React from 'react'

type CapabilityDetail = {
  name: string
  description: string
  flow: string[]
  prerequisites: string[]
  deniedBehavior: string[]
  mermaid: string
}

const labels: Record<string, CapabilityDetail> = {
  build: {
    name: 'Image Build',
    description: 'Create and manage projects and builds.',
    flow: [
      'Create or select a project.',
      'Create build configuration and trigger builds.',
      'Track execution status and logs.',
    ],
    prerequisites: [
      'Tenant must be entitled for Image Build.',
      'User still needs RBAC permissions for projects/builds.',
    ],
    deniedBehavior: [
      'Build/project routes are hidden or denied.',
      'Dashboard build/project metrics are replaced with entitlement guidance.',
    ],
    mermaid: `sequenceDiagram
    autonumber
    actor Tenant as Tenant User
    participant UI as Tenant UI
    participant API as Backend API
    participant Build as Build Service
    Tenant->>UI: Open Projects/Builds
    UI->>API: GET /settings/capability-surfaces
    API-->>UI: build=true
    Tenant->>UI: Trigger build
    UI->>API: POST /builds
    API->>Build: Start build workflow
    Build-->>API: Build status/log updates
    API-->>UI: Build accepted + status stream`,
  },
  quarantine_request: {
    name: 'Quarantine Request',
    description: 'Submit quarantine requests that trigger import/scan pipeline.',
    flow: [
      'Submit source image + EPR record in Quarantine Requests workspace.',
      'Approval workflow runs where required.',
      'Import/scan pipeline runs and lands in success/quarantined/failed outcome.',
    ],
    prerequisites: [
      'Tenant entitlement: Quarantine Request enabled.',
      'Valid EPR registration is required for create/retry admission.',
    ],
    deniedBehavior: [
      'API returns 403 tenant_capability_not_entitled when capability is disabled.',
      'API returns 412 epr_registration_required when EPR prereq is missing/invalid.',
    ],
    mermaid: `sequenceDiagram
    autonumber
    actor Tenant as Tenant User
    participant UI as Quarantine Requests UI
    participant API as Backend API
    participant EPR as EPR Validator
    participant Approval as Approval Workflow
    participant Pipeline as Import/Scan Pipeline
    Tenant->>UI: Submit quarantine request
    UI->>API: POST /images/import-requests
    API->>API: Check capability (quarantine_request)
    API->>EPR: Validate EPR record
    alt Capability or EPR denied
      API-->>UI: 403/412 denial with reason
    else Admitted
      API->>Approval: Create approval request
      Approval-->>API: Approved
      API->>Pipeline: Dispatch import/scan
      Pipeline-->>API: success/quarantined/failed
      API-->>UI: Updated request timeline/status
    end`,
  },
  quarantine_release: {
    name: 'Quarantine Release (Admin)',
    description: 'Admin-governed release of eligible quarantined images.',
    flow: [
      'Admin reviews eligible quarantined image.',
      'Admin executes release to tenant namespace/channel.',
    ],
    prerequisites: [
      'Admin capability entitlement must be enabled.',
      'Policy and workflow gates for release must be satisfied.',
    ],
    deniedBehavior: [
      'Release actions are hidden or denied when entitlement is disabled.',
    ],
    mermaid: `sequenceDiagram
    autonumber
    actor Admin as Admin User
    participant UI as Admin UI
    participant API as Backend API
    participant Policy as Policy Gate
    participant Registry as Registry
    Admin->>UI: Trigger quarantine release
    UI->>API: POST /quarantine/release
    API->>API: Check capability (quarantine_release)
    API->>Policy: Validate release policy gates
    alt Gate denied
      API-->>UI: Denied with reason
    else Gate passed
      API->>Registry: Promote image/tag to target namespace
      Registry-->>API: Release completed
      API-->>UI: Release success
    end`,
  },
  ondemand_image_scanning: {
    name: 'On-Demand Image Scanning',
    description: 'Run manual vulnerability scans from image security views.',
    flow: [
      'Open image details Security tab.',
      'Trigger a manual scan and review refreshed findings.',
    ],
    prerequisites: [
      'Tenant entitlement for on-demand scan must be enabled.',
    ],
    deniedBehavior: [
      'Scan CTA is hidden or disabled.',
      'Direct API trigger returns 403 tenant_capability_not_entitled.',
    ],
    mermaid: `sequenceDiagram
    autonumber
    actor Tenant as Tenant User
    participant UI as Image Detail Security Tab
    participant API as Backend API
    participant Scanner as Scan Runtime
    Tenant->>UI: Click Run Scan
    UI->>API: POST /images/{id}/scan
    API->>API: Check capability (ondemand_image_scanning)
    alt Not entitled
      API-->>UI: 403 tenant_capability_not_entitled
    else Entitled
      API->>Scanner: Start scan
      Scanner-->>API: Scan result
      API-->>UI: Updated vulnerabilities
    end`,
  },
}

const CapabilityAccessPage: React.FC = () => {
  const data = useCapabilitySurfacesStore((state) => state.data)
  const loadedTenantId = useCapabilitySurfacesStore((state) => state.loadedTenantId)
  const loading = useCapabilitySurfacesStore((state) => state.isLoading)
  const [selectedCapability, setSelectedCapability] = React.useState<string | null>(null)
  const [copyState, setCopyState] = React.useState<'idle' | 'copied' | 'error'>('idle')

  const capabilities = data.capabilities || {}
  const selectedDetail = selectedCapability ? labels[selectedCapability] : null

  const handleCopyMermaid = async () => {
    if (!selectedDetail?.mermaid) return
    try {
      await navigator.clipboard.writeText(selectedDetail.mermaid)
      setCopyState('copied')
      window.setTimeout(() => setCopyState('idle'), 1500)
    } catch {
      setCopyState('error')
      window.setTimeout(() => setCopyState('idle'), 2000)
    }
  }

  return (
    <div className="space-y-6 px-4 py-6 sm:px-6 lg:px-8">
      <div>
        <h1 className="text-2xl font-semibold text-slate-900 dark:text-white">Capability Access</h1>
        <p className="mt-2 text-sm text-slate-600 dark:text-slate-400">
          This page shows what your current tenant context is entitled to use.
        </p>
        <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
          {loading ? 'Loading tenant capability profile...' : `Tenant context: ${loadedTenantId || 'not selected'}`}
        </p>
      </div>

      <section className="rounded-xl border border-slate-200 bg-white p-4 dark:border-slate-700 dark:bg-slate-900">
        <h2 className="text-sm font-semibold uppercase tracking-wide text-slate-700 dark:text-slate-200">Operational Capabilities</h2>
        <div className="mt-3 grid grid-cols-1 gap-3 md:grid-cols-2">
          {Object.keys(capabilities).map((key) => {
            const enabled = Boolean((capabilities as unknown as Record<string, boolean>)[key])
            const entry = labels[key] || {
              name: key,
              description: 'Capability entitlement status for this tenant.',
              flow: ['No detailed flow is documented for this capability yet.'],
              prerequisites: ['Capability must be enabled for the tenant.'],
              deniedBehavior: ['Capability surfaces will be hidden or denied.'],
            }
            return (
              <button
                key={key}
                type="button"
                onClick={() => setSelectedCapability(key)}
                className={`w-full rounded-lg border p-3 text-left transition ${
                  selectedCapability === key
                    ? 'border-sky-400 bg-sky-50 dark:border-sky-600 dark:bg-sky-950/20'
                    : 'border-slate-200 hover:border-slate-300 hover:bg-slate-50 dark:border-slate-700 dark:hover:border-slate-600 dark:hover:bg-slate-800/60'
                }`}
              >
                <div className="flex items-start justify-between gap-3">
                  <div>
                    <p className="text-sm font-semibold text-slate-900 dark:text-white">{entry.name}</p>
                    <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">{entry.description}</p>
                  </div>
                  <div className="flex flex-col items-end gap-1">
                    <span
                      className={`rounded-full px-2 py-0.5 text-xs font-semibold ${
                        enabled
                          ? 'bg-emerald-100 text-emerald-800 dark:bg-emerald-900/30 dark:text-emerald-200'
                          : 'bg-amber-100 text-amber-800 dark:bg-amber-900/30 dark:text-amber-200'
                      }`}
                    >
                      {enabled ? 'Enabled' : 'Not entitled'}
                    </span>
                    <span className="inline-flex items-center gap-1 text-xs text-sky-700 dark:text-sky-300">
                      View details
                      <ChevronRight className="h-3 w-3" />
                    </span>
                  </div>
                </div>
              </button>
            )
          })}
        </div>

      </section>

      <section className="rounded-xl border border-indigo-200 bg-indigo-50/70 p-4 dark:border-indigo-800 dark:bg-indigo-950/20">
        <h2 className="text-sm font-semibold uppercase tracking-wide text-indigo-900 dark:text-indigo-200">Central Security Reviewer Workflow</h2>
        <p className="mt-2 text-xs text-indigo-900 dark:text-indigo-100">
          Central reviewers in the system group (<span className="font-semibold">security_reviewers</span>) should use
          the admin reviewer queue at <span className="font-semibold">/admin/quarantine/review</span> to process pending approvals.
        </p>
        <ul className="mt-2 list-disc space-y-1 pl-4 text-xs text-indigo-900 dark:text-indigo-100">
          <li>Required permissions: <span className="font-semibold">quarantine:read</span>, <span className="font-semibold">quarantine:approve</span>, <span className="font-semibold">quarantine:reject</span>.</li>
          <li>List/detail APIs and decision APIs enforce these permissions at route level.</li>
          <li>If access is denied, use this page and admin role/group assignments to verify reviewer setup.</li>
        </ul>
      </section>

      <Drawer
        isOpen={Boolean(selectedDetail)}
        onClose={() => setSelectedCapability(null)}
        title={selectedDetail ? `${selectedDetail.name}: Process Detail` : 'Capability Detail'}
        description={selectedDetail?.description}
        width="xl"
      >
        {selectedDetail ? (
          <div className="space-y-4">
            <div className="rounded-lg border border-slate-200 bg-slate-50 p-4 dark:border-slate-700 dark:bg-slate-800/40">
              <p className="text-xs font-semibold uppercase tracking-wide text-slate-700 dark:text-slate-200">Flow</p>
              <ul className="mt-2 list-disc space-y-1 pl-4 text-xs text-slate-600 dark:text-slate-300">
                {selectedDetail.flow.map((item) => (
                  <li key={item}>{item}</li>
                ))}
              </ul>
            </div>
            <div className="rounded-lg border border-slate-200 bg-slate-50 p-4 dark:border-slate-700 dark:bg-slate-800/40">
              <p className="text-xs font-semibold uppercase tracking-wide text-slate-700 dark:text-slate-200">Prerequisites</p>
              <ul className="mt-2 list-disc space-y-1 pl-4 text-xs text-slate-600 dark:text-slate-300">
                {selectedDetail.prerequisites.map((item) => (
                  <li key={item}>{item}</li>
                ))}
              </ul>
            </div>
            <div className="rounded-lg border border-slate-200 bg-slate-50 p-4 dark:border-slate-700 dark:bg-slate-800/40">
              <p className="text-xs font-semibold uppercase tracking-wide text-slate-700 dark:text-slate-200">When Disabled</p>
              <ul className="mt-2 list-disc space-y-1 pl-4 text-xs text-slate-600 dark:text-slate-300">
                {selectedDetail.deniedBehavior.map((item) => (
                  <li key={item}>{item}</li>
                ))}
              </ul>
            </div>
            <div className="rounded-lg border border-slate-200 bg-slate-50 p-4 dark:border-slate-700 dark:bg-slate-800/40">
              <div className="flex items-center justify-between gap-2">
                <p className="text-xs font-semibold uppercase tracking-wide text-slate-700 dark:text-slate-200">Sequence Diagram (Mermaid)</p>
                <button
                  type="button"
                  onClick={handleCopyMermaid}
                  className="rounded-md border border-slate-300 bg-white px-2.5 py-1 text-[11px] font-medium text-slate-700 hover:bg-slate-100 dark:border-slate-600 dark:bg-slate-900 dark:text-slate-200 dark:hover:bg-slate-800"
                >
                  {copyState === 'copied' ? 'Copied' : copyState === 'error' ? 'Copy failed' : 'Copy Mermaid'}
                </button>
              </div>
              <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
                Rendered preview below. You can still copy Mermaid source for docs/viewers.
              </p>
              <div className="mt-2">
                <MermaidSequencePreview chart={selectedDetail.mermaid} />
              </div>
            </div>
          </div>
        ) : null}
      </Drawer>
    </div>
  )
}

export default CapabilityAccessPage
