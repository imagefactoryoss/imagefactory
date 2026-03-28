import { adminService } from '@/services/adminService'
import React from 'react'

type FeatureCategory = {
    title: string
    status: 'available' | 'in_progress' | 'planned'
    capabilities: string[]
}

type FeatureFilter = 'all' | FeatureCategory['status']

type RoadmapItem = {
    title: string
    priority: 'P0' | 'P1' | 'P2'
    target: string
    items: string[]
}

const statusTone: Record<FeatureCategory['status'], string> = {
    available: 'border-emerald-300 bg-emerald-50 text-emerald-900 dark:border-emerald-800/60 dark:bg-emerald-950/30 dark:text-emerald-200',
    in_progress: 'border-amber-300 bg-amber-50 text-amber-900 dark:border-amber-800/60 dark:bg-amber-950/30 dark:text-amber-200',
    planned: 'border-slate-300 bg-slate-50 text-slate-800 dark:border-slate-700 dark:bg-slate-900/70 dark:text-slate-200',
}

const categories: FeatureCategory[] = [
    {
        title: 'Access, Identity, and Multi-Tenant Control',
        status: 'available',
        capabilities: [
            'Tenant-scoped RBAC with role-based navigation and action gating',
            'Owner/Admin, Dev, Operator, Viewer, and System Admin role model',
            'Tenant, member, and invitation management UX with entitlement checks',
        ],
    },
    {
        title: 'Build and Delivery Platform',
        status: 'available',
        capabilities: [
            'End-to-end build lifecycle: queued, running, logs, artifacts, cancel/restart',
            'Dispatcher-driven orchestration and provider-aware execution routing',
            'Project source management, source-aware builds, and webhook trigger flows',
            'Build-as-code diagnostics surfaced in traces and operator-facing troubleshooting UX',
        ],
    },
    {
        title: 'Provider and Infrastructure Operations',
        status: 'available',
        capabilities: [
            'Infrastructure provider onboarding with readiness gates and blocker reasons',
            'Managed-provider tenant namespace deprovision operation with safety checks',
            'Tenant build capability entitlements enforced at admission, retry, and dispatch stages',
        ],
    },
    {
        title: 'Image Catalog and Security Evidence',
        status: 'available',
        capabilities: [
            'Image catalog with details, versions, tags, and metadata views',
            'Layer-level security evidence with SBOM and vulnerability mapping',
            'Execution evidence capture from build output into catalog and audit stores',
            'Project and build workflows consume released artifacts through governed handoff paths',
        ],
    },
    {
        title: 'Notification and Realtime Operations',
        status: 'available',
        capabilities: [
            'In-app notification center with websocket-driven refresh',
            'Dedicated notification management page with selective and bulk cleanup',
            'Notification replay and delivery observability for build-related events',
        ],
    },
    {
        title: 'Quarantine Governance and Release Flow',
        status: 'available',
        capabilities: [
            'Quarantine request intake, reviewer workbench, decisions, and timeline tracking',
            'Governed release pipeline with deterministic deny contracts and audit trail',
            'Tenant release consumption handoff into build workflows',
            'Lifecycle hardening across reviewer queue observability, dispatch resilience, and pipeline readiness states',
        ],
    },
    {
        title: 'Admin Governance and Access Management',
        status: 'available',
        capabilities: [
            'Operational capability entitlement management in dedicated admin UX',
            'Role/permission definition and assignment workflows with route-level enforcement',
            'Audit logs and admin-system controls for secure operational governance',
        ],
    },
    {
        title: 'SRE Smart Bot',
        status: 'in_progress',
        capabilities: [
            'Implemented baseline: incident ledger, approvals, detector rules, and settings UX',
            'Implemented signals: runtime, logs, HTTP golden signals, async backlog, transport pressure',
            'Current epic: guided remediation packs with dry-run and guarded execution',
            'Read-only MCP tool families and deterministic AI draft/interpretation operator workflows',
        ],
    },
    {
        title: 'Air-Gapped and Runtime Infrastructure Controls',
        status: 'in_progress',
        capabilities: [
            'Tekton task image override management (admin API and UI scaffold in place)',
            'Runtime storage profile controls for bootstrap/runtime assets',
            'Internal registry temp-image GC worker and health visibility workstream',
            'On-demand external scan request workflow separation and dedicated tenant/admin UX',
        ],
    },
    {
        title: 'Release Governance Telemetry (Phase F/L Track)',
        status: 'in_progress',
        capabilities: [
            'Release telemetry counters, thresholds, and alert pathways are operationalized in slices',
            'Runtime compliance and trust enforcement expansion is active for drift triage completion',
            'Validation runners and runbook artifacts are being used as release readiness gates',
        ],
    },
    {
        title: 'Next Planned Productization',
        status: 'planned',
        capabilities: [
            'Build policy rules productization with strict admission and simulation contracts',
            'Advanced admin build management analytics and queue/infrastructure controls',
            'Multi-tenant explicit context switching enhancement model',
            'Build-node provider provisioning parity and deeper execution-control UX',
        ],
    },
]

const roadmapNow: RoadmapItem[] = [
    {
        title: 'SRE Smart Bot Phase 7',
        priority: 'P0',
        target: 'Current sprint',
        items: [
            'Close guided remediation pack dry-run and approval-gated execute loop in incident workspace',
            'Complete operator UX confidence with automation coverage and evidence linkage',
        ],
    },
    {
        title: 'Quarantine Phase L',
        priority: 'P0',
        target: 'Current sprint',
        items: [
            'Finish drift triage/remediation UX surfaces for tenant and admin operators',
            'Connect compliance metrics and alerts into dashboard slices for operational closure',
        ],
    },
    {
        title: 'Air-Gapped Runtime Readiness',
        priority: 'P1',
        target: 'Current sprint',
        items: [
            'Complete staging validation for Tekton task image overrides across build methods',
            'Advance runtime storage profiles and internal registry GC worker integration',
        ],
    },
]

const roadmapNext: RoadmapItem[] = [
    {
        title: 'Build Policy Rules Productization (BP-1)',
        priority: 'P1',
        target: 'Next delivery window',
        items: [
            'Harden typed policy schema/validation and deterministic deny contracts',
            'Add policy simulation endpoint and authoring/audit UX improvements',
        ],
    },
    {
        title: 'SRE Smart Bot Consumer Pressure Expansion',
        priority: 'P1',
        target: 'Next delivery window',
        items: [
            'Introduce NATS consumer lag and stalled-progress signal modeling',
            'Correlate consumer pressure with backlog and transport instability in drafts',
        ],
    },
]

const roadmapLater: RoadmapItem[] = [
    {
        title: 'Advanced Admin Build Management',
        priority: 'P2',
        target: 'Future roadmap',
        items: [
            'Expanded queue operations, analytics, and infrastructure operations cockpit',
            'Proactive alerting and event-log productization beyond baseline observability',
        ],
    },
    {
        title: 'Multi-Tenant Context Switching Enhancements',
        priority: 'P2',
        target: 'Future roadmap',
        items: [
            'Explicit request-context tenant switching model for multi-tenant users',
            'UI and API flow hardening for context clarity and cross-tenant safety',
        ],
    },
]

const priorityTone: Record<RoadmapItem['priority'], string> = {
    P0: 'border-rose-300 bg-rose-50 text-rose-900 dark:border-rose-800/60 dark:bg-rose-950/30 dark:text-rose-200',
    P1: 'border-amber-300 bg-amber-50 text-amber-900 dark:border-amber-800/60 dark:bg-amber-950/30 dark:text-amber-200',
    P2: 'border-slate-300 bg-slate-50 text-slate-800 dark:border-slate-700 dark:bg-slate-900/70 dark:text-slate-200',
}

const ProductInfoPage: React.FC = () => {
    const appVersion = import.meta.env.VITE_APP_VERSION || 'dev'
    const buildDate = import.meta.env.VITE_APP_BUILD_DATE || 'unknown'
    const envBacklogSync = import.meta.env.VITE_PRODUCT_INFO_LAST_SYNC || buildDate
    const [lastBacklogSync, setLastBacklogSync] = React.useState(envBacklogSync)
    const [featureFilter, setFeatureFilter] = React.useState<FeatureFilter>('all')

    React.useEffect(() => {
        let isCancelled = false
        const loadRuntimeMetadata = async () => {
            try {
                const metadata = await adminService.getProductInfoMetadata()
                const runtimeValue = metadata?.last_backlog_sync?.trim()
                if (!isCancelled && runtimeValue) {
                    setLastBacklogSync(runtimeValue)
                }
            } catch {
                // Admin-only endpoint; keep fallback metadata for non-admin contexts.
            }
        }
        void loadRuntimeMetadata()

        return () => {
            isCancelled = true
        }
    }, [envBacklogSync])

    const filteredCategories = categories.filter((category) => featureFilter === 'all' || category.status === featureFilter)
    const filterButtons: Array<{ label: string; value: FeatureFilter }> = [
        { label: 'All', value: 'all' },
        { label: 'Available', value: 'available' },
        { label: 'In Progress', value: 'in_progress' },
        { label: 'Planned', value: 'planned' },
    ]
    return (
        <div className="space-y-6 px-4 py-6 sm:px-6 lg:px-8">
            <section className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm dark:border-slate-800 dark:bg-slate-900">
                <div className="flex flex-wrap items-center justify-between gap-3">
                    <div>
                        <h1 className="text-2xl font-semibold text-slate-900 dark:text-white">Product Information</h1>
                        <p className="mt-2 text-sm text-slate-600 dark:text-slate-400">
                            Product capability snapshot sourced from implementation backlog and planning artifacts.
                        </p>
                    </div>
                    <div className="rounded-xl border border-sky-300 bg-sky-50 px-4 py-2 text-sm font-medium text-sky-900 dark:border-sky-800 dark:bg-sky-950/40 dark:text-sky-200">
                        Version {appVersion}
                    </div>
                </div>
                <div className="mt-4 grid gap-3 text-xs text-slate-600 dark:text-slate-400 sm:grid-cols-4">
                    <div className="rounded-lg border border-slate-200 bg-slate-50 px-3 py-2 dark:border-slate-700 dark:bg-slate-800/60">
                        <span className="font-semibold text-slate-800 dark:text-slate-200">Build date:</span> {buildDate}
                    </div>
                    <div className="rounded-lg border border-slate-200 bg-slate-50 px-3 py-2 dark:border-slate-700 dark:bg-slate-800/60">
                        <span className="font-semibold text-slate-800 dark:text-slate-200">Track:</span> Alpha / active delivery
                    </div>
                    <div className="rounded-lg border border-slate-200 bg-slate-50 px-3 py-2 dark:border-slate-700 dark:bg-slate-800/60">
                        <span className="font-semibold text-slate-800 dark:text-slate-200">Updated from:</span> engineering backlog + plans
                    </div>
                    <div className="rounded-lg border border-slate-200 bg-slate-50 px-3 py-2 dark:border-slate-700 dark:bg-slate-800/60">
                        <span className="font-semibold text-slate-800 dark:text-slate-200">Last backlog sync:</span> {lastBacklogSync}
                    </div>
                </div>
            </section>

            <section className="space-y-4">
                <div className="rounded-2xl border border-slate-200 bg-white p-4 shadow-sm dark:border-slate-800 dark:bg-slate-900">
                    <div className="flex flex-wrap items-center justify-between gap-3">
                        <div>
                            <h2 className="text-base font-semibold text-slate-900 dark:text-slate-100">Feature Coverage</h2>
                            <p className="text-xs text-slate-600 dark:text-slate-400">Filter by delivery state.</p>
                        </div>
                        <div className="flex flex-wrap gap-2">
                            {filterButtons.map((button) => {
                                const active = featureFilter === button.value
                                return (
                                    <button
                                        key={button.value}
                                        type="button"
                                        onClick={() => setFeatureFilter(button.value)}
                                        className={`rounded-full border px-3 py-1.5 text-xs font-semibold transition ${active
                                            ? 'border-sky-500 bg-sky-100 text-sky-900 dark:border-sky-500 dark:bg-sky-900/50 dark:text-sky-200'
                                            : 'border-slate-300 bg-slate-50 text-slate-700 hover:border-slate-400 dark:border-slate-700 dark:bg-slate-800 dark:text-slate-300 dark:hover:border-slate-600'
                                            }`}
                                    >
                                        {button.label}
                                    </button>
                                )
                            })}
                        </div>
                    </div>
                </div>

                {filteredCategories.map((category) => (
                    <article key={category.title} className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm dark:border-slate-800 dark:bg-slate-900">
                        <div className="mb-3 flex items-center justify-between gap-2">
                            <h2 className="text-lg font-semibold text-slate-900 dark:text-slate-100">{category.title}</h2>
                            <span className={`rounded-full border px-2.5 py-1 text-xs font-semibold uppercase tracking-wide ${statusTone[category.status]}`}>
                                {category.status.replace('_', ' ')}
                            </span>
                        </div>
                        <ul className="space-y-2 text-sm text-slate-700 dark:text-slate-300">
                            {category.capabilities.map((capability) => (
                                <li key={capability} className="rounded-lg border border-slate-200 bg-slate-50 px-3 py-2 dark:border-slate-700 dark:bg-slate-800/60">
                                    {capability}
                                </li>
                            ))}
                        </ul>
                    </article>
                ))}
            </section>

            <section className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm dark:border-slate-800 dark:bg-slate-900">
                <h2 className="text-xl font-semibold text-slate-900 dark:text-slate-100">Roadmap Snapshot</h2>
                <p className="mt-2 text-sm text-slate-600 dark:text-slate-400">
                    Delivery sequencing based on active engineering backlog priorities and current implementation plans.
                </p>

                <div className="mt-4 grid gap-4 lg:grid-cols-3">
                    {[
                        { title: 'Now', subtitle: 'Active implementation', data: roadmapNow },
                        { title: 'Next', subtitle: 'Next delivery window', data: roadmapNext },
                        { title: 'Later', subtitle: 'Planned expansion', data: roadmapLater },
                    ].map((column) => (
                        <div key={column.title} className="rounded-xl border border-slate-200 bg-slate-50 p-4 dark:border-slate-700 dark:bg-slate-800/50">
                            <div className="mb-3">
                                <h3 className="text-base font-semibold text-slate-900 dark:text-slate-100">{column.title}</h3>
                                <p className="text-xs text-slate-600 dark:text-slate-400">{column.subtitle}</p>
                            </div>
                            <div className="space-y-3">
                                {column.data.map((item) => (
                                    <article key={item.title} className="rounded-lg border border-slate-200 bg-white p-3 dark:border-slate-700 dark:bg-slate-900/60">
                                        <div className="mb-2 flex items-start justify-between gap-2">
                                            <div>
                                                <h4 className="text-sm font-semibold text-slate-900 dark:text-slate-100">{item.title}</h4>
                                                <p className="text-xs text-slate-500 dark:text-slate-400">{item.target}</p>
                                            </div>
                                            <span className={`rounded-full border px-2 py-0.5 text-[11px] font-semibold ${priorityTone[item.priority]}`}>
                                                {item.priority}
                                            </span>
                                        </div>
                                        <ul className="space-y-1 text-xs text-slate-700 dark:text-slate-300">
                                            {item.items.map((entry) => (
                                                <li key={entry} className="rounded-md border border-slate-200 bg-slate-50 px-2 py-1 dark:border-slate-700 dark:bg-slate-800/60">
                                                    {entry}
                                                </li>
                                            ))}
                                        </ul>
                                    </article>
                                ))}
                            </div>
                        </div>
                    ))}
                </div>
            </section>
        </div>
    )
}

export default ProductInfoPage
