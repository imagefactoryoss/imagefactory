import Drawer from '@/components/ui/Drawer'
import { useCanManageAdmin } from '@/hooks/useAccess'
import { api } from '@/services/api'
import React, { useCallback, useEffect, useState } from 'react'
import toast from 'react-hot-toast'

interface ExternalService {
    name: string
    description: string
    url: string
    api_key?: string
    headers?: Record<string, string>
    enabled: boolean
}

interface ExternalServiceFormData {
    name: string
    description: string
    url: string
    api_key: string
    headers: Record<string, string>
    enabled: boolean
}

const ExternalServicesPage: React.FC = () => {
    const canManageAdmin = useCanManageAdmin()
    const [services, setServices] = useState<ExternalService[]>([])
    const [loading, setLoading] = useState(true)
    const [drawerOpen, setDrawerOpen] = useState(false)
    const [drawerMode, setDrawerMode] = useState<'create' | 'edit' | 'view'>('create')
    const [editingService, setEditingService] = useState<ExternalService | null>(null)
    const [formData, setFormData] = useState<ExternalServiceFormData>({
        name: '',
        description: '',
        url: '',
        api_key: '',
        headers: {},
        enabled: true,
    })
    const [headerKey, setHeaderKey] = useState('')
    const [headerValue, setHeaderValue] = useState('')

    const loadExternalServices = useCallback(async () => {
        try {
            console.log('Loading external services...')
            const response = await api.get('/admin/external-services?all_tenants=true')
            console.log('External services response:', response.status)
            const data = response.data
            console.log('External services data:', data)
            setServices(data.services || [])
        } catch (error) {
            console.error('Error loading external services:', error)
            toast.error('Failed to load external services')
        } finally {
            setLoading(false)
        }
    }, [])

    useEffect(() => {
        loadExternalServices()
    }, [loadExternalServices])

    const handleCreateService = async (e: React.FormEvent) => {
        e.preventDefault()
        if (!canManageAdmin) {
            toast.error('Read-only mode.')
            return
        }

        try {
            await api.post('/admin/external-services', formData)
            toast.success('External service created successfully')
            closeDrawer()
            loadExternalServices()
        } catch (error) {
            console.error('Error creating external service:', error)
            toast.error('Failed to create external service')
        }
    }

    const handleUpdateService = async (e: React.FormEvent) => {
        e.preventDefault()
        if (!canManageAdmin) {
            toast.error('Read-only mode.')
            return
        }

        if (!editingService) {
            toast.error('No service selected. Please try again.')
            return
        }

        try {
            await api.put(`/admin/external-services/${editingService.name}`, formData)
            toast.success('External service updated successfully')
            closeDrawer()
            loadExternalServices()
        } catch (error) {
            console.error('Error updating external service:', error)
            toast.error('Failed to update external service')
        }
    }

    const handleDeleteService = async (serviceName: string) => {
        if (!canManageAdmin) {
            toast.error('Read-only mode.')
            return
        }
        if (!confirm(`Are you sure you want to delete the external service "${serviceName}"?`)) {
            return
        }

        try {
            await api.delete(`/admin/external-services/${serviceName}`)
            toast.success('External service deleted successfully')
            loadExternalServices()
        } catch (error) {
            console.error('Error deleting external service:', error)
            toast.error('Failed to delete external service')
        }
    }

    const handleTestConnection = async (serviceName: string) => {
        try {
            toast('Testing connection...')
            // Generate the config key from the service name (same logic as backend)
            const configKey = `external_service_${serviceName.toLowerCase().replace(/[\s-]/g, '_')}`
            const response = await api.post('/system-configs/test-connection?all_tenants=true', {
                config_key: configKey
            })
            if (response.data.success) {
                toast.success('Connection test successful!')
            } else {
                toast.error(`Connection test failed: ${response.data.message}`)
            }
        } catch (error: any) {
            console.error('Error testing connection:', error)
            const errorMessage = error.response?.data?.message || 'Failed to test connection'
            toast.error(`Connection test failed: ${errorMessage}`)
        }
    }

    const resetForm = () => {
        setFormData({
            name: '',
            description: '',
            url: '',
            api_key: '',
            headers: {},
            enabled: true,
        })
        setHeaderKey('')
        setHeaderValue('')
    }

    const startEditing = async (service: ExternalService) => {
        console.log('startEditing called with service:', service)
        toast('Loading service details...')
        try {
            console.log('Fetching service details for editing:', service.name)
            // Fetch the full service details including API key
            const response = await api.get(`/admin/external-services/${service.name}?all_tenants=true`)
            console.log('Response status:', response.status)
            const fullService = response.data
            console.log('Full service data:', fullService)
            setEditingService(fullService)
            setFormData({
                name: fullService.name,
                description: fullService.description,
                url: fullService.url,
                api_key: fullService.api_key || '',
                headers: fullService.headers || {},
                enabled: fullService.enabled,
            })
            setDrawerMode('edit')
            setDrawerOpen(true)
            console.log('Drawer should now be open')
            toast.success('Service details loaded')
        } catch (error) {
            console.error('Error loading service details:', error)
            toast.error('Failed to load service details for editing')
        }
    }

    const startViewing = async (service: ExternalService) => {
        console.log('startViewing called with service:', service)
        toast('Loading service details...')
        try {
            console.log('Fetching service details for viewing:', service.name)
            // Fetch the full service details including API key
            const response = await api.get(`/admin/external-services/${service.name}?all_tenants=true`)
            console.log('Response status:', response.status)
            const fullService = response.data
            console.log('Full service data:', fullService)
            setEditingService(fullService)
            setFormData({
                name: fullService.name,
                description: fullService.description,
                url: fullService.url,
                api_key: fullService.api_key || '',
                headers: fullService.headers || {},
                enabled: fullService.enabled,
            })
            setDrawerMode('view')
            setDrawerOpen(true)
            console.log('Drawer should now be open in view mode')
            toast.success('Service details loaded')
        } catch (error) {
            console.error('Error loading service details:', error)
            toast.error('Failed to load service details for viewing')
        }
    }

    const openCreateDrawer = () => {
        resetForm()
        setDrawerMode('create')
        setDrawerOpen(true)
    }

    const closeDrawer = () => {
        setDrawerOpen(false)
        setEditingService(null)
        resetForm()
    }

    const addHeader = () => {
        if (headerKey.trim() && headerValue.trim()) {
            setFormData(prev => ({
                ...prev,
                headers: {
                    ...prev.headers,
                    [headerKey.trim()]: headerValue.trim()
                }
            }))
            setHeaderKey('')
            setHeaderValue('')
        }
    }

    const removeHeader = (key: string) => {
        setFormData(prev => {
            const newHeaders = { ...prev.headers }
            delete newHeaders[key]
            return {
                ...prev,
                headers: newHeaders
            }
        })
    }



    if (loading) {
        return (
            <div className="flex justify-center items-center h-64">
                <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
            </div>
        )
    }

    return (
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
            <div className="mb-8">
                <h1 className="text-3xl font-bold text-gray-900 dark:text-slate-100">External Services</h1>
                <p className="mt-2 text-gray-600">
                    Configure external services and their API credentials for system integrations.
                </p>
            </div>

            {canManageAdmin && (
                <div className="mb-6">
                    <button
                        onClick={openCreateDrawer}
                        className="bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-md font-medium"
                    >
                        Add External Service
                    </button>
                </div>
            )}

            {!canManageAdmin && (
                <div className="mb-6 rounded-md border border-amber-300 bg-amber-50 px-4 py-3 text-sm text-amber-900 dark:border-amber-700 dark:bg-amber-900/20 dark:text-amber-200">
                    Read-only mode: external service create, edit, and delete actions are hidden for System Administrator Viewer.
                </div>
            )}

            {/* Create/Edit/View Drawer */}
            <Drawer
                isOpen={drawerOpen}
                onClose={closeDrawer}
                title={
                    drawerMode === 'edit' ? 'Edit External Service' :
                        drawerMode === 'view' ? 'View External Service' :
                            'Create External Service'
                }
            >
                {drawerMode === 'view' ? (
                    <div className="space-y-4">
                        <div>
                            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">
                                Service Name
                            </label>
                            <div className="mt-1 block w-full px-3 py-2 bg-gray-50 dark:bg-gray-700 border border-gray-300 dark:border-gray-600 rounded-md text-gray-900 dark:text-white">
                                {formData.name}
                            </div>
                        </div>

                        <div>
                            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">
                                Description
                            </label>
                            <div className="mt-1 block w-full px-3 py-2 bg-gray-50 dark:bg-gray-700 border border-gray-300 dark:border-gray-600 rounded-md text-gray-900 dark:text-white whitespace-pre-wrap">
                                {formData.description || 'No description'}
                            </div>
                        </div>

                        <div>
                            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">
                                Service URL
                            </label>
                            <div className="mt-1 block w-full px-3 py-2 bg-gray-50 dark:bg-gray-700 border border-gray-300 dark:border-gray-600 rounded-md text-gray-900 dark:text-white font-mono text-sm">
                                {formData.url}
                            </div>
                        </div>

                        <div>
                            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">
                                API Key
                            </label>
                            <div className="mt-1 block w-full px-3 py-2 bg-gray-50 dark:bg-gray-700 border border-gray-300 dark:border-gray-600 rounded-md text-gray-900 dark:text-white font-mono text-sm">
                                {'*'.repeat(Math.min(formData.api_key.length, 20))}
                            </div>
                        </div>

                        <div>
                            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                                Custom Headers
                            </label>
                            <div className="space-y-2">
                                {Object.keys(formData.headers).length > 0 ? (
                                    Object.entries(formData.headers).map(([key, value]) => (
                                        <div key={key} className="flex-1 px-3 py-2 bg-gray-50 dark:bg-gray-700 border border-gray-300 dark:border-gray-600 rounded-md text-sm font-mono text-gray-900 dark:text-white">
                                            {key}: {value}
                                        </div>
                                    ))
                                ) : (
                                    <div className="px-3 py-2 bg-gray-50 dark:bg-gray-700 border border-gray-300 dark:border-gray-600 rounded-md text-gray-500 dark:text-gray-400">
                                        No custom headers
                                    </div>
                                )}
                            </div>
                        </div>

                        <div className="flex items-center">
                            <input
                                type="checkbox"
                                checked={formData.enabled}
                                className="h-4 w-4 text-blue-600 border-gray-300 dark:border-gray-600 rounded"
                                disabled
                            />
                            <label className="ml-2 block text-sm text-gray-900 dark:text-white">
                                Enabled
                            </label>
                        </div>

                        <div className="flex space-x-3 pt-4">
                            <button
                                type="button"
                                onClick={() => handleTestConnection(formData.name)}
                                className="bg-green-600 hover:bg-green-700 text-white px-4 py-2 rounded-md font-medium"
                            >
                                Test Connection
                            </button>
                            {canManageAdmin && (
                                <button
                                    type="button"
                                    onClick={(e) => {
                                        e.preventDefault();
                                        console.log('Edit button clicked in view mode');
                                        setDrawerMode('edit');
                                        console.log('Drawer mode changed to edit');
                                    }}
                                    className="bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-md font-medium"
                                >
                                    Edit
                                </button>
                            )}
                            <button
                                type="button"
                                onClick={closeDrawer}
                                className="bg-gray-300 hover:bg-gray-400 dark:bg-gray-600 dark:hover:bg-gray-500 text-gray-800 dark:text-white px-4 py-2 rounded-md font-medium"
                            >
                                Cancel
                            </button>
                        </div>
                    </div>
                ) : canManageAdmin ? (
                    <form onSubmit={drawerMode === 'edit' ? handleUpdateService : handleCreateService} className="space-y-4">
                        <div>
                            <label htmlFor="name" className="block text-sm font-medium text-gray-700 dark:text-gray-300">
                                Service Name *
                            </label>
                            <input
                                type="text"
                                id="name"
                                value={formData.name}
                                onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                                className="mt-1 block w-full border border-gray-300 dark:border-gray-600 rounded-md shadow-sm px-3 py-2 bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:outline-none focus:ring-blue-500 focus:border-blue-500"
                                required
                                disabled={drawerMode === 'edit'} // Don't allow name changes when editing
                            />
                        </div>

                        <div>
                            <label htmlFor="description" className="block text-sm font-medium text-gray-700 dark:text-gray-300">
                                Description
                            </label>
                            <textarea
                                id="description"
                                value={formData.description}
                                onChange={(e) => setFormData({ ...formData, description: e.target.value })}
                                rows={3}
                                className="mt-1 block w-full border border-gray-300 dark:border-gray-600 rounded-md shadow-sm px-3 py-2 bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:outline-none focus:ring-blue-500 focus:border-blue-500"
                            />
                        </div>

                        <div>
                            <label htmlFor="url" className="block text-sm font-medium text-gray-700 dark:text-gray-300">
                                Service URL *
                            </label>
                            <input
                                type="url"
                                id="url"
                                value={formData.url}
                                onChange={(e) => setFormData({ ...formData, url: e.target.value })}
                                className="mt-1 block w-full border border-gray-300 dark:border-gray-600 rounded-md shadow-sm px-3 py-2 bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:outline-none focus:ring-blue-500 focus:border-blue-500"
                                required
                                placeholder="https://api.example.com"
                            />
                        </div>

                        <div>
                            <label htmlFor="api_key" className="block text-sm font-medium text-gray-700 dark:text-gray-300">
                                API Key *
                            </label>
                            <input
                                type="password"
                                id="api_key"
                                value={formData.api_key || ''}
                                onChange={(e) => setFormData({ ...formData, api_key: e.target.value })}
                                className="mt-1 block w-full border border-gray-300 dark:border-gray-600 rounded-md shadow-sm px-3 py-2 bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:outline-none focus:ring-blue-500 focus:border-blue-500"
                                required
                            />
                        </div>

                        <div>
                            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                                Custom Headers
                            </label>
                            <div className="space-y-2">
                                {Object.entries(formData.headers).map(([key, value]) => (
                                    <div key={key} className="flex items-center space-x-2">
                                        <span className="flex-1 px-3 py-2 bg-gray-100 dark:bg-gray-600 rounded-md text-sm font-mono text-gray-900 dark:text-white">
                                            {key}: {value}
                                        </span>
                                        <button
                                            type="button"
                                            onClick={() => removeHeader(key)}
                                            className="text-red-600 hover:text-red-800 dark:text-red-400 dark:hover:text-red-300 p-1"
                                            title="Remove header"
                                        >
                                            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                                            </svg>
                                        </button>
                                    </div>
                                ))}
                                <div className="flex space-x-2">
                                    <input
                                        type="text"
                                        placeholder="Header name"
                                        value={headerKey}
                                        onChange={(e) => setHeaderKey(e.target.value)}
                                        className="flex-1 border border-gray-300 dark:border-gray-600 rounded-md shadow-sm px-3 py-2 text-sm bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:outline-none focus:ring-blue-500 focus:border-blue-500"
                                    />
                                    <input
                                        type="text"
                                        placeholder="Header value"
                                        value={headerValue}
                                        onChange={(e) => setHeaderValue(e.target.value)}
                                        className="flex-1 border border-gray-300 dark:border-gray-600 rounded-md shadow-sm px-3 py-2 text-sm bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:outline-none focus:ring-blue-500 focus:border-blue-500"
                                    />
                                    <button
                                        type="button"
                                        onClick={addHeader}
                                        className="bg-green-600 hover:bg-green-700 text-white px-3 py-2 rounded-md text-sm font-medium"
                                    >
                                        Add
                                    </button>
                                </div>
                            </div>
                        </div>

                        <div className="flex items-center">
                            <input
                                type="checkbox"
                                id="enabled"
                                checked={formData.enabled}
                                onChange={(e) => setFormData({ ...formData, enabled: e.target.checked })}
                                className="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 dark:border-gray-600 rounded"
                            />
                            <label htmlFor="enabled" className="ml-2 block text-sm text-gray-900 dark:text-white">
                                Enabled
                            </label>
                        </div>

                        <div className="flex space-x-3 pt-4">
                            {(drawerMode as any) === 'view' && (
                                <>
                                    <button
                                        type="button"
                                        onClick={() => handleTestConnection(formData.name)}
                                        className="bg-green-600 hover:bg-green-700 text-white px-4 py-2 rounded-md font-medium"
                                    >
                                        Test Connection
                                    </button>
                                    <button
                                        type="button"
                                        onClick={(e) => {
                                            e.preventDefault();
                                            console.log('Edit button clicked in view mode');
                                            setDrawerMode('edit');
                                            console.log('Drawer mode changed to edit');
                                        }}
                                        className="bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-md font-medium"
                                    >
                                        Edit
                                    </button>
                                    <button
                                        type="button"
                                        onClick={closeDrawer}
                                        className="bg-gray-300 hover:bg-gray-400 dark:bg-gray-600 dark:hover:bg-gray-500 text-gray-800 dark:text-white px-4 py-2 rounded-md font-medium"
                                    >
                                        Cancel
                                    </button>
                                </>
                            )}
                            {(drawerMode as any) !== 'view' && (
                                <>
                                    <button
                                        type="submit"
                                        className="bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-md font-medium"
                                    >
                                        {drawerMode === 'edit' ? 'Update Service' : 'Create Service'}
                                    </button>
                                    <button
                                        type="button"
                                        onClick={closeDrawer}
                                        className="bg-gray-300 hover:bg-gray-400 dark:bg-gray-600 dark:hover:bg-gray-500 text-gray-800 dark:text-white px-4 py-2 rounded-md font-medium"
                                    >
                                        Cancel
                                    </button>
                                </>
                            )}
                        </div>
                    </form>
                ) : null}
            </Drawer>

            {/* Services List */}
            <div className="bg-white dark:bg-gray-800 shadow rounded-lg overflow-hidden">
                <div className="px-6 py-4 border-b border-slate-200 dark:border-slate-700">
                    <h2 className="text-xl font-semibold text-gray-900 dark:text-white">Configured Services</h2>
                </div>

                {services.length === 0 ? (
                    <div className="px-6 py-8 text-center text-slate-500 dark:text-slate-400">
                        No external services configured yet.
                    </div>
                ) : (
                    <div className="overflow-x-auto">
                        <table className="min-w-full divide-y divide-slate-200 dark:divide-slate-700">
                            <thead className="bg-slate-50 dark:bg-slate-900">
                                <tr>
                                    <th className="px-6 py-3 text-left text-xs font-medium text-slate-500 dark:text-slate-300 uppercase tracking-wider">
                                        Service Name
                                    </th>
                                    <th className="px-6 py-3 text-left text-xs font-medium text-slate-500 dark:text-slate-300 uppercase tracking-wider">
                                        URL
                                    </th>
                                    <th className="px-6 py-3 text-left text-xs font-medium text-slate-500 dark:text-slate-300 uppercase tracking-wider">
                                        Headers
                                    </th>
                                    <th className="px-6 py-3 text-left text-xs font-medium text-slate-500 dark:text-slate-300 uppercase tracking-wider">
                                        Status
                                    </th>
                                    <th className="px-6 py-3 text-left text-xs font-medium text-slate-500 dark:text-slate-300 uppercase tracking-wider">
                                        Actions
                                    </th>
                                </tr>
                            </thead>
                            <tbody className="bg-white dark:bg-gray-800 divide-y divide-slate-200 dark:divide-slate-700">
                                {services.map((service) => (
                                    <tr key={service.name}>
                                        <td className="px-6 py-4 whitespace-nowrap">
                                            <div>
                                                <div className="text-sm font-medium text-gray-900 dark:text-white">
                                                    {service.name}
                                                </div>
                                                {service.description && (
                                                    <div className="text-sm text-gray-500 dark:text-gray-400">
                                                        {service.description}
                                                    </div>
                                                )}
                                            </div>
                                        </td>
                                        <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900 dark:text-white">
                                            {service.url}
                                        </td>
                                        <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900 dark:text-white">
                                            {Object.keys(service.headers || {}).length > 0 ? (
                                                <span className="inline-flex items-center px-2 py-1 rounded-full text-xs font-medium bg-blue-100 text-blue-800 dark:bg-blue-900/20 dark:text-blue-400">
                                                    {Object.keys(service.headers || {}).length} custom
                                                </span>
                                            ) : (
                                                <span className="text-gray-400 dark:text-gray-500">None</span>
                                            )}
                                        </td>
                                        <td className="px-6 py-4 whitespace-nowrap">
                                            <span className={`inline-flex px-2 py-1 text-xs font-semibold rounded-full ${service.enabled
                                                ? 'bg-green-100 text-green-800 dark:bg-green-900/20 dark:text-green-400'
                                                : 'bg-red-100 text-red-800 dark:bg-red-900/20 dark:text-red-400'
                                                }`}>
                                                {service.enabled ? 'Enabled' : 'Disabled'}
                                            </span>
                                        </td>
                                        <td className="px-6 py-4 whitespace-nowrap text-sm font-medium space-x-2">
                                            <button
                                                onClick={() => startViewing(service)}
                                                className="text-gray-600 hover:text-gray-900 dark:text-gray-400 dark:hover:text-gray-300"
                                            >
                                                View
                                            </button>
                                            {canManageAdmin && (
                                                <button
                                                    onClick={() => startEditing(service)}
                                                    className="text-blue-600 hover:text-blue-900 dark:text-blue-400 dark:hover:text-blue-300"
                                                >
                                                    Edit
                                                </button>
                                            )}
                                            {canManageAdmin && (
                                                <button
                                                    onClick={() => handleDeleteService(service.name)}
                                                    className="text-red-600 hover:text-red-900 dark:text-red-400 dark:hover:text-red-300"
                                                >
                                                    Delete
                                                </button>
                                            )}
                                        </td>
                                    </tr>
                                ))}
                            </tbody>
                        </table>
                    </div>
                )}
            </div>
        </div>
    )
}

export default ExternalServicesPage
