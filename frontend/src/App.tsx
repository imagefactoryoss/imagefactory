import NotificationContainer from '@/components/notifications/NotificationContainer'
import { NotificationProvider } from '@/context/NotificationContext'
import { RefreshProvider } from '@/context/RefreshContext'
import { useAuthStore } from '@/store/auth'
import { useCapabilitySurfacesStore } from '@/store/capabilitySurfaces'
import { useOperationCapabilitiesStore } from '@/store/operationCapabilities'
import { useTenantStore } from '@/store/tenant'
import { useThemeStore } from '@/store/theme'
import { Suspense, lazy, useEffect, useRef } from 'react'
import { Navigate, Route, Routes, useParams } from 'react-router-dom'

// Layout components
import ContextGuard from '@/components/auth/ContextGuard'
import AdminLayout from '@/components/layout/AdminLayout'
import CapabilityProtectedRoute from '@/components/layout/CapabilityProtectedRoute'
import Layout from '@/components/layout/Layout'
import PermissionProtectedRoute from '@/components/layout/PermissionProtectedRoute'
import ProtectedRoute from '@/components/layout/ProtectedRoute'
import ReviewerLayout from '@/components/layout/ReviewerLayout'


// Hooks
import { useDashboardPath, useHasTenantAccess, useIsSecurityReviewer, useIsSystemAdmin } from '@/hooks/useAccess'
// import useBuildWebSocket from '@/hooks/useBuildWebSocket' // TODO: Implement /api/builds/events in Phase 6.2.9

const AcceptInvitationPage = lazy(() => import('@/pages/auth/AcceptInvitationPage'))
const ForcePasswordChangePage = lazy(() => import('@/pages/auth/ForcePasswordChangePage'))
const ForgotPasswordPage = lazy(() => import('@/pages/auth/ForgotPasswordPage'))
const LandingPage = lazy(() => import('@/pages/auth/LandingPage'))
const LoginPage = lazy(() => import('@/pages/auth/LoginPage'))
const ResetPasswordPage = lazy(() => import('@/pages/auth/ResetPasswordPage'))
const BuildDetailPage = lazy(() => import('@/pages/builds/BuildDetailPage'))
const BuildsPage = lazy(() => import('@/pages/builds/BuildsPage'))
const CreateBuildPage = lazy(() => import('@/pages/builds/CreateBuildPage'))
const DashboardPage = lazy(() => import('@/pages/DashboardPage'))
const CapabilityAccessPage = lazy(() => import('@/pages/help/CapabilityAccessPage'))
const ProductInfoPage = lazy(() => import('@/pages/help/ProductInfoPage'))
const CreateImagePage = lazy(() => import('@/pages/images/CreateImagePage'))
const ImageDetailPage = lazy(() => import('@/pages/images/ImageDetailPage'))
const ImagesPage = lazy(() => import('@/pages/images/ImagesPage'))
const OnDemandScansPage = lazy(() => import('@/pages/images/OnDemandScansPage'))
const VMImagesPage = lazy(() => import('@/pages/images/VMImagesPage'))
const InvitationsPage = lazy(() => import('@/pages/members/InvitationsPage'))
const MembersPage = lazy(() => import('@/pages/members/MembersPage'))
const NoTenantAccessPage = lazy(() => import('@/pages/NoTenantAccessPage'))
const NotFoundPage = lazy(() => import('@/pages/NotFoundPage'))
const NotificationsPage = lazy(() => import('@/pages/NotificationsPage'))
const ProfilePage = lazy(() => import('@/pages/ProfilePage'))
const CreateProjectPage = lazy(() => import('@/pages/projects/CreateProjectPage'))
const EditProjectPage = lazy(() => import('@/pages/projects/EditProjectPage'))
const ProjectDetailPage = lazy(() => import('@/pages/projects/ProjectDetailPage'))
const ProjectsPage = lazy(() => import('@/pages/projects/ProjectsPage'))
const ProjectBuildsPage = lazy(() => import('@/pages/projects/ProjectBuildsPage'))
const QuarantineRequestDetailPage = lazy(() => import('@/pages/quarantine/QuarantineRequestDetailPage'))
const QuarantineRequestsPage = lazy(() => import('@/pages/quarantine/QuarantineRequestsPage'))
const ReleasedArtifactsPage = lazy(() => import('@/pages/quarantine/ReleasedArtifactsPage'))
const AuthManagementPage = lazy(() => import('@/pages/settings/AuthManagementPage'))
const SettingsPage = lazy(() => import('@/pages/SettingsPage'))
const TenantDetailPage = lazy(() => import('@/pages/tenants/TenantDetailPage'))
const TenantsPage = lazy(() => import('@/pages/tenants/TenantsPage'))

const AdminBuildAnalyticsPage = lazy(() => import('@/pages/admin/AdminBuildAnalyticsPage'))
const AdminBuildNodesPage = lazy(() => import('@/pages/admin/AdminBuildNodesPage'))
const AdminBuildPoliciesPage = lazy(() => import('@/pages/admin/AdminBuildPoliciesPage'))
const AdminBuildsPage = lazy(() => import('@/pages/admin/AdminBuildsPage'))
const AdminDashboardPage = lazy(() => import('@/pages/admin/AdminDashboardPage'))
const AdminNotificationDefaultsPage = lazy(() => import('@/pages/admin/AdminNotificationDefaultsPage'))
const AdminInfrastructureProviderDetailPage = lazy(() => import('@/pages/admin/AdminInfrastructureProviderDetailPage'))
const AdminInfrastructureProvidersPage = lazy(() => import('@/pages/admin/AdminInfrastructureProvidersPage'))
const AdminPackerTargetProfilesPage = lazy(() => import('@/pages/admin/AdminPackerTargetProfilesPage'))
const AuditLogsPage = lazy(() => import('@/pages/admin/AuditLogsPage'))
const AuthProvidersPage = lazy(() => import('@/pages/admin/AuthProvidersPage'))
const ExternalServicesPage = lazy(() => import('@/pages/admin/ExternalServicesPage'))
const InitialSetupPage = lazy(() => import('@/pages/admin/InitialSetupPage'))
const OperationalCapabilitiesPage = lazy(() => import('@/pages/admin/OperationalCapabilitiesPage'))
const PermissionDefinitionsPage = lazy(() => import('@/pages/admin/PermissionDefinitionsPage'))
const PermissionManagementPage = lazy(() => import('@/pages/admin/PermissionManagementPage'))
const QuarantineReviewWorkbenchPage = lazy(() => import('@/pages/admin/QuarantineReviewWorkbenchPage'))
const RoleManagementPage = lazy(() => import('@/pages/admin/RoleManagementPage'))
const SRESmartBotActiveDetectorRulesSettingsPage = lazy(() => import('@/pages/admin/SRESmartBotActiveDetectorRulesSettingsPage'))
const SRESmartBotApprovalsPage = lazy(() => import('@/pages/admin/SRESmartBotApprovalsPage'))
const SRESmartBotDetectorRulesPage = lazy(() => import('@/pages/admin/SRESmartBotDetectorRulesPage'))
const SRESmartBotIncidentsPage = lazy(() => import('@/pages/admin/SRESmartBotIncidentsPage'))
const SRESmartBotIncidentTimelinePage = lazy(() => import('@/pages/admin/SRESmartBotIncidentTimelinePage'))
const SRESmartBotOperatorRulesSettingsPage = lazy(() => import('@/pages/admin/SRESmartBotOperatorRulesSettingsPage'))
const SRESmartBotSettingsPage = lazy(() => import('@/pages/admin/SRESmartBotSettingsPage'))
const SystemConfigurationPage = lazy(() => import('@/pages/admin/SystemConfigurationPage'))
const TenantDetailsPage = lazy(() => import('@/pages/admin/TenantDetailsPage'))
const TenantManagementPage = lazy(() => import('@/pages/admin/TenantManagementPage'))
const ToolManagementPage = lazy(() => import('@/pages/admin/ToolManagementPage'))
const UserInvitationsPage = lazy(() => import('@/pages/admin/UserInvitationsPage'))
const UserManagementPage = lazy(() => import('@/pages/admin/UserManagementPage'))

const ReviewerDashboardPage = lazy(() => import('@/pages/reviewer/ReviewerDashboardPage'))
const ReviewerEprApprovalsPage = lazy(() => import('@/pages/reviewer/ReviewerEprApprovalsPage'))
const ReviewerRequestsPage = lazy(() => import('@/pages/reviewer/ReviewerRequestsPage'))

const RouteLoadingFallback = () => (
    <div className="flex items-center justify-center min-h-screen bg-white dark:bg-slate-900">
        <div className="text-center">
            <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600 mx-auto mb-4"></div>
            <p className="text-slate-600 dark:text-slate-300">Loading page...</p>
        </div>
    </div>
)

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
                <Suspense fallback={<RouteLoadingFallback />}>
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
                                            <Route path="operations/sre-smart-bot/incidents/:incidentId/timeline" element={<SRESmartBotIncidentTimelinePage />} />
                                            <Route path="operations/sre-smart-bot/approvals" element={<SRESmartBotApprovalsPage />} />
                                            <Route path="operations/sre-smart-bot/detector-rules" element={<SRESmartBotDetectorRulesPage />} />
                                            <Route path="operations/sre-smart-bot/settings" element={<SRESmartBotSettingsPage />} />
                                            <Route path="operations/sre-smart-bot/settings/operator-rules" element={<SRESmartBotOperatorRulesSettingsPage />} />
                                            <Route path="operations/sre-smart-bot/settings/detector-rules" element={<SRESmartBotActiveDetectorRulesSettingsPage />} />
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
                                            <Route path="infrastructure/packer-target-profiles" element={<AdminPackerTargetProfilesPage />} />
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
                                            <Route path="help/product-info" element={<ProductInfoPage />} />
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
                                                <Route path="/images/vm" element={<VMImagesPage />} />
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
                                                <Route path="/help/product-info" element={<ProductInfoPage />} />

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
                </Suspense>
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
