import { BUILD_METHODS } from '@/lib/buildMethods';
import { BuildMethod } from '@/types/buildConfig';
import React from 'react';

interface MethodPerformanceMetrics {
    method: BuildMethod;
    totalBuilds: number;
    successCount: number;
    failureCount: number;
    averageDuration: number; // in seconds
    averageSize: number; // in MB
}

interface MethodPerformanceChartProps {
    metrics: MethodPerformanceMetrics[];
    selectedMethods?: BuildMethod[];
}

export const MethodPerformanceChart: React.FC<MethodPerformanceChartProps> = ({
    metrics,
    selectedMethods,
}) => {
    const filteredMetrics = selectedMethods
        ? metrics.filter((m) => selectedMethods.includes(m.method))
        : metrics;

    const getMethodName = (method: BuildMethod): string => {
        return BUILD_METHODS[method]?.name || method
    }

    const getSuccessRate = (metric: MethodPerformanceMetrics): number => {
        if (metric.totalBuilds === 0) return 0;
        return Math.round((metric.successCount / metric.totalBuilds) * 100);
    };

    const formatDuration = (seconds: number): string => {
        if (seconds < 60) return `${Math.round(seconds)}s`;
        const minutes = Math.round(seconds / 60);
        return `${minutes}m`;
    };

    const getStatusColor = (rate: number): string => {
        if (rate >= 95) return 'text-green-600 bg-green-50';
        if (rate >= 80) return 'text-yellow-600 bg-yellow-50';
        return 'text-red-600 bg-red-50';
    };

    return (
        <div className="space-y-4">
            <div className="flex items-center justify-between">
                <h3 className="text-lg font-semibold text-gray-900">Build Method Performance</h3>
                <span className="text-sm text-gray-500">{filteredMetrics.length} methods</span>
            </div>

            {filteredMetrics.length === 0 ? (
                <div className="bg-gray-50 rounded-lg p-8 text-center">
                    <p className="text-sm text-gray-600">No performance data available</p>
                </div>
            ) : (
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                    {filteredMetrics.map((metric) => {
                        const successRate = getSuccessRate(metric);
                        const statusColor = getStatusColor(successRate);

                        return (
                            <div
                                key={metric.method}
                                className="border border-gray-200 rounded-lg p-4 hover:shadow-md transition-shadow"
                            >
                                {/* Header */}
                                <div className="flex items-start justify-between mb-4">
                                    <div>
                                        <h4 className="font-semibold text-gray-900">
                                            {getMethodName(metric.method)}
                                        </h4>
                                        <p className="text-xs text-gray-500 mt-1">
                                            {metric.totalBuilds} builds total
                                        </p>
                                    </div>
                                    <span className={`px-2 py-1 rounded-full text-sm font-semibold ${statusColor}`}>
                                        {successRate}%
                                    </span>
                                </div>

                                {/* Success/Failure Bars */}
                                <div className="mb-4">
                                    <div className="flex gap-1 mb-2 h-2 rounded-full overflow-hidden bg-gray-200">
                                        {metric.totalBuilds > 0 && (
                                            <>
                                                <div
                                                    className="bg-green-500"
                                                    style={{
                                                        width: `${(metric.successCount / metric.totalBuilds) * 100}%`,
                                                    }}
                                                />
                                                <div
                                                    className="bg-red-500"
                                                    style={{
                                                        width: `${(metric.failureCount / metric.totalBuilds) * 100}%`,
                                                    }}
                                                />
                                            </>
                                        )}
                                    </div>
                                    <div className="flex gap-4 text-xs">
                                        <div className="flex items-center gap-1">
                                            <span className="w-2 h-2 bg-green-500 rounded-full" />
                                            <span className="text-gray-600">
                                                {metric.successCount} succeeded
                                            </span>
                                        </div>
                                        <div className="flex items-center gap-1">
                                            <span className="w-2 h-2 bg-red-500 rounded-full" />
                                            <span className="text-gray-600">
                                                {metric.failureCount} failed
                                            </span>
                                        </div>
                                    </div>
                                </div>

                                {/* Metrics Grid */}
                                <div className="grid grid-cols-2 gap-2">
                                    <div className="bg-gray-50 rounded p-2">
                                        <p className="text-xs text-gray-600">Avg Duration</p>
                                        <p className="text-sm font-semibold text-gray-900">
                                            {formatDuration(metric.averageDuration)}
                                        </p>
                                    </div>
                                    <div className="bg-gray-50 rounded p-2">
                                        <p className="text-xs text-gray-600">Avg Size</p>
                                        <p className="text-sm font-semibold text-gray-900">
                                            {metric.averageSize.toFixed(1)} MB
                                        </p>
                                    </div>
                                </div>
                            </div>
                        );
                    })}
                </div>
            )}
        </div>
    );
};

export default MethodPerformanceChart;
