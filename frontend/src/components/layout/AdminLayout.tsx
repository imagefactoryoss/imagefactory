import { useConfirmDialog } from '@/context/ConfirmDialogContext'
import { useRefresh } from '@/context/RefreshContext'
import { useDashboardPath } from '@/hooks/useAccess'
import { adminService } from '@/services/adminService'
import { authService } from '@/services/authService'
import { profileService } from '@/services/profileService'
import { useAuthStore } from '@/store/auth'
import { useTenantStore } from '@/store/tenant'
import { useThemeStore } from '@/store/theme'
import React, { useEffect, useState } from 'react'
import { Link, useLocation, useNavigate } from 'react-router-dom'
import PostLoginContextSelector from '../auth/PostLoginContextSelector'
import CurrentTenantDisplay from '../CurrentTenantDisplay'
import UserNotificationCenter from '../notifications/UserNotificationCenter'
import { TokenExpirationWarning } from '../TokenExpirationWarning'

interface AdminLayoutProps {
    children: React.ReactNode
}

const APP_VERSION = import.meta.env.VITE_APP_VERSION || 'dev'

const AdminLayout: React.FC<AdminLayoutProps> = ({ children }) => {
    const { user, logout, avatar, canAccessAdmin, setupRequired } = useAuthStore()
    const { userTenants } = useTenantStore()
    const { isDark, toggleTheme } = useThemeStore()
    const { isRefreshing, triggerRefresh } = useRefresh()
    const confirmDialog = useConfirmDialog()
    const location = useLocation()
    const navigate = useNavigate()
    const dashboardPath = useDashboardPath()
    const [sidebarOpen, setSidebarOpen] = useState(true)
    const [showProfileMenu, setShowProfileMenu] = useState(false)
    const [showContextSelector, setShowContextSelector] = useState(false)
    const [expandedSections, setExpandedSections] = useState<Set<string>>(new Set())
    const [pendingDetectorRuleSuggestions, setPendingDetectorRuleSuggestions] = useState(0)
    const isAdmin = !!canAccessAdmin

    useEffect(() => {
        if (setupRequired && location.pathname !== '/admin/setup') {
            navigate('/admin/setup', { replace: true })
        }
    }, [setupRequired, location.pathname, navigate])

    useEffect(() => {
        if (!isAdmin || setupRequired) return

        let cancelled = false
        const loadPendingDetectorSuggestions = async () => {
            try {
                const response = await adminService.getSREDetectorRuleSuggestions({
                    status: 'pending',
                    limit: 100,
                    offset: 0,
                })
                if (!cancelled) {
                    setPendingDetectorRuleSuggestions((response.suggestions || []).length)
                }
            } catch {
                if (!cancelled) {
                    setPendingDetectorRuleSuggestions(0)
                }
            }
        }

        void loadPendingDetectorSuggestions()
        return () => {
            cancelled = true
        }
    }, [isAdmin, location.pathname, setupRequired])

    const renderNavLabel = (name: string, href: string, active: boolean) => (
        <div className="flex items-center gap-2 font-medium text-sm">
            <span>{name}</span>
            {href === '/admin/operations/sre-smart-bot/detector-rules' && pendingDetectorRuleSuggestions > 0 ? (
                <span className={`inline-flex min-w-[1.5rem] items-center justify-center rounded-full border px-1.5 py-0.5 text-[10px] font-semibold ${active
                        ? 'border-blue-200/70 bg-blue-500/30 text-white'
                        : 'border-amber-300 bg-amber-100 text-amber-900 dark:border-amber-700 dark:bg-amber-900/40 dark:text-amber-200'
                    }`}>
                    {pendingDetectorRuleSuggestions}
                </span>
            ) : null}
        </div>
    )

    // Load tenant data on component mount
    useEffect(() => {
        // Tenant data is already loaded during login via auth store
        // No need to reload it here - just use what's in the tenant store
    }, [user])

    const handleRefresh = () => {
        triggerRefresh()
    }

    if (!isAdmin) {
        return (
            <div className="flex items-center justify-center min-h-screen bg-gradient-to-br from-slate-100 to-slate-50 dark:from-slate-900 dark:to-slate-800">
                <div className="text-center">
                    <h1 className="text-3xl font-bold text-slate-900 dark:text-white mb-4">Access Denied</h1>
                    <p className="text-slate-600 dark:text-slate-300 mb-6">
                        You do not have permission to access the administration section.
                    </p>
                    <button
                        onClick={() => navigate('/dashboard')}
                        className="px-6 py-2 bg-blue-600 hover:bg-blue-700 dark:bg-blue-600 dark:hover:bg-blue-700 text-white rounded-lg font-medium transition-colors"
                    >
                        Back to Dashboard
                    </button>
                </div>
            </div>
        )
    }

    const setupSection = setupRequired ? [
        {
            title: 'Setup',
            items: [
                {
                    name: 'Initial Setup',
                    href: '/admin/setup',
                    icon: '🛠️',
                    description: 'Complete required first-run setup',
                },
            ],
        },
    ] : []

    const adminSections = [
        ...setupSection,
        {
            title: 'Overview',
            items: [
                {
                    name: 'Dashboard',
                    href: '/admin',
                    icon: '📊',
                    description: 'System overview and analytics',
                }
            ]
        },
        {
            title: 'Organization',
            items: [
                {
                    name: 'Tenants',
                    href: '/admin/tenants',
                    icon: '🏢',
                    description: 'Manage tenants',
                }
            ]
        },
        {
            title: 'Access Management',
            items: [
                {
                    name: 'Users',
                    href: '/admin/users',
                    icon: '👥',
                    description: 'Manage system users',
                },
                {
                    name: 'User Invitations',
                    href: '/admin/users/invitations',
                    icon: '📧',
                    description: 'Manage user invitations',
                },
                {
                    name: 'Roles',
                    href: '/admin/access/roles',
                    icon: '🔐',
                    description: 'Manage roles and permissions',
                },
                {
                    name: 'Permissions',
                    href: '/admin/access/permissions',
                    icon: '🔒',
                    description: 'Assign permissions to roles',
                },
                {
                    name: 'Permission Definitions',
                    href: '/admin/access/permission-definitions',
                    icon: '🛡️',
                    description: 'Manage system permission definitions',
                },
                {
                    name: 'Operational Capabilities',
                    href: '/admin/access/operational-capabilities',
                    icon: '🧩',
                    description: 'Manage tenant operational capability entitlements',
                }
            ]
        },
        {
            title: 'Build Management',
            items: [
                {
                    name: 'Build Overview',
                    href: '/admin/builds',
                    icon: '📊',
                    description: 'Build status and monitoring',
                },
                {
                    name: 'Infrastructure Providers',
                    href: '/admin/infrastructure',
                    icon: '🏗️',
                    description: 'Configure K8s clusters and build nodes',
                },
                {
                    name: 'Packer Target Profiles',
                    href: '/admin/infrastructure/packer-target-profiles',
                    icon: '📦',
                    description: 'Configure VMware/AWS/Azure/GCP destination profiles',
                },
                {
                    name: 'Build Nodes',
                    href: '/admin/builds/nodes',
                    icon: '🖥️',
                    description: 'Configure build infrastructure',
                },
                {
                    name: 'Build Analytics',
                    href: '/admin/builds/analytics',
                    icon: '📈',
                    description: 'Performance metrics and trends',
                },
                {
                    name: 'Build Policies',
                    href: '/admin/builds/policies',
                    icon: '📋',
                    description: 'Resource limits and scheduling rules',
                },
            ]
        },
        {
            title: 'SRE Smart Bot',
            items: [
                ...(isAdmin ? [{
                    name: 'Incident Workspace',
                    href: '/admin/operations/sre-smart-bot',
                    icon: '🤖',
                    description: 'Inspect incidents, evidence, actions, and approvals',
                },
                {
                    name: 'Approvals',
                    href: '/admin/operations/sre-smart-bot/approvals',
                    icon: '✅',
                    description: 'Review pending and recent SRE Smart Bot approval requests',
                },
                {
                    name: 'Detector Rules',
                    href: '/admin/operations/sre-smart-bot/detector-rules',
                    icon: '🧠',
                    description: 'Review learned detector rules and activate accepted suggestions',
                },
                {
                    name: 'Settings',
                    href: '/admin/operations/sre-smart-bot/settings',
                    icon: '🛡️',
                    description: 'Configure policy, channels, domains, and operator rules',
                }] : []),
            ]
        },
        {
            title: 'System Management',
            items: [
                ...(isAdmin ? [{
                    name: 'System Config',
                    href: '/admin/system-config',
                    icon: '⚙️',
                    description: 'System configuration',
                }] : []),
                ...(isAdmin ? [{
                    name: 'Notification Defaults',
                    href: '/admin/notifications/defaults',
                    icon: '🔔',
                    description: 'Manage tenant notification defaults',
                }] : []),
                ...(isAdmin ? [{
                    name: 'External Services',
                    href: '/admin/external-services',
                    icon: '🔗',
                    description: 'Configure external service integrations',
                }] : []),
                ...(isAdmin ? [{
                    name: 'Auth Providers',
                    href: '/admin/auth-providers',
                    icon: '🔐',
                    description: 'Manage LDAP and SSO providers',
                }] : []),
                {
                    name: 'Tool Management',
                    href: '/admin/tools',
                    icon: '🔧',
                    description: 'Configure available build tools',
                },
                {
                    name: 'Image Catalog',
                    href: '/admin/images',
                    icon: '🖼️',
                    description: 'Browse and manage container images',
                },
                {
                    name: 'Security Review Queue',
                    href: '/admin/quarantine/review',
                    icon: '✅',
                    description: 'Approve or reject pending quarantine imports',
                },
                {
                    name: 'Quarantine Requests',
                    href: '/admin/quarantine/requests',
                    icon: '🧪',
                    description: 'Track import lifecycle and quarantine decisions',
                },
                {
                    name: 'On-Demand Scans',
                    href: '/admin/images/scans',
                    icon: '🛡️',
                    description: 'Review ad-hoc image scan request outcomes',
                }
            ]
        },
        {
            title: 'Monitoring',
            items: [
                {
                    name: 'Audit Logs',
                    href: '/admin/audit-logs',
                    icon: '📋',
                    description: 'View system audit logs',
                }
            ]
        },
        {
            title: 'Help',
            items: [
                {
                    name: 'Product Info',
                    href: '/admin/help/product-info',
                    icon: '📦',
                    description: `Capabilities and release info (v${APP_VERSION})`,
                },
            ],
        }
    ]

    const adminNavigation = adminSections.flatMap(section =>
        section.items.flatMap((item: any) => item.children ? [item, ...item.children] : [item])
    )

    const isActive = (href: string) => {
        if (href === '/admin') {
            return location.pathname === '/admin'
        }
        // Check if current path exactly matches
        if (location.pathname === href) {
            return true
        }
        // For parent routes, check if path starts with href followed by '/',
        // but only if no child route is more specific
        if (location.pathname.startsWith(href + '/')) {
            // Find if there's a more specific nav item that matches
            const moreSpecificItem = adminNavigation.find(item =>
                item.href !== href &&
                item.href.startsWith(href + '/') &&
                location.pathname.startsWith(item.href)
            )
            // If no more specific item matches, then this parent can be active
            return !moreSpecificItem
        }
        return false
    }

    const toggleSection = (sectionTitle: string) => {
        if (setupRequired && sectionTitle !== 'Setup') {
            return
        }
        setExpandedSections(prev => {
            const newSet = new Set(prev)
            if (newSet.has(sectionTitle)) {
                newSet.delete(sectionTitle)
            } else {
                newSet.add(sectionTitle)
            }
            return newSet
        })
    }

    const handleLogout = async () => {
        const confirmed = await confirmDialog({
            title: 'Log Out',
            message: 'Are you sure you want to logout?',
            confirmLabel: 'Log Out',
        })
        if (!confirmed) return

        try {
            await authService.logout()
        } catch (error) {
        }
        logout()
        navigate('/login')
    }

    const initials = user ? profileService.getInitials(user.name || user.email || 'U', '') : 'U'
    const avatarColor = user ? profileService.getAvatarColor(user.id) : 'bg-gray-500'

    return (
        <div className="min-h-screen bg-white dark:bg-slate-900 text-slate-900 dark:text-slate-50">
            {/* Header */}
            <header className="sticky top-0 z-40 border-b border-slate-200 dark:border-slate-700 bg-white/95 dark:bg-slate-800/95 backdrop-blur supports-[backdrop-filter]:bg-white/60 dark:supports-[backdrop-filter]:bg-slate-800/60">
                <div className="px-4 sm:px-6 lg:px-8 py-4 flex items-center justify-between">
                    <div className="flex items-center gap-4">
                        <button
                            onClick={() => setSidebarOpen(!sidebarOpen)}
                            className="p-2 hover:bg-slate-100 dark:hover:bg-slate-700 rounded-lg transition-colors lg:hidden"
                        >
                            <svg
                                className="w-6 h-6"
                                fill="none"
                                stroke="currentColor"
                                viewBox="0 0 24 24"
                            >
                                <path
                                    strokeLinecap="round"
                                    strokeLinejoin="round"
                                    strokeWidth={2}
                                    d="M4 6h16M4 12h16M4 18h16"
                                />
                            </svg>
                        </button>
                        <Link
                            to="/admin"
                            className="flex items-center gap-3 text-xl font-bold hover:opacity-80 transition-opacity"
                        >
                            <span className="text-2xl">🏭</span>
                            <span className="hidden sm:block">Image Factory Admin</span>
                            <span className="sm:hidden">Admin</span>
                        </Link>
                        <Link
                            to="/admin/help/product-info"
                            className="hidden sm:inline-flex items-center rounded-full border border-slate-300 bg-slate-50 px-2.5 py-1 text-xs font-semibold text-slate-700 transition hover:border-sky-400 hover:text-sky-700 dark:border-slate-700 dark:bg-slate-800 dark:text-slate-300 dark:hover:border-sky-500 dark:hover:text-sky-300"
                            title="Open product information"
                        >
                            v{APP_VERSION}
                        </Link>
                    </div>
                    <div className="flex items-center gap-4">
                        <CurrentTenantDisplay />

                        {/* Refresh Button */}
                        <button
                            onClick={handleRefresh}
                            disabled={isRefreshing}
                            className="p-1 rounded-full hover:bg-slate-100 dark:hover:bg-slate-700 transition-colors text-slate-700 dark:text-slate-300 hover:text-slate-900 dark:hover:text-slate-100 disabled:opacity-50 disabled:cursor-not-allowed"
                            title="Refresh page"
                        >
                            <svg
                                className={`w-5 h-5 ${isRefreshing ? 'animate-spin' : ''}`}
                                fill="none"
                                stroke="currentColor"
                                viewBox="0 0 24 24"
                            >
                                <path
                                    strokeLinecap="round"
                                    strokeLinejoin="round"
                                    strokeWidth={2}
                                    d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"
                                />
                            </svg>
                        </button>

                        {/* Theme Toggle */}
                        <button
                            onClick={toggleTheme}
                            className="p-1 rounded-full hover:bg-slate-100 dark:hover:bg-slate-700 transition-colors text-slate-700 dark:text-slate-300 hover:text-slate-900 dark:hover:text-slate-100"
                            title="Toggle dark mode"
                        >
                            {isDark ? '🌙' : '☀️'}
                        </button>

                        <UserNotificationCenter />

                        {/* Avatar Profile Menu */}
                        <div className="relative">
                            <button
                                onClick={() => setShowProfileMenu(!showProfileMenu)}
                                className="flex items-center space-x-2 hover:bg-slate-100 dark:hover:bg-slate-700 rounded-full p-1 transition-colors"
                                title={user?.email}
                            >
                                {avatar ? (
                                    <img
                                        src={avatar}
                                        alt={user?.name || 'Profile'}
                                        className="w-8 h-8 rounded-full object-cover"
                                    />
                                ) : (
                                    <div
                                        className={`w-8 h-8 rounded-full ${avatarColor} flex items-center justify-center text-white text-xs font-bold`}
                                    >
                                        {initials}
                                    </div>
                                )}
                            </button>

                            {/* Dropdown Menu */}
                            {showProfileMenu && (
                                <div className="absolute right-0 mt-2 w-48 bg-white dark:bg-slate-800 rounded-lg shadow-lg border border-slate-200 dark:border-slate-700 z-50">
                                    <div className="p-4 border-b border-slate-200 dark:border-slate-700">
                                        <p className="text-sm font-medium text-slate-900 dark:text-slate-50">{user?.name}</p>
                                        <p className="text-xs text-slate-600 dark:text-slate-400">{user?.email}</p>
                                    </div>
                                    <div className="py-2">
                                        <Link
                                            to="/profile"
                                            className="block px-4 py-2 text-sm text-slate-900 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-700"
                                            onClick={() => setShowProfileMenu(false)}
                                        >
                                            View Profile
                                        </Link>
                                        {userTenants && userTenants.length > 1 && (
                                            <button
                                                onClick={() => {
                                                    setShowProfileMenu(false)
                                                    setShowContextSelector(true)
                                                }}
                                                className="w-full text-left px-4 py-2 text-sm text-slate-900 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-700"
                                            >
                                                Switch Context
                                            </button>
                                        )}
                                        <Link
                                            to={dashboardPath}
                                            className="block px-4 py-2 text-sm text-slate-900 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-700"
                                            onClick={() => setShowProfileMenu(false)}
                                        >
                                            Back to Dashboard
                                        </Link>
                                        <button
                                            onClick={() => {
                                                setShowProfileMenu(false)
                                                handleLogout()
                                            }}
                                            className="w-full text-left px-4 py-2 text-sm text-slate-900 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-700 border-t border-slate-200 dark:border-slate-700"
                                        >
                                            Sign Out
                                        </button>
                                    </div>
                                </div>
                            )}
                        </div>
                    </div>
                </div>
            </header>

            <div className="flex h-[calc(100vh-4rem)]">
                {/* Sidebar */}
                <aside
                    className={`${sidebarOpen ? 'translate-x-0' : '-translate-x-full'
                        } lg:translate-x-0 w-64 border-r border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-800 transition-transform duration-200 ease-in-out fixed lg:sticky top-16 h-[calc(100vh-4rem)] z-30 lg:z-0 flex flex-col`}
                >
                    <nav className="p-4 space-y-6 overflow-y-auto flex-1">
                        {adminSections.map((section, sectionIndex) => {
                            const isExpanded = expandedSections.has(section.title)
                            const sectionDisabled = setupRequired && section.title !== 'Setup'
                            return (
                                <div key={section.title} className={sectionIndex > 0 ? '' : ''}>
                                    <button
                                        onClick={() => toggleSection(section.title)}
                                        disabled={sectionDisabled}
                                        className={`w-full flex items-center justify-between px-3 py-2 text-xs font-semibold uppercase tracking-wider transition-colors group ${sectionDisabled
                                            ? 'text-slate-400 dark:text-slate-600 cursor-not-allowed'
                                            : 'text-slate-500 dark:text-slate-400 hover:text-slate-700 dark:hover:text-slate-200'
                                            }`}
                                    >
                                        <span>{section.title}</span>
                                        <svg
                                            className={`w-4 h-4 transition-transform duration-200 ${isExpanded ? 'rotate-90' : ''}`}
                                            fill="none"
                                            stroke="currentColor"
                                            viewBox="0 0 24 24"
                                        >
                                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
                                        </svg>
                                    </button>
                                    <div className={`space-y-1 overflow-hidden transition-all duration-300 ease-in-out ${isExpanded ? 'max-h-[200rem] opacity-100' : 'max-h-0 opacity-0'}`}>
                                        {section.items.map((item: any) => {
                                            const active = isActive(item.href)
                                            const hasChildren = item.children && item.children.length > 0
                                            const isItemExpanded = expandedSections.has(item.name)
                                            const itemDisabled = setupRequired && item.href !== '/admin/setup'

                                            if (hasChildren) {
                                                // Parent item with children
                                                return (
                                                    <div key={item.href} className="space-y-1">
                                                        <button
                                                            onClick={() => toggleSection(item.name)}
                                                            disabled={itemDisabled}
                                                            className={`w-full flex items-center justify-between p-3 rounded-lg transition-all group ${active
                                                                ? 'bg-blue-600 text-white shadow-lg'
                                                                : 'text-slate-700 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-700/50 hover:text-slate-900 dark:hover:text-slate-100'
                                                                } ${itemDisabled ? 'opacity-50 cursor-not-allowed hover:bg-transparent dark:hover:bg-transparent' : ''}`}
                                                        >
                                                            <div className="flex items-center gap-3 flex-1">
                                                                <span className="text-xl">{item.icon}</span>
                                                                <div className="flex-1 text-left">
                                                                    {renderNavLabel(item.name, item.href, active)}
                                                                    <div
                                                                        className={`text-xs ${active
                                                                            ? 'text-blue-100'
                                                                            : 'text-slate-600 dark:text-slate-400 group-hover:text-slate-700 dark:group-hover:text-slate-200'
                                                                            }`}
                                                                    >
                                                                        {item.description}
                                                                    </div>
                                                                </div>
                                                            </div>
                                                            <svg
                                                                className={`w-4 h-4 transition-transform duration-200 ${isItemExpanded ? 'rotate-90' : ''}`}
                                                                fill="none"
                                                                stroke="currentColor"
                                                                viewBox="0 0 24 24"
                                                            >
                                                                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
                                                            </svg>
                                                        </button>
                                                        <div className={`ml-4 space-y-1 overflow-hidden transition-all duration-300 ease-in-out ${isItemExpanded ? 'max-h-[200rem] opacity-100' : 'max-h-0 opacity-0'}`}>
                                                            {item.children.map((child: any) => {
                                                                const childActive = isActive(child.href)
                                                                const childDisabled = setupRequired && child.href !== '/admin/setup'
                                                                if (childDisabled) {
                                                                    return (
                                                                        <div
                                                                            key={child.href}
                                                                            className="block p-2 rounded-lg transition-all group opacity-50 cursor-not-allowed text-slate-600 dark:text-slate-500"
                                                                            title="Disabled until initial setup is complete"
                                                                        >
                                                                            <div className="flex items-center gap-3">
                                                                                <span className="text-lg">{child.icon}</span>
                                                                                <div className="flex-1">
                                                                                    <div className="font-medium text-sm">{child.name}</div>
                                                                                    <div className="text-xs text-slate-500 dark:text-slate-500">
                                                                                        {child.description}
                                                                                    </div>
                                                                                </div>
                                                                            </div>
                                                                        </div>
                                                                    )
                                                                }
                                                                return (
                                                                    <Link
                                                                        key={child.href}
                                                                        to={child.href}
                                                                        onClick={() => setSidebarOpen(false)}
                                                                        className={`block p-2 rounded-lg transition-all group ${childActive
                                                                            ? 'bg-blue-500 text-white shadow-md'
                                                                            : 'text-slate-600 dark:text-slate-400 hover:bg-slate-100 dark:hover:bg-slate-700/30 hover:text-slate-900 dark:hover:text-slate-100'
                                                                            }`}
                                                                    >
                                                                        <div className="flex items-center gap-3">
                                                                            <span className="text-lg">{child.icon}</span>
                                                                            <div className="flex-1">
                                                                                <div className="font-medium text-sm">{child.name}</div>
                                                                                <div
                                                                                    className={`text-xs ${childActive
                                                                                        ? 'text-blue-100'
                                                                                        : 'text-slate-500 dark:text-slate-500 group-hover:text-slate-600 dark:group-hover:text-slate-300'
                                                                                        }`}
                                                                                >
                                                                                    {child.description}
                                                                                </div>
                                                                            </div>
                                                                        </div>
                                                                    </Link>
                                                                )
                                                            })}
                                                        </div>
                                                    </div>
                                                )
                                            } else {
                                                // Regular item without children
                                                if (itemDisabled) {
                                                    return (
                                                        <div
                                                            key={item.href}
                                                            className="block p-3 rounded-lg transition-all group opacity-50 cursor-not-allowed text-slate-700 dark:text-slate-400"
                                                            title="Disabled until initial setup is complete"
                                                        >
                                                            <div className="flex items-center gap-3">
                                                                <span className="text-xl">{item.icon}</span>
                                                                <div className="flex-1">
                                                                    {renderNavLabel(item.name, item.href, false)}
                                                                    <div className="text-xs text-slate-600 dark:text-slate-500">
                                                                        {item.description}
                                                                    </div>
                                                                </div>
                                                            </div>
                                                        </div>
                                                    )
                                                }
                                                return (
                                                    <Link
                                                        key={item.href}
                                                        to={item.href}
                                                        onClick={() => setSidebarOpen(false)}
                                                        className={`block p-3 rounded-lg transition-all group ${active
                                                            ? 'bg-blue-600 text-white shadow-lg'
                                                            : 'text-slate-700 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-700/50 hover:text-slate-900 dark:hover:text-slate-100'
                                                            }`}
                                                    >
                                                        <div className="flex items-center gap-3">
                                                            <span className="text-xl">{item.icon}</span>
                                                            <div className="flex-1">
                                                                {renderNavLabel(item.name, item.href, active)}
                                                                <div
                                                                    className={`text-xs ${active
                                                                        ? 'text-blue-100'
                                                                        : 'text-slate-600 dark:text-slate-400 group-hover:text-slate-700 dark:group-hover:text-slate-200'
                                                                        }`}
                                                                >
                                                                    {item.description}
                                                                </div>
                                                            </div>
                                                        </div>
                                                    </Link>
                                                )
                                            }
                                        })}
                                    </div>
                                </div>
                            )
                        })}
                    </nav>

                    {/* Footer */}
                    <div className="p-4 border-t border-slate-200 dark:border-slate-700 bg-slate-100 dark:bg-slate-900">
                        {setupRequired ? (
                            <div className="flex items-center gap-2 text-sm text-slate-400 dark:text-slate-600 cursor-not-allowed">
                                <span>←</span>
                                Back to Dashboard
                            </div>
                        ) : (
                            <Link
                                to="/dashboard"
                                className="flex items-center gap-2 text-sm text-slate-700 dark:text-slate-400 hover:text-slate-900 dark:hover:text-slate-200 transition-colors"
                            >
                                <span>←</span>
                                Back to Dashboard
                            </Link>
                        )}
                    </div>
                </aside>

                {/* Main content */}
                <main className="flex-1 overflow-y-auto">
                    <div className="p-4 sm:p-6 lg:p-8">
                        {children}
                    </div>
                </main>
            </div>

            {/* Token expiration warning */}
            <TokenExpirationWarning />

            {/* Context Selector Modal */}
            <PostLoginContextSelector
                isOpen={showContextSelector}
                onClose={() => setShowContextSelector(false)}
                onContextSwitch={() => {
                    // Navigate to appropriate dashboard after context switch
                    navigate(dashboardPath)
                }}
            />
        </div>
    )
}

export default AdminLayout
