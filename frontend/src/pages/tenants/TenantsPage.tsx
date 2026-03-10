import { Breadcrumb } from '@/components/common/Breadcrumb'
import { EmptyState } from '@/components/common/EmptyState'
import { TenantService } from '@/services/tenantService'
import { useTenantStore } from '@/store/tenant'
import { Tenant } from '@/types'
import React, { useEffect, useState } from 'react'
import toast from 'react-hot-toast'
import { Link } from 'react-router-dom'

const TenantsPage: React.FC = () => {
    const { selectedTenantId, selectedRoleId } = useTenantStore()
    const [tenants, setTenants] = useState<Tenant[]>([])
    const [currentTenant, setCurrentTenant] = useState<Tenant | null>(null)
    const [loading, setLoading] = useState(true)
    const [error, setError] = useState<string | null>(null)

    useEffect(() => {
        loadTenants()
    }, [selectedTenantId])

    const loadTenants = async () => {
        try {
            setLoading(true)
            setError(null)

            // Load all tenants (API will filter by user's ownership on backend eventually)
            const response = await TenantService.getTenants(1, 100)
            setTenants(response.data || [])

            // Also load the current tenant if selected
            if (selectedTenantId) {
                try {
                    const currentTenantData = await TenantService.getTenant(selectedTenantId)
                    setCurrentTenant(currentTenantData)
                } catch (err) {
                }
            }
        } catch (err) {
            setError('Failed to load tenants')
            toast.error('Failed to load tenants')
        } finally {
            setLoading(false)
        }
    }

    const formatDate = (dateString: string) => {
        return new Date(dateString).toLocaleDateString()
    }

    const getStatusColor = (status: string) => {
        switch (status) {
            case 'active':
                return 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200'
            case 'suspended':
                return 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200'
            case 'pending':
                return 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200'
            default:
                return 'bg-slate-100 text-slate-800 dark:bg-slate-900 dark:text-slate-200'
        }
    }

    if (loading) {
        return (
            <div className="px-4 py-6 sm:px-6 lg:px-8">
                <div className="flex justify-center items-center h-64">
                    <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-500"></div>
                </div>
            </div>
        )
    }

    return (
        <div className="px-4 py-6 sm:px-6 lg:px-8">
            {/* Breadcrumb */}
            <Breadcrumb
                items={[
                    { name: 'Dashboard', href: '/dashboard' },
                    { name: 'Tenants' }
                ]}
            />

            {/* Header */}
            <div className="sm:flex sm:items-center sm:justify-between mt-6">
                <div className="sm:flex-auto">
                    <h1 className="text-2xl font-semibold text-slate-900 dark:text-white">
                        My Tenants
                    </h1>
                    <p className="mt-2 text-sm text-slate-700 dark:text-slate-400">
                        View tenant organizations where you are an owner. You can manage members, projects, and builds within each tenant.
                    </p>
                </div>
            </div>

            {/* Error Message */}
            {error && (
                <div className="mt-6 bg-red-50 dark:bg-red-900 border border-red-200 dark:border-red-700 rounded-md p-4">
                    <div className="text-red-800 dark:text-red-200">{error}</div>
                </div>
            )}

            {/* Current Tenant Section */}
            {currentTenant && selectedRoleId === 'owner' && (
                <div className="mt-8 p-4 bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-lg">
                    <div className="flex items-start justify-between">
                        <div>
                            <h3 className="text-lg font-semibold text-slate-900 dark:text-white">
                                📍 Current Tenant: {currentTenant.name}
                            </h3>
                            <p className="mt-1 text-sm text-slate-600 dark:text-slate-400">
                                You are an Owner of this tenant. You can manage members, view builds, and manage projects.
                            </p>
                        </div>
                        <Link
                            to={`/projects`}
                            className="inline-flex items-center px-3 py-1.5 text-sm font-medium text-blue-600 hover:text-blue-500 dark:text-blue-400 dark:hover:text-blue-300 bg-blue-100 dark:bg-blue-900/30 rounded hover:bg-blue-200 dark:hover:bg-blue-900/50 transition"
                        >
                            Manage →
                        </Link>
                    </div>
                </div>
            )}

            {/* Tenants List */}
            {!currentTenant && tenants.length === 0 ? (
                <div className="mt-8">
                    <EmptyState
                        icon="🏢"
                        title="No tenants"
                        description="You are not currently an owner of any tenants. Contact your System Administrator to add you as an owner to a tenant."
                    />
                </div>
            ) : tenants.length > 0 ? (
                <div className="mt-8 overflow-hidden shadow ring-1 ring-black ring-opacity-5 md:rounded-lg">
                    <table className="min-w-full divide-y divide-slate-300 dark:divide-slate-600">
                        <thead className="bg-slate-50 dark:bg-slate-800">
                            <tr>
                                <th className="px-6 py-3 text-left text-xs font-medium text-slate-900 dark:text-slate-100 uppercase tracking-wider">
                                    Name
                                </th>
                                <th className="px-6 py-3 text-left text-xs font-medium text-slate-900 dark:text-slate-100 uppercase tracking-wider">
                                    Slug
                                </th>
                                <th className="px-6 py-3 text-left text-xs font-medium text-slate-900 dark:text-slate-100 uppercase tracking-wider">
                                    Status
                                </th>
                                <th className="px-6 py-3 text-left text-xs font-medium text-slate-900 dark:text-slate-100 uppercase tracking-wider">
                                    Quota
                                </th>
                                <th className="px-6 py-3 text-left text-xs font-medium text-slate-900 dark:text-slate-100 uppercase tracking-wider">
                                    Created
                                </th>
                                <th className="px-6 py-3 text-left text-xs font-medium text-slate-900 dark:text-slate-100 uppercase tracking-wider">
                                    Actions
                                </th>
                            </tr>
                        </thead>
                        <tbody className="divide-y divide-slate-200 dark:divide-slate-700 bg-white dark:bg-slate-800">
                            {tenants.map((tenant) => (
                                <tr key={tenant.id} className="hover:bg-slate-50 dark:hover:bg-slate-700 transition-colors">
                                    <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-slate-900 dark:text-white">
                                        {tenant.name}
                                    </td>
                                    <td className="px-6 py-4 whitespace-nowrap text-sm text-slate-500 dark:text-slate-400">
                                        {tenant.slug}
                                    </td>
                                    <td className="px-6 py-4 whitespace-nowrap text-sm">
                                        <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${getStatusColor(tenant.status)}`}>
                                            {tenant.status}
                                        </span>
                                    </td>
                                    <td className="px-6 py-4 whitespace-nowrap text-sm text-slate-500 dark:text-slate-400">
                                        <div className="text-xs">
                                            <div>Builds: {tenant.quota?.maxBuilds || '-'}</div>
                                            <div>Storage: {tenant.quota?.maxStorageGB || '-'}GB</div>
                                        </div>
                                    </td>
                                    <td className="px-6 py-4 whitespace-nowrap text-sm text-slate-500 dark:text-slate-400">
                                        {formatDate(tenant.createdAt)}
                                    </td>
                                    <td className="px-6 py-4 whitespace-nowrap text-sm font-medium">
                                        <div className="flex items-center space-x-3">
                                            <Link
                                                to={`/members?tenantId=${tenant.id}`}
                                                className="text-blue-600 hover:text-blue-500 dark:text-blue-400 dark:hover:text-blue-300"
                                            >
                                                Manage
                                            </Link>
                                        </div>
                                    </td>
                                </tr>
                            ))}
                        </tbody>
                    </table>
                </div>
            ) : null}
        </div>
    )
}

export default TenantsPage