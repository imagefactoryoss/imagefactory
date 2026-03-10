import { useAuthStore } from '@/store/auth'
import { useTenantStore } from '@/store/tenant'
import React, { useState } from 'react'
import toast from 'react-hot-toast'

const ContextSwitcher: React.FC = () => {
    const { userTenants, selectedTenantId, selectedRoleId, setContext, validateContext } = useTenantStore()
    const { user } = useAuthStore()
    const [isOpen, setIsOpen] = useState(false)

    const selectedTenant = userTenants.find(t => t.id === selectedTenantId)
    const selectedRole = selectedTenant?.roles.find(r => r.id === selectedRoleId)

    const handleContextChange = (tenantId: string, roleId: string) => {
        try {
            setContext(tenantId, roleId)
            if (validateContext()) {
                const newTenant = userTenants.find(t => t.id === tenantId)
                const newRole = newTenant?.roles.find(r => r.id === roleId)
                toast.success(`Switched context: ${newTenant?.name} as ${newRole?.name}`)
                setIsOpen(false)
            } else {
                toast.error('Invalid context selection')
            }
        } catch (error) {
            toast.error('Failed to switch context')
        }
    }

    const getAvailableContexts = () => {
        const contexts: Array<{ tenantId: string; roleId: string; tenantName: string; roleName: string; isAdmin: boolean }> = []

        userTenants.forEach(tenant => {
            tenant.roles.forEach(role => {
                contexts.push({
                    tenantId: tenant.id,
                    roleId: role.id,
                    tenantName: tenant.name,
                    roleName: role.name,
                    isAdmin: role.is_admin
                })
            })
        })

        return contexts
    }

    // Only show switcher if user has multiple tenants OR multiple roles in a tenant
    const availableContexts = getAvailableContexts()
    const hasMultipleContexts = availableContexts.length > 1

    if (!selectedTenant || !selectedRole) {
        return null
    }

    return (
        <div className="relative">
            <button
                onClick={() => hasMultipleContexts && setIsOpen(!isOpen)}
                className={`flex items-center space-x-2 px-3 py-2 text-sm font-medium border rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 ${hasMultipleContexts
                    ? 'text-slate-700 dark:text-slate-300 bg-white dark:bg-slate-800 border-slate-300 dark:border-slate-600 hover:bg-slate-50 dark:hover:bg-slate-700'
                    : 'text-slate-500 dark:text-slate-400 bg-slate-50 dark:bg-slate-800/60 border-slate-200 dark:border-slate-700 cursor-default'
                    }`}
                title={hasMultipleContexts ? 'Switch context' : 'Single context available'}
            >
                <div className="flex flex-col items-start">
                    <span className="font-semibold">
                        {selectedTenant.name}
                    </span>
                    <span className="text-xs text-slate-500 dark:text-slate-400">
                        Tenant Context • {selectedRole.name} {selectedRole.is_admin && '(Admin)'}
                    </span>
                </div>
                {hasMultipleContexts && (
                    <svg
                        className={`h-4 w-4 transition-transform ${isOpen ? 'rotate-180' : ''}`}
                        fill="none"
                        viewBox="0 0 24 24"
                        stroke="currentColor"
                    >
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
                    </svg>
                )}
            </button>

            {isOpen && hasMultipleContexts && (
                <div className="absolute right-0 mt-2 w-80 bg-white dark:bg-slate-800 border border-slate-300 dark:border-slate-600 rounded-md shadow-lg z-50">
                    <div className="p-4">
                        <h3 className="text-sm font-medium text-slate-900 dark:text-white mb-3">
                            Switch Context
                        </h3>
                        <p className="text-xs text-slate-500 dark:text-slate-400 mb-3">
                            Tenant context scopes actions to a specific tenant.
                        </p>
                        <div className="space-y-2 max-h-60 overflow-y-auto">
                            {getAvailableContexts().map((context) => (
                                <button
                                    key={`${context.tenantId}-${context.roleId}`}
                                    onClick={() => handleContextChange(context.tenantId, context.roleId)}
                                    className={`w-full text-left px-3 py-2 rounded-md transition-colors ${context.tenantId === selectedTenantId && context.roleId === selectedRoleId
                                        ? 'bg-blue-100 dark:bg-blue-900/30 text-blue-900 dark:text-blue-100'
                                        : 'hover:bg-slate-100 dark:hover:bg-slate-700 text-slate-700 dark:text-slate-300'
                                        }`}
                                >
                                    <div className="flex items-center justify-between">
                                        <div>
                                            <div className="font-medium">
                                                {context.tenantName}
                                            </div>
                                            <div className="text-sm text-slate-500 dark:text-slate-400">
                                                Tenant Context • {context.roleName} {context.isAdmin && '(Admin)'}
                                            </div>
                                        </div>
                                        {context.tenantId === selectedTenantId && context.roleId === selectedRoleId && (
                                            <svg className="h-4 w-4 text-blue-600 dark:text-blue-400" fill="currentColor" viewBox="0 0 20 20">
                                                <path fillRule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clipRule="evenodd" />
                                            </svg>
                                        )}
                                    </div>
                                </button>
                            ))}
                        </div>
                        <div className="mt-3 pt-3 border-t border-slate-200 dark:border-slate-600">
                            <p className="text-xs text-slate-500 dark:text-slate-400">
                                Current user: {user?.name || user?.email}
                            </p>
                        </div>
                    </div>
                </div>
            )}

            {/* Click outside to close */}
            {isOpen && hasMultipleContexts && (
                <div
                    className="fixed inset-0 z-40"
                    onClick={() => setIsOpen(false)}
                ></div>
            )}
        </div>
    )
}

export default ContextSwitcher
