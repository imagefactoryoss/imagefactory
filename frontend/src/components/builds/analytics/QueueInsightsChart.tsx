import { QueueStats } from '@/types/buildConfig';
import React, { useMemo } from 'react';

interface QueueInsightsChartProps {
    stats: QueueStats;
    historicalData?: Array<{
        timestamp: Date;
        pendingCount: number;
        processingCount: number;
        averageWaitTime: number;
        throughput: number; // builds per hour
    }>;
}

export const QueueInsightsChart: React.FC<QueueInsightsChartProps> = ({
    stats,
    historicalData = [],
}) => {
    const queueHealth = useMemo(() => {
        const pending = stats.pending ?? stats.total_pending ?? 0
        const processing = stats.processing ?? stats.total_processing ?? 0
        const avgWaitTime = stats.average_wait_time ?? stats.average_wait_time_minutes ?? 0

        if (pending === 0 && processing === 0) {
            return { status: 'idle', color: 'text-gray-600 bg-gray-50', message: 'No builds in queue' }
        }
        if (avgWaitTime < 60) {
            return { status: 'healthy', color: 'text-green-600 bg-green-50', message: 'Queue flowing smoothly' }
        }
        if (avgWaitTime < 300) {
            return { status: 'warning', color: 'text-yellow-600 bg-yellow-50', message: 'Queue experiencing delays' }
        }
        return { status: 'critical', color: 'text-red-600 bg-red-50', message: 'Queue significantly backed up' }
    }, [stats])

    const formatDuration = (seconds: number): string => {
        if (seconds === 0) return '0s';
        if (seconds < 60) return `${Math.round(seconds)}s`;
        const minutes = Math.round(seconds / 60);
        if (minutes < 60) return `${minutes}m`;
        const hours = Math.round(minutes / 60);
        return `${hours}h`;
    };

    const totalQueued = (stats.pending ?? stats.total_pending ?? 0) + (stats.assigned ?? stats.total_assigned ?? 0)
    const queueUtilization =
        (stats.processing ?? stats.total_processing ?? 0) > 0
            ? (((stats.processing ?? stats.total_processing ?? 0) / ((stats.processing ?? stats.total_processing ?? 0) + (stats.pending ?? stats.total_pending ?? 0))) * 100)
            : 0

    // Calculate trend from historical data
    const avgWaitTrend = useMemo(() => {
        if (historicalData.length < 2) return null;
        const recent = historicalData.slice(-5);
        const oldAvg = recent[0].averageWaitTime;
        const newAvg = recent[recent.length - 1].averageWaitTime;
        const change = newAvg - oldAvg;
        return {
            direction: change > 0 ? 'up' : 'down',
            value: Math.abs(change),
            percentage: oldAvg > 0 ? Math.round((Math.abs(change) / oldAvg) * 100) : 0,
        };
    }, [historicalData]);

    return (
        <div className="space-y-6">
            {/* Header */}
            <div className="flex items-start justify-between">
                <div>
                    <h3 className="text-lg font-semibold text-gray-900">Queue Insights</h3>
                    <p className="text-sm text-gray-600 mt-1">Real-time queue health and performance metrics</p>
                </div>
                <div className={`px-3 py-1 rounded-full text-sm font-semibold ${queueHealth.color}`}>
                    {queueHealth.message}
                </div>
            </div>

            {/* Main Stats Grid */}
            <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                {/* Pending */}
                <div className="border border-gray-200 rounded-lg p-4 hover:shadow-sm transition-shadow">
                    <p className="text-xs font-semibold text-gray-600 uppercase tracking-wide">Pending</p>
                    <p className="text-2xl font-bold text-gray-900 mt-2">{stats.pending}</p>
                    <p className="text-xs text-gray-500 mt-1">Waiting to be assigned</p>
                </div>

                {/* Assigned */}
                <div className="border border-gray-200 rounded-lg p-4 hover:shadow-sm transition-shadow">
                    <p className="text-xs font-semibold text-gray-600 uppercase tracking-wide">Assigned</p>
                    <p className="text-2xl font-bold text-gray-900 mt-2">{stats.assigned}</p>
                    <p className="text-xs text-gray-500 mt-1">In worker pipeline</p>
                </div>

                {/* Processing */}
                <div className="border border-blue-200 rounded-lg p-4 bg-blue-50 hover:shadow-sm transition-shadow">
                    <p className="text-xs font-semibold text-blue-600 uppercase tracking-wide">Processing</p>
                    <p className="text-2xl font-bold text-blue-900 mt-2">{stats.processing}</p>
                    <p className="text-xs text-blue-700 mt-1">Currently building</p>
                </div>

                {/* Avg Wait */}
                <div className="border border-gray-200 rounded-lg p-4 hover:shadow-sm transition-shadow">
                    <p className="text-xs font-semibold text-gray-600 uppercase tracking-wide">Avg Wait</p>
                    <p className="text-2xl font-bold text-gray-900 mt-2">
                        {formatDuration(stats.average_wait_time ?? stats.average_wait_time_minutes ?? 0)}
                    </p>
                    {avgWaitTrend && (
                        <p
                            className={`text-xs mt-1 font-medium ${avgWaitTrend.direction === 'up' ? 'text-red-600' : 'text-green-600'
                                }`}
                        >
                            {avgWaitTrend.direction === 'up' ? '↑' : '↓'} {avgWaitTrend.percentage}%
                        </p>
                    )}
                </div>
            </div>

            {/* Queue Composition */}
            <div className="border border-gray-200 rounded-lg p-6">
                <h4 className="text-sm font-semibold text-gray-900 mb-4">Queue Composition</h4>
                <div className="space-y-4">
                    {/* Visualization */}
                    <div className="flex gap-2 h-8 rounded-full overflow-hidden bg-gray-100">
                        {totalQueued > 0 ? (
                            <>
                                {/* Pending Segment */}
                                <div
                                    className="bg-yellow-400 hover:bg-yellow-500 transition-colors"
                                    style={{ width: `${(((stats.pending ?? stats.total_pending ?? 0) / totalQueued) * 100) || 0}%` }}
                                    title={`${stats.pending ?? stats.total_pending ?? 0} pending`}
                                />
                                {/* Assigned Segment */}
                                <div
                                    className="bg-blue-400 hover:bg-blue-500 transition-colors"
                                    style={{ width: `${(((stats.assigned ?? stats.total_assigned ?? 0) / totalQueued) * 100) || 0}%` }}
                                    title={`${stats.assigned ?? stats.total_assigned ?? 0} assigned`}
                                />
                            </>
                        ) : (
                            <div className="w-full bg-gray-200" />
                        )}
                    </div>

                    {/* Legend */}
                    <div className="flex flex-wrap gap-4">
                        <div className="flex items-center gap-2">
                            <span className="w-3 h-3 bg-yellow-400 rounded-full" />
                            <span className="text-sm text-gray-700">
                                Pending: <span className="font-semibold">{stats.pending}</span>
                            </span>
                        </div>
                        <div className="flex items-center gap-2">
                            <span className="w-3 h-3 bg-blue-400 rounded-full" />
                            <span className="text-sm text-gray-700">
                                Assigned: <span className="font-semibold">{stats.assigned}</span>
                            </span>
                        </div>
                        <div className="flex items-center gap-2">
                            <span className="w-3 h-3 bg-blue-600 rounded-full" />
                            <span className="text-sm text-gray-700">
                                Processing: <span className="font-semibold">{stats.processing}</span>
                            </span>
                        </div>
                    </div>
                </div>
            </div>

            {/* Throughput & Utilization */}
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                {/* Queue Utilization */}
                <div className="border border-gray-200 rounded-lg p-4">
                    <h4 className="text-sm font-semibold text-gray-900 mb-3">Queue Utilization</h4>
                    <div className="flex items-end gap-4">
                        <div>
                            <p className="text-3xl font-bold text-gray-900">{Math.round(queueUtilization)}%</p>
                            <p className="text-xs text-gray-600 mt-1">of queue is processing</p>
                        </div>
                        <div className="flex-1">
                            <div className="h-2 bg-gray-200 rounded-full overflow-hidden">
                                <div
                                    className={`h-full transition-all ${queueUtilization > 80 ? 'bg-red-500' : 'bg-green-500'
                                        }`}
                                    style={{ width: `${queueUtilization}%` }}
                                />
                            </div>
                        </div>
                    </div>
                </div>

                {/* Throughput Info */}
                <div className="border border-gray-200 rounded-lg p-4">
                    <h4 className="text-sm font-semibold text-gray-900 mb-3">Throughput</h4>
                    {historicalData.length > 0 && (
                        <div>
                            <p className="text-3xl font-bold text-gray-900">
                                {historicalData[historicalData.length - 1].throughput.toFixed(1)}/h
                            </p>
                            <p className="text-xs text-gray-600 mt-1">builds per hour (1h average)</p>
                        </div>
                    )}
                    {historicalData.length === 0 && (
                        <p className="text-sm text-gray-500">Insufficient data</p>
                    )}
                </div>
            </div>

            {/* Historical Trend */}
            {historicalData.length > 0 && (
                <div className="border border-gray-200 rounded-lg p-4">
                    <h4 className="text-sm font-semibold text-gray-900 mb-4">Recent Trend (Last Hour)</h4>
                    <div className="space-y-3">
                        {historicalData.slice(-6).map((data, idx) => (
                            <div key={idx} className="flex items-center gap-3">
                                <span className="text-xs text-gray-500 w-12">
                                    {data.timestamp.toLocaleTimeString()}
                                </span>
                                <div className="flex-1 flex gap-1">
                                    <div className="flex-1 h-6 bg-yellow-100 rounded relative">
                                        <div
                                            className="h-full bg-yellow-400 rounded"
                                            style={{ width: `${Math.min((data.pendingCount / 10) * 100, 100)}%` }}
                                        />
                                        <span className="absolute left-2 top-1 text-xs font-semibold text-yellow-900">
                                            {data.pendingCount}p
                                        </span>
                                    </div>
                                    <div className="flex-1 h-6 bg-blue-100 rounded relative">
                                        <div
                                            className="h-full bg-blue-500 rounded"
                                            style={{
                                                width: `${Math.min((data.processingCount / 10) * 100, 100)}%`,
                                            }}
                                        />
                                        <span className="absolute left-2 top-1 text-xs font-semibold text-blue-900">
                                            {data.processingCount}a
                                        </span>
                                    </div>
                                </div>
                            </div>
                        ))}
                    </div>
                    <p className="text-xs text-gray-500 mt-3">p = pending, a = active</p>
                </div>
            )}
        </div>
    );
};

export default QueueInsightsChart;
