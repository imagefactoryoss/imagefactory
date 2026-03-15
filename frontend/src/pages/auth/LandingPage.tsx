import { useThemeStore } from '@/store/theme'
import { Bell, Boxes, Building2, ShieldCheck, Sparkles, Workflow } from 'lucide-react'
import { Moon, Sun } from 'lucide-react'
import React from 'react'
import { Link } from 'react-router-dom'

const capabilityCards = [
    {
        icon: Workflow,
        title: 'Build Orchestration',
        description: 'Run and monitor Kaniko, Buildx, Docker, and Packer pipelines with structured execution visibility.',
    },
    {
        icon: Boxes,
        title: 'Image Evidence',
        description: 'Capture and review SBOM, vulnerability, and artifact evidence tied to build and image history.',
    },
    {
        icon: ShieldCheck,
        title: 'Quarantine + Scanning',
        description: 'Control external image admission and request asynchronous on-demand scans with traceable outcomes.',
    },
    {
        icon: Building2,
        title: 'Tenant Isolation',
        description: 'Separate projects, credentials, members, and capability entitlements per tenant context.',
    },
    {
        icon: Bell,
        title: 'Realtime Notifications',
        description: 'Track build and workflow state changes with websocket-driven notification updates.',
    },
    {
        icon: Sparkles,
        title: 'SRE Smart Bot',
        description: 'Learn from logs and golden signals, detect incidents early, guide remediation, and notify operators with evidence.',
    },
]

const srePillars = [
    {
        title: 'Learn',
        description: 'Observes incidents, logs, golden signals, and operator actions to improve detector coverage over time.',
    },
    {
        title: 'Detect',
        description: 'Correlates runtime health, Loki evidence, HTTP trends, backlog pressure, and messaging transport instability.',
    },
    {
        title: 'Remediate',
        description: 'Suggests or executes bounded actions with approvals, policy controls, and an auditable incident ledger.',
    },
    {
        title: 'Notify',
        description: 'Delivers summaries, approvals, and operator-ready guidance through configurable channels and admin workflows.',
    },
]

const aiRuntimeHighlights = [
    'Embedded small LLM runtime option for private, low-footprint interpretation.',
    'Grounded on MCP evidence and deterministic drafts instead of freeform guesses.',
    'Useful for summarization, hypothesis framing, and operator-facing explanations.',
]

const LandingPage: React.FC = () => {
    const { isDark, toggleTheme } = useThemeStore()

    return (
        <div className="min-h-screen bg-gradient-to-br from-slate-100 via-white to-cyan-50 dark:from-slate-950 dark:via-slate-900 dark:to-slate-900">
            <header className="border-b border-slate-200/70 bg-white/90 backdrop-blur dark:border-slate-700 dark:bg-slate-900/90">
                <div className="mx-auto flex max-w-7xl items-center justify-between px-6 py-4">
                    <div className="inline-flex items-center gap-2 text-sm font-semibold text-slate-900 dark:text-slate-100">
                        <div className="rounded-md bg-blue-600 px-2 py-1 text-xs uppercase tracking-wide text-white">IF</div>
                        Image Factory
                    </div>
                    <div className="flex items-center gap-2">
                        <button
                            type="button"
                            onClick={toggleTheme}
                            className="inline-flex items-center gap-1 rounded-md border border-slate-300 bg-white px-3 py-1.5 text-sm text-slate-700 transition-colors hover:bg-slate-50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500/40 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:hover:bg-slate-700 dark:focus-visible:ring-blue-300/40"
                            title="Toggle dark mode"
                        >
                            {isDark ? <Sun className="h-4 w-4" /> : <Moon className="h-4 w-4" />}
                            {isDark ? 'Light' : 'Dark'}
                        </button>
                        <Link
                            to="/help/capabilities"
                            className="rounded-md border border-slate-300 px-3 py-1.5 text-sm text-slate-700 hover:bg-slate-50 dark:border-slate-600 dark:text-slate-200 dark:hover:bg-slate-800"
                        >
                            Capability Matrix
                        </Link>
                        <Link
                            to="/login"
                            className="rounded-md bg-blue-600 px-3 py-1.5 text-sm font-medium text-white transition-colors hover:bg-blue-700 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500/40 dark:bg-blue-500 dark:hover:bg-blue-400 dark:focus-visible:ring-blue-300/40"
                        >
                            Sign in
                        </Link>
                    </div>
                </div>
            </header>

            <main className="mx-auto max-w-7xl px-6 py-14">
                <section className="grid gap-10 lg:grid-cols-[minmax(0,1.25fr)_420px] lg:items-center">
                    <div>
                        <p className="text-xs font-semibold uppercase tracking-[0.2em] text-cyan-700 dark:text-cyan-300">Container Delivery + AI Operations</p>
                        <h1 className="mt-4 text-4xl font-bold tracking-tight text-slate-900 dark:text-slate-100 sm:text-5xl">
                            Governed Container Delivery With An SRE Smart Bot That Learns, Detects, Remediates, and Notifies
                        </h1>
                        <p className="mt-5 max-w-3xl text-base text-slate-600 dark:text-slate-300">
                            Image Factory centralizes build execution, image evidence, quarantine controls, and operational health, then layers on SRE Smart Bot to turn logs, signals, and incidents into guided operator action.
                        </p>
                        <div className="mt-8 flex flex-wrap gap-3">
                            <Link
                                to="/login"
                                className="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-blue-700 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500/40 dark:bg-blue-500 dark:hover:bg-blue-400 dark:focus-visible:ring-blue-300/40"
                            >
                                Open Workspace
                            </Link>
                            <Link
                                to="/help/capabilities"
                                className="rounded-md border border-slate-300 px-4 py-2 text-sm text-slate-700 hover:bg-slate-50 dark:border-slate-600 dark:text-slate-200 dark:hover:bg-slate-800"
                            >
                                Explore Capabilities
                            </Link>
                        </div>
                    </div>

                    <div className="rounded-2xl border border-slate-200 bg-white p-6 shadow-xl dark:border-slate-700 dark:bg-slate-900">
                        <div className="inline-flex items-center gap-2 rounded-full border border-cyan-200 bg-cyan-50 px-3 py-1 text-[11px] font-semibold uppercase tracking-wide text-cyan-800 dark:border-cyan-800 dark:bg-cyan-950/40 dark:text-cyan-200">
                            <Sparkles className="h-3.5 w-3.5" />
                            SRE Smart Bot
                        </div>
                        <h2 className="mt-4 text-lg font-semibold text-slate-900 dark:text-slate-100">AI-Assisted Operator Outcomes</h2>
                        <ul className="mt-4 space-y-3 text-sm text-slate-700 dark:text-slate-300">
                            <li>Grounded incident summaries tied to logs, metrics, backlog, and messaging health.</li>
                            <li>Read-only MCP tools for evidence gathering before any action is proposed.</li>
                            <li>Approval-aware remediation for safe operational recovery.</li>
                            <li>Detector learning that can suggest or auto-create rules in training mode.</li>
                        </ul>
                    </div>
                </section>

                <section className="mt-10 rounded-3xl border border-slate-200 bg-white/90 p-6 shadow-sm dark:border-slate-700 dark:bg-slate-900/80">
                    <div className="flex flex-col gap-3 lg:flex-row lg:items-end lg:justify-between">
                        <div>
                            <p className="text-xs font-semibold uppercase tracking-[0.2em] text-cyan-700 dark:text-cyan-300">SRE Smart Bot Loop</p>
                            <h2 className="mt-2 text-2xl font-bold text-slate-900 dark:text-slate-100">Built for real operations, not just dashboards</h2>
                            <p className="mt-2 max-w-3xl text-sm text-slate-600 dark:text-slate-300">
                                The bot combines deterministic policy, MCP-based evidence gathering, and an optional embedded small-LLM layer so operators can understand what is happening before they decide what to do.
                            </p>
                        </div>
                    </div>

                    <div className="mt-6 grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
                        {srePillars.map((pillar) => (
                            <article key={pillar.title} className="rounded-2xl border border-slate-200 bg-slate-50/80 p-5 dark:border-slate-700 dark:bg-slate-950/40">
                                <p className="text-sm font-semibold text-slate-900 dark:text-slate-100">{pillar.title}</p>
                                <p className="mt-2 text-xs leading-5 text-slate-600 dark:text-slate-300">{pillar.description}</p>
                            </article>
                        ))}
                    </div>

                    <div className="mt-6 rounded-2xl border border-cyan-200 bg-cyan-50/70 p-5 dark:border-cyan-800 dark:bg-cyan-950/20">
                        <div className="flex items-center gap-2">
                            <Sparkles className="h-4 w-4 text-cyan-700 dark:text-cyan-300" />
                            <p className="text-sm font-semibold text-slate-900 dark:text-slate-100">Embedded Small LLM, Used Carefully</p>
                        </div>
                        <p className="mt-2 text-xs leading-5 text-slate-700 dark:text-slate-300">
                            SRE Smart Bot can run with a small embedded model for local interpretation and incident storytelling, while keeping remediation deterministic and approval-bound.
                        </p>
                        <ul className="mt-3 space-y-1 text-xs text-slate-700 dark:text-slate-300">
                            {aiRuntimeHighlights.map((item) => (
                                <li key={item}>{item}</li>
                            ))}
                        </ul>
                    </div>
                </section>

                <section className="mt-14">
                    <h2 className="text-2xl font-bold text-slate-900 dark:text-slate-100">Core Capability Areas</h2>
                    <div className="mt-6 grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
                        {capabilityCards.map((item) => {
                            const Icon = item.icon
                            return (
                                <article key={item.title} className="rounded-xl border border-slate-200 bg-white/90 p-5 shadow-sm dark:border-slate-700 dark:bg-slate-900/80">
                                    <div className="mb-3 inline-flex h-9 w-9 items-center justify-center rounded-lg bg-cyan-100 text-cyan-700 dark:bg-cyan-900/40 dark:text-cyan-300">
                                        <Icon className="h-4 w-4" />
                                    </div>
                                    <h3 className="text-sm font-semibold text-slate-900 dark:text-slate-100">{item.title}</h3>
                                    <p className="mt-2 text-xs text-slate-600 dark:text-slate-300">{item.description}</p>
                                </article>
                            )
                        })}
                    </div>
                </section>
            </main>
        </div>
    )
}

export default LandingPage
