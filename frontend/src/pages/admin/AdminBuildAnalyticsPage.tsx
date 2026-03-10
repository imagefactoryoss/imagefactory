import { BuildAnalyticsWidget, BuildPerformanceWidget, FailureAnalysisWidget } from '@/components/admin/widgets'
import React from 'react'

const AdminBuildAnalyticsPage: React.FC = () => {
    return (
        <div className="space-y-6">
            <div>
                <h1 className="text-2xl font-bold text-gray-900 dark:text-white">
                    Build Analytics & Performance
                </h1>
                <p className="mt-2 text-gray-600 dark:text-gray-400">
                    Monitor build performance metrics, trends, and failure analysis
                </p>
            </div>

            {/* Build Analytics Overview */}
            <div className="bg-white dark:bg-gray-900 rounded-lg border border-gray-200 dark:border-gray-700 p-6">
                <BuildAnalyticsWidget />
            </div>

            {/* Performance Trends */}
            <div className="bg-white dark:bg-gray-900 rounded-lg border border-gray-200 dark:border-gray-700 p-6">
                <BuildPerformanceWidget />
            </div>

            {/* Failure Analysis */}
            <div className="bg-white dark:bg-gray-900 rounded-lg border border-gray-200 dark:border-gray-700 p-6">
                <FailureAnalysisWidget />
            </div>
        </div>
    )
}

export default AdminBuildAnalyticsPage