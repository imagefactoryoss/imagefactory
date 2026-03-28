import React, { Suspense, lazy, useMemo, useState } from 'react'

const BuildAnalyticsWidget = lazy(() => import('@/components/admin/widgets/BuildAnalyticsWidget'))
const BuildPerformanceWidget = lazy(() => import('@/components/admin/widgets/BuildPerformanceWidget'))
const FailureAnalysisWidget = lazy(() => import('@/components/admin/widgets/FailureAnalysisWidget'))

type AnalyticsView = 'overview' | 'performance' | 'failures'

const widgetShellClassName = 'bg-white dark:bg-gray-900 rounded-lg border border-gray-200 dark:border-gray-700 p-6'

const WidgetLoadingFallback: React.FC = () => (
    <div className="flex min-h-[320px] items-center justify-center">
        <div className="text-center">
            <div className="mx-auto mb-3 h-10 w-10 animate-spin rounded-full border-b-2 border-blue-600"></div>
            <p className="text-sm text-gray-600 dark:text-gray-400">Loading analytics widget...</p>
        </div>
    </div>
)

const AdminBuildAnalyticsPage: React.FC = () => {
    const [activeView, setActiveView] = useState<AnalyticsView>('overview')

    const viewMeta = useMemo(
        () => ({
            overview: {
                label: 'Overview',
                subtitle: 'Current build volume, success rate, duration, and queue depth.',
                render: () => <BuildAnalyticsWidget />,
            },
            performance: {
                label: 'Performance Trends',
                subtitle: 'Duration, success, and queue trends over the recent period.',
                render: () => <BuildPerformanceWidget />,
            },
            failures: {
                label: 'Failure Analysis',
                subtitle: 'Slowest builds, failure reasons, and project failure rates.',
                render: () => <FailureAnalysisWidget />,
            },
        }),
        [],
    )

    const activeMeta = viewMeta[activeView]

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

            <div className="rounded-lg border border-gray-200 bg-white p-2 dark:border-gray-700 dark:bg-gray-900">
                <div className="grid grid-cols-1 gap-2 sm:grid-cols-3">
                    {(Object.keys(viewMeta) as AnalyticsView[]).map((key) => {
                        const isActive = key === activeView
                        return (
                            <button
                                key={key}
                                type="button"
                                onClick={() => setActiveView(key)}
                                className={`rounded-md px-3 py-2 text-sm font-medium transition ${
                                    isActive
                                        ? 'bg-blue-600 text-white shadow-sm'
                                        : 'bg-gray-100 text-gray-700 hover:bg-gray-200 dark:bg-gray-800 dark:text-gray-300 dark:hover:bg-gray-700'
                                }`}
                            >
                                {viewMeta[key].label}
                            </button>
                        )
                    })}
                </div>
            </div>

            <div className={widgetShellClassName}>
                <div className="mb-4">
                    <h2 className="text-lg font-semibold text-gray-900 dark:text-white">{activeMeta.label}</h2>
                    <p className="mt-1 text-sm text-gray-600 dark:text-gray-400">{activeMeta.subtitle}</p>
                </div>
                <Suspense fallback={<WidgetLoadingFallback />}>
                    {activeMeta.render()}
                </Suspense>
            </div>
        </div>
    )
}

export default AdminBuildAnalyticsPage
