import { useAuthStore } from '@/store/auth'
import React from 'react'
import { Navigate } from 'react-router-dom'

interface ProtectedRouteProps {
    children: React.ReactNode
    requiredRoles?: string[]
}

const ProtectedRoute: React.FC<ProtectedRouteProps> = ({
    children,
    requiredRoles = []
}) => {
    const { isAuthenticated, user } = useAuthStore()

    if (!isAuthenticated) {
        return <Navigate to="/login" replace />
    }

    if (requiredRoles.length > 0 && user) {
        const hasRequiredRole = requiredRoles.some(role =>
            // Check if user has roles array (newer format) or just a role string (legacy)
            (user as any).roles?.some((userRole: any) => userRole.name === role) ||
            (user as any).role === role
        )

        if (!hasRequiredRole) {
            return (
                <div className="min-h-screen flex items-center justify-center">
                    <div className="text-center">
                        <h1 className="text-2xl font-bold text-error mb-4">Access Denied</h1>
                        <p className="text-muted-foreground">
                            You don't have permission to access this page.
                        </p>
                    </div>
                </div>
            )
        }
    }

    return <>{children}</>
}

export default ProtectedRoute