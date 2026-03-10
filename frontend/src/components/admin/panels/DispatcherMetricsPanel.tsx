import React, { useCallback, useEffect, useState } from 'react'
import { Activity, AlertTriangle, RefreshCw } from 'lucide-react'
import { adminService } from '@/services/adminService'
import { DispatcherMetrics, DispatcherStatus } from '@/types'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'

const DispatcherMetricsPanel: React.FC = () => {
    const [metrics, setMetrics] = useState<DispatcherMetrics | null>(null)
    const [status, setStatus] = useState<DispatcherStatus | null>(null)
    const [isLoading, setIsLoading] = useState(true)
    const [isRefreshing, setIsRefreshing] = useState(false)
    const [isToggling, setIsToggling] = useState(false)
    const [error, setError] = useState<string | null>(null)
    const [lastUpdatedAt, setLastUpdatedAt] = useState<Date | null>(null)

    const loadMetrics = useCallback(async (silent = false) => {
        try {
            if (!silent) {
                setIsLoading(true)
            } else {
                setIsRefreshing(true)
            }
            setError(null)
            const data = await adminService.getDispatcherMetrics()
            setMetrics(data)
            setLastUpdatedAt(new Date())
        } catch (err: any) {
            setError(err?.message || 'Failed to load dispatcher metrics')
        } finally {
            if (!silent) {
                setIsLoading(false)
            } else {
                setIsRefreshing(false)
            }
        }
    }, [])

    const loadStatus = useCallback(async () => {
        try {
            const data = await adminService.getDispatcherStatus()
            setStatus(data)
        } catch (err: any) {
            setError(err?.message || 'Failed to load dispatcher status')
        }
    }, [])

    const refreshAll = useCallback(async (silent = false) => {
        await Promise.all([loadMetrics(silent), loadStatus()])
    }, [loadMetrics, loadStatus])

    useEffect(() => {
        void refreshAll(false)

        const interval = window.setInterval(() => {
            void refreshAll(true)
        }, 15000)

        return () => window.clearInterval(interval)
    }, [refreshAll])

    const toggleDispatcher = async () => {
        if (!status) {
            return
        }
        if (status.mode === 'external') {
            return
        }
        try {
            setIsToggling(true)
            const data = status.running
                ? await adminService.stopDispatcher()
                : await adminService.startDispatcher()
            setStatus(data)
        } catch (err: any) {
            setError(err?.message || 'Failed to update dispatcher status')
        } finally {
            setIsToggling(false)
        }
    }

    return (
        <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <div className="flex items-center gap-3">
                    <CardTitle className="text-sm font-medium">Dispatcher Metrics</CardTitle>
                    {status && (
                        <>
                            <span
                                className={`rounded-full px-2 py-0.5 text-xs font-medium ${
                                    status.running
                                        ? 'bg-emerald-100 text-emerald-700'
                                        : 'bg-slate-200 text-slate-700'
                                }`}
                            >
                                {status.running ? 'Running' : 'Stopped'}
                            </span>
                            <span className="rounded-full px-2 py-0.5 text-xs font-medium bg-sky-100 text-sky-700">
                                {status.mode === 'external' ? 'External' : 'Embedded'}
                            </span>
                        </>
                    )}
                </div>
                <div className="flex items-center gap-2">
                    <Button
                        variant="outline"
                        size="sm"
                        onClick={() => void refreshAll(true)}
                        disabled={isRefreshing || isLoading}
                        title="Refresh dispatcher metrics"
                    >
                        <RefreshCw className={`mr-2 h-4 w-4 ${isRefreshing ? 'animate-spin' : ''}`} />
                        Refresh
                    </Button>
                    <Button
                        variant={status?.running ? 'outline' : 'default'}
                        size="sm"
                        onClick={toggleDispatcher}
                        disabled={isToggling || !status || status.mode === 'external'}
                    >
                        {status?.mode === 'external' ? 'Managed Externally' : status?.running ? 'Stop' : 'Start'}
                    </Button>
                    <Activity className="h-4 w-4 text-muted-foreground" />
                </div>
            </CardHeader>
            <CardContent>
                {isLoading && (
                    <div className="text-sm text-muted-foreground">Loading metrics...</div>
                )}

                {error && (
                    <div className="flex items-center gap-2 text-sm text-red-600">
                        <AlertTriangle className="h-4 w-4" />
                        <span>{error}</span>
                    </div>
                )}

                {status && (
                    <div className="mb-4 text-xs text-muted-foreground">
                        <div>
                            Availability: {status.available === false ? 'Unavailable' : 'Available'}
                            {status.stale ? ' (stale heartbeat)' : ''}
                        </div>
                        {status.last_heartbeat && (
                            <div>Last heartbeat: {new Date(status.last_heartbeat).toLocaleString()}</div>
                        )}
                        {status.message && <div>{status.message}</div>}
                        {lastUpdatedAt && (
                            <div>Updated: {lastUpdatedAt.toLocaleTimeString()}</div>
                        )}
                    </div>
                )}

                {!isLoading && !error && metrics && (
                    <div className="space-y-4">
                        <div className="grid grid-cols-2 md:grid-cols-3 gap-4">
                            <div>
                                <div className="text-xs text-muted-foreground">Claims</div>
                                <div className="text-lg font-semibold">{metrics.claims}</div>
                            </div>
                            <div>
                                <div className="text-xs text-muted-foreground">Dispatches</div>
                                <div className="text-lg font-semibold">{metrics.dispatches}</div>
                            </div>
                            <div>
                                <div className="text-xs text-muted-foreground">Requeues</div>
                                <div className="text-lg font-semibold">{metrics.requeues}</div>
                            </div>
                            <div>
                                <div className="text-xs text-muted-foreground">Claim Errors</div>
                                <div className="text-lg font-semibold">{metrics.claim_errors}</div>
                            </div>
                            <div>
                                <div className="text-xs text-muted-foreground">Dispatch Errors</div>
                                <div className="text-lg font-semibold">{metrics.dispatch_errors}</div>
                            </div>
                            <div>
                                <div className="text-xs text-muted-foreground">Skipped (Limits)</div>
                                <div className="text-lg font-semibold">{metrics.skipped_for_limit}</div>
                            </div>
                        </div>

                        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                            <div className="rounded-md border border-slate-200 bg-white p-3 dark:border-slate-700 dark:bg-slate-900/40">
                                <div className="text-xs text-muted-foreground mb-1">Claim Latency (ms)</div>
                                <div className="text-sm">
                                    avg {metrics.claim_avg_ms.toFixed(1)} · min {metrics.claim_min_ms} · max {metrics.claim_max_ms}
                                </div>
                            </div>
                            <div className="rounded-md border border-slate-200 bg-white p-3 dark:border-slate-700 dark:bg-slate-900/40">
                                <div className="text-xs text-muted-foreground mb-1">Dispatch Latency (ms)</div>
                                <div className="text-sm">
                                    avg {metrics.dispatch_avg_ms.toFixed(1)} · min {metrics.dispatch_min_ms} · max {metrics.dispatch_max_ms}
                                </div>
                            </div>
                        </div>
                    </div>
                )}
            </CardContent>
        </Card>
    )
}

export default DispatcherMetricsPanel
