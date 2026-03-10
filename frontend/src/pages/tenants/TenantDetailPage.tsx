import React from 'react'
import { useParams } from 'react-router-dom'

const TenantDetailPage: React.FC = () => {
    const { tenantId } = useParams<{ tenantId: string }>()

    return (
        <div className="px-4 py-6 sm:px-6 lg:px-8">
            <div className="sm:flex sm:items-center">
                <div className="sm:flex-auto">
                    <h1 className="text-2xl font-semibold text-slate-900 dark:text-white">Tenant Details</h1>
                    <p className="mt-2 text-sm text-slate-700 dark:text-slate-400">
                        Tenant ID: {tenantId}
                    </p>
                </div>
            </div>

            <div className="mt-8">
                <div className="card">
                    <div className="card-body">
                        <p className="text-muted-foreground">Tenant detail page coming soon...</p>
                    </div>
                </div>
            </div>
        </div>
    )
}

export default TenantDetailPage