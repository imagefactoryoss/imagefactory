import { useCanManageAdmin } from '@/hooks/useAccess'
import { api } from '@/services/api'
import React, { useEffect, useState } from 'react'
import toast from 'react-hot-toast'

type LDAPProvider = {
    config_key: string
    provider_name?: string
    provider_type?: string
    host: string
    port: number
    base_dn?: string
    user_search_base?: string
    group_search_base?: string
    bind_dn?: string
    bind_password?: string
    user_filter?: string
    group_filter?: string
    start_tls?: boolean
    ssl?: boolean
    enabled: boolean
}

type SSOProvider = {
    id: string
    type: 'saml' | 'oidc'
    name: string
    enabled: boolean
}

type AddMode = 'ldap' | 'saml' | 'oidc' | null

const slugify = (value: string) => value.toLowerCase().replace(/[^a-z0-9]+/g, '_').replace(/^_+|_+$/g, '')

const AuthProvidersPage: React.FC = () => {
    const canManageAdmin = useCanManageAdmin()
    const [loading, setLoading] = useState(true)
    const [ldapProviders, setLdapProviders] = useState<LDAPProvider[]>([])
    const [ssoProviders, setSsoProviders] = useState<SSOProvider[]>([])
    const [addMode, setAddMode] = useState<AddMode>(null)
    const [submitting, setSubmitting] = useState(false)
    const [editingLdapKey, setEditingLdapKey] = useState<string | null>(null)

    const [ldapForm, setLdapForm] = useState({
        provider_name: 'Active Directory',
        provider_type: 'active_directory',
        host: '',
        port: 389,
        base_dn: '',
        user_filter: '(uid=%s)',
        group_filter: '(member=%s)',
        enabled: false,
    })

    const [ldapEditForm, setLdapEditForm] = useState({
        provider_name: '',
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
        enabled: false,
    })

    const [oidcForm, setOidcForm] = useState({
        name: 'PingFed',
        issuer: '',
        client_id: '',
        client_secret: '',
        authorization_url: '',
        token_url: '',
        userinfo_url: '',
        jwks_url: '',
        redirect_uris: 'http://localhost:3000/auth/callback',
        enabled: false,
    })

    const [samlForm, setSamlForm] = useState({
        name: '',
        entity_id: '',
        sso_url: '',
        slo_url: '',
        certificate: '',
        private_key: '',
        enabled: false,
    })

    const inputClass = 'w-full rounded-md border border-slate-300 dark:border-slate-600 bg-white dark:bg-slate-800 text-slate-900 dark:text-slate-100 px-3 py-2 text-sm placeholder-slate-400 dark:placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500'
    const cardClass = 'rounded-xl border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-900 p-5'

    const resetLdapForm = () => {
        setLdapForm({
            provider_name: 'Active Directory',
            provider_type: 'active_directory',
            host: '',
            port: 389,
            base_dn: '',
            user_filter: '(uid=%s)',
            group_filter: '(member=%s)',
            enabled: false,
        })
    }

    const resetOidcForm = () => {
        setOidcForm({
            name: 'PingFed',
            issuer: '',
            client_id: '',
            client_secret: '',
            authorization_url: '',
            token_url: '',
            userinfo_url: '',
            jwks_url: '',
            redirect_uris: 'http://localhost:3000/auth/callback',
            enabled: false,
        })
    }

    const resetSamlForm = () => {
        setSamlForm({
            name: '',
            entity_id: '',
            sso_url: '',
            slo_url: '',
            certificate: '',
            private_key: '',
            enabled: false,
        })
    }

    const closeAddDrawer = () => {
        setAddMode(null)
        setSubmitting(false)
    }

    const loadData = async () => {
        try {
            setLoading(true)
            const [ldapResp, ssoResp] = await Promise.all([
                api.get('/admin/ldap'),
                api.get('/sso/configuration'),
            ])

            setLdapProviders((ldapResp.data?.configs || []).map((c: any) => ({
                config_key: c.config_key,
                provider_name: c.provider_name,
                provider_type: c.provider_type,
                host: c.host,
                port: c.port,
                base_dn: c.base_dn,
                user_search_base: c.user_search_base,
                group_search_base: c.group_search_base,
                bind_dn: c.bind_dn,
                bind_password: c.bind_password,
                user_filter: c.user_filter,
                group_filter: c.group_filter,
                start_tls: c.start_tls,
                ssl: c.ssl,
                enabled: !!c.enabled,
            })))

            const providers: SSOProvider[] = []
            for (const p of ssoResp.data?.saml_providers || []) {
                providers.push({ id: p.id, type: 'saml', name: p.name, enabled: p.enabled !== false })
            }
            for (const p of ssoResp.data?.oidc_providers || []) {
                providers.push({ id: p.id, type: 'oidc', name: p.name, enabled: p.enabled !== false })
            }
            setSsoProviders(providers)
        } catch {
            toast.error('Failed to load auth providers')
        } finally {
            setLoading(false)
        }
    }

    useEffect(() => {
        loadData()
    }, [])

    const createLDAP = async () => {
        if (!canManageAdmin) {
            toast.error('Read-only mode.')
            return
        }
        try {
            setSubmitting(true)
            const key = `ldap_${slugify(ldapForm.provider_name)}`
            await api.post('/admin/ldap', {
                config_key: key,
                provider_name: ldapForm.provider_name,
                provider_type: ldapForm.provider_type,
                host: ldapForm.host,
                port: ldapForm.port,
                base_dn: ldapForm.base_dn,
                user_filter: ldapForm.user_filter,
                group_filter: ldapForm.group_filter,
                enabled: ldapForm.enabled,
            })
            toast.success('LDAP provider added')
            resetLdapForm()
            closeAddDrawer()
            await loadData()
        } catch {
            toast.error('Failed to add LDAP provider')
        } finally {
            setSubmitting(false)
        }
    }

    const toggleLDAP = async (provider: LDAPProvider, enabled: boolean) => {
        if (!canManageAdmin) {
            toast.error('Read-only mode.')
            return
        }
        try {
            await api.put(`/admin/ldap/${provider.config_key}`, { enabled })
            toast.success(`LDAP provider ${enabled ? 'activated' : 'deactivated'}`)
            await loadData()
        } catch {
            toast.error('Failed to update LDAP provider')
        }
    }

    const startEditLDAP = (provider: LDAPProvider) => {
        if (!canManageAdmin) {
            toast.error('Read-only mode.')
            return
        }
        setEditingLdapKey(provider.config_key)
        setLdapEditForm({
            provider_name: provider.provider_name || '',
            provider_type: provider.provider_type || 'active_directory',
            host: provider.host || '',
            port: provider.port || 389,
            base_dn: provider.base_dn || '',
            user_search_base: provider.user_search_base || '',
            group_search_base: provider.group_search_base || '',
            bind_dn: provider.bind_dn || '',
            bind_password: provider.bind_password || '',
            user_filter: provider.user_filter || '(uid=%s)',
            group_filter: provider.group_filter || '(member=%s)',
            start_tls: !!provider.start_tls,
            ssl: !!provider.ssl,
            enabled: !!provider.enabled,
        })
    }

    const cancelEditLDAP = () => {
        setEditingLdapKey(null)
    }

    const saveEditLDAP = async () => {
        if (!editingLdapKey) return
        if (!canManageAdmin) {
            toast.error('Read-only mode.')
            return
        }
        try {
            setSubmitting(true)
            await api.put(`/admin/ldap/${editingLdapKey}`, {
                provider_name: ldapEditForm.provider_name,
                provider_type: ldapEditForm.provider_type,
                host: ldapEditForm.host,
                port: ldapEditForm.port,
                base_dn: ldapEditForm.base_dn,
                user_search_base: ldapEditForm.user_search_base,
                group_search_base: ldapEditForm.group_search_base,
                bind_dn: ldapEditForm.bind_dn,
                bind_password: ldapEditForm.bind_password,
                user_filter: ldapEditForm.user_filter,
                group_filter: ldapEditForm.group_filter,
                start_tls: ldapEditForm.start_tls,
                ssl: ldapEditForm.ssl,
                enabled: ldapEditForm.enabled,
            })
            toast.success('LDAP provider updated')
            setEditingLdapKey(null)
            await loadData()
        } catch {
            toast.error('Failed to update LDAP provider')
        } finally {
            setSubmitting(false)
        }
    }

    const createOIDC = async () => {
        if (!canManageAdmin) {
            toast.error('Read-only mode.')
            return
        }
        try {
            setSubmitting(true)
            await api.post('/sso/oidc/providers', {
                ...oidcForm,
                redirect_uris: oidcForm.redirect_uris.split(',').map(v => v.trim()).filter(Boolean),
                scopes: ['openid', 'profile', 'email'],
                response_types: ['code'],
                grant_types: ['authorization_code'],
                enabled: oidcForm.enabled,
            })
            toast.success('OIDC provider added')
            resetOidcForm()
            closeAddDrawer()
            await loadData()
        } catch {
            toast.error('Failed to add OIDC provider')
        } finally {
            setSubmitting(false)
        }
    }

    const createSAML = async () => {
        if (!canManageAdmin) {
            toast.error('Read-only mode.')
            return
        }
        try {
            setSubmitting(true)
            await api.post('/sso/saml/providers', {
                ...samlForm,
                position: 'idp',
                attributes: {},
                enabled: samlForm.enabled,
            })
            toast.success('SAML provider added')
            resetSamlForm()
            closeAddDrawer()
            await loadData()
        } catch {
            toast.error('Failed to add SAML provider')
        } finally {
            setSubmitting(false)
        }
    }

    const toggleSSO = async (provider: SSOProvider, enabled: boolean) => {
        if (!canManageAdmin) {
            toast.error('Read-only mode.')
            return
        }
        try {
            await api.patch(`/sso/${provider.type}/providers/${provider.id}/status`, { enabled })
            toast.success(`SSO provider ${enabled ? 'activated' : 'deactivated'}`)
            await loadData()
        } catch {
            toast.error('Failed to update SSO provider')
        }
    }

    const ssoSamlProviders = ssoProviders.filter(p => p.type === 'saml')
    const ssoOidcProviders = ssoProviders.filter(p => p.type === 'oidc')
    const drawerOpen = addMode !== null

    if (loading) {
        return (
            <div className="p-8 text-slate-700 dark:text-slate-300">
                Loading authentication providers...
            </div>
        )
    }

    return (
        <>
            <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8 space-y-6">
                <div>
                    <h1 className="text-3xl font-bold text-slate-900 dark:text-slate-100">Authentication Providers</h1>
                    <p className="mt-2 text-slate-600 dark:text-slate-300">
                        Configure LDAP and SSO provider catalogs. Activate exactly one provider per category.
                    </p>
                </div>

                <div className={`${cardClass} flex flex-wrap items-center gap-3`}>
                    <span className="text-sm text-slate-700 dark:text-slate-300">Quick Add:</span>
                    {canManageAdmin && (
                        <button onClick={() => setAddMode('ldap')} className="rounded-md px-3 py-2 text-sm transition-colors bg-blue-600 hover:bg-blue-700 text-white">+ LDAP Provider</button>
                    )}
                    {canManageAdmin && (
                        <button onClick={() => setAddMode('saml')} className="rounded-md px-3 py-2 text-sm transition-colors bg-blue-600 hover:bg-blue-700 text-white">+ SAML Provider</button>
                    )}
                    {canManageAdmin && (
                        <button onClick={() => setAddMode('oidc')} className="rounded-md px-3 py-2 text-sm transition-colors bg-blue-600 hover:bg-blue-700 text-white">+ OIDC Provider</button>
                    )}
                </div>

                {!canManageAdmin && (
                    <div className="rounded-md border border-amber-300 bg-amber-50 px-4 py-3 text-sm text-amber-900 dark:border-amber-700 dark:bg-amber-900/20 dark:text-amber-200">
                        Read-only mode: auth provider add/edit/activate actions are hidden for System Administrator Viewer.
                    </div>
                )}

                <div className="grid lg:grid-cols-2 gap-6">
                    <div className={cardClass}>
                        <div className="flex items-center justify-between">
                            <h2 className="text-xl font-semibold text-slate-900 dark:text-slate-100">LDAP Providers</h2>
                            <span className="text-xs rounded-full bg-slate-100 dark:bg-slate-800 text-slate-700 dark:text-slate-300 px-2 py-1">{ldapProviders.length} configured</span>
                        </div>
                        <div className="mt-4 space-y-3">
                            {ldapProviders.length === 0 && (
                                <div className="rounded-md border border-dashed border-slate-300 dark:border-slate-600 p-4 text-sm text-slate-600 dark:text-slate-400">
                                    No LDAP providers configured yet.
                                </div>
                            )}
                            {ldapProviders.map(p => (
                                <div key={p.config_key} className="rounded-lg border border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-800/40 p-3 space-y-3">
                                    <div className="flex items-center justify-between gap-3">
                                        <div>
                                            <div className="font-medium text-slate-900 dark:text-slate-100">
                                                {p.provider_name || p.config_key} <span className="text-xs text-slate-500 dark:text-slate-400">({p.provider_type || 'ldap'})</span>
                                                {p.enabled && <span className="ml-2 text-xs bg-emerald-100 dark:bg-emerald-900/40 text-emerald-800 dark:text-emerald-300 px-2 py-0.5 rounded">Active</span>}
                                            </div>
                                            <div className="text-sm text-slate-600 dark:text-slate-400">{p.host}:{p.port}</div>
                                        </div>
                                        <div className="flex items-center gap-2">
                                            {canManageAdmin && (
                                                <button className="px-3 py-1.5 rounded text-sm border border-slate-300 dark:border-slate-600 bg-white dark:bg-slate-800 text-slate-700 dark:text-slate-200 hover:bg-slate-100 dark:hover:bg-slate-700" onClick={() => startEditLDAP(p)}>Edit</button>
                                            )}
                                            {canManageAdmin && (
                                                <button className={`px-3 py-1.5 rounded text-sm text-white ${p.enabled ? 'bg-amber-600 hover:bg-amber-700' : 'bg-emerald-600 hover:bg-emerald-700'}`} onClick={() => toggleLDAP(p, !p.enabled)}>
                                                    {p.enabled ? 'Deactivate' : 'Activate'}
                                                </button>
                                            )}
                                        </div>
                                    </div>

                                    {canManageAdmin && editingLdapKey === p.config_key && (
                                        <div className="rounded-lg border border-slate-300 dark:border-slate-600 bg-white dark:bg-slate-900 p-3">
                                            <div className="grid md:grid-cols-3 gap-3">
                                                <input className={inputClass} placeholder="Provider name" value={ldapEditForm.provider_name} onChange={e => setLdapEditForm(prev => ({ ...prev, provider_name: e.target.value }))} />
                                                <select className={inputClass} value={ldapEditForm.provider_type} onChange={e => setLdapEditForm(prev => ({ ...prev, provider_type: e.target.value }))}>
                                                    <option value="active_directory">Active Directory</option>
                                                    <option value="openldap">OpenLDAP</option>
                                                    <option value="generic">Generic LDAP</option>
                                                </select>
                                                <label className="flex items-center gap-2 text-sm text-slate-700 dark:text-slate-300">
                                                    <input type="checkbox" checked={ldapEditForm.enabled} onChange={e => setLdapEditForm(prev => ({ ...prev, enabled: e.target.checked }))} />
                                                    Enabled
                                                </label>
                                                <input className={inputClass} placeholder="Host" value={ldapEditForm.host} onChange={e => setLdapEditForm(prev => ({ ...prev, host: e.target.value }))} />
                                                <input className={inputClass} type="number" placeholder="Port" value={ldapEditForm.port} onChange={e => setLdapEditForm(prev => ({ ...prev, port: Number(e.target.value) || 389 }))} />
                                                <input className={inputClass} placeholder="Base DN" value={ldapEditForm.base_dn} onChange={e => setLdapEditForm(prev => ({ ...prev, base_dn: e.target.value }))} />
                                                <input className={inputClass} placeholder="User Search Base" value={ldapEditForm.user_search_base} onChange={e => setLdapEditForm(prev => ({ ...prev, user_search_base: e.target.value }))} />
                                                <input className={inputClass} placeholder="Group Search Base" value={ldapEditForm.group_search_base} onChange={e => setLdapEditForm(prev => ({ ...prev, group_search_base: e.target.value }))} />
                                                <input className={inputClass} placeholder="Bind DN" value={ldapEditForm.bind_dn} onChange={e => setLdapEditForm(prev => ({ ...prev, bind_dn: e.target.value }))} />
                                                <input className={inputClass} placeholder="Bind Password" value={ldapEditForm.bind_password} onChange={e => setLdapEditForm(prev => ({ ...prev, bind_password: e.target.value }))} />
                                                <input className={inputClass} placeholder="User Filter" value={ldapEditForm.user_filter} onChange={e => setLdapEditForm(prev => ({ ...prev, user_filter: e.target.value }))} />
                                                <input className={inputClass} placeholder="Group Filter" value={ldapEditForm.group_filter} onChange={e => setLdapEditForm(prev => ({ ...prev, group_filter: e.target.value }))} />
                                                <div className="flex items-center gap-4 text-sm text-slate-700 dark:text-slate-300 md:col-span-3">
                                                    <label className="flex items-center gap-2">
                                                        <input type="checkbox" checked={ldapEditForm.start_tls} onChange={e => setLdapEditForm(prev => ({ ...prev, start_tls: e.target.checked }))} />
                                                        StartTLS
                                                    </label>
                                                    <label className="flex items-center gap-2">
                                                        <input type="checkbox" checked={ldapEditForm.ssl} onChange={e => setLdapEditForm(prev => ({ ...prev, ssl: e.target.checked }))} />
                                                        SSL
                                                    </label>
                                                </div>
                                            </div>
                                            <div className="mt-3 flex items-center justify-end gap-2">
                                                <button className="px-3 py-1.5 rounded text-sm border border-slate-300 dark:border-slate-600 bg-white dark:bg-slate-800 text-slate-700 dark:text-slate-200 hover:bg-slate-100 dark:hover:bg-slate-700" onClick={cancelEditLDAP}>Cancel</button>
                                                <button className="px-3 py-1.5 rounded text-sm bg-blue-600 hover:bg-blue-700 text-white disabled:opacity-60 disabled:cursor-not-allowed" onClick={saveEditLDAP} disabled={submitting}>
                                                    {submitting ? 'Saving...' : 'Save'}
                                                </button>
                                            </div>
                                        </div>
                                    )}
                                </div>
                            ))}
                        </div>
                    </div>

                    <div className={cardClass}>
                        <div className="flex items-center justify-between">
                            <h2 className="text-xl font-semibold text-slate-900 dark:text-slate-100">SSO Providers</h2>
                            <span className="text-xs rounded-full bg-slate-100 dark:bg-slate-800 text-slate-700 dark:text-slate-300 px-2 py-1">{ssoProviders.length} configured</span>
                        </div>

                        <div className="mt-4 space-y-4">
                            <p className="text-xs text-slate-500 dark:text-slate-400">
                                SSO supports create/list/activate from this screen. Inline editing for existing SSO providers will be enabled once update endpoints are available.
                            </p>
                            <div>
                                <h3 className="text-sm font-semibold text-slate-800 dark:text-slate-200 mb-2">SAML</h3>
                                <div className="space-y-2">
                                    {ssoSamlProviders.length === 0 && (
                                        <div className="rounded-md border border-dashed border-slate-300 dark:border-slate-600 p-3 text-sm text-slate-600 dark:text-slate-400">
                                            No SAML providers configured.
                                        </div>
                                    )}
                                    {ssoSamlProviders.map(p => (
                                        <div key={`${p.type}-${p.id}`} className="rounded-lg border border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-800/40 p-3 flex items-center justify-between gap-3">
                                            <div className="font-medium text-slate-900 dark:text-slate-100">
                                                {p.name}
                                                {p.enabled && <span className="ml-2 text-xs bg-emerald-100 dark:bg-emerald-900/40 text-emerald-800 dark:text-emerald-300 px-2 py-0.5 rounded">Active</span>}
                                            </div>
                                            {canManageAdmin && (
                                                <button className={`px-3 py-1.5 rounded text-sm text-white ${p.enabled ? 'bg-amber-600 hover:bg-amber-700' : 'bg-emerald-600 hover:bg-emerald-700'}`} onClick={() => toggleSSO(p, !p.enabled)}>
                                                    {p.enabled ? 'Deactivate' : 'Activate'}
                                                </button>
                                            )}
                                        </div>
                                    ))}
                                </div>
                            </div>

                            <div>
                                <h3 className="text-sm font-semibold text-slate-800 dark:text-slate-200 mb-2">OIDC</h3>
                                <div className="space-y-2">
                                    {ssoOidcProviders.length === 0 && (
                                        <div className="rounded-md border border-dashed border-slate-300 dark:border-slate-600 p-3 text-sm text-slate-600 dark:text-slate-400">
                                            No OIDC providers configured.
                                        </div>
                                    )}
                                    {ssoOidcProviders.map(p => (
                                        <div key={`${p.type}-${p.id}`} className="rounded-lg border border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-800/40 p-3 flex items-center justify-between gap-3">
                                            <div className="font-medium text-slate-900 dark:text-slate-100">
                                                {p.name}
                                                {p.enabled && <span className="ml-2 text-xs bg-emerald-100 dark:bg-emerald-900/40 text-emerald-800 dark:text-emerald-300 px-2 py-0.5 rounded">Active</span>}
                                            </div>
                                            {canManageAdmin && (
                                                <button className={`px-3 py-1.5 rounded text-sm text-white ${p.enabled ? 'bg-amber-600 hover:bg-amber-700' : 'bg-emerald-600 hover:bg-emerald-700'}`} onClick={() => toggleSSO(p, !p.enabled)}>
                                                    {p.enabled ? 'Deactivate' : 'Activate'}
                                                </button>
                                            )}
                                        </div>
                                    ))}
                                </div>
                            </div>
                        </div>
                    </div>
                </div>
            </div>

            {canManageAdmin && drawerOpen && (
                <div className="fixed inset-0 z-50">
                    <div className="absolute inset-0 bg-slate-900/50 dark:bg-slate-950/70" onClick={closeAddDrawer} />
                    <aside className="absolute right-0 top-0 h-full w-full max-w-2xl border-l border-slate-300 dark:border-slate-700 bg-white dark:bg-slate-900 shadow-2xl overflow-y-auto">
                        <div className="sticky top-0 z-10 flex items-center justify-between px-5 py-4 border-b border-slate-200 dark:border-slate-700 bg-white/95 dark:bg-slate-900/95 backdrop-blur">
                            <h2 className="text-lg font-semibold text-slate-900 dark:text-slate-100">
                                {addMode === 'ldap' ? 'Add LDAP Provider' : addMode === 'saml' ? 'Add SAML Provider' : 'Add OIDC Provider'}
                            </h2>
                            <button onClick={closeAddDrawer} className="rounded-md border border-slate-300 dark:border-slate-600 bg-white dark:bg-slate-800 px-3 py-1.5 text-sm text-slate-700 dark:text-slate-200 hover:bg-slate-100 dark:hover:bg-slate-700">
                                Close
                            </button>
                        </div>

                        <div className="p-5 space-y-4">
                            {addMode === 'ldap' && (
                                <div className="grid md:grid-cols-2 gap-3">
                                    <input className={inputClass} placeholder="Provider name" value={ldapForm.provider_name} onChange={e => setLdapForm(prev => ({ ...prev, provider_name: e.target.value }))} />
                                    <select className={inputClass} value={ldapForm.provider_type} onChange={e => setLdapForm(prev => ({ ...prev, provider_type: e.target.value }))}>
                                        <option value="active_directory">Active Directory</option>
                                        <option value="openldap">OpenLDAP</option>
                                        <option value="generic">Generic LDAP</option>
                                    </select>
                                    <input className={inputClass} placeholder="Host" value={ldapForm.host} onChange={e => setLdapForm(prev => ({ ...prev, host: e.target.value }))} />
                                    <input className={inputClass} type="number" placeholder="Port" value={ldapForm.port} onChange={e => setLdapForm(prev => ({ ...prev, port: Number(e.target.value) || 389 }))} />
                                    <input className={`${inputClass} md:col-span-2`} placeholder="Base DN" value={ldapForm.base_dn} onChange={e => setLdapForm(prev => ({ ...prev, base_dn: e.target.value }))} />
                                    <input className={inputClass} placeholder="User filter" value={ldapForm.user_filter} onChange={e => setLdapForm(prev => ({ ...prev, user_filter: e.target.value }))} />
                                    <input className={inputClass} placeholder="Group filter" value={ldapForm.group_filter} onChange={e => setLdapForm(prev => ({ ...prev, group_filter: e.target.value }))} />
                                    <label className="md:col-span-2 flex items-center gap-2 text-sm text-slate-700 dark:text-slate-300">
                                        <input type="checkbox" checked={ldapForm.enabled} onChange={e => setLdapForm(prev => ({ ...prev, enabled: e.target.checked }))} />
                                        Activate on create
                                    </label>
                                    <div className="md:col-span-2">
                                        <button disabled={submitting} className="rounded-md bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 text-sm disabled:opacity-60 disabled:cursor-not-allowed" onClick={createLDAP}>
                                            {submitting ? 'Adding...' : 'Add LDAP Provider'}
                                        </button>
                                    </div>
                                </div>
                            )}

                            {addMode === 'saml' && (
                                <div className="grid md:grid-cols-2 gap-3">
                                    <input className={inputClass} placeholder="Name" value={samlForm.name} onChange={e => setSamlForm(prev => ({ ...prev, name: e.target.value }))} />
                                    <input className={inputClass} placeholder="Entity ID (optional)" value={samlForm.entity_id} onChange={e => setSamlForm(prev => ({ ...prev, entity_id: e.target.value }))} />
                                    <input className={inputClass} placeholder="SSO URL" value={samlForm.sso_url} onChange={e => setSamlForm(prev => ({ ...prev, sso_url: e.target.value }))} />
                                    <input className={inputClass} placeholder="SLO URL" value={samlForm.slo_url} onChange={e => setSamlForm(prev => ({ ...prev, slo_url: e.target.value }))} />
                                    <input className={`${inputClass} md:col-span-2`} placeholder="Private Key (optional)" value={samlForm.private_key} onChange={e => setSamlForm(prev => ({ ...prev, private_key: e.target.value }))} />
                                    <textarea className="md:col-span-2 border border-slate-300 dark:border-slate-600 rounded-md bg-white dark:bg-slate-800 text-slate-900 dark:text-slate-100 px-3 py-2 text-sm min-h-24 placeholder-slate-400 dark:placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500" placeholder="Certificate (PEM format)" value={samlForm.certificate} onChange={e => setSamlForm(prev => ({ ...prev, certificate: e.target.value }))} />
                                    <label className="md:col-span-2 flex items-center gap-2 text-sm text-slate-700 dark:text-slate-300">
                                        <input type="checkbox" checked={samlForm.enabled} onChange={e => setSamlForm(prev => ({ ...prev, enabled: e.target.checked }))} />
                                        Activate on create
                                    </label>
                                    <div className="md:col-span-2">
                                        <button disabled={submitting} className="rounded-md bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 text-sm disabled:opacity-60 disabled:cursor-not-allowed" onClick={createSAML}>
                                            {submitting ? 'Adding...' : 'Add SAML Provider'}
                                        </button>
                                    </div>
                                </div>
                            )}

                            {addMode === 'oidc' && (
                                <div className="grid md:grid-cols-2 gap-3">
                                    <input className={inputClass} placeholder="Name" value={oidcForm.name} onChange={e => setOidcForm(prev => ({ ...prev, name: e.target.value }))} />
                                    <input className={inputClass} placeholder="Issuer URL" value={oidcForm.issuer} onChange={e => setOidcForm(prev => ({ ...prev, issuer: e.target.value }))} />
                                    <input className={inputClass} placeholder="Client ID" value={oidcForm.client_id} onChange={e => setOidcForm(prev => ({ ...prev, client_id: e.target.value }))} />
                                    <input className={inputClass} placeholder="Client Secret" value={oidcForm.client_secret} onChange={e => setOidcForm(prev => ({ ...prev, client_secret: e.target.value }))} />
                                    <input className={inputClass} placeholder="Authorization URL" value={oidcForm.authorization_url} onChange={e => setOidcForm(prev => ({ ...prev, authorization_url: e.target.value }))} />
                                    <input className={inputClass} placeholder="Token URL" value={oidcForm.token_url} onChange={e => setOidcForm(prev => ({ ...prev, token_url: e.target.value }))} />
                                    <input className={inputClass} placeholder="UserInfo URL" value={oidcForm.userinfo_url} onChange={e => setOidcForm(prev => ({ ...prev, userinfo_url: e.target.value }))} />
                                    <input className={inputClass} placeholder="JWKS URL" value={oidcForm.jwks_url} onChange={e => setOidcForm(prev => ({ ...prev, jwks_url: e.target.value }))} />
                                    <input className={`${inputClass} md:col-span-2`} placeholder="Redirect URIs (comma-separated)" value={oidcForm.redirect_uris} onChange={e => setOidcForm(prev => ({ ...prev, redirect_uris: e.target.value }))} />
                                    <label className="md:col-span-2 flex items-center gap-2 text-sm text-slate-700 dark:text-slate-300">
                                        <input type="checkbox" checked={oidcForm.enabled} onChange={e => setOidcForm(prev => ({ ...prev, enabled: e.target.checked }))} />
                                        Activate on create
                                    </label>
                                    <div className="md:col-span-2">
                                        <button disabled={submitting} className="rounded-md bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 text-sm disabled:opacity-60 disabled:cursor-not-allowed" onClick={createOIDC}>
                                            {submitting ? 'Adding...' : 'Add OIDC Provider'}
                                        </button>
                                    </div>
                                </div>
                            )}
                        </div>
                    </aside>
                </div>
            )}
        </>
    )
}

export default AuthProvidersPage
