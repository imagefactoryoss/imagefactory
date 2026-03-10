import { InfrastructurePanel } from '@/components/admin/panels'
import React from 'react'

const AdminBuildNodesPage: React.FC = () => {
    return (
        <div className="space-y-6">
            <div>
                <h1 className="text-2xl font-bold text-gray-900 dark:text-white">
                    Build Nodes Management
                </h1>
                <p className="mt-2 text-gray-600 dark:text-gray-400">
                    Configure and monitor build infrastructure nodes, resource allocation, and scaling policies
                </p>
            </div>

            <InfrastructurePanel />
        </div>
    )
}

export default AdminBuildNodesPage