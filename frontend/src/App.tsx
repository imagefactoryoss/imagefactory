import NotificationContainer from '@/components/notifications/NotificationContainer'
import { NotificationProvider } from '@/context/NotificationContext'
import { RefreshProvider } from '@/context/RefreshContext'
import { useAuthStore } from '@/store/auth'
import { useCapabilitySurfacesStore } from '@/store/capabilitySurfaces'
import { useOperationCapabilitiesStore } from '@/store/operationCapabilities'
import { useTenantStore } from '@/store/tenant'
import { useThemeStore } from '@/store/theme'
import { useEffect, useRef } from 'react'
import { Navigate, Route, Routes, useParams } from 'react-router-dom'

// Layout components
import ContextGuard from '@/components/auth/ContextGuard'
import AdminLayout from '@/components/layout/AdminLayout'
import CapabilityProtectedRoute from '@/components/layout/CapabilityProtectedRoute'
import Layout from '@/components/layout/Layout'
import PermissionProtectedRoute from '@/components/layout/PermissionProtectedRoute'
import ProtectedRoute from '@/components/layout/ProtectedRoute'
import ReviewerLayout from '@/components/layout/ReviewerLayout'

// Pages
import AcceptInvitationPage from '@/pages/auth/AcceptInvitationPage'
import ForgotPasswordPage from '@/pages/auth/ForgotPasswordPage'
import ForcePasswordChangePage from '@/pages/auth/ForcePasswordChangePage'
import LandingPage from '@/pages/auth/LandingPage'
import LoginPage from '@/pages/auth/LoginPage'
import ResetPasswordPage from '@/pages/auth/ResetPasswordPage'
import BuildDetailPage from '@/pages/builds/BuildDetailPage'
import BuildsPage from '@/pages/builds/BuildsPage'
import CreateBuildPage from '@/pages/builds/CreateBuildPage'
import DashboardPage from '@/pages/DashboardPage'
import CapabilityAccessPage from '@/pages/help/CapabilityAccessPage'
import CreateImagePage from '@/pages/images/CreateImagePage'
import ImageDetailPage from '@/pages/images/ImageDetailPage'
import ImagesPage from '@/pages/images/ImagesPage'
import OnDemandScansPage from '@/pages/images/OnDemandScansPage'
import ReleasedArtifactsPage from '@/pages/quarantine/ReleasedArtifactsPage'
import QuarantineRequestDetailPage from '@/pages/quarantine/QuarantineRequestDetailPage'
import QuarantineRequestsPage from '@/pages/quarantine/QuarantineRequestsPage.tsx'
import InvitationsPage from '@/pages/members/InvitationsPage'
import MembersPage from '@/pages/members/MembersPage'
import NoTenantAccessPage from '@/pages/NoTenantAccessPage'
import NotFoundPage from '@/pages/NotFoundPage'
import NotificationsPage from '@/pages/NotificationsPage'
import ProfilePage from '@/pages/ProfilePage'
import { CreateProjectPage, EditProjectPage, ProjectDetailPage, ProjectsPage } from '@/pages/projects'
import ProjectBuildsPage from '@/pages/projects/ProjectBuildsPage'
import SettingsPage from '@/pages/SettingsPage'
import AuthManagementPage from '@/pages/settings/AuthManagementPage'
import TenantDetailPage from '@/pages/tenants/TenantDetailPage'
import TenantsPage from '@/pages/tenants/TenantsPage'

// Admin Pages
import AdminBuildAnalyticsPage from '@/pages/admin/AdminBuildAnalyticsPage'
import AdminBuildNodesPage from '@/pages/admin/AdminBuildNodesPage'
import AdminBuildPoliciesPage from '@/pages/admin/AdminBuildPoliciesPage'
import AdminBuildsPage from '@/pages/admin/AdminBuildsPage'
import AdminDashboardPage from '@/pages/admin/AdminDashboardPage'
import AdminInfrastructureProviderDetailPage from '@/pages/admin/AdminInfrastructureProviderDetailPage'
import AdminInfrastructureProvidersPage from '@/pages/admin/AdminInfrastructureProvidersPage'
import AdminNotificationDefaultsPage from '@/pages/admin/AdminNotificationDefaultsPage'
import AuditLogsPage from '@/pages/admin/AuditLogsPage'
import ExternalServicesPage from '@/pages/admin/ExternalServicesPage'
import AuthProvidersPage from '@/pages/admin/AuthProvidersPage'
import InitialSetupPage from '@/pages/admin/InitialSetupPage'
import OperationalCapabilitiesPage from '@/pages/admin/OperationalCapabilitiesPage'
import SRESmartBotApprovalsPage from '@/pages/admin/SRESmartBotApprovalsPage'
import SRESmartBotDetectorRulesPage from '@/pages/admin/SRESmartBotDetectorRulesPage'
import SRESmartBotIncidentsPage from '@/pages/admin/SRESmartBotIncidentsPage'
import SRESmartBotSettingsPage from '@/pages/admin/SRESmartBotSettingsPage'
import PermissionDefinitionsPage from '@/pages/admin/PermissionDefinitionsPage'
import PermissionManagementPage from '@/pages/admin/PermissionManagementPage'
import QuarantineReviewWorkbenchPage from '@/pages/admin/QuarantineReviewWorkbenchPage'
import RoleManagementPage from '@/pages/admin/RoleManagementPage'
import SystemConfigurationPage from '@/pages/admin/SystemConfigurationPage'
import TenantDetailsPage from '@/pages/admin/TenantDetailsPage'
import TenantManagementPage from '@/pages/admin/TenantManagementPage'
import ToolManagementPage from '@/pages/admin/ToolManagementPage'
import UserInvitationsPage from '@/pages/admin/UserInvitationsPage'
import UserManagementPage from '@/pages/admin/UserManagementPage'
import ReviewerDashboardPage from '@/pages/reviewer/ReviewerDashboardPage'
import ReviewerEprApprovalsPage from '@/pages/reviewer/ReviewerEprApprovalsPage'
import ReviewerRequestsPage from '@/pages/reviewer/ReviewerRequestsPage'

// Hooks
import { useDashboardPath, useHasTenantAccess, useIsSecurityReviewer, useIsSystemAdmin } from '@/hooks/useAccess'
// import useBuildWebSocket from '@/hooks/useBuildWebSocket' // TODO: Implement /api/builds/events in Phase 6.2.9

// Root redirect component that waits for auth to be ready
function RootRedirect() {
    const { token, isLoading, setupRequired, requiresPasswordChange } = useAuthStore()
    const dashboardPath = useDashboardPath()
    const isAuthenticated = !!token

    // If not authenticated at all, go to login
    if (!isAuthenticated) {
        return <Navigate to="/landing" replace />
    }

    // If authenticated but still loading profile, show loading state
    if (isLoading) {
        return (
            <div className="flex items-center justify-center min-h-screen bg-gradient-to-br from-slate-100 to-slate-50 dark:from-slate-900 dark:to-slate-800">
                <div className="text-center">
                    <div className="inline-block">
                        <div className="w-12 h-12 border-4 border-slate-200 dark:border-slate-700 border-t-blue-600 dark:border-t-blue-400 rounded-full animate-spin"></div>
                    </div>
                    <p className="mt-4 text-slate-600 dark:text-slate-300">Loading profile...</p>
                </div>
            </div>
        )
    }

    // Profile is loaded (isLoading=false) - check access
    if (requiresPasswordChange) {
        return <Navigate to="/force-password-change" replace />
    }

    if (setupRequired && dashboardPath === '/admin/dashboard') {
        return <Navigate to="/admin/setup" replace />
    }

    return <Navigate to={dashboardPath} replace />
}

function LandingGuard() {
    const { token, isLoading } = useAuthStore()
    const isAuthenticated = !!token

    if (isAuthenticated) {
        if (isLoading) {
            return (
                <div className="flex items-center justify-center min-h-screen bg-gradient-to-br from-slate-100 to-slate-50 dark:from-slate-900 dark:to-slate-800">
                    <div className="text-center">
                        <div className="inline-block">
                            <div className="w-12 h-12 border-4 border-slate-200 dark:border-slate-700 border-t-blue-600 dark:border-t-blue-400 rounded-full animate-spin"></div>
                        </div>
                        <p className="mt-4 text-slate-600 dark:text-slate-300">Loading profile...</p>
                    </div>
                </div>
            )
        }
        return <Navigate to="/" replace />
    }

    return <LandingPage />
}

// Wrapper component for login that reactively checks auth state
function LoginGuard() {
    const { token, isLoading, setupRequired, requiresPasswordChange } = useAuthStore()
    const dashboardPath = useDashboardPath()
    const isAuthenticated = !!token

    // If authenticated, evaluate where to go
    if (isAuthenticated) {
        if (isLoading) {
            // Still loading profile, show spinner
            return (
                <div className="flex items-center justify-center min-h-screen bg-gradient-to-br from-slate-100 to-slate-50 dark:from-slate-900 dark:to-slate-800">
                    <div className="text-center">
                        <div className="inline-block">
                            <div className="w-12 h-12 border-4 border-slate-200 dark:border-slate-700 border-t-blue-600 dark:border-t-blue-400 rounded-full animate-spin"></div>
                        </div>
                        <p className="mt-4 text-slate-600 dark:text-slate-300">Loading profile...</p>
                    </div>
                </div>
            )
        }

        // Profile loaded, check where user should go
        if (requiresPasswordChange) {
            return <Navigate to="/force-password-change" replace />
        }

        if (setupRequired && dashboardPath === '/admin/dashboard') {
            return <Navigate to="/admin/setup" replace />
        }

        return <Navigate to={dashboardPath} replace />
    }

    // Not authenticated, show login page
    return <LoginPage />
}

// Legacy route redirect: /builds/:buildId/configure -> wizard clone flow
function BuildConfigureRedirect() {
    const { buildId } = useParams<{ buildId: string }>()
    const target = buildId ? `/builds/new?cloneFrom=${encodeURIComponent(buildId)}` : '/builds/new'
    return <Navigate to={target} replace />
}

// Separate component to use hooks inside the providers
function AppRoutes() {
    const { initTheme } = useThemeStore()
    const { token, groups, refreshProfile, setLoading, isLoading } = useAuthStore()
    const selectedTenantId = useTenantStore((state) => state.selectedTenantId)
    const refreshOperationCapabilities = useOperationCapabilitiesStore((state) => state.refreshForTenant)
    const resetOperationCapabilities = useOperationCapabilitiesStore((state) => state.reset)
    const refreshCapabilitySurfaces = useCapabilitySurfacesStore((state) => state.refreshForTenant)
    const resetCapabilitySurfaces = useCapabilitySurfacesStore((state) => state.reset)
    const refreshAttemptedRef = useRef(false)
    // const ws = useBuildWebSocket() // TODO: Implement /api/builds/events in Phase 6.2.9

    // Initialize theme on app load
    useEffect(() => {
        initTheme()
    }, [initTheme])

    // Refresh profile on app load if token exists but groups haven't been fetched yet
    // Only attempt once per app mount and not during active login
    useEffect(() => {
        if (token && !groups && !isLoading && !refreshAttemptedRef.current) {
            refreshAttemptedRef.current = true
            setLoading(true)
            refreshProfile().catch((error) => {
                console.error('Failed to refresh profile on app init', error)
            })
        }
    }, [token, groups, isLoading, refreshProfile, setLoading])

    useEffect(() => {
        if (!token) {
            resetOperationCapabilities()
            resetCapabilitySurfaces()
            return
        }
        if (!selectedTenantId) {
            return
        }
        refreshOperationCapabilities(selectedTenantId).catch(() => {
            // Store already captures/normalizes failures to fail-closed defaults.
        })
        refreshCapabilitySurfaces(selectedTenantId).catch(() => {
            // Store already captures/normalizes failures to fail-closed defaults.
        })
    }, [
        token,
        selectedTenantId,
        refreshOperationCapabilities,
        resetOperationCapabilities,
        refreshCapabilitySurfaces,
        resetCapabilitySurfaces,
    ])

    // Get routing values for conditional route rendering
    const isAuthenticated = useAuthStore((state) => !!state.token)
    const isAdmin = useIsSystemAdmin()
    const isSecurityReviewer = useIsSecurityReviewer()
    const hasTenantAccess = useHasTenantAccess()
    const dashboardPath = useDashboardPath()

    return (
        <div className="min-h-screen bg-slate-50 dark:bg-slate-950">
            {/* Global loading check - if we have token but still loading profile, show loading for all routes */}
            {isAuthenticated && isLoading ? (
                <div className="flex items-center justify-center min-h-screen bg-white dark:bg-slate-900">
                    <div className="text-center">
                        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600 mx-auto mb-4"></div>
                        <p className="text-slate-600 dark:text-slate-300">Loading profile...</p>
                    </div>
                </div>
            ) : (
                <Routes>
                    {/* Root route - uses RootRedirect component for better reactivity */}
                    <Route path="/" element={<RootRedirect />} />

                    {/* Public routes */}
                    <Route
                        path="/landing"
                        element={<LandingGuard />}
                    />
                    <Route
                        path="/login"
                        element={<LoginGuard />}
                    />
                    <Route
                        path="/force-password-change"
                        element={
                            isAuthenticated ? (
                                <ForcePasswordChangePage />
                            ) : (
                                <Navigate to="/login" replace />
                            )
                        }
                    />
                    <Route
                        path="/forgot-password"
                        element={
                            isAuthenticated ? (
                                <Navigate to={dashboardPath} replace />
                            ) : (
                                <ForgotPasswordPage />
                            )
                        }
                    />
                    <Route
                        path="/reset-password"
                        element={
                            isAuthenticated ? (
                                <Navigate to={dashboardPath} replace />
                            ) : (
                                <ResetPasswordPage />
                            )
                        }
                    />
                    <Route
                        path="/accept-invitation"
                        element={<AcceptInvitationPage />}
                    />

                    {/* Protected routes */}
                    {/* No tenant access page */}
                    <Route
                        path="/no-access"
                        element={
                            <ProtectedRoute>
                                <NoTenantAccessPage />
                            </ProtectedRoute>
                        }
                    />

                    {/* Admin routes - protected by ProtectedRoute + AdminLayout checks */}
                    <Route
                        path="/admin/*"
                        element={
                            <ProtectedRoute>
                                {isLoading ? (
                                    <div className="flex items-center justify-center min-h-screen bg-white dark:bg-slate-900">
                                        <div className="text-center">
                                            <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600 mx-auto mb-4"></div>
                                            <p className="text-slate-600 dark:text-slate-300">Loading profile...</p>
                                        </div>
                                    </div>
                                ) : isAdmin ? (
                                    <AdminLayout>
                                        <Routes>
                                            <Route path="/" element={<AdminDashboardPage />} />
                                            <Route path="dashboard" element={<AdminDashboardPage />} />
                                            <Route path="users" element={<UserManagementPage />} />
                                            <Route path="users/invitations" element={<UserInvitationsPage />} />
                                            <Route path="access/roles" element={<RoleManagementPage />} />
                                            <Route path="access/permissions" element={<PermissionManagementPage />} />
                                            <Route path="access/permission-definitions" element={<PermissionDefinitionsPage />} />
                                            <Route path="access/operational-capabilities" element={<OperationalCapabilitiesPage />} />
                                            <Route path="operations/sre-smart-bot" element={<SRESmartBotIncidentsPage />} />
                                            <Route path="operations/sre-smart-bot/approvals" element={<SRESmartBotApprovalsPage />} />
                                            <Route path="operations/sre-smart-bot/detector-rules" element={<SRESmartBotDetectorRulesPage />} />
                                            <Route path="operations/sre-smart-bot/settings" element={<SRESmartBotSettingsPage />} />
                                            <Route path="tenants" element={<TenantManagementPage />} />
                                            <Route path="tenants/:id" element={<TenantDetailsPage />} />
                                            <Route path="system-config" element={<SystemConfigurationPage />} />
                                            <Route path="notifications/defaults" element={<AdminNotificationDefaultsPage />} />
                                            <Route path="external-services" element={<ExternalServicesPage />} />
                                            <Route path="auth-providers" element={<AuthProvidersPage />} />
                                            <Route path="setup" element={<InitialSetupPage />} />
                                            <Route path="audit-logs" element={<AuditLogsPage />} />
                                            <Route path="tools" element={<ToolManagementPage />} />
                                            <Route path="builds" element={<AdminBuildsPage />} />
                                            <Route path="builds/nodes" element={<AdminBuildNodesPage />} />
                                            <Route path="builds/analytics" element={<AdminBuildAnalyticsPage />} />
                                            <Route path="builds/policies" element={<AdminBuildPoliciesPage />} />
                                            <Route path="infrastructure" element={<AdminInfrastructureProvidersPage />} />
                                            <Route path="infrastructure/:id" element={<AdminInfrastructureProviderDetailPage />} />
                                            <Route path="images/:imageId" element={<ImageDetailPage />} />
                                            <Route path="images/:imageId/edit" element={
                                                <PermissionProtectedRoute resource="image" action="update">
                                                    <CreateImagePage />
                                                </PermissionProtectedRoute>
                                            } />
                                            <Route path="images/create" element={
                                                <PermissionProtectedRoute resource="image" action="create">
                                                    <CreateImagePage />
                                                </PermissionProtectedRoute>
                                            } />
                                            <Route path="images/scans" element={<OnDemandScansPage />} />
                                            <Route path="quarantine/requests" element={<QuarantineReviewWorkbenchPage mode="requests" />} />
                                            <Route path="quarantine/requests/:requestId" element={<QuarantineRequestDetailPage scope="admin" />} />
                                            <Route path="quarantine/review" element={<QuarantineReviewWorkbenchPage mode="requests" />} />
                                            <Route path="images" element={<ImagesPage />} />
                                        </Routes>
                                    </AdminLayout>
                                ) : (
                                    <Navigate to="/dashboard" replace />
                                )}
                            </ProtectedRoute>
                        }
                    />

                    {/* Admin dashboard redirect - if admin tries to access /dashboard, redirect to /admin */}
                    <Route
                        path="/dashboard"
                        element={
                            <ProtectedRoute>
                                {isLoading ? (
                                    <div className="flex items-center justify-center min-h-screen bg-white dark:bg-slate-900">
                                        <div className="text-center">
                                            <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600 mx-auto mb-4"></div>
                                            <p className="text-slate-600 dark:text-slate-300">Loading profile...</p>
                                        </div>
                                    </div>
                                ) : isAdmin ? (
                                    <Navigate to="/admin/dashboard" replace />
                                ) : isSecurityReviewer ? (
                                    <Navigate to="/reviewer/dashboard" replace />
                                ) : hasTenantAccess ? (
                                    <ContextGuard>
                                        <Layout>
                                            <DashboardPage />
                                        </Layout>
                                    </ContextGuard>
                                ) : (
                                    <NoTenantAccessPage />
                                )}
                            </ProtectedRoute>
                        }
                    />

                    <Route
                        path="/reviewer/*"
                        element={
                            <ProtectedRoute>
                                {isLoading ? (
                                    <div className="flex items-center justify-center min-h-screen bg-white dark:bg-slate-900">
                                        <div className="text-center">
                                            <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600 mx-auto mb-4"></div>
                                            <p className="text-slate-600 dark:text-slate-300">Loading profile...</p>
                                        </div>
                                    </div>
                                ) : isAdmin ? (
                                    <Navigate to="/admin/dashboard" replace />
                                ) : !isSecurityReviewer ? (
                                    <Navigate to="/dashboard" replace />
                                ) : (
                                    <ReviewerLayout>
                                        <Routes>
                                            <Route path="/" element={<Navigate to="/reviewer/dashboard" replace />} />
                                            <Route path="dashboard" element={<ReviewerDashboardPage />} />
                                            <Route path="quarantine/requests" element={<ReviewerRequestsPage />} />
                                            <Route path="quarantine/requests/:requestId" element={<QuarantineRequestDetailPage scope="admin" />} />
                                            <Route path="quarantine/review" element={<ReviewerRequestsPage />} />
                                            <Route path="epr/approvals" element={<ReviewerEprApprovalsPage />} />
                                        </Routes>
                                    </ReviewerLayout>
                                )}
                            </ProtectedRoute>
                        }
                    />

                    {/* Regular user and general routes */}
                    <Route
                        path="/*"
                        element={
                            <ProtectedRoute>
                                {/* Show loading while profile is being fetched */}
                                {isLoading ? (
                                    <div className="flex items-center justify-center min-h-screen bg-white dark:bg-slate-900">
                                        <div className="text-center">
                                            <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600 mx-auto mb-4"></div>
                                            <p className="text-slate-600 dark:text-slate-300">Loading profile...</p>
                                        </div>
                                    </div>
                                ) : !hasTenantAccess && !isAdmin && !isSecurityReviewer ? (
                                    <NoTenantAccessPage />
                                ) : isSecurityReviewer ? (
                                    <Navigate to="/reviewer/dashboard" replace />
                                ) : (
                                    <ContextGuard>
                                        <Layout>
                                            <Routes>
                                                <Route path="/" element={<Navigate to={dashboardPath} replace />} />
                                                <Route path="/dashboard" element={<DashboardPage />} />

                                                {/* Project routes */}
                                                <Route
                                                    path="/projects"
                                                    element={
                                                        <CapabilityProtectedRoute routeKey="projects.list">
                                                            <ProjectsPage />
                                                        </CapabilityProtectedRoute>
                                                    }
                                                />
                                                <Route path="/projects/new" element={
                                                    <CapabilityProtectedRoute routeKey="projects.create">
                                                        <PermissionProtectedRoute resource="project" action="create">
                                                            <CreateProjectPage />
                                                        </PermissionProtectedRoute>
                                                    </CapabilityProtectedRoute>
                                                } />
                                                <Route path="/projects/:projectId/edit" element={
                                                    <CapabilityProtectedRoute routeKey="projects.edit">
                                                        <PermissionProtectedRoute resource="project" action="update">
                                                            <EditProjectPage />
                                                        </PermissionProtectedRoute>
                                                    </CapabilityProtectedRoute>
                                                } />
                                                <Route
                                                    path="/projects/:projectId"
                                                    element={
                                                        <CapabilityProtectedRoute routeKey="projects.detail">
                                                            <ProjectDetailPage />
                                                        </CapabilityProtectedRoute>
                                                    }
                                                />
                                                <Route
                                                    path="/projects/:projectId/builds"
                                                    element={
                                                        <CapabilityProtectedRoute routeKey="builds.list">
                                                            <ProjectBuildsPage />
                                                        </CapabilityProtectedRoute>
                                                    }
                                                />

                                                {/* Tenant routes */}
                                                <Route path="/tenants" element={<TenantsPage />} />
                                                <Route path="/tenants/:tenantId" element={<TenantDetailPage />} />
                                                <Route path="/members" element={<MembersPage />} />
                                                <Route path="/invitations" element={<InvitationsPage />} />

                                                {/* Build routes */}
                                                <Route
                                                    path="/builds"
                                                    element={
                                                        <CapabilityProtectedRoute routeKey="builds.list">
                                                            <BuildsPage />
                                                        </CapabilityProtectedRoute>
                                                    }
                                                />
                                                <Route path="/builds/new" element={
                                                    <CapabilityProtectedRoute routeKey="builds.create">
                                                        <PermissionProtectedRoute resource="build" action="create">
                                                            <CreateBuildPage />
                                                        </PermissionProtectedRoute>
                                                    </CapabilityProtectedRoute>
                                                } />
                                                <Route
                                                    path="/builds/:buildId"
                                                    element={
                                                        <CapabilityProtectedRoute routeKey="builds.detail">
                                                            <BuildDetailPage />
                                                        </CapabilityProtectedRoute>
                                                    }
                                                />
                                                <Route
                                                    path="/builds/:buildId/configure"
                                                    element={
                                                        <CapabilityProtectedRoute routeKey="builds.detail">
                                                            <BuildConfigureRedirect />
                                                        </CapabilityProtectedRoute>
                                                    }
                                                />

                                                {/* Image routes */}
                                                <Route path="/images" element={<ImagesPage />} />
                                                <Route
                                                    path="/images/scans"
                                                    element={
                                                        <CapabilityProtectedRoute routeKey="images.scan.ondemand">
                                                            <OnDemandScansPage />
                                                        </CapabilityProtectedRoute>
                                                    }
                                                />
                                                <Route
                                                    path="/quarantine/requests"
                                                    element={
                                                        <CapabilityProtectedRoute routeKey="quarantine.request.list">
                                                            <QuarantineRequestsPage mode="requests" />
                                                        </CapabilityProtectedRoute>
                                                    }
                                                />
                                                <Route
                                                    path="/quarantine/requests/:requestId"
                                                    element={
                                                        <CapabilityProtectedRoute routeKey="quarantine.request.list">
                                                            <QuarantineRequestDetailPage scope="tenant" />
                                                        </CapabilityProtectedRoute>
                                                    }
                                                />
                                                <Route
                                                    path="/quarantine/epr"
                                                    element={
                                                        <CapabilityProtectedRoute routeKey="quarantine.request.list">
                                                            <QuarantineRequestsPage mode="epr" />
                                                        </CapabilityProtectedRoute>
                                                    }
                                                />
                                                <Route
                                                    path="/quarantine/releases"
                                                    element={
                                                        <CapabilityProtectedRoute routeKey="quarantine.request.list">
                                                            <ReleasedArtifactsPage />
                                                        </CapabilityProtectedRoute>
                                                    }
                                                />
                                                <Route path="/images/create" element={
                                                    <PermissionProtectedRoute resource="image" action="create">
                                                        <CreateImagePage />
                                                    </PermissionProtectedRoute>
                                                } />
                                                <Route path="/images/:imageId" element={<ImageDetailPage />} />
                                                <Route path="/images/:imageId/edit" element={
                                                    <PermissionProtectedRoute resource="image" action="update">
                                                        <CreateImagePage />
                                                    </PermissionProtectedRoute>
                                                } />

                                                {/* Profile */}
                                                <Route path="/profile" element={<ProfilePage />} />
                                                <Route path="/notifications" element={<NotificationsPage />} />
                                                <Route path="/help/capabilities" element={<CapabilityAccessPage />} />

                                                {/* Settings */}
                                                <Route path="/settings" element={<SettingsPage />} />
                                                <Route
                                                    path="/settings/auth"
                                                    element={
                                                        <CapabilityProtectedRoute routeKey="settings.auth">
                                                            <AuthManagementPage />
                                                        </CapabilityProtectedRoute>
                                                    }
                                                />

                                                {/* 404 */}
                                                <Route path="*" element={<NotFoundPage />} />
                                            </Routes>
                                        </Layout>
                                    </ContextGuard>
                                )}
                            </ProtectedRoute>
                        }
                    />
                </Routes >
            )}
        </div >
    )
}

function App() {
    return (
        <RefreshProvider>
            <NotificationProvider>
                <AppRoutes />
                <NotificationContainer />
            </NotificationProvider>
        </RefreshProvider>
    )
}

export default App
