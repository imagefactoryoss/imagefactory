import { useTenantStore } from '@/store/tenant'
import React from 'react'

interface ContextIndicatorProps {
    className?: string
    showRole?: boolean
    compact?: boolean
}

const ContextIndicator: React.FC<ContextIndicatorProps> = ({
    className = '',
    showRole = true,
    compact = false
}) => {
    const { userTenants, selectedTenantId, selectedRoleId } = useTenantStore()

    const selectedTenant = userTenants.find(t => t.id === selectedTenantId)
    const selectedRole = selectedTenant?.roles.find(r => r.id === selectedRoleId)

    if (!selectedTenant || !selectedRole) {
        return (
            <div className={`flex items-center space-x-2 text-gray-500 dark:text-gray-400 ${className}`}>
                <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.964-.833-2.732 0L3.732 16.5c-.77.833.192 2.5 1.732 2.5z" />
                </svg>
                <span className="text-sm">No context selected</span>
            </div>
        )
    }

    if (compact) {
        return (
            <div className={`flex items-center space-x-1 text-gray-600 dark:text-gray-300 ${className}`}>
                <span className="text-sm font-medium">{selectedTenant.name}</span>
                {showRole && (
                    <>
                        <span className="text-gray-400 dark:text-gray-500">•</span>
                        <span className="text-sm">{selectedRole.name}</span>
                        {selectedRole.is_admin && (
                            <span className="text-xs bg-blue-100 dark:bg-blue-900 text-blue-800 dark:text-blue-200 px-1.5 py-0.5 rounded">
                                Admin
                            </span>
                        )}
                    </>
                )}
            </div>
        )
    }

    return (
        <div className={`flex items-center space-x-3 px-3 py-2 bg-gray-100 dark:bg-gray-800 rounded-md ${className}`}>
            <div className="flex items-center space-x-2">
                <svg className="h-5 w-5 text-gray-500 dark:text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 21V5a2 2 0 00-2-2H7a2 2 0 00-2 2v16m14 0h2m-2 0h-5m-9 0H3m2 0h5M9 7h1m-1 4h1m4-4h1m-1 4h1m-5 10v-5a1 1 0 011-1h2a1 1 0 011 1v5m-4 0h4" />
                </svg>
                <div>
                    <div className="text-sm font-medium text-gray-900 dark:text-white">
                        {selectedTenant.name}
                    </div>
                    {showRole && (
                        <div className="text-xs text-gray-500 dark:text-gray-400 flex items-center space-x-1">
                            <span>{selectedRole.name}</span>
                            {selectedRole.is_admin && (
                                <span className="bg-blue-100 dark:bg-blue-900 text-blue-800 dark:text-blue-200 px-1.5 py-0.5 rounded text-xs">
                                    Admin
                                </span>
                            )}
                        </div>
                    )}
                </div>
            </div>
        </div>
    )
}

export default ContextIndicator
