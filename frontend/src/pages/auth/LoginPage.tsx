import { authService, ssoService } from '@/services/authService'
import { useAuthStore } from '@/store/auth'
import { useThemeStore } from '@/store/theme'
import { LoginForm } from '@/types'
import { ArrowRight, BellRing, Box, Building2, KeyRound, Lock, Moon, ShieldCheck, Sparkles, Sun, Workflow } from 'lucide-react'
import React, { useEffect, useMemo, useState } from 'react'
import toast from 'react-hot-toast'
import { Link, useNavigate } from 'react-router-dom'

const LoginPage: React.FC = () => {
    const navigate = useNavigate()
    const { login } = useAuthStore()
    const { isDark, toggleTheme } = useThemeStore()

    const [form, setForm] = useState<LoginForm>({
        email: '',
        password: '',
        use_ldap: false,
    })
    const [loading, setLoadingState] = useState(false)
    const [error, setError] = useState<string>('')
    const [ssoProviders, setSsoProviders] = useState<any[]>([])
    const [ldapEnabled, setLdapEnabled] = useState(false)
    const [ldapOptionsLoaded, setLdapOptionsLoaded] = useState(false)
    const [setupRequired, setSetupRequired] = useState(false)
    const [setupStatusLoaded, setSetupStatusLoaded] = useState(false)

    useEffect(() => {
        const loadBootstrapStatus = async () => {
            try {
                const status = await authService.getBootstrapStatus()
                setSetupRequired(!!status.setup_required)
            } catch {
                setSetupRequired(false)
            } finally {
                setSetupStatusLoaded(true)
            }
        }

        const loadLoginCapabilities = async () => {
            try {
                const options = await authService.getLoginOptions()
                setLdapEnabled(!!options.ldap_enabled)
            } catch {
                setLdapEnabled(false)
            } finally {
                setLdapOptionsLoaded(true)
            }
        }

        const loadSSOProviders = async () => {
            if (setupRequired) {
                setSsoProviders([])
                return
            }
            try {
                const providers = await ssoService.listProviders()
                setSsoProviders(providers.filter((p) => p.enabled))
            } catch {
                setSsoProviders([])
            }
        }

        loadBootstrapStatus()
        loadLoginCapabilities()
        loadSSOProviders()
    }, [setupRequired])

    useEffect(() => {
        if (setupRequired) {
            setForm((prev) => ({ ...prev, use_ldap: false }))
        }
    }, [setupRequired])

    const capabilityHighlights = useMemo(
        () => [
            {
                icon: Building2,
                title: 'Multi-Tenant Projects',
                description: 'Isolate projects, teams, and credentials by tenant with clear ownership boundaries.',
                iconClass: 'bg-sky-100 text-sky-700 dark:bg-sky-900/40 dark:text-sky-300',
                cardClass: 'border-sky-200 bg-sky-50/70 dark:border-sky-800 dark:bg-sky-950/20',
            },
            {
                icon: Box,
                title: 'Build + Evidence Pipeline',
                description: 'Run Kaniko, Buildx, Docker, and capture SBOM/vulnerability evidence in one flow.',
                iconClass: 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/40 dark:text-emerald-300',
                cardClass: 'border-emerald-200 bg-emerald-50/70 dark:border-emerald-800 dark:bg-emerald-950/20',
            },
            {
                icon: ShieldCheck,
                title: 'Quarantine + On-Demand Scans',
                description: 'Control external image admission and trigger asynchronous scans with traceable outcomes.',
                iconClass: 'bg-violet-100 text-violet-700 dark:bg-violet-900/40 dark:text-violet-300',
                cardClass: 'border-violet-200 bg-violet-50/70 dark:border-violet-800 dark:bg-violet-950/20',
            },
        ],
        []
    )

    const workflowSteps = useMemo(
        () => [
            {
                icon: KeyRound,
                title: 'Authenticate',
                description: 'Sign in with local, LDAP, or approved SSO provider.',
            },
            {
                icon: Workflow,
                title: 'Operate',
                description: 'Trigger builds, review evidence, and monitor runtime workflows.',
            },
            {
                icon: BellRing,
                title: 'Track Outcomes',
                description: 'Receive notifications and audit-ready build history updates.',
            },
        ],
        []
    )

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault()
        if (loading) return

        setError('')

        if (!form.email || !form.password) {
            setError('Email and password are required')
            toast.error('Email and password are required')
            return
        }

        if (form.use_ldap && !ldapEnabled) {
            const message = 'LDAP sign-in is currently unavailable. Contact your administrator.'
            setError(message)
            toast.error(message)
            return
        }
        if (setupRequired && form.use_ldap) {
            const message = 'LDAP login is unavailable until initial setup is completed.'
            setError(message)
            toast.error(message)
            return
        }

        setLoadingState(true)

        try {
            const response = await authService.login(form)
            await login(response)
            toast.success('Login successful')
            if (response.requires_password_change) {
                navigate('/force-password-change')
            } else if (response.setup_required) {
                navigate('/admin/setup')
            }
        } catch (error: any) {
            const errorMsg = error.response?.data?.error || error.message || 'Login failed'
            setError(errorMsg)
            toast.error(errorMsg)
            setLoadingState(false)
        }
    }

    const handleSSO = (providerId: string) => {
        try {
            ssoService.initiateSSOLogin(providerId)
        } catch {
            toast.error('Failed to initiate SSO login')
        }
    }

    const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
        const { name, value } = e.target
        setForm((prev) => ({
            ...prev,
            [name]: value,
        }))
    }

    return (
        <div className="flex min-h-screen items-center bg-gradient-to-br from-slate-100 via-white to-cyan-50 px-4 py-6 dark:from-slate-950 dark:via-slate-900 dark:to-slate-900 sm:px-6 lg:px-8">
            <div className="mx-auto grid w-full max-w-7xl items-center gap-6 lg:grid-cols-[minmax(0,1.35fr)_440px]">
                <section className="rounded-2xl border border-slate-200 bg-white/95 p-6 shadow-xl dark:border-slate-700 dark:bg-slate-900 sm:p-8">
                    <div className="mb-4 flex items-center justify-between">
                        <div className="inline-flex items-center gap-2 rounded-full border border-cyan-200 bg-cyan-50 px-3 py-1 text-xs font-semibold uppercase tracking-wide text-cyan-800 dark:border-cyan-700 dark:bg-cyan-900/30 dark:text-cyan-200">
                            <Lock className="h-3.5 w-3.5" />
                            Secure Tenant Access
                        </div>
                        <div className="flex items-center gap-2">
                            <button
                                type="button"
                                onClick={toggleTheme}
                                className="inline-flex items-center gap-1 rounded-md border border-slate-300 bg-white px-2.5 py-1 text-xs font-medium text-slate-700 transition-colors hover:bg-slate-50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500/40 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:hover:bg-slate-700 dark:focus-visible:ring-blue-300/40"
                                title="Toggle dark mode"
                            >
                                {isDark ? <Sun className="h-3.5 w-3.5" /> : <Moon className="h-3.5 w-3.5" />}
                                {isDark ? 'Light' : 'Dark'}
                            </button>
                            <Link to="/landing" className="text-xs font-medium text-blue-600 hover:text-blue-500 dark:text-blue-300 dark:hover:text-blue-200">
                                View overview
                            </Link>
                        </div>
                    </div>
                    <h1 className="text-4xl font-bold tracking-tight text-slate-900 dark:text-slate-100 sm:text-5xl">Container Build, Scan, and Quarantine Workflows. All in One Place.</h1>
                    <p className="mt-4 max-w-3xl text-sm text-slate-600 dark:text-slate-300 sm:text-base">
                        Sign in to manage build pipelines, image evidence, quarantine workflows, and operational controls across tenant workspaces.
                    </p>

                    <div className="mt-6 grid gap-3 sm:grid-cols-3">
                        <div className="rounded-lg border border-slate-200 bg-white p-3 dark:border-slate-700 dark:bg-slate-800/80">
                            <p className="text-xs uppercase tracking-wide text-slate-500 dark:text-slate-400">Capabilities</p>
                            <p className="mt-1 text-lg font-semibold text-slate-900 dark:text-slate-100">12+</p>
                        </div>
                        <div className="rounded-lg border border-slate-200 bg-white p-3 dark:border-slate-700 dark:bg-slate-800/80">
                            <p className="text-xs uppercase tracking-wide text-slate-500 dark:text-slate-400">Realtime Signals</p>
                            <p className="mt-1 text-lg font-semibold text-slate-900 dark:text-slate-100">Build + Scan</p>
                        </div>
                        <div className="rounded-lg border border-slate-200 bg-white p-3 dark:border-slate-700 dark:bg-slate-800/80">
                            <p className="text-xs uppercase tracking-wide text-slate-500 dark:text-slate-400">Security Scope</p>
                            <p className="mt-1 text-lg font-semibold text-slate-900 dark:text-slate-100">Tenant-Isolated</p>
                        </div>
                    </div>

                    <div className="mt-5 grid gap-4 sm:grid-cols-3">
                        {capabilityHighlights.map((item) => {
                            const Icon = item.icon
                            return (
                                <div key={item.title} className={`rounded-xl border p-4 ${item.cardClass}`}>
                                    <div className={`mb-3 inline-flex h-9 w-9 items-center justify-center rounded-lg ${item.iconClass}`}>
                                        <Icon className="h-4 w-4" />
                                    </div>
                                    <p className="text-sm font-semibold text-slate-900 dark:text-slate-100">{item.title}</p>
                                    <p className="mt-1 text-xs text-slate-600 dark:text-slate-300">{item.description}</p>
                                </div>
                            )
                        })}
                    </div>

                    <div className="mt-8 rounded-xl border border-cyan-200 bg-cyan-50/70 p-4 dark:border-cyan-800 dark:bg-cyan-950/20">
                        <div className="flex items-center gap-2">
                            <Sparkles className="h-4 w-4 text-cyan-700 dark:text-cyan-300" />
                            <p className="text-xs font-semibold uppercase tracking-wide text-cyan-800 dark:text-cyan-300">What you get after sign-in</p>
                        </div>
                        <ul className="mt-2 space-y-1 text-xs text-slate-700 dark:text-slate-300">
                            <li>Real-time build and notification event streams.</li>
                            <li>Capability-aware route gating for tenant security posture.</li>
                            <li>Admin controls for runtime services, tool availability, and policies.</li>
                        </ul>
                    </div>

                    <div className="mt-4 rounded-xl border border-slate-200 bg-white p-4 dark:border-slate-700 dark:bg-slate-800/70">
                        <p className="text-xs font-semibold uppercase tracking-wide text-slate-700 dark:text-slate-300">Sign-in Workflow</p>
                        <div className="mt-3 grid gap-3 sm:grid-cols-3">
                            {workflowSteps.map((step) => {
                                const Icon = step.icon
                                return (
                                    <div key={step.title} className="rounded-lg border border-slate-200 bg-slate-50/80 p-3 dark:border-slate-700 dark:bg-slate-900/70">
                                        <div className="flex items-center gap-2">
                                            <Icon className="h-4 w-4 text-blue-600 dark:text-blue-300" />
                                            <p className="text-xs font-semibold text-slate-900 dark:text-slate-100">{step.title}</p>
                                            <ArrowRight className="ml-auto h-3.5 w-3.5 text-slate-400 dark:text-slate-500" />
                                        </div>
                                        <p className="mt-1 text-xs text-slate-600 dark:text-slate-300">{step.description}</p>
                                    </div>
                                )
                            })}
                        </div>
                    </div>
                </section>

                <section className="rounded-2xl border border-slate-200 bg-white/95 p-6 shadow-xl dark:border-slate-700 dark:bg-slate-900 sm:p-8">
                    <div className="mb-4">
                        <h2 className="text-2xl font-bold text-slate-900 dark:text-slate-100">Sign in</h2>
                        <p className="mt-1 text-sm text-slate-600 dark:text-slate-400">Use your tenant account credentials.</p>
                    </div>

                    {setupStatusLoaded && setupRequired && (
                        <div className="mb-4 rounded-md border border-amber-300 bg-amber-50 p-3 dark:border-amber-700 dark:bg-amber-900/30">
                            <p className="text-sm text-amber-900 dark:text-amber-200">
                                Initial setup mode: only local administrator login is available until setup is complete.
                            </p>
                        </div>
                    )}

                    {!setupRequired && ssoProviders.length > 0 && (
                        <div className="mb-5 space-y-2">
                            {ssoProviders.map((provider) => (
                                <button
                                    key={provider.id}
                                    type="button"
                                    onClick={() => handleSSO(provider.id)}
                                    className="group flex w-full items-center justify-between rounded-md border border-blue-200 bg-gradient-to-r from-blue-50 to-cyan-50 px-4 py-2 text-sm font-medium text-blue-900 transition-colors hover:from-blue-100 hover:to-cyan-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500/40 dark:border-blue-800 dark:from-blue-900/30 dark:to-cyan-900/20 dark:text-blue-100 dark:hover:from-blue-900/40 dark:hover:to-cyan-900/30 dark:focus-visible:ring-blue-300/40"
                                >
                                    <span className="inline-flex items-center gap-2">
                                        <Building2 className="h-4 w-4 text-blue-700 dark:text-blue-300" />
                                        Continue with {provider.name}
                                    </span>
                                    <ArrowRight className="h-4 w-4 text-blue-600 transition-transform group-hover:translate-x-0.5 dark:text-blue-300" />
                                </button>
                            ))}
                            <div className="relative py-2">
                                <div className="absolute inset-0 flex items-center">
                                    <div className="w-full border-t border-slate-300 dark:border-slate-600" />
                                </div>
                                <div className="relative flex justify-center text-xs">
                                    <span className="bg-white px-2 text-slate-500 dark:bg-slate-900 dark:text-slate-400">Or continue with email</span>
                                </div>
                            </div>
                        </div>
                    )}

                    <form className="space-y-5" onSubmit={handleSubmit}>
                        {error && (
                            <div className="rounded-md border border-red-200 bg-red-50 p-3 dark:border-red-800 dark:bg-red-950/40">
                                <p className="text-sm text-red-800 dark:text-red-200">{error}</p>
                            </div>
                        )}

                        <div>
                            <label htmlFor="email" className="mb-1 block text-sm font-medium text-slate-700 dark:text-slate-300">Email</label>
                            <input
                                id="email"
                                name="email"
                                type="email"
                                autoComplete="email"
                                required
                                className="auth-input w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 placeholder:text-slate-400 focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-500/30 dark:border-slate-600 dark:bg-slate-800 dark:text-white dark:placeholder:text-slate-500"
                                placeholder="you@company.com"
                                value={form.email}
                                onChange={handleInputChange}
                            />
                        </div>

                        <div>
                            <label htmlFor="password" className="mb-1 block text-sm font-medium text-slate-700 dark:text-slate-300">Password</label>
                            <input
                                id="password"
                                name="password"
                                type="password"
                                autoComplete="current-password"
                                required
                                className="auth-input w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 placeholder:text-slate-400 focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-500/30 dark:border-slate-600 dark:bg-slate-800 dark:text-white dark:placeholder:text-slate-500"
                                placeholder="Enter password"
                                value={form.password}
                                onChange={handleInputChange}
                            />
                        </div>

                        {!setupRequired && (
                            <>
                                <div className="flex items-center justify-between">
                                    <label
                                        htmlFor="use_ldap"
                                        className={`inline-flex items-center gap-2 text-sm ${ldapEnabled ? 'text-slate-700 dark:text-slate-300' : 'text-slate-400 dark:text-slate-500'}`}
                                    >
                                        <input
                                            id="use_ldap"
                                            name="use_ldap"
                                            type="checkbox"
                                            checked={!!form.use_ldap}
                                            onChange={(e) => setForm((prev) => ({ ...prev, use_ldap: e.target.checked }))}
                                            disabled={!ldapEnabled}
                                            className="h-4 w-4 rounded border-slate-300 text-blue-600 focus:ring-blue-500 disabled:opacity-50"
                                        />
                                        Sign in with LDAP
                                    </label>
                                    <Link to="/forgot-password" className="text-sm font-medium text-blue-600 hover:text-blue-500 dark:text-blue-300 dark:hover:text-blue-200">
                                        Forgot password?
                                    </Link>
                                </div>
                                {ldapOptionsLoaded && !ldapEnabled && (
                                    <p className="text-xs text-amber-600 dark:text-amber-400">LDAP login is disabled or not configured.</p>
                                )}
                            </>
                        )}

                        <button
                            type="submit"
                            disabled={loading}
                            className="inline-flex w-full items-center justify-center rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-blue-700 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500/40 disabled:cursor-not-allowed disabled:opacity-60 dark:bg-blue-500 dark:hover:bg-blue-400 dark:focus-visible:ring-blue-300/40"
                        >
                            {loading ? 'Signing in...' : 'Sign in'}
                        </button>
                    </form>

                    {!setupRequired && (
                        <p className="mt-4 text-center text-xs text-slate-500 dark:text-slate-400">
                            Need access? <span className="font-medium">Contact your tenant administrator.</span>
                        </p>
                    )}
                </section>
            </div>
        </div>
    )
}

export default LoginPage
