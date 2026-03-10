import { useRefresh } from '@/context/RefreshContext'
import { useConfirmDialog } from '@/context/ConfirmDialogContext'
import { api } from '@/services/api'
import { authService } from '@/services/authService'
import { profileService } from '@/services/profileService'
import { useAuthStore } from '@/store/auth'
import { useCapabilitySurfacesStore } from '@/store/capabilitySurfaces'
import { useTenantStore } from '@/store/tenant'
import { useThemeStore } from '@/store/theme'
import { canCreateBuilds, canManageMembers, canManageTenants, canViewBuilds, canViewMembers, canViewTenants } from '@/utils/permissions'
import React, { useEffect, useRef, useState } from 'react'
import { Link, useLocation, useNavigate } from 'react-router-dom'
import PostLoginContextSelector from '../auth/PostLoginContextSelector'
import ContextSwitcher from '../common/ContextSwitcher'
import UserNotificationCenter from '../notifications/UserNotificationCenter'
import TenantOwnerWelcomeTour from '../onboarding/TenantOwnerWelcomeTour'
import { TokenExpirationWarning } from '../TokenExpirationWarning'

interface LayoutProps {
    children: React.ReactNode
}

const NAV_OPEN_SECTION_KEY = 'if_layout_open_nav_section'

const Layout: React.FC<LayoutProps> = ({ children }) => {
    const { user, logout, avatar, groups, canAccessAdmin, isSystemAdmin } = useAuthStore()
    const capabilitySurfaces = useCapabilitySurfacesStore((state) => state.data.surfaces)
    const capabilitySurfacesLoadedTenantId = useCapabilitySurfacesStore((state) => state.loadedTenantId)
    const capabilitySurfacesLoading = useCapabilitySurfacesStore((state) => state.isLoading)
    const { selectedTenantId, userTenants } = useTenantStore()
    const { isDark, toggleTheme } = useThemeStore()
    const { isRefreshing, triggerRefresh } = useRefresh()
    const confirmDialog = useConfirmDialog()
    const location = useLocation()
    const navigate = useNavigate()
    const [sidebarOpen, setSidebarOpen] = useState(true)
    const [showProfileMenu, setShowProfileMenu] = useState(false)
    const [showContextSelector, setShowContextSelector] = useState(false)
    const [maintenanceMode, setMaintenanceMode] = useState(false)
    const [openSectionTitle, setOpenSectionTitle] = useState<string | null>(() => {
        if (typeof window === 'undefined') return null
        return window.localStorage.getItem(NAV_OPEN_SECTION_KEY)
    })
    const profileMenuRef = useRef<HTMLDivElement | null>(null)
    const profileMenuCloseTimerRef = useRef<number | null>(null)

    const hasAdminConsoleAccess = !!canAccessAdmin

    // Fallback to tenant groups if no RBAC role selected
    const currentTenantGroups = selectedTenantId ?
        groups?.filter((group: any) => group.tenant_id === selectedTenantId) || []
        : []
    const currentTenantRoleType = currentTenantGroups.length > 0 ?
        currentTenantGroups[0].role_type || null
        : null

    const capabilitySurfaceStateReadyForTenant =
        !!selectedTenantId && !capabilitySurfacesLoading && capabilitySurfacesLoadedTenantId === selectedTenantId
    const hasNavKey = (key: string) => capabilitySurfaceStateReadyForTenant && capabilitySurfaces.nav_keys.includes(key)
    const hasRouteKey = (key: string) => capabilitySurfaceStateReadyForTenant && capabilitySurfaces.route_keys.includes(key)

    // Determine user permissions based on actual RBAC permissions
    const permissions = {
        canManageTenants: canManageTenants(),
        canManageMembers: canManageMembers(),
        canViewTenants: canViewTenants(),
        canViewMembers: canViewMembers(),
        canViewProjects: hasNavKey('projects'),
        canCreateBuilds: canCreateBuilds() && hasNavKey('builds'),
        canViewBuilds: canViewBuilds() && hasNavKey('builds'),
        canViewImages: true, // All authenticated users can view images
        canManageSettings: hasNavKey('auth_management'),
    }

    // Debug logging

    // Create a text representation of all user roles across tenants
    const rbacRolesText = userTenants.flatMap(tenant =>
        tenant.roles.map(role => `${tenant.name}: ${role.name}`)
    ).join('; ')
    const groupRolesText = groups ? groups
        .filter((group: any) => group.tenant_id)
        .map((group: any) => {
            const tenantName = userTenants.find(t => t.id === group.tenant_id)?.name || group.tenant_id
            return `${tenantName}: ${group.role_type}`
        })
        .join('; ') : ''
    const allRolesText = [rbacRolesText, groupRolesText].filter(Boolean).join('; ')

    // Role-based navigation with sections
    const navigationSections = [
        {
            title: 'Home',
            items: [
                { name: 'Dashboard', href: '/dashboard', icon: '📊', description: 'Overview and analytics', show: true },
                { name: 'Notifications', href: '/notifications', icon: '🔔', description: 'Manage notifications', show: true },
            ]
        },
        {
            title: 'Build & Delivery',
            items: [
                { name: 'Projects', href: '/projects', icon: '📁', description: 'Manage projects', show: permissions.canViewProjects },
                { name: 'Builds', href: '/builds', icon: '🔨', description: 'Build management', show: permissions.canViewBuilds },
            ]
        },
        {
            title: 'Image Catalog',
            items: [
                { name: 'Images', href: '/images', icon: '🖼️', description: 'Manage images', show: permissions.canViewImages },
            ]
        },
        {
            title: 'Quarantine',
            items: [
                { name: 'Quarantine Requests', href: '/quarantine/requests', icon: '🧪', description: 'Request and track quarantine imports', show: hasNavKey('quarantine_requests') },
                { name: 'EPR Registrations', href: '/quarantine/epr', icon: '🗂️', description: 'Register and track enterprise product/technology entries', show: hasNavKey('quarantine_requests') },
            ]
        },
        {
            title: 'Image Scanning',
            items: [
                { name: 'On-Demand Scans', href: '/images/scans', icon: '🛡️', description: 'Submit and track async scan requests for external images', show: hasRouteKey('images.scan.ondemand') },
            ]
        },
        {
            title: 'Security',
            items: [
                { name: 'Registry Auth', href: '/settings/auth', icon: '🔐', description: 'Manage registry credentials for quarantine/build workflows', show: permissions.canManageSettings },
            ]
        },
        {
            title: 'Management',
            items: [
                { name: 'Tenants', href: '/tenants', icon: '🏢', description: 'View tenants', show: permissions.canManageTenants },
                { name: 'Members', href: '/members', icon: '👥', description: 'Team members', show: permissions.canManageMembers },
                { name: 'Invitations', href: '/invitations', icon: '✉️', description: 'Pending invites', show: permissions.canManageMembers },
            ]
        },
        {
            title: 'Administration',
            items: [
                { name: 'Build Management', href: '/admin/builds', icon: '🔨', description: 'Manage builds system-wide', show: hasAdminConsoleAccess },
                { name: 'Infrastructure', href: '/admin/tools', icon: '⚙️', description: 'Tool availability & config', show: hasAdminConsoleAccess },
                { name: 'Audit Logs', href: '/admin/audit-logs', icon: '📋', description: 'System audit logs', show: hasAdminConsoleAccess },
            ]
        },
        {
            title: 'Help',
            items: [
                { name: 'Capability Access', href: '/help/capabilities', icon: '🧭', description: 'What this tenant is entitled to use', show: true },
            ]
        },
    ]

    // Flatten navigation for rendering (not used in current implementation)
    // const navigation = navigationSections.flatMap(section => section.items).filter(item => item.show)

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

    const handleRefresh = () => {
        triggerRefresh()
    }

    const initials = user ? profileService.getInitials(user.name || user.email || 'U', '') : 'U'
    const avatarColor = user ? profileService.getAvatarColor(user.id) : 'bg-gray-500'

    const isActive = (href: string) => {
        if (href === '/dashboard') {
            return location.pathname === '/dashboard'
        }
        return location.pathname.startsWith(href)
    }

    useEffect(() => {
        if (!isSystemAdmin) {
            setMaintenanceMode(false)
            return
        }

        const loadMaintenanceMode = async () => {
            try {
                const response = await api.get('/system-configs')
                const configs = response.data?.configs || []
                const generalConfig = configs.find((config: any) => config.config_type === 'general' && config.config_key === 'general')
                if (generalConfig?.config_value && typeof generalConfig.config_value.maintenance_mode === 'boolean') {
                    setMaintenanceMode(generalConfig.config_value.maintenance_mode)
                }
            } catch {
                // Ignore maintenance mode load errors to avoid blocking UI
            }
        }

        loadMaintenanceMode()
    }, [isSystemAdmin])

    useEffect(() => {
        const visibleSectionTitles = navigationSections
            .filter(section => section.items.some(item => item.show))
            .map(section => section.title)

        if (openSectionTitle && !visibleSectionTitles.includes(openSectionTitle)) {
            setOpenSectionTitle(null)
            if (typeof window !== 'undefined') {
                window.localStorage.removeItem(NAV_OPEN_SECTION_KEY)
            }
        }
    }, [openSectionTitle, navigationSections])

    const toggleSection = (title: string) => {
        const next = openSectionTitle === title ? null : title
        setOpenSectionTitle(next)
        if (typeof window !== 'undefined') {
            if (next) {
                window.localStorage.setItem(NAV_OPEN_SECTION_KEY, next)
            } else {
                window.localStorage.removeItem(NAV_OPEN_SECTION_KEY)
            }
        }
    }

    useEffect(() => {
        if (!showProfileMenu) {
            return
        }

        const closeIfOutside = (target: EventTarget | null) => {
            if (!profileMenuRef.current) {
                return
            }
            if (!profileMenuRef.current.contains(target as Node)) {
                setShowProfileMenu(false)
            }
        }

        const handleMouseDown = (event: MouseEvent) => closeIfOutside(event.target)
        const handleFocusIn = (event: FocusEvent) => closeIfOutside(event.target)
        const handleKeyDown = (event: KeyboardEvent) => {
            if (event.key === 'Escape') {
                setShowProfileMenu(false)
            }
        }

        document.addEventListener('mousedown', handleMouseDown)
        document.addEventListener('focusin', handleFocusIn)
        document.addEventListener('keydown', handleKeyDown)

        return () => {
            document.removeEventListener('mousedown', handleMouseDown)
            document.removeEventListener('focusin', handleFocusIn)
            document.removeEventListener('keydown', handleKeyDown)
        }
    }, [showProfileMenu])

    useEffect(() => {
        return () => {
            if (profileMenuCloseTimerRef.current) {
                window.clearTimeout(profileMenuCloseTimerRef.current)
            }
        }
    }, [])

    const cancelProfileMenuClose = () => {
        if (profileMenuCloseTimerRef.current) {
            window.clearTimeout(profileMenuCloseTimerRef.current)
            profileMenuCloseTimerRef.current = null
        }
    }

    const scheduleProfileMenuClose = () => {
        cancelProfileMenuClose()
        profileMenuCloseTimerRef.current = window.setTimeout(() => {
            setShowProfileMenu(false)
            profileMenuCloseTimerRef.current = null
        }, 180)
    }

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
                            to="/dashboard"
                            className="flex items-center gap-3 text-xl font-bold hover:opacity-80 transition-opacity"
                        >
                            <span className="text-2xl">🏭</span>
                            <span className="hidden sm:block">Image Factory</span>
                        </Link>
                    </div>
                    <div className="flex items-center gap-4">
                        <ContextSwitcher />

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

                        {/* Admin Dashboard Link */}
                        {isSystemAdmin && (
                            <Link
                                to="/admin/dashboard"
                                className="text-sm text-blue-600 dark:text-blue-400 hover:text-blue-700 dark:hover:text-blue-300 font-medium"
                            >
                                Admin
                            </Link>
                        )}

                        {/* Avatar Profile Menu */}
                        <div
                            className="relative"
                            ref={profileMenuRef}
                            onMouseEnter={cancelProfileMenuClose}
                            onMouseLeave={scheduleProfileMenuClose}
                        >
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
                                <div className="absolute right-0 mt-2 w-56 bg-white dark:bg-slate-800 rounded-lg shadow-lg border border-slate-200 dark:border-slate-700 z-50">
                                    <div className="p-4 border-b border-slate-200 dark:border-slate-700">
                                        <p className="text-sm font-medium text-slate-900 dark:text-slate-50">{user?.name}</p>
                                        <p className="text-xs text-slate-600 dark:text-slate-400">{user?.email}</p>
                                        {selectedTenantId && (
                                            <>
                                                <p className="text-xs text-slate-600 dark:text-slate-400 mt-2 font-medium">Current Tenant:</p>
                                                <p className="text-xs text-blue-600 dark:text-blue-400">{userTenants.find(t => t.id === selectedTenantId)?.name || 'Unknown'}</p>
                                                {currentTenantRoleType && (
                                                    <>
                                                        <p className="text-xs text-slate-600 dark:text-slate-400 mt-1 font-medium">Current Role:</p>
                                                        <p className="text-xs text-blue-600 dark:text-blue-400">{currentTenantRoleType}</p>
                                                    </>
                                                )}
                                                {!currentTenantRoleType && groups && groups.length > 0 && (
                                                    <>
                                                        <p className="text-xs text-slate-600 dark:text-slate-400 mt-1 font-medium">Your Roles:</p>
                                                        <p className="text-xs text-blue-600 dark:text-blue-400">{allRolesText}</p>
                                                    </>
                                                )}
                                            </>
                                        )}
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
                                        {isSystemAdmin && (
                                            <Link
                                                to="/admin/dashboard"
                                                className="block px-4 py-2 text-sm text-slate-900 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-700"
                                                onClick={() => setShowProfileMenu(false)}
                                            >
                                                Admin Dashboard
                                            </Link>
                                        )}
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
            {maintenanceMode && (
                <div className="border-b border-amber-200 bg-amber-50 text-amber-900 dark:border-amber-700 dark:bg-amber-900/30 dark:text-amber-100">
                    <div className="px-4 sm:px-6 lg:px-8 py-2 text-sm">
                        Maintenance mode is enabled. Write actions are temporarily disabled for non-admin users.
                    </div>
                </div>
            )}

            <div className="flex h-[calc(100vh-4rem)]">
                {/* Sidebar */}
                <aside
                    className={`${sidebarOpen ? 'translate-x-0' : '-translate-x-full'
                        } lg:translate-x-0 w-64 border-r border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-800 overflow-y-auto transition-transform duration-200 ease-in-out fixed lg:sticky top-16 h-[calc(100vh-4rem)] z-30 lg:z-0`}
                >
                    <nav className="p-4 space-y-6">
                        {navigationSections.map((section) => {
                            const visibleItems = section.items.filter(item => item.show)
                            if (visibleItems.length === 0) return null

                            return (
                                <div key={section.title}>
                                    <button
                                        type="button"
                                        onClick={() => toggleSection(section.title)}
                                        className="w-full flex items-center justify-between text-xs font-semibold text-slate-500 dark:text-slate-400 uppercase tracking-wider mb-2 hover:text-slate-700 dark:hover:text-slate-200"
                                    >
                                        <span>{section.title}</span>
                                        <span className="text-sm">{openSectionTitle === section.title ? '▾' : '▸'}</span>
                                    </button>
                                    {openSectionTitle === section.title && (
                                        <div className="space-y-1">
                                            {visibleItems.map((item) => {
                                                const active = isActive(item.href)
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
                                                                <div className="font-medium text-sm">{item.name}</div>
                                                                <div
                                                                    className={`text-xs ${active
                                                                        ? 'text-blue-100'
                                                                        : 'text-slate-600 dark:text-slate-400 group-hover:text-slate-700 dark:group-hover:text-slate-300'
                                                                        }`}
                                                                >
                                                                    {item.description}
                                                                </div>
                                                            </div>
                                                        </div>
                                                    </Link>
                                                )
                                            })}
                                        </div>
                                    )}
                                </div>
                            )
                        })}
                    </nav>
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
                    // Refresh the current page to update context-dependent content
                    window.location.reload()
                }}
            />
            <TenantOwnerWelcomeTour groups={groups} />
        </div>
    )
}

export default Layout
