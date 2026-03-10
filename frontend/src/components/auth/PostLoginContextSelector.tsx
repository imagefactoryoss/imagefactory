import { useDashboardPath } from '@/hooks/useAccess'
import { useAuthStore } from '@/store/auth'
import { useTenantStore } from '@/store/tenant'
import React, { useEffect, useState } from 'react'
import toast from 'react-hot-toast'
import { useNavigate } from 'react-router-dom'

interface PostLoginContextSelectorProps {
    isOpen: boolean
    onClose: () => void
    onContextSwitch?: () => void // Optional callback for custom behavior after context switch
}

const PostLoginContextSelector: React.FC<PostLoginContextSelectorProps> = ({
    isOpen,
    onClose,
    onContextSwitch
}) => {
    const navigate = useNavigate()
    const { userTenants, setContext, validateContext } = useTenantStore()
    const { user } = useAuthStore()
    const dashboardPath = useDashboardPath()

    const [selectedTenantId, setSelectedTenantId] = useState<string>('')
    const [selectedRoleId, setSelectedRoleId] = useState<string>('')
    const [isSubmitting, setIsSubmitting] = useState(false)

    useEffect(() => {
        if (userTenants.length > 0 && !selectedTenantId) {
            setSelectedTenantId(userTenants[0].id)
        }
    }, [userTenants, selectedTenantId])

    useEffect(() => {
        if (selectedTenantId) {
            const tenant = userTenants.find(t => t.id === selectedTenantId)
            if (tenant && tenant.roles.length > 0 && !selectedRoleId) {
                setSelectedRoleId(tenant.roles[0].id)
            }
        }
    }, [selectedTenantId, selectedRoleId, userTenants])

    const handleTenantChange = (tenantId: string) => {
        setSelectedTenantId(tenantId)
        setSelectedRoleId('') // Reset role when tenant changes
    }

    const handleSubmit = async () => {
        if (!selectedTenantId || !selectedRoleId) {
            toast.error('Please select both a tenant and role')
            return
        }

        setIsSubmitting(true)
        try {
            setContext(selectedTenantId, selectedRoleId)

            if (validateContext()) {
                toast.success(`Context set to ${getSelectedTenantName()} as ${getSelectedRoleName()}`)
                onClose()
                if (onContextSwitch) {
                    onContextSwitch()
                } else {
                    navigate(dashboardPath)
                }
            } else {
                toast.error('Invalid context selection')
            }
        } catch (error) {
            toast.error('Failed to set context')
        } finally {
            setIsSubmitting(false)
        }
    }

    const getSelectedTenantName = () => {
        const tenant = userTenants.find(t => t.id === selectedTenantId)
        return tenant?.name || 'Unknown Tenant'
    }

    const getSelectedRoleName = () => {
        const tenant = userTenants.find(t => t.id === selectedTenantId)
        const role = tenant?.roles.find(r => r.id === selectedRoleId)
        return role?.name || 'Unknown Role'
    }

    const getAvailableRoles = () => {
        const tenant = userTenants.find(t => t.id === selectedTenantId)
        return tenant?.roles || []
    }

    if (!isOpen) return null

    return (
        <div className="fixed inset-0 z-50 overflow-y-auto">
            <div className="flex items-center justify-center min-h-screen px-4 pt-4 pb-20 text-center sm:block sm:p-0">
                {/* Background overlay */}
                <div className="fixed inset-0 bg-slate-900 bg-opacity-75 transition-opacity" onClick={onClose}></div>

                {/* Modal panel */}
                <div className="inline-block align-bottom bg-white dark:bg-slate-800 rounded-lg text-left overflow-hidden shadow-xl transform transition-all sm:my-8 sm:align-middle sm:max-w-lg sm:w-full">
                    <div className="bg-white dark:bg-slate-800 px-4 pt-5 pb-4 sm:p-6 sm:pb-4">
                        <div className="sm:flex sm:items-start">
                            <div className="mx-auto flex-shrink-0 flex items-center justify-center h-12 w-12 rounded-full bg-blue-100 dark:bg-blue-900 sm:mx-0 sm:h-10 sm:w-10">
                                <svg className="h-6 w-6 text-blue-600 dark:text-blue-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z" />
                                </svg>
                            </div>
                            <div className="mt-3 text-center sm:mt-0 sm:ml-4 sm:text-left flex-1">
                                <h3 className="text-lg leading-6 font-medium text-slate-900 dark:text-white">
                                    Welcome back, {user?.name || user?.email}!
                                </h3>
                                <div className="mt-2">
                                    <p className="text-sm text-slate-600 dark:text-slate-400">
                                        You have access to multiple tenants. Please select which tenant and role you'd like to work with.
                                    </p>
                                </div>

                                {/* Tenant Selection */}
                                <div className="mt-6">
                                    <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                                        Select Tenant
                                    </label>
                                    <select
                                        value={selectedTenantId}
                                        onChange={(e) => handleTenantChange(e.target.value)}
                                        className="block w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white"
                                    >
                                        <option value="">Choose a tenant...</option>
                                        {userTenants.map((tenant) => (
                                            <option key={tenant.id} value={tenant.id}>
                                                {tenant.name}
                                            </option>
                                        ))}
                                    </select>
                                </div>

                                {/* Role Selection */}
                                <div className="mt-4">
                                    <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                                        Select Role
                                    </label>
                                    <select
                                        value={selectedRoleId}
                                        onChange={(e) => setSelectedRoleId(e.target.value)}
                                        disabled={!selectedTenantId || getAvailableRoles().length === 0}
                                        className="block w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white disabled:opacity-50 disabled:cursor-not-allowed"
                                    >
                                        <option value="">
                                            {getAvailableRoles().length === 0 ? 'No roles available' : 'Choose a role...'}
                                        </option>
                                        {getAvailableRoles().map((role) => (
                                            <option key={role.id} value={role.id}>
                                                {role.name} {role.is_admin && '(Admin)'}
                                            </option>
                                        ))}
                                    </select>
                                </div>

                                {/* Context Preview */}
                                {selectedTenantId && selectedRoleId && (
                                    <div className="mt-4 p-3 bg-blue-50 dark:bg-blue-900/20 rounded-md">
                                        <p className="text-sm text-blue-800 dark:text-blue-200">
                                            You will be working as <strong>{getSelectedRoleName()}</strong> in <strong>{getSelectedTenantName()}</strong>
                                        </p>
                                    </div>
                                )}
                            </div>
                        </div>
                    </div>

                    {/* Footer */}
                    <div className="bg-slate-50 dark:bg-slate-700 px-4 py-3 sm:px-6 sm:flex sm:flex-row-reverse">
                        <button
                            type="button"
                            onClick={handleSubmit}
                            disabled={isSubmitting || !selectedTenantId || !selectedRoleId}
                            className="w-full inline-flex justify-center rounded-md border border-transparent shadow-sm px-4 py-2 bg-blue-600 text-base font-medium text-white hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 sm:ml-3 sm:w-auto sm:text-sm disabled:opacity-50 disabled:cursor-not-allowed"
                        >
                            {isSubmitting ? 'Setting Context...' : 'Continue to Dashboard'}
                        </button>
                        <button
                            type="button"
                            onClick={onClose}
                            className="mt-3 w-full inline-flex justify-center rounded-md border border-slate-300 dark:border-slate-600 shadow-sm px-4 py-2 bg-white dark:bg-slate-800 text-base font-medium text-slate-700 dark:text-slate-300 hover:bg-slate-50 dark:hover:bg-slate-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 sm:mt-0 sm:ml-3 sm:w-auto sm:text-sm"
                        >
                            Cancel
                        </button>
                    </div>
                </div>
            </div>
        </div>
    )
}

export default PostLoginContextSelector