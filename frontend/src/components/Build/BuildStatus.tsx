import React, { useEffect, useState } from 'react';
import { BuildClient, BuildStatusResponse } from '../../api/buildClient';

interface BuildStatusProps {
    client: BuildClient;
    buildId: string;
}

export const BuildStatus: React.FC<BuildStatusProps> = ({ client, buildId }) => {
    const [status, setStatus] = useState<BuildStatusResponse | null>(null);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);

    // Fetch status
    const fetchStatus = async () => {
        try {
            setError(null);
            const data = await client.getBuildStatus(buildId);
            setStatus(data);
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Failed to fetch status');
        } finally {
            setLoading(false);
        }
    };

    // Initial fetch and polling
    useEffect(() => {
        fetchStatus();

        // Poll for updates every 5 seconds
        const interval = setInterval(fetchStatus, 5000);

        return () => {
            if (interval) clearInterval(interval);
        };
    }, [client, buildId]);

    if (loading && !status) {
        return (
            <div className="flex items-center justify-center py-12">
                <div className="text-center">
                    <div className="w-12 h-12 border-4 border-blue-200 border-t-blue-600 rounded-full animate-spin mx-auto mb-4" />
                    <p className="text-gray-600 dark:text-slate-300">Loading status...</p>
                </div>
            </div>
        );
    }

    if (error) {
        return (
            <div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-700 rounded-lg p-4">
                <p className="text-red-800 dark:text-red-200">{error}</p>
                <button
                    onClick={fetchStatus}
                    className="mt-2 text-red-600 dark:text-red-300 hover:text-red-800 dark:hover:text-red-200 text-sm font-medium"
                >
                    Retry
                </button>
            </div>
        );
    }

    if (!status) {
        return (
            <div className="text-center py-12 text-gray-600 dark:text-slate-300">
                No status information available
            </div>
        );
    }

    const statusConfig: Record<string, { color: string; bgColor: string; icon: string }> = {
        queued: { color: 'text-gray-600 dark:text-slate-300', bgColor: 'bg-gray-100 dark:bg-slate-700', icon: '⏳' },
        running: { color: 'text-blue-600 dark:text-blue-300', bgColor: 'bg-blue-100 dark:bg-blue-900/30', icon: '⚙️' },
        completed: { color: 'text-green-600 dark:text-green-300', bgColor: 'bg-green-100 dark:bg-green-900/30', icon: '✓' },
        failed: { color: 'text-red-600 dark:text-red-300', bgColor: 'bg-red-100 dark:bg-red-900/30', icon: '✕' },
        cancelled: { color: 'text-yellow-600 dark:text-yellow-300', bgColor: 'bg-yellow-100 dark:bg-yellow-900/30', icon: '◉' },
    };

    const normalizedStatus = status.status === 'in_progress' ? 'running' : status.status === 'success' ? 'completed' : status.status;
    const config = statusConfig[normalizedStatus] || statusConfig.queued;

    const percentage = Math.min(100, Math.max(0, status.progress?.percentage || 0));

    // Calculate estimated time remaining
    let estimatedTimeRemaining = 'Unknown';
    if (status.estimated_completion) {
        const now = new Date().getTime();
        const completionTime = new Date(status.estimated_completion).getTime();
        const remaining = completionTime - now;

        if (remaining > 0) {
            const minutes = Math.floor(remaining / 60000);
            const seconds = Math.floor((remaining % 60000) / 1000);
            estimatedTimeRemaining = `${minutes}m ${seconds}s`;
        } else {
            estimatedTimeRemaining = 'Completed';
        }
    }

    const elapsedTime = status.started_at
        ? Math.floor((new Date().getTime() - new Date(status.started_at).getTime()) / 1000)
        : 0;

    const elapsedMinutes = Math.floor(elapsedTime / 60);
    const elapsedSeconds = elapsedTime % 60;

    return (
        <div className="space-y-6">
            {/* Status Badge */}
            <div className={`${config.bgColor} rounded-lg p-6 text-center`}>
                <div className="text-4xl mb-2">{config.icon}</div>
                <h2 className={`text-3xl font-bold ${config.color} capitalize`}>
                    {normalizedStatus.replace('_', ' ')}
                </h2>
                <p className="text-gray-600 dark:text-slate-300 mt-2">Current build status</p>
            </div>

            {/* Progress Bar */}
            {normalizedStatus === 'running' && (
                <div>
                    <div className="flex justify-between mb-2">
                        <span className="text-sm font-medium text-gray-700 dark:text-slate-300">Progress</span>
                        <span className="text-sm font-medium text-gray-600 dark:text-slate-400">{percentage}%</span>
                    </div>
                    <div className="w-full bg-gray-200 dark:bg-slate-700 rounded-full h-3">
                        <div
                            className="bg-blue-600 h-3 rounded-full transition-all duration-300"
                            style={{ width: `${percentage}%` }}
                        />
                    </div>
                </div>
            )}

            {/* Timing Information */}
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
                <div className="bg-gray-50 dark:bg-slate-700 rounded-lg p-4">
                    <p className="text-sm font-medium text-gray-600 dark:text-slate-300">Elapsed Time</p>
                    <p className="text-2xl font-bold text-gray-900 dark:text-white mt-1">
                        {elapsedMinutes}m {elapsedSeconds}s
                    </p>
                </div>

                {normalizedStatus === 'running' && (
                    <div className="bg-blue-50 dark:bg-blue-900/20 rounded-lg p-4 border border-blue-200 dark:border-blue-700">
                        <p className="text-sm font-medium text-blue-600 dark:text-blue-300">Est. Remaining</p>
                        <p className="text-2xl font-bold text-blue-900 dark:text-blue-200 mt-1">
                            {estimatedTimeRemaining}
                        </p>
                    </div>
                )}
            </div>

            {/* Error Message */}
            {status.error && (
                <div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-700 rounded-lg p-4">
                    <p className="text-sm font-medium text-red-600 dark:text-red-300 mb-1">Error</p>
                    <p className="text-red-800 dark:text-red-200">{status.error}</p>
                </div>
            )}

            {/* Build Details */}
            <div className="bg-white dark:bg-slate-800 border border-gray-200 dark:border-slate-700 rounded-lg p-4 space-y-3">
                <div className="grid grid-cols-2 gap-4">
                    <div>
                        <p className="text-xs font-medium text-gray-600 dark:text-slate-400 uppercase">Started</p>
                        <p className="text-sm text-gray-900 dark:text-white mt-1">
                            {status.started_at
                                ? new Date(status.started_at).toLocaleString()
                                : 'Not started'}
                        </p>
                    </div>
                    <div>
                        <p className="text-xs font-medium text-gray-600 dark:text-slate-400 uppercase">Estimated Completion</p>
                        <p className="text-sm text-gray-900 dark:text-white mt-1">
                            {status.estimated_completion
                                ? new Date(status.estimated_completion).toLocaleString()
                                : normalizedStatus === 'completed'
                                    ? 'Completed'
                                    : 'In progress'}
                        </p>
                    </div>
                </div>
            </div>

            {/* Auto-refresh info */}
            <div className="text-xs text-gray-500 dark:text-slate-400 text-center">
                Status auto-refreshes every 5 seconds
            </div>
        </div>
    );
};

export default BuildStatus;
