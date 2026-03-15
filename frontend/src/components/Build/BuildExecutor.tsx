import React, { useCallback, useState } from 'react';
import { Build, BuildClient, CreateBuildRequest } from '../../api/buildClient';
import BuildLogs from './BuildLogs';
import BuildStatus from './BuildStatus';

export interface BuildExecutorProps {
    client: BuildClient;
    projectId: string;
}

export const BuildExecutor: React.FC<BuildExecutorProps> = ({ client, projectId }) => {
    const [builds, setBuilds] = useState<Build[]>([]);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const [selectedBuild, setSelectedBuild] = useState<Build | null>(null);
    const [showLogs, setShowLogs] = useState(false);
    const [showStatus, setShowStatus] = useState(false);
    const [formData, setFormData] = useState({ gitBranch: 'main', gitCommit: '' });

    // Load builds
    const loadBuilds = useCallback(async () => {
        try {
            setLoading(true);
            setError(null);
            const result = await client.listBuilds({ project_id: projectId, limit: 20 });
            setBuilds(result.builds);
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Failed to load builds');
        } finally {
            setLoading(false);
        }
    }, [client, projectId]);

    // Create build
    const handleCreateBuild = async (e: React.FormEvent) => {
        e.preventDefault();
        try {
            setLoading(true);
            setError(null);

            const request: CreateBuildRequest = {
                project_id: projectId,
                git_branch: formData.gitBranch,
                git_commit: formData.gitCommit || undefined,
            };

            const newBuild = await client.createBuild(request);
            setBuilds([newBuild, ...builds]);
            setFormData({ gitBranch: 'main', gitCommit: '' });
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Failed to create build');
        } finally {
            setLoading(false);
        }
    };

    // Start build
    const handleStartBuild = async (buildId: string) => {
        try {
            setError(null);
            const updated = await client.startBuild(buildId);
            setBuilds(builds.map(b => b.id === buildId ? updated : b));
            setSelectedBuild(updated);
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Failed to start build');
        }
    };

    // Cancel build
    const handleCancelBuild = async (buildId: string) => {
        try {
            setError(null);
            const updated = await client.cancelBuild(buildId);
            setBuilds(builds.map(b => b.id === buildId ? updated : b));
            setSelectedBuild(updated);
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Failed to cancel build');
        }
    };

    // Retry build
    const handleRetryBuild = async (buildId: string) => {
        try {
            setError(null);
            const newBuild = await client.retryBuild(buildId);
            setBuilds([newBuild, ...builds]);
            setSelectedBuild(newBuild);
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Failed to retry build');
        }
    };

    const statusColor: Record<string, string> = {
        queued: 'bg-gray-100 dark:bg-slate-700 text-gray-800 dark:text-slate-200',
        running: 'bg-blue-100 dark:bg-blue-900/30 text-blue-800 dark:text-blue-300',
        completed: 'bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-300',
        failed: 'bg-red-100 dark:bg-red-900/30 text-red-800 dark:text-red-300',
        cancelled: 'bg-yellow-100 dark:bg-yellow-900/30 text-yellow-800 dark:text-yellow-300',
    };

    return (
        <div className="space-y-6">
            {/* Header */}
            <div className="bg-white dark:bg-slate-800 rounded-lg shadow p-6 border border-gray-200 dark:border-slate-700">
                <h1 className="text-3xl font-bold text-gray-900 dark:text-white">Build Executor</h1>
                <p className="text-gray-600 dark:text-slate-300 mt-1">Manage and monitor image builds in real-time</p>
            </div>

            {/* Error Alert */}
            {error && (
                <div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-700 rounded-lg p-4">
                    <p className="text-red-800 dark:text-red-200">{error}</p>
                    <button
                        onClick={() => setError(null)}
                        className="text-red-600 dark:text-red-300 hover:text-red-800 dark:hover:text-red-200 mt-2"
                    >
                        Dismiss
                    </button>
                </div>
            )}

            {/* Create Build Form */}
            <form onSubmit={handleCreateBuild} className="bg-white dark:bg-slate-800 rounded-lg shadow p-6 border border-gray-200 dark:border-slate-700">
                <h2 className="text-xl font-semibold text-gray-900 dark:text-white mb-4">Create New Build</h2>
                <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                    <div>
                        <label className="block text-sm font-medium text-gray-700 dark:text-slate-300">Git Branch</label>
                        <input
                            type="text"
                            value={formData.gitBranch}
                            onChange={(e) => setFormData({ ...formData, gitBranch: e.target.value })}
                            className="mt-1 w-full px-3 py-2 border border-gray-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white"
                            placeholder="main"
                            required
                        />
                    </div>
                    <div>
                        <label className="block text-sm font-medium text-gray-700 dark:text-slate-300">Git Commit (Optional)</label>
                        <input
                            type="text"
                            value={formData.gitCommit}
                            onChange={(e) => setFormData({ ...formData, gitCommit: e.target.value })}
                            className="mt-1 w-full px-3 py-2 border border-gray-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white"
                            placeholder="abc123..."
                        />
                    </div>
                </div>
                <button
                    type="submit"
                    disabled={loading}
                    className="mt-4 bg-blue-600 text-white px-4 py-2 rounded-md hover:bg-blue-700 disabled:opacity-50"
                >
                    {loading ? 'Creating...' : 'Create Build'}
                </button>
            </form>

            {/* Build List */}
            <div className="bg-white dark:bg-slate-800 rounded-lg shadow border border-gray-200 dark:border-slate-700">
                <div className="px-6 py-4 border-b border-gray-200 dark:border-slate-700">
                    <div className="flex justify-between items-center">
                        <h2 className="text-xl font-semibold text-gray-900 dark:text-white">Recent Builds</h2>
                        <button
                            onClick={loadBuilds}
                            disabled={loading}
                            className="text-blue-600 dark:text-blue-400 hover:text-blue-800 dark:hover:text-blue-300 disabled:opacity-50"
                        >
                            {loading ? 'Loading...' : 'Refresh'}
                        </button>
                    </div>
                </div>

                {builds.length === 0 ? (
                    <div className="px-6 py-12 text-center text-gray-500 dark:text-slate-400">
                        No builds yet. Create your first build above!
                    </div>
                ) : (
                    <div className="overflow-x-auto">
                        <table className="w-full">
                            <thead className="bg-gray-50 dark:bg-slate-900 border-t border-gray-200 dark:border-slate-700">
                                <tr>
                                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-700 dark:text-slate-300 uppercase tracking-wider">
                                        Build
                                    </th>
                                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-700 dark:text-slate-300 uppercase tracking-wider">
                                        Branch
                                    </th>
                                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-700 dark:text-slate-300 uppercase tracking-wider">
                                        Status
                                    </th>
                                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-700 dark:text-slate-300 uppercase tracking-wider">
                                        Created
                                    </th>
                                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-700 dark:text-slate-300 uppercase tracking-wider">
                                        Actions
                                    </th>
                                </tr>
                            </thead>
                            <tbody className="bg-white dark:bg-slate-800 divide-y divide-gray-200 dark:divide-slate-700">
                                {builds.map((build) => (
                                    <tr key={build.id} className="hover:bg-gray-50 dark:hover:bg-slate-700">
                                        <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900 dark:text-white">
                                            #{build.build_number}
                                        </td>
                                        <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-600 dark:text-slate-300">
                                            {build.git_branch}
                                        </td>
                                        <td className="px-6 py-4 whitespace-nowrap">
                                            <span className={`px-3 py-1 rounded-full text-xs font-medium ${statusColor[build.status] || 'bg-gray-100 dark:bg-slate-700 text-gray-800 dark:text-slate-200'}`}>
                                                {build.status}
                                            </span>
                                        </td>
                                        <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-600 dark:text-slate-300">
                                            {new Date(build.created_at).toLocaleDateString()}
                                        </td>
                                        <td className="px-6 py-4 whitespace-nowrap text-sm space-x-2">
                                            <button
                                                onClick={() => {
                                                    setSelectedBuild(build);
                                                    setShowStatus(true);
                                                }}
                                                className="text-blue-600 hover:text-blue-800"
                                            >
                                                Status
                                            </button>
                                            <button
                                                onClick={() => {
                                                    setSelectedBuild(build);
                                                    setShowLogs(true);
                                                }}
                                                className="text-green-600 hover:text-green-800"
                                            >
                                                Logs
                                            </button>
                                            {build.status === 'queued' && (
                                                <button
                                                    onClick={() => handleStartBuild(build.id)}
                                                    className="text-purple-600 hover:text-purple-800"
                                                >
                                                    Start
                                                </button>
                                            )}
                                            {build.status === 'running' && (
                                                <button
                                                    onClick={() => handleCancelBuild(build.id)}
                                                    className="text-red-600 hover:text-red-800"
                                                >
                                                    Cancel
                                                </button>
                                            )}
                                            {build.status === 'failed' && (
                                                <button
                                                    onClick={() => handleRetryBuild(build.id)}
                                                    className="text-orange-600 hover:text-orange-800"
                                                >
                                                    Retry
                                                </button>
                                            )}
                                        </td>
                                    </tr>
                                ))}
                            </tbody>
                        </table>
                    </div>
                )}
            </div>

            {/* Status Modal */}
            {showStatus && selectedBuild && (
                <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
                    <div className="bg-white dark:bg-slate-800 rounded-lg shadow-lg p-6 max-w-2xl w-full mx-4 border border-gray-200 dark:border-slate-700">
                        <div className="flex justify-between items-center mb-4">
                            <h2 className="text-2xl font-bold text-gray-900 dark:text-white">Build #{selectedBuild.build_number} Status</h2>
                            <button onClick={() => setShowStatus(false)} className="text-gray-400 dark:text-slate-400 hover:text-gray-600 dark:hover:text-slate-200">
                                ✕
                            </button>
                        </div>
                        <BuildStatus client={client} buildId={selectedBuild.id} />
                    </div>
                </div>
            )}

            {/* Logs Modal */}
            {showLogs && selectedBuild && (
                <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
                    <div className="bg-white dark:bg-slate-800 rounded-lg shadow-lg p-6 max-w-4xl w-full mx-4 max-h-screen overflow-y-auto border border-gray-200 dark:border-slate-700">
                        <div className="flex justify-between items-center mb-4">
                            <h2 className="text-2xl font-bold text-gray-900 dark:text-white">Build #{selectedBuild.build_number} Logs</h2>
                            <button onClick={() => setShowLogs(false)} className="text-gray-400 dark:text-slate-400 hover:text-gray-600 dark:hover:text-slate-200">
                                ✕
                            </button>
                        </div>
                        <BuildLogs client={client} buildId={selectedBuild.id} />
                    </div>
                </div>
            )}
        </div>
    );
};

export default BuildExecutor;
