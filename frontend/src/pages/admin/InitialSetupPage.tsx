import { api } from '@/services/api'
import React, { useState } from 'react'
import toast from 'react-hot-toast'
import { useAuthStore } from '@/store/auth'

const InitialSetupPage: React.FC = () => {
    const { setSetupRequired } = useAuthStore()
    const [finalizing, setFinalizing] = useState(false)
    const [saving, setSaving] = useState(false)
    const [loadingDefaults, setLoadingDefaults] = useState(false)
    const [showRestartConfirm, setShowRestartConfirm] = useState(false)
    const [showRestartProgress, setShowRestartProgress] = useState(false)
    const [restartTimedOut, setRestartTimedOut] = useState(false)
    const [restartAttempts, setRestartAttempts] = useState(0)
    const [activeStep, setActiveStep] = useState<'general' | 'smtp' | 'ldap' | 'external_services' | 'runtime_services' | 'sso' | 'review'>('general')
    const [missingSteps, setMissingSteps] = useState<string[]>([])
    const [completedSteps, setCompletedSteps] = useState<string[]>([])

    const [general, setGeneral] = useState({
        system_name: 'Image Factory',
        system_description: 'Container image factory platform',
        admin_email: 'admin@imagefactory.local',
        support_email: 'support@example.com',
        time_zone: 'UTC',
        date_format: 'YYYY-MM-DD',
        default_language: 'en',
        maintenance_mode: false,
    })

    const [smtp, setSmtp] = useState({
        host: '',
        port: 587,
        username: '',
        password: '',
        from: 'noreply@example.com',
        start_tls: true,
        ssl: false,
        enabled: true,
    })

    const [ldap, setLdap] = useState({
        provider_name: 'Active Directory',
        provider_type: 'active_directory',
        host: '',
        port: 389,
        base_dn: '',
        user_search_base: '',
        group_search_base: '',
        bind_dn: '',
        bind_password: '',
        user_filter: '(uid=%s)',
        group_filter: '(member=%s)',
        start_tls: false,
        ssl: false,
        allowed_domains: ['imagefactory.local'],
        enabled: true,
    })
    const [ldapDomainDraft, setLdapDomainDraft] = useState('')

    const [externalService, setExternalService] = useState({
        name: 'tenant-service',
        description: 'Tenant service integration',
        url: '',
        api_key: '',
        enabled: true,
    })

    const [runtimeServices, setRuntimeServices] = useState({
        dispatcher_url: 'http://localhost',
        dispatcher_port: 8084,
        dispatcher_mtls_enabled: false,
        dispatcher_ca_cert: '',
        dispatcher_client_cert: '',
        dispatcher_client_key: '',
        workflow_orchestrator_enabled: true,
        email_worker_url: 'http://localhost',
        email_worker_port: 8081,
        email_worker_tls_enabled: false,
        notification_worker_url: 'http://localhost',
        notification_worker_port: 8083,
        notification_tls_enabled: false,
        health_check_timeout_seconds: 5,
        provider_readiness_watcher_enabled: true,
        provider_readiness_watcher_interval_seconds: 180,
        provider_readiness_watcher_timeout_seconds: 90,
        provider_readiness_watcher_batch_size: 200,
    })

    const [ssoType, setSsoType] = useState<'oidc' | 'saml'>('oidc')
    const [oidc, setOidc] = useState({
        name: 'PingFed',
        issuer: '',
        client_id: '',
        client_secret: '',
        authorization_url: '',
        token_url: '',
        userinfo_url: '',
        jwks_url: '',
        redirect_uris: ['http://localhost:3000/auth/callback'],
        scopes: ['openid', 'profile', 'email'],
        response_types: ['code'],
        grant_types: ['authorization_code'],
        attributes: {},
        enabled: true,
    })
    const [saml, setSaml] = useState({
        name: '',
        entity_id: '',
        sso_url: '',
        slo_url: '',
        certificate: '',
        private_key: '',
        position: 'idp',
        attributes: {},
        enabled: true,
    })

    const refreshStatus = async () => {
        try {
            const response = await api.get('/bootstrap/status')
            setMissingSteps(response.data?.missing_steps || [])
            setCompletedSteps(response.data?.completed_steps || [])
        } catch (error) {
            // ignore status refresh errors in UI
        }
    }

    const handleStartSetup = async () => {
        try {
            await api.post('/bootstrap/start')
            await refreshStatus()
        } catch (error) {
            // Best-effort; setup page remains usable.
        }
    }

    const loadDefaultsFromEnv = async () => {
        try {
            setLoadingDefaults(true)
            const response = await api.get('/bootstrap/defaults')
            const defaults = response.data || {}

            if (defaults.general) setGeneral(prev => ({ ...prev, ...defaults.general }))
            if (defaults.smtp) setSmtp(prev => ({ ...prev, ...defaults.smtp }))
            if (defaults.ldap) {
                const normalizedLdap = {
                    ...defaults.ldap,
                    allowed_domains: Array.isArray(defaults.ldap.allowed_domains)
                        ? defaults.ldap.allowed_domains
                        : [],
                }
                setLdap(prev => ({ ...prev, ...normalizedLdap }))
            }
            if (defaults.external_service) setExternalService(prev => ({ ...prev, ...defaults.external_service }))
            if (defaults.runtime_services) setRuntimeServices(prev => ({ ...prev, ...defaults.runtime_services }))

            if (defaults.sso) {
                const type = defaults.sso.type === 'saml' ? 'saml' : 'oidc'
                setSsoType(type)
                if (defaults.sso.oidc) setOidc(prev => ({ ...prev, ...defaults.sso.oidc }))
                if (defaults.sso.saml) setSaml(prev => ({ ...prev, ...defaults.sso.saml }))
            }

            toast.success('Setup defaults loaded from environment')
        } catch (error: any) {
            toast.error(error?.response?.data?.error || 'Failed to load defaults from environment')
        } finally {
            setLoadingDefaults(false)
        }
    }

    const saveStep = async (step: 'general' | 'smtp' | 'ldap' | 'external_services' | 'runtime_services' | 'sso') => {
        try {
            setSaving(true)
            let payload: any = {}
            if (step === 'general') payload = { config: general }
            if (step === 'smtp') payload = { config: smtp }
            if (step === 'ldap') payload = { config: ldap, config_key: 'ldap_active_directory' }
            if (step === 'external_services') payload = { config: externalService }
            if (step === 'runtime_services') payload = { config: runtimeServices }
            if (step === 'sso') {
                payload = {
                    type: ssoType,
                    config: ssoType === 'oidc' ? oidc : saml,
                }
            }
            await api.post(`/bootstrap/steps/${step}/save`, payload)
            toast.success(`${step.toUpperCase()} settings saved`)
            await refreshStatus()
        } catch (error: any) {
            toast.error(error?.response?.data?.error || `Failed to save ${step} settings`)
        } finally {
            setSaving(false)
        }
    }

    const sleep = (ms: number) => new Promise(resolve => setTimeout(resolve, ms))

    const isBackendAvailable = async (): Promise<boolean> => {
        try {
            const controller = new AbortController()
            const timeout = window.setTimeout(() => controller.abort(), 5000)
            const response = await fetch('/api/v1/bootstrap/status', {
                method: 'GET',
                cache: 'no-store',
                signal: controller.signal,
            })
            window.clearTimeout(timeout)
            return response.ok
        } catch {
            return false
        }
    }

    const waitForBackendRecovery = async (maxAttempts = 24) => {
        for (let i = 1; i <= maxAttempts; i += 1) {
            setRestartAttempts(i)
            await sleep(2500)
            const available = await isBackendAvailable()
            if (available) {
                toast.success('Server is back online. Redirecting to login...')
                window.location.href = '/login'
                return
            }
        }
        setRestartTimedOut(true)
    }

    const checkBackendNow = async () => {
        const available = await isBackendAvailable()
        if (available) {
            toast.success('Server is reachable. Redirecting to login...')
            window.location.href = '/login'
            return
        }
        toast.error('Server is still unavailable. Please check backend process and try again.')
    }

    const handleCompleteSetup = async () => {
        try {
            setFinalizing(true)
            setRestartTimedOut(false)
            setRestartAttempts(0)
            await api.post('/bootstrap/save-all', {
                general,
                smtp,
                ldap,
                runtime_services: runtimeServices,
                external_service: externalService,
                sso_type: ssoType,
                oidc,
                saml,
            })
            await refreshStatus()
            await api.post('/bootstrap/complete')
            setSetupRequired(false)
            setShowRestartConfirm(false)
            setShowRestartProgress(true)
            toast.success('Initial setup completed. Restarting server...')
            await api.post('/admin/system/reboot').catch(() => undefined)
            await waitForBackendRecovery()
        } catch (error: any) {
            setMissingSteps(error?.response?.data?.missing_steps || [])
            toast.error(error?.response?.data?.error || 'Failed to complete setup')
        } finally {
            setFinalizing(false)
        }
    }

    React.useEffect(() => {
        handleStartSetup()
    }, [])

    const stepButtonClass = (step: 'general' | 'smtp' | 'ldap' | 'external_services' | 'runtime_services' | 'sso' | 'review') => {
        const isActive = activeStep === step
        return `px-3 py-2 rounded border text-sm transition-colors ${isActive ? 'bg-blue-600 text-white border-blue-600' : 'bg-white dark:bg-slate-800 text-slate-700 dark:text-slate-200 border-slate-300 dark:border-slate-600 hover:bg-slate-50 dark:hover:bg-slate-700'}`
    }
    const requiredSteps = ['general', 'smtp', 'ldap', 'runtime_services']
    const missingRequiredSteps = missingSteps.filter(step => requiredSteps.includes(step))
    const isCompleteDisabled = finalizing || saving

    const cardClass = 'rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 p-4 space-y-3'
    const inputClass = 'border border-slate-300 dark:border-slate-600 rounded px-3 py-2 bg-white dark:bg-slate-700 text-slate-900 dark:text-white placeholder-slate-500 dark:placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500'
    const textareaClass = 'border border-slate-300 dark:border-slate-600 rounded px-3 py-2 bg-white dark:bg-slate-700 text-slate-900 dark:text-white placeholder-slate-500 dark:placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500'
    const primaryButtonClass = 'rounded bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 disabled:opacity-60 disabled:cursor-not-allowed transition-colors'
    const checkboxLabelClass = 'text-sm text-slate-700 dark:text-slate-300'

    const addLdapDomain = () => {
        const normalized = ldapDomainDraft.trim().toLowerCase().replace(/^@/, '')
        if (!normalized) return
        const domains = Array.isArray(ldap.allowed_domains) ? ldap.allowed_domains : []
        if (domains.includes(normalized)) return
        setLdap(prev => {
            const safeDomains = Array.isArray(prev.allowed_domains) ? prev.allowed_domains : []
            return { ...prev, allowed_domains: [...safeDomains, normalized] }
        })
        setLdapDomainDraft('')
    }

    const removeLdapDomain = (domain: string) => {
        setLdap(prev => {
            const safeDomains = Array.isArray(prev.allowed_domains) ? prev.allowed_domains : []
            return { ...prev, allowed_domains: safeDomains.filter(d => d !== domain) }
        })
    }

    return (
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6 space-y-6">
            <div className="rounded-lg border border-amber-300 bg-amber-50 dark:border-amber-700 dark:bg-amber-900/30 p-4">
                <div className="flex flex-wrap items-start justify-between gap-3">
                    <h1 className="text-lg font-semibold text-amber-900 dark:text-amber-200">Initial System Setup Required</h1>
                    <button
                        className="px-3 py-2 rounded border text-sm bg-emerald-600 text-white border-emerald-600 hover:bg-emerald-700 transition-colors disabled:opacity-60 disabled:cursor-not-allowed"
                        onClick={loadDefaultsFromEnv}
                        disabled={loadingDefaults}
                    >
                        {loadingDefaults ? 'Loading Env...' : 'Load from Env'}
                    </button>
                </div>
                <p className="mt-1 text-sm text-amber-800 dark:text-amber-300">
                    Configure required settings and complete setup. Completion is blocked until all required steps are valid.
                </p>
                <p className="mt-1 text-sm text-amber-800 dark:text-amber-300">
                    Runtime services run independently and must be started in parallel with the API: dispatcher, email-worker, and notification-worker.
                </p>
                {missingSteps.length > 0 && (
                    <p className="mt-2 text-sm text-amber-900 dark:text-amber-200">
                        Missing steps: {missingSteps.join(', ')}
                    </p>
                )}
            </div>

            <div className="flex flex-wrap items-center gap-2">
                <span className="text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-slate-400 mr-1">Required</span>
                <button className={stepButtonClass('general')} onClick={() => setActiveStep('general')}>General</button>
                <button className={stepButtonClass('smtp')} onClick={() => setActiveStep('smtp')}>SMTP</button>
                <button className={stepButtonClass('ldap')} onClick={() => setActiveStep('ldap')}>LDAP</button>
                <button className={stepButtonClass('runtime_services')} onClick={() => setActiveStep('runtime_services')}>Runtime Services</button>
                <button className={stepButtonClass('review')} onClick={() => setActiveStep('review')}>Review</button>

                <span className="mx-2 h-6 w-px bg-slate-300 dark:bg-slate-600" />

                <span className="text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-slate-400 mr-1">Optional</span>
                <button className={stepButtonClass('external_services')} onClick={() => setActiveStep('external_services')}>External Services</button>
                <button className={stepButtonClass('sso')} onClick={() => setActiveStep('sso')}>SSO</button>
            </div>

            {activeStep === 'general' && (
                <div className={cardClass}>
                    <h2 className="text-base font-semibold text-slate-900 dark:text-white">General</h2>
                    <div className="grid md:grid-cols-2 gap-3">
                        <input className={inputClass} placeholder="System Name" value={general.system_name} onChange={(e) => setGeneral(prev => ({ ...prev, system_name: e.target.value }))} />
                        <input className={inputClass} placeholder="Admin Email" value={general.admin_email} onChange={(e) => setGeneral(prev => ({ ...prev, admin_email: e.target.value }))} />
                        <input className={inputClass} placeholder="Support Email" value={general.support_email} onChange={(e) => setGeneral(prev => ({ ...prev, support_email: e.target.value }))} />
                        <input className={inputClass} placeholder="Time Zone" value={general.time_zone} onChange={(e) => setGeneral(prev => ({ ...prev, time_zone: e.target.value }))} />
                        <input className={inputClass} placeholder="Date Format" value={general.date_format} onChange={(e) => setGeneral(prev => ({ ...prev, date_format: e.target.value }))} />
                        <input className={inputClass} placeholder="Language" value={general.default_language} onChange={(e) => setGeneral(prev => ({ ...prev, default_language: e.target.value }))} />
                    </div>
                    <textarea className={`${textareaClass} w-full min-h-24`} placeholder="System Description" value={general.system_description} onChange={(e) => setGeneral(prev => ({ ...prev, system_description: e.target.value }))} />
                    <button className={primaryButtonClass} disabled={saving} onClick={() => saveStep('general')}>
                        {saving ? 'Saving...' : 'Save General'}
                    </button>
                </div>
            )}

            {activeStep === 'smtp' && (
                <div className={cardClass}>
                    <h2 className="text-base font-semibold text-slate-900 dark:text-white">SMTP</h2>
                    <div className="grid md:grid-cols-2 gap-3">
                        <input className={inputClass} placeholder="Host" value={smtp.host} onChange={(e) => setSmtp(prev => ({ ...prev, host: e.target.value }))} />
                        <input className={inputClass} type="number" placeholder="Port" value={smtp.port} onChange={(e) => setSmtp(prev => ({ ...prev, port: Number(e.target.value) || 587 }))} />
                        <input className={inputClass} placeholder="Username" value={smtp.username} onChange={(e) => setSmtp(prev => ({ ...prev, username: e.target.value }))} />
                        <input className={inputClass} placeholder="Password" type="password" value={smtp.password} onChange={(e) => setSmtp(prev => ({ ...prev, password: e.target.value }))} />
                        <input className={`${inputClass} md:col-span-2`} placeholder="From Email" value={smtp.from} onChange={(e) => setSmtp(prev => ({ ...prev, from: e.target.value }))} />
                    </div>
                    <div className="flex gap-4 text-sm">
                        <label className={checkboxLabelClass}><input type="checkbox" checked={smtp.start_tls} onChange={(e) => setSmtp(prev => ({ ...prev, start_tls: e.target.checked }))} /> <span className="ml-1">StartTLS</span></label>
                        <label className={checkboxLabelClass}><input type="checkbox" checked={smtp.ssl} onChange={(e) => setSmtp(prev => ({ ...prev, ssl: e.target.checked }))} /> <span className="ml-1">SSL</span></label>
                        <label className={checkboxLabelClass}><input type="checkbox" checked={smtp.enabled} onChange={(e) => setSmtp(prev => ({ ...prev, enabled: e.target.checked }))} /> <span className="ml-1">Enabled</span></label>
                    </div>
                    <button className={primaryButtonClass} disabled={saving} onClick={() => saveStep('smtp')}>
                        {saving ? 'Saving...' : 'Save SMTP'}
                    </button>
                </div>
            )}

            {activeStep === 'ldap' && (
                <div className={cardClass}>
                    <h2 className="text-base font-semibold text-slate-900 dark:text-white">LDAP</h2>
                    <div className="grid md:grid-cols-2 gap-3">
                        <input className={inputClass} placeholder="Provider Name" value={ldap.provider_name} onChange={(e) => setLdap(prev => ({ ...prev, provider_name: e.target.value }))} />
                        <input className={inputClass} placeholder="Provider Type" value={ldap.provider_type} onChange={(e) => setLdap(prev => ({ ...prev, provider_type: e.target.value }))} />
                        <input className={inputClass} placeholder="Host" value={ldap.host} onChange={(e) => setLdap(prev => ({ ...prev, host: e.target.value }))} />
                        <input className={inputClass} type="number" placeholder="Port" value={ldap.port} onChange={(e) => setLdap(prev => ({ ...prev, port: Number(e.target.value) || 389 }))} />
                        <input className={`${inputClass} md:col-span-2`} placeholder="Base DN" value={ldap.base_dn} onChange={(e) => setLdap(prev => ({ ...prev, base_dn: e.target.value }))} />
                        <input className={inputClass} placeholder="Bind DN" value={ldap.bind_dn} onChange={(e) => setLdap(prev => ({ ...prev, bind_dn: e.target.value }))} />
                        <input className={inputClass} placeholder="Bind Password" type="password" value={ldap.bind_password} onChange={(e) => setLdap(prev => ({ ...prev, bind_password: e.target.value }))} />
                        <input className={inputClass} placeholder="User Filter" value={ldap.user_filter} onChange={(e) => setLdap(prev => ({ ...prev, user_filter: e.target.value }))} />
                        <input className={inputClass} placeholder="Group Filter" value={ldap.group_filter} onChange={(e) => setLdap(prev => ({ ...prev, group_filter: e.target.value }))} />
                    </div>
                    <div className="space-y-2">
                        <label className="text-sm font-medium text-slate-700 dark:text-slate-300">Allowed Email Domains</label>
                        <div className="flex flex-wrap gap-2">
                            <input
                                className={`${inputClass} flex-1 min-w-[220px]`}
                                placeholder="e.g. imagefactory.local"
                                value={ldapDomainDraft}
                                onChange={(e) => setLdapDomainDraft(e.target.value)}
                                onKeyDown={(e) => {
                                    if (e.key === 'Enter') {
                                        e.preventDefault()
                                        addLdapDomain()
                                    }
                                }}
                            />
                            <button type="button" className={primaryButtonClass} onClick={addLdapDomain}>Add Domain</button>
                        </div>
                        {Array.isArray(ldap.allowed_domains) && ldap.allowed_domains.length > 0 && (
                            <div className="flex flex-wrap gap-2">
                                {ldap.allowed_domains.map((domain) => (
                                    <span key={domain} className="inline-flex items-center gap-2 rounded-full border border-slate-300 dark:border-slate-600 bg-slate-100 dark:bg-slate-700 px-3 py-1 text-xs text-slate-700 dark:text-slate-200">
                                        {domain}
                                        <button
                                            type="button"
                                            className="text-slate-500 hover:text-red-600 dark:text-slate-300 dark:hover:text-red-400"
                                            onClick={() => removeLdapDomain(domain)}
                                            aria-label={`Remove ${domain}`}
                                        >
                                            x
                                        </button>
                                    </span>
                                ))}
                            </div>
                        )}
                    </div>
                    <div className="flex gap-4 text-sm">
                        <label className={checkboxLabelClass}><input type="checkbox" checked={ldap.start_tls} onChange={(e) => setLdap(prev => ({ ...prev, start_tls: e.target.checked }))} /> <span className="ml-1">StartTLS</span></label>
                        <label className={checkboxLabelClass}><input type="checkbox" checked={ldap.ssl} onChange={(e) => setLdap(prev => ({ ...prev, ssl: e.target.checked }))} /> <span className="ml-1">SSL</span></label>
                        <label className={checkboxLabelClass}><input type="checkbox" checked={ldap.enabled} onChange={(e) => setLdap(prev => ({ ...prev, enabled: e.target.checked }))} /> <span className="ml-1">Enabled</span></label>
                    </div>
                    <button className={primaryButtonClass} disabled={saving} onClick={() => saveStep('ldap')}>
                        {saving ? 'Saving...' : 'Save LDAP'}
                    </button>
                </div>
            )}

            {activeStep === 'external_services' && (
                <div className={cardClass}>
                    <h2 className="text-base font-semibold text-slate-900 dark:text-white">External Services</h2>
                    <div className="grid md:grid-cols-2 gap-3">
                        <input className={inputClass} placeholder="Service Name" value={externalService.name} onChange={(e) => setExternalService(prev => ({ ...prev, name: e.target.value }))} />
                        <input className={inputClass} placeholder="Description" value={externalService.description} onChange={(e) => setExternalService(prev => ({ ...prev, description: e.target.value }))} />
                        <input className={`${inputClass} md:col-span-2`} placeholder="URL" value={externalService.url} onChange={(e) => setExternalService(prev => ({ ...prev, url: e.target.value }))} />
                        <input className={`${inputClass} md:col-span-2`} placeholder="API Key" value={externalService.api_key} onChange={(e) => setExternalService(prev => ({ ...prev, api_key: e.target.value }))} />
                    </div>
                    <div className="flex gap-4 text-sm">
                        <label className={checkboxLabelClass}><input type="checkbox" checked={externalService.enabled} onChange={(e) => setExternalService(prev => ({ ...prev, enabled: e.target.checked }))} /> <span className="ml-1">Enabled</span></label>
                    </div>
                    <button className={primaryButtonClass} disabled={saving} onClick={() => saveStep('external_services')}>
                        {saving ? 'Saving...' : 'Save External Service'}
                    </button>
                </div>
            )}

            {activeStep === 'runtime_services' && (
                <div className={cardClass}>
                    <h2 className="text-base font-semibold text-slate-900 dark:text-white">Runtime Services</h2>
                    <p className="text-sm text-slate-700 dark:text-slate-300">
                        These are independent background services. Configure endpoints for health checks and operational visibility.
                    </p>
                    <div className="space-y-3">
                        <div className="grid md:grid-cols-4 gap-3 items-center">
                            <div className="text-sm font-medium text-slate-700 dark:text-slate-300">Dispatcher</div>
                            <input className={`${inputClass} md:col-span-2`} placeholder="URL" value={runtimeServices.dispatcher_url} onChange={(e) => setRuntimeServices(prev => ({ ...prev, dispatcher_url: e.target.value }))} />
                            <input className={inputClass} type="number" placeholder="Port" value={runtimeServices.dispatcher_port} onChange={(e) => setRuntimeServices(prev => ({ ...prev, dispatcher_port: Number(e.target.value) || 8084 }))} />
                        </div>
                        <div className="grid md:grid-cols-4 gap-3 items-center">
                            <div className="text-sm font-medium text-slate-700 dark:text-slate-300">Dispatcher TLS</div>
                            <label className={`${checkboxLabelClass} md:col-span-3 flex items-center gap-2`}><input type="checkbox" checked={runtimeServices.dispatcher_mtls_enabled} onChange={(e) => setRuntimeServices(prev => ({ ...prev, dispatcher_mtls_enabled: e.target.checked }))} /> Enable mTLS</label>
                        </div>
                        <div className="grid md:grid-cols-4 gap-3 items-center">
                            <div className="text-sm font-medium text-slate-700 dark:text-slate-300">Dispatcher CA Cert</div>
                            <input className={`${inputClass} md:col-span-3`} placeholder="PEM" value={runtimeServices.dispatcher_ca_cert} onChange={(e) => setRuntimeServices(prev => ({ ...prev, dispatcher_ca_cert: e.target.value }))} />
                        </div>
                        <div className="grid md:grid-cols-4 gap-3 items-center">
                            <div className="text-sm font-medium text-slate-700 dark:text-slate-300">Dispatcher Client Cert</div>
                            <input className={`${inputClass} md:col-span-3`} placeholder="PEM" value={runtimeServices.dispatcher_client_cert} onChange={(e) => setRuntimeServices(prev => ({ ...prev, dispatcher_client_cert: e.target.value }))} />
                        </div>
                        <div className="grid md:grid-cols-4 gap-3 items-center">
                            <div className="text-sm font-medium text-slate-700 dark:text-slate-300">Dispatcher Client Key</div>
                            <input className={`${inputClass} md:col-span-3`} placeholder="PEM" value={runtimeServices.dispatcher_client_key} onChange={(e) => setRuntimeServices(prev => ({ ...prev, dispatcher_client_key: e.target.value }))} />
                        </div>
                        <div className="grid md:grid-cols-4 gap-3 items-center">
                            <div className="text-sm font-medium text-slate-700 dark:text-slate-300">Workflow Orchestrator</div>
                            <label className={`${checkboxLabelClass} md:col-span-3 flex items-center gap-2`}>
                                <input
                                    type="checkbox"
                                    checked={runtimeServices.workflow_orchestrator_enabled}
                                    onChange={(e) => setRuntimeServices(prev => ({ ...prev, workflow_orchestrator_enabled: e.target.checked }))}
                                />
                                Enable workflow orchestrator
                            </label>
                        </div>

                        <div className="grid md:grid-cols-4 gap-3 items-center">
                            <div className="text-sm font-medium text-slate-700 dark:text-slate-300">Email Worker</div>
                            <input className={`${inputClass} md:col-span-2`} placeholder="URL" value={runtimeServices.email_worker_url} onChange={(e) => setRuntimeServices(prev => ({ ...prev, email_worker_url: e.target.value }))} />
                            <input className={inputClass} type="number" placeholder="Port" value={runtimeServices.email_worker_port} onChange={(e) => setRuntimeServices(prev => ({ ...prev, email_worker_port: Number(e.target.value) || 8081 }))} />
                        </div>
                        <div className="grid md:grid-cols-4 gap-3 items-center">
                            <div className="text-sm font-medium text-slate-700 dark:text-slate-300">Email Worker TLS</div>
                            <label className={`${checkboxLabelClass} md:col-span-3 flex items-center gap-2`}><input type="checkbox" checked={runtimeServices.email_worker_tls_enabled} onChange={(e) => setRuntimeServices(prev => ({ ...prev, email_worker_tls_enabled: e.target.checked }))} /> Enable TLS</label>
                        </div>

                        <div className="grid md:grid-cols-4 gap-3 items-center">
                            <div className="text-sm font-medium text-slate-700 dark:text-slate-300">Notification Worker</div>
                            <input className={`${inputClass} md:col-span-2`} placeholder="URL" value={runtimeServices.notification_worker_url} onChange={(e) => setRuntimeServices(prev => ({ ...prev, notification_worker_url: e.target.value }))} />
                            <input className={inputClass} type="number" placeholder="Port" value={runtimeServices.notification_worker_port} onChange={(e) => setRuntimeServices(prev => ({ ...prev, notification_worker_port: Number(e.target.value) || 8083 }))} />
                        </div>
                        <div className="grid md:grid-cols-4 gap-3 items-center">
                            <div className="text-sm font-medium text-slate-700 dark:text-slate-300">Notification TLS</div>
                            <label className={`${checkboxLabelClass} md:col-span-3 flex items-center gap-2`}><input type="checkbox" checked={runtimeServices.notification_tls_enabled} onChange={(e) => setRuntimeServices(prev => ({ ...prev, notification_tls_enabled: e.target.checked }))} /> Enable TLS</label>
                        </div>

                        <div className="grid md:grid-cols-4 gap-3 items-center">
                            <div className="text-sm font-medium text-slate-700 dark:text-slate-300">Health Timeout</div>
                            <input className={`${inputClass} md:col-span-3`} type="number" placeholder="Seconds" value={runtimeServices.health_check_timeout_seconds} onChange={(e) => setRuntimeServices(prev => ({ ...prev, health_check_timeout_seconds: Number(e.target.value) || 5 }))} />
                        </div>
                        <div className="grid md:grid-cols-4 gap-3 items-center">
                            <div className="text-sm font-medium text-slate-700 dark:text-slate-300">Provider Readiness Watcher</div>
                            <label className={`${checkboxLabelClass} md:col-span-3 flex items-center gap-2`}>
                                <input
                                    type="checkbox"
                                    checked={runtimeServices.provider_readiness_watcher_enabled}
                                    onChange={(e) => setRuntimeServices(prev => ({ ...prev, provider_readiness_watcher_enabled: e.target.checked }))}
                                />
                                Enable scheduled provider readiness reconciliation
                            </label>
                        </div>
                        <p className="text-xs text-slate-500 dark:text-slate-400 md:col-span-4">
                            Recommended defaults: interval 180s, timeout 90s, batch 200.
                        </p>
                        <div className="grid md:grid-cols-4 gap-3 items-center">
                            <div className="text-sm font-medium text-slate-700 dark:text-slate-300">Watcher Interval</div>
                            <input className={`${inputClass} md:col-span-3`} type="number" min={30} placeholder="Seconds" value={runtimeServices.provider_readiness_watcher_interval_seconds} onChange={(e) => setRuntimeServices(prev => ({ ...prev, provider_readiness_watcher_interval_seconds: Number(e.target.value) || 180 }))} />
                        </div>
                        <div className="grid md:grid-cols-4 gap-3 items-center">
                            <div className="text-sm font-medium text-slate-700 dark:text-slate-300">Watcher Timeout</div>
                            <input className={`${inputClass} md:col-span-3`} type="number" min={10} placeholder="Seconds" value={runtimeServices.provider_readiness_watcher_timeout_seconds} onChange={(e) => setRuntimeServices(prev => ({ ...prev, provider_readiness_watcher_timeout_seconds: Number(e.target.value) || 90 }))} />
                        </div>
                        <div className="grid md:grid-cols-4 gap-3 items-center">
                            <div className="text-sm font-medium text-slate-700 dark:text-slate-300">Watcher Batch Size</div>
                            <input className={`${inputClass} md:col-span-3`} type="number" min={1} max={1000} placeholder="Providers per tick" value={runtimeServices.provider_readiness_watcher_batch_size} onChange={(e) => setRuntimeServices(prev => ({ ...prev, provider_readiness_watcher_batch_size: Number(e.target.value) || 200 }))} />
                        </div>
                    </div>
                    <button className={primaryButtonClass} disabled={saving} onClick={() => saveStep('runtime_services')}>
                        {saving ? 'Saving...' : 'Save Runtime Services'}
                    </button>
                </div>
            )}

            {activeStep === 'sso' && (
                <div className={cardClass}>
                    <h2 className="text-base font-semibold text-slate-900 dark:text-white">SSO</h2>
                    <div className="flex gap-4 text-sm text-slate-700 dark:text-slate-300">
                        <label><input type="radio" checked={ssoType === 'oidc'} onChange={() => setSsoType('oidc')} /> <span className="ml-1">OIDC</span></label>
                        <label><input type="radio" checked={ssoType === 'saml'} onChange={() => setSsoType('saml')} /> <span className="ml-1">SAML</span></label>
                    </div>

                    {ssoType === 'oidc' ? (
                        <div className="grid md:grid-cols-2 gap-3">
                            <input className={inputClass} placeholder="Name" value={oidc.name} onChange={(e) => setOidc(prev => ({ ...prev, name: e.target.value }))} />
                            <input className={inputClass} placeholder="Issuer" value={oidc.issuer} onChange={(e) => setOidc(prev => ({ ...prev, issuer: e.target.value }))} />
                            <input className={inputClass} placeholder="Client ID" value={oidc.client_id} onChange={(e) => setOidc(prev => ({ ...prev, client_id: e.target.value }))} />
                            <input className={inputClass} placeholder="Client Secret" value={oidc.client_secret} onChange={(e) => setOidc(prev => ({ ...prev, client_secret: e.target.value }))} />
                            <input className={`${inputClass} md:col-span-2`} placeholder="Authorization URL" value={oidc.authorization_url} onChange={(e) => setOidc(prev => ({ ...prev, authorization_url: e.target.value }))} />
                            <input className={`${inputClass} md:col-span-2`} placeholder="Token URL" value={oidc.token_url} onChange={(e) => setOidc(prev => ({ ...prev, token_url: e.target.value }))} />
                            <input className={`${inputClass} md:col-span-2`} placeholder="UserInfo URL" value={oidc.userinfo_url} onChange={(e) => setOidc(prev => ({ ...prev, userinfo_url: e.target.value }))} />
                            <input className={`${inputClass} md:col-span-2`} placeholder="JWKS URL" value={oidc.jwks_url} onChange={(e) => setOidc(prev => ({ ...prev, jwks_url: e.target.value }))} />
                        </div>
                    ) : (
                        <div className="grid md:grid-cols-2 gap-3">
                            <input className={inputClass} placeholder="Name" value={saml.name} onChange={(e) => setSaml(prev => ({ ...prev, name: e.target.value }))} />
                            <input className={inputClass} placeholder="Entity ID" value={saml.entity_id} onChange={(e) => setSaml(prev => ({ ...prev, entity_id: e.target.value }))} />
                            <input className={`${inputClass} md:col-span-2`} placeholder="SSO URL" value={saml.sso_url} onChange={(e) => setSaml(prev => ({ ...prev, sso_url: e.target.value }))} />
                            <input className={`${inputClass} md:col-span-2`} placeholder="SLO URL" value={saml.slo_url} onChange={(e) => setSaml(prev => ({ ...prev, slo_url: e.target.value }))} />
                            <textarea className={`${textareaClass} md:col-span-2 min-h-24`} placeholder="Certificate (PEM)" value={saml.certificate} onChange={(e) => setSaml(prev => ({ ...prev, certificate: e.target.value }))} />
                        </div>
                    )}
                    <button className={primaryButtonClass} disabled={saving} onClick={() => saveStep('sso')}>
                        {saving ? 'Saving...' : `Save ${ssoType.toUpperCase()} Provider`}
                    </button>
                </div>
            )}

            {activeStep === 'review' && (
                <div className={cardClass}>
                    <h2 className="text-base font-semibold text-slate-900 dark:text-white">Review And Finalize</h2>
                    <p className="text-sm text-slate-700 dark:text-slate-300">
                        Completed: {completedSteps.length ? completedSteps.join(', ') : 'none'}
                    </p>
                    <p className="text-sm text-slate-700 dark:text-slate-300">
                        Missing: {missingSteps.length ? missingSteps.join(', ') : 'none'}
                    </p>
                    <p className="text-xs text-slate-500 dark:text-slate-400">
                        Clicking complete validates and saves all sections in one request. Optional steps can be configured later in admin settings.
                    </p>
                    <div>
                        <button
                            onClick={() => setShowRestartConfirm(true)}
                            disabled={isCompleteDisabled}
                            title={saving ? 'Wait for active save to finish' : undefined}
                            className="rounded bg-amber-700 px-4 py-2 text-white hover:bg-amber-800 disabled:opacity-60 disabled:cursor-not-allowed transition-colors"
                        >
                            {finalizing ? 'Finalizing...' : 'Complete Setup And Restart'}
                        </button>
                        {missingRequiredSteps.length > 0 && (
                            <p className="mt-2 text-sm text-amber-700 dark:text-amber-300">
                                Complete setup is blocked. Missing required steps: {missingRequiredSteps.join(', ')}.
                            </p>
                        )}
                        {finalizing && (
                            <p className="mt-2 text-sm text-slate-600 dark:text-slate-400">
                                Finalizing setup, please wait...
                            </p>
                        )}
                    </div>
                </div>
            )}

            {showRestartConfirm && (
                <div className="fixed inset-0 z-40 bg-black/50 dark:bg-black/70 flex items-center justify-center p-4">
                    <div className="w-full max-w-lg rounded-xl border border-slate-300 dark:border-slate-700 bg-white dark:bg-slate-900 p-5 shadow-xl">
                        <div className="flex items-start gap-3">
                            <div className="mt-0.5 h-6 w-6 rounded-full bg-amber-100 dark:bg-amber-900/40 text-amber-700 dark:text-amber-300 flex items-center justify-center text-sm font-bold">!</div>
                            <div className="space-y-2">
                                <h3 className="text-base font-semibold text-slate-900 dark:text-slate-100">Complete setup and restart now?</h3>
                                <p className="text-sm text-slate-700 dark:text-slate-300">
                                    This will persist setup configuration and trigger backend restart. The UI may be temporarily unavailable for up to 60 seconds.
                                </p>
                                <p className="text-sm text-slate-700 dark:text-slate-300">
                                    Keep this tab open. We will automatically check when the server is back.
                                </p>
                            </div>
                        </div>
                        <div className="mt-5 flex items-center justify-end gap-2">
                            <button
                                type="button"
                                onClick={() => setShowRestartConfirm(false)}
                                className="rounded border border-slate-300 dark:border-slate-600 bg-white dark:bg-slate-800 text-slate-800 dark:text-slate-100 px-4 py-2 text-sm hover:bg-slate-100 dark:hover:bg-slate-700 transition-colors"
                            >
                                Cancel
                            </button>
                            <button
                                type="button"
                                onClick={handleCompleteSetup}
                                disabled={finalizing || saving}
                                className="rounded bg-amber-700 px-4 py-2 text-sm text-white hover:bg-amber-800 disabled:opacity-60 disabled:cursor-not-allowed transition-colors"
                            >
                                {finalizing ? 'Finalizing...' : 'Yes, Complete And Restart'}
                            </button>
                        </div>
                    </div>
                </div>
            )}

            {showRestartProgress && (
                <div className="fixed inset-0 z-50 bg-slate-950/75 dark:bg-slate-950/85 flex items-center justify-center p-4">
                    <div className="w-full max-w-xl rounded-xl border border-slate-300 dark:border-slate-700 bg-white dark:bg-slate-900 p-6 shadow-xl">
                        <div className="flex items-center gap-3">
                            <div className="h-8 w-8 rounded-full border-2 border-slate-300 dark:border-slate-600 border-t-blue-600 animate-spin" />
                            <div>
                                <h3 className="text-base font-semibold text-slate-900 dark:text-slate-100">Restart in progress</h3>
                                <p className="text-sm text-slate-700 dark:text-slate-300">Checking server availability... attempt {restartAttempts}/24</p>
                            </div>
                        </div>

                        {!restartTimedOut && (
                            <p className="mt-4 text-sm text-slate-700 dark:text-slate-300">
                                We are waiting for the backend to restart and become reachable again.
                            </p>
                        )}

                        {restartTimedOut && (
                            <div className="mt-4 rounded-lg border border-amber-300 dark:border-amber-700 bg-amber-50 dark:bg-amber-900/30 p-3">
                                <p className="text-sm font-medium text-amber-900 dark:text-amber-200">Server has not recovered yet</p>
                                <p className="mt-1 text-sm text-amber-800 dark:text-amber-300">
                                    Check your backend process/service manager. Once restarted, click <span className="font-semibold">Check Again</span>.
                                </p>
                            </div>
                        )}

                        <div className="mt-5 flex items-center justify-end gap-2">
                            <button
                                type="button"
                                onClick={checkBackendNow}
                                className="rounded border border-blue-300 dark:border-blue-700 bg-blue-50 dark:bg-blue-900/30 text-blue-800 dark:text-blue-200 px-4 py-2 text-sm hover:bg-blue-100 dark:hover:bg-blue-900/50 transition-colors"
                            >
                                Check Again
                            </button>
                            <button
                                type="button"
                                onClick={() => setShowRestartProgress(false)}
                                className="rounded border border-slate-300 dark:border-slate-600 bg-white dark:bg-slate-800 text-slate-800 dark:text-slate-100 px-4 py-2 text-sm hover:bg-slate-100 dark:hover:bg-slate-700 transition-colors"
                            >
                                Close
                            </button>
                        </div>
                    </div>
                </div>
            )}
        </div>
    )
}

export default InitialSetupPage
