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
        title: 'Operational Controls',
        description: 'Manage runtime services, policies, health, and capability access from admin surfaces.',
    },
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
                        <p className="text-xs font-semibold uppercase tracking-[0.2em] text-cyan-700 dark:text-cyan-300">Container Delivery Platform</p>
                        <h1 className="mt-4 text-4xl font-bold tracking-tight text-slate-900 dark:text-slate-100 sm:text-5xl">
                            Governed Container Image Build and Scan Operations For Multi-Tenant Teams
                        </h1>
                        <p className="mt-5 max-w-3xl text-base text-slate-600 dark:text-slate-300">
                            Image Factory centralizes build execution, image evidence, quarantine controls, and operational health so teams can ship faster without losing governance.
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
                        <h2 className="text-lg font-semibold text-slate-900 dark:text-slate-100">Platform Outcomes</h2>
                        <ul className="mt-4 space-y-3 text-sm text-slate-700 dark:text-slate-300">
                            <li>Unified pipeline visibility from request to publish.</li>
                            <li>Capability-aware access and secure tenant boundaries.</li>
                            <li>Evidence-backed release decisions with scan/SBOM context.</li>
                            <li>Operational telemetry for runtime service reliability.</li>
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
