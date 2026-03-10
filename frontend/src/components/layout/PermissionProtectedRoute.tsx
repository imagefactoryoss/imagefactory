import { useAuthStore } from '@/store/auth'
import { hasPermission } from '@/utils/permissions'
import React from 'react'
import { Navigate } from 'react-router-dom'

interface PermissionProtectedRouteProps {
    children: React.ReactNode
    resource: string
    action: string
}

const PermissionProtectedRoute: React.FC<PermissionProtectedRouteProps> = ({
    children,
    resource,
    action
}) => {
    const { isAuthenticated } = useAuthStore()

    if (!isAuthenticated) {
        return <Navigate to="/login" replace />
    }

    if (!hasPermission(resource, action)) {
        return (
            <div className="min-h-screen flex items-center justify-center">
                <div className="text-center">
                    <h1 className="text-2xl font-bold text-red-600 dark:text-red-400 mb-4">Access Denied</h1>
                    <p className="text-slate-600 dark:text-slate-300">
                        You don't have permission to access this page.
                    </p>
                </div>
            </div>
        )
    }

    return <>{children}</>
}

export default PermissionProtectedRoute