import { ToolAvailabilityManager } from '@/components/admin'
import React from 'react'

const ToolManagementPage: React.FC = () => {
    return (
        <div className="space-y-6">
            <div>
                <h1 className="text-2xl font-bold text-gray-900 dark:text-white">
                    Tool Management
                </h1>
                <p className="mt-2 text-gray-600 dark:text-gray-400">
                    Configure which build tools and services are available to users
                </p>
            </div>

            <ToolAvailabilityManager />
        </div>
    )
}

export default ToolManagementPage