import { api } from '@/services/api';
import { useAuthStore } from '@/store/auth';
import {
    AlertCircle,
    Check,
    Edit2,
    Plus,
    RefreshCw,
    Trash2,
    X,
} from 'lucide-react';
import React, { useCallback, useEffect, useState } from 'react';

interface InfrastructureNode {
    id: string;
    name: string;
    status: string; // 'ready', 'offline', 'maintenance'
    total_cpu_capacity: number;
    total_memory_capacity_gb: number;
    total_disk_capacity_gb: number;
    used_cpu_cores?: number;
    used_memory_gb?: number;
    used_disk_gb?: number;
    last_heartbeat?: string;
    maintenance_mode?: boolean;
    labels?: Record<string, string>;
    created_at?: string;
    updated_at?: string;
}

interface GetNodesResponse {
    nodes: InfrastructureNode[];
    total: number;
    limit: number;
    offset: number;
    has_more: boolean;
}

interface InfrastructureHealth {
    total_nodes: number;
    healthy_nodes: number;
    offline_nodes: number;
    maintenance_nodes: number;
    total_cpu_capacity: number;
    used_cpu_cores: number;
    total_memory_capacity_gb: number;
    used_memory_gb: number;
    total_disk_capacity_gb: number;
    used_disk_gb: number;
    average_cpu_usage_percent: number;
    average_memory_usage_percent: number;
    average_disk_usage_percent: number;
    node_health_breakdown: NodeHealthSummary[];
}

interface NodeHealthSummary {
    node_id: string;
    node_name: string;
    status: string;
    cpu_usage_percent: number;
    memory_usage_percent: number;
    disk_usage_percent: number;
    up_since?: string;
}

interface FormData {
    name: string;
    total_cpu_capacity: number;
    total_memory_capacity_gb: number;
    total_disk_capacity_gb: number;
    status: string;
}

const statusColors: Record<string, string> = {
    ready: 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200',
    offline: 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200',
    maintenance: 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200',
};

const statusBgColors: Record<string, string> = {
    ready: 'bg-green-500',
    offline: 'bg-red-500',
    maintenance: 'bg-yellow-500',
};

/**
 * InfrastructurePanel - Admin component for managing infrastructure nodes
 * Displays nodes with resource metrics, health status, and CRUD operations
 */
export const InfrastructurePanel: React.FC = () => {
    const { user } = useAuthStore();
    const [nodesData, setNodesData] = useState<GetNodesResponse | null>(null);
    const [healthData, setHealthData] = useState<InfrastructureHealth | null>(null);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const [currentPage, setCurrentPage] = useState(0);
    const [autoRefresh, setAutoRefresh] = useState(true);
    const [refreshInterval, setRefreshInterval] = useState(5000);

    // Form state
    const [showForm, setShowForm] = useState(false);
    const [editingNode, setEditingNode] = useState<InfrastructureNode | null>(null);
    const [formData, setFormData] = useState<FormData>({
        name: '',
        total_cpu_capacity: 4,
        total_memory_capacity_gb: 16,
        total_disk_capacity_gb: 100,
        status: 'ready',
    });
    const [formError, setFormError] = useState<string | null>(null);

    const pageSize = 20;

    // Fetch nodes
    const fetchNodes = useCallback(async () => {
        try {
            setLoading(true);
            setError(null);

            const response = await api.get('/admin/infrastructure/nodes', {
                params: {
                    limit: pageSize,
                    offset: currentPage * pageSize,
                },
            });
            const data: GetNodesResponse = response.data;
            setNodesData(data);
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Failed to fetch nodes');
        } finally {
            setLoading(false);
        }
    }, [currentPage]);

    // Fetch health metrics
    const fetchHealth = useCallback(async () => {
        try {
            const response = await api.get('/admin/infrastructure/health');
            const data: InfrastructureHealth = response.data;
            setHealthData(data);
        } catch (err) {
            console.error('Failed to fetch health metrics:', err);
        }
    }, []);

    // Auto-refresh effect
    useEffect(() => {
        if (!autoRefresh) return;

        fetchNodes();
        fetchHealth();

        const interval = setInterval(() => {
            fetchNodes();
            fetchHealth();
        }, refreshInterval);

        return () => clearInterval(interval);
    }, [fetchNodes, fetchHealth, autoRefresh, refreshInterval]);

    // Handle form submission (create or update)
    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();
        setFormError(null);

        try {
            if (!formData.name.trim()) {
                setFormError('Node name is required');
                return;
            }

            if (
                formData.total_cpu_capacity <= 0 ||
                formData.total_memory_capacity_gb <= 0 ||
                formData.total_disk_capacity_gb <= 0
            ) {
                setFormError('All resource values must be greater than 0');
                return;
            }

            if (editingNode) {
                await api.put(`/admin/infrastructure/nodes/${editingNode.id}`, formData);
            } else {
                await api.post('/admin/infrastructure/nodes', formData);
            }

            // Reset form and refresh
            setShowForm(false);
            setEditingNode(null);
            setFormData({
                name: '',
                total_cpu_capacity: 4,
                total_memory_capacity_gb: 16,
                total_disk_capacity_gb: 100,
                status: 'ready',
            });

            await fetchNodes();
            await fetchHealth();
        } catch (err) {
            setFormError(err instanceof Error ? err.message : 'An error occurred');
        }
    };

    // Handle delete
    const handleDelete = async (node: InfrastructureNode) => {
        if (!window.confirm(`Delete node "${node.name}"? This action cannot be undone.`)) {
            return;
        }

        try {
            setError(null);

            await api.delete(`/admin/infrastructure/nodes/${node.id}`);

            await fetchNodes();
            await fetchHealth();
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Failed to delete node');
        }
    };

    // Handle edit
    const handleEdit = (node: InfrastructureNode) => {
        setEditingNode(node);
        setFormData({
            name: node.name,
            total_cpu_capacity: node.total_cpu_capacity,
            total_memory_capacity_gb: node.total_memory_capacity_gb,
            total_disk_capacity_gb: node.total_disk_capacity_gb,
            status: node.status,
        });
        setShowForm(true);
    };

    // Handle cancel
    const handleCancel = () => {
        setShowForm(false);
        setEditingNode(null);
        setFormData({
            name: '',
            total_cpu_capacity: 4,
            total_memory_capacity_gb: 16,
            total_disk_capacity_gb: 100,
            status: 'ready',
        });
        setFormError(null);
    };

    // Calculate resource percentage
    const getResourcePercentage = (used: number = 0, total: number): number => {
        if (total === 0) return 0;
        return Math.round((used / total) * 100);
    };

    // Format bytes to human readable
    const formatBytes = (bytes: number): string => {
        if (bytes === 0) return '0 B';
        const k = 1024;
        const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
        const i = Math.floor(Math.log(bytes) / Math.log(k));
        return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
    };

    if (!user) {
        return (
            <div className="flex items-center justify-center p-4">
                <div className="text-center">
                    <AlertCircle className="mx-auto h-12 w-12 text-gray-400 dark:text-gray-500" />
                    <h3 className="mt-2 text-sm font-medium text-gray-900 dark:text-gray-100">Not authenticated</h3>
                    <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">Please log in to manage infrastructure.</p>
                </div>
            </div>
        );
    }

    return (
        <div className="space-y-4 p-4">
            {/* Header */}
            <div className="flex items-center justify-between">
                <div>
                    <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">Infrastructure Nodes</h2>
                    <p className="text-sm text-gray-600 dark:text-gray-400">
                        {nodesData ? `Total: ${nodesData.total} nodes` : 'Loading...'}
                    </p>
                </div>
                <div className="flex items-center gap-2">
                    <button
                        onClick={() => fetchNodes()}
                        disabled={loading}
                        className="rounded-lg bg-blue-600 p-2 text-white hover:bg-blue-700 disabled:opacity-50 dark:bg-blue-700 dark:hover:bg-blue-800"
                        title="Refresh nodes"
                    >
                        <RefreshCw className={`h-4 w-4 ${loading ? 'animate-spin' : ''}`} />
                    </button>
                    <button
                        onClick={() => setShowForm(true)}
                        className="flex items-center gap-2 rounded-lg bg-green-600 px-4 py-2 text-white hover:bg-green-700 dark:bg-green-700 dark:hover:bg-green-800"
                    >
                        <Plus className="h-4 w-4" />
                        Add Node
                    </button>
                    <label className="flex items-center gap-2 text-sm text-gray-600 dark:text-gray-400">
                        <input
                            type="checkbox"
                            checked={autoRefresh}
                            onChange={(e) => setAutoRefresh(e.target.checked)}
                            className="rounded"
                        />
                        Auto-refresh
                    </label>
                </div>
            </div>

            {/* Error Alert */}
            {error && (
                <div className="rounded-lg bg-red-50 dark:bg-red-900/20 p-4">
                    <div className="flex items-start gap-3">
                        <AlertCircle className="h-5 w-5 text-red-600 dark:text-red-400" />
                        <div>
                            <h3 className="font-medium text-red-900 dark:text-red-100">Error</h3>
                            <p className="text-sm text-red-700 dark:text-red-300">{error}</p>
                        </div>
                    </div>
                </div>
            )}

            {/* Health Summary */}
            {healthData && (
                <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                    {/* Node Count */}
                    <div className="rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 p-4">
                        <h3 className="text-sm font-medium text-gray-600 dark:text-gray-400">Node Status</h3>
                        <div className="mt-3 space-y-2 text-sm">
                            <div className="flex items-center justify-between">
                                <span className="flex items-center gap-2">
                                    <span className={`h-2 w-2 rounded-full ${statusBgColors['ready']}`}></span>
                                    Healthy
                                </span>
                                <span className="font-semibold text-gray-900 dark:text-gray-100">{healthData.healthy_nodes}</span>
                            </div>
                            <div className="flex items-center justify-between">
                                <span className="flex items-center gap-2">
                                    <span className={`h-2 w-2 rounded-full ${statusBgColors['offline']}`}></span>
                                    Offline
                                </span>
                                <span className="font-semibold text-gray-900 dark:text-gray-100">{healthData.offline_nodes}</span>
                            </div>
                            <div className="flex items-center justify-between">
                                <span className="flex items-center gap-2">
                                    <span className={`h-2 w-2 rounded-full ${statusBgColors['maintenance']}`}></span>
                                    Maintenance
                                </span>
                                <span className="font-semibold text-gray-900 dark:text-gray-100">{healthData.maintenance_nodes}</span>
                            </div>
                        </div>
                    </div>

                    {/* CPU Usage */}
                    <div className="rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 p-4">
                        <h3 className="text-sm font-medium text-gray-600 dark:text-gray-400">CPU Cores</h3>
                        <div className="mt-3">
                            <div className="text-2xl font-semibold text-gray-900 dark:text-gray-100">
                                {healthData.used_cpu_cores}/{healthData.total_cpu_capacity}
                            </div>
                            <div className="mt-2 h-2 rounded-full bg-gray-200 dark:bg-gray-700">
                                <div
                                    className="h-full rounded-full bg-blue-600"
                                    style={{
                                        width: `${getResourcePercentage(
                                            healthData.used_cpu_cores,
                                            healthData.total_cpu_capacity
                                        )}%`,
                                    }}
                                ></div>
                            </div>
                            <p className="mt-1 text-xs text-gray-600 dark:text-gray-400">
                                {getResourcePercentage(healthData.used_cpu_cores, healthData.total_cpu_capacity)}%
                                used
                            </p>
                        </div>
                    </div>

                    {/* Memory Usage */}
                    <div className="rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 p-4">
                        <h3 className="text-sm font-medium text-gray-600 dark:text-gray-400">Memory</h3>
                        <div className="mt-3">
                            <div className="text-2xl font-semibold text-gray-900 dark:text-gray-100">
                                {healthData.used_memory_gb}/{healthData.total_memory_capacity_gb} GB
                            </div>
                            <div className="mt-2 h-2 rounded-full bg-gray-200 dark:bg-gray-700">
                                <div
                                    className="h-full rounded-full bg-purple-600"
                                    style={{
                                        width: `${getResourcePercentage(
                                            healthData.used_memory_gb,
                                            healthData.total_memory_capacity_gb
                                        )}%`,
                                    }}
                                ></div>
                            </div>
                            <p className="mt-1 text-xs text-gray-600 dark:text-gray-400">
                                {getResourcePercentage(healthData.used_memory_gb, healthData.total_memory_capacity_gb)}%
                                used
                            </p>
                        </div>
                    </div>
                </div>
            )}

            {/* Add/Edit Form Modal */}
            {showForm && (
                <div className="fixed inset-0 z-50 flex items-center justify-center bg-black bg-opacity-50">
                    <div className="w-full max-w-md rounded-lg bg-white dark:bg-gray-800 p-6 shadow-xl">
                        <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
                            {editingNode ? 'Edit Node' : 'Add New Node'}
                        </h3>

                        {formError && (
                            <div className="mt-3 rounded-lg bg-red-50 dark:bg-red-900/20 p-3">
                                <p className="text-sm text-red-700 dark:text-red-300">{formError}</p>
                            </div>
                        )}

                        <form onSubmit={handleSubmit} className="mt-4 space-y-4">
                            {/* Node Name */}
                            <div>
                                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">Node Name</label>
                                <input
                                    type="text"
                                    value={formData.name}
                                    onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                                    placeholder="e.g., worker-1, build-server-02"
                                    className="mt-1 w-full rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 px-3 py-2 text-gray-900 dark:text-gray-100 placeholder-gray-500 dark:placeholder-gray-400"
                                />
                            </div>

                            {/* Status */}
                            <div>
                                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">Status</label>
                                <select
                                    value={formData.status}
                                    onChange={(e) => setFormData({ ...formData, status: e.target.value })}
                                    className="mt-1 w-full rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 px-3 py-2 text-gray-900 dark:text-gray-100"
                                >
                                    <option value="ready">Ready</option>
                                    <option value="maintenance">Maintenance</option>
                                    <option value="offline">Offline</option>
                                </select>
                            </div>

                            {/* CPU Cores */}
                            <div>
                                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">CPU Cores</label>
                                <input
                                    type="number"
                                    value={formData.total_cpu_capacity}
                                    onChange={(e) =>
                                        setFormData({
                                            ...formData,
                                            total_cpu_capacity: parseInt(e.target.value) || 0,
                                        })
                                    }
                                    min="1"
                                    className="mt-1 w-full rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 px-3 py-2 text-gray-900 dark:text-gray-100"
                                />
                            </div>

                            {/* Memory GB */}
                            <div>
                                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">Memory (GB)</label>
                                <input
                                    type="number"
                                    value={formData.total_memory_capacity_gb}
                                    onChange={(e) =>
                                        setFormData({
                                            ...formData,
                                            total_memory_capacity_gb: parseInt(e.target.value) || 0,
                                        })
                                    }
                                    min="1"
                                    className="mt-1 w-full rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 px-3 py-2 text-gray-900 dark:text-gray-100"
                                />
                            </div>

                            {/* Disk GB */}
                            <div>
                                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">Disk (GB)</label>
                                <input
                                    type="number"
                                    value={formData.total_disk_capacity_gb}
                                    onChange={(e) =>
                                        setFormData({
                                            ...formData,
                                            total_disk_capacity_gb: parseInt(e.target.value) || 0,
                                        })
                                    }
                                    min="1"
                                    className="mt-1 w-full rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 px-3 py-2 text-gray-900 dark:text-gray-100"
                                />
                            </div>

                            {/* Buttons */}
                            <div className="flex gap-3 pt-4">
                                <button
                                    type="submit"
                                    className="flex-1 flex items-center justify-center gap-2 rounded-lg bg-blue-600 px-4 py-2 text-white hover:bg-blue-700 dark:bg-blue-700 dark:hover:bg-blue-800"
                                >
                                    <Check className="h-4 w-4" />
                                    {editingNode ? 'Update' : 'Create'}
                                </button>
                                <button
                                    type="button"
                                    onClick={handleCancel}
                                    className="flex-1 flex items-center justify-center gap-2 rounded-lg bg-gray-300 dark:bg-gray-600 px-4 py-2 text-gray-900 dark:text-gray-100 hover:bg-gray-400 dark:hover:bg-gray-500"
                                >
                                    <X className="h-4 w-4" />
                                    Cancel
                                </button>
                            </div>
                        </form>
                    </div>
                </div>
            )}

            {/* Nodes Table */}
            <div className="overflow-x-auto rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800">
                <table className="w-full">
                    <thead className="bg-gray-100 dark:bg-gray-700">
                        <tr>
                            <th className="px-6 py-3 text-left text-sm font-medium text-gray-900 dark:text-gray-100">Name</th>
                            <th className="px-6 py-3 text-left text-sm font-medium text-gray-900 dark:text-gray-100">Status</th>
                            <th className="px-6 py-3 text-left text-sm font-medium text-gray-900 dark:text-gray-100">CPU</th>
                            <th className="px-6 py-3 text-left text-sm font-medium text-gray-900 dark:text-gray-100">Memory</th>
                            <th className="px-6 py-3 text-left text-sm font-medium text-gray-900 dark:text-gray-100">Disk</th>
                            <th className="px-6 py-3 text-left text-sm font-medium text-gray-900 dark:text-gray-100">
                                Last Heartbeat
                            </th>
                            <th className="px-6 py-3 text-left text-sm font-medium text-gray-900 dark:text-gray-100">Actions</th>
                        </tr>
                    </thead>
                    <tbody className="divide-y divide-gray-200 dark:divide-gray-600">
                        {loading && !nodesData ? (
                            <tr>
                                <td colSpan={7} className="px-6 py-4 text-center text-gray-500 dark:text-gray-400">
                                    Loading nodes...
                                </td>
                            </tr>
                        ) : nodesData?.nodes.length === 0 ? (
                            <tr>
                                <td colSpan={7} className="px-6 py-4 text-center text-gray-500 dark:text-gray-400">
                                    No infrastructure nodes configured
                                </td>
                            </tr>
                        ) : (
                            nodesData?.nodes.map((node) => (
                                <tr key={node.id} className="hover:bg-gray-50 dark:hover:bg-gray-700">
                                    {/* Name */}
                                    <td className="px-6 py-4">
                                        <span className="font-medium text-gray-900 dark:text-gray-100">{node.name}</span>
                                    </td>

                                    {/* Status */}
                                    <td className="px-6 py-4">
                                        <span
                                            className={`inline-flex rounded-full px-3 py-1 text-xs font-medium ${statusColors[node.status] || 'bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-200'
                                                }`}
                                        >
                                            {node.status}
                                        </span>
                                    </td>

                                    {/* CPU */}
                                    <td className="px-6 py-4">
                                        <div className="flex flex-col gap-1">
                                            <span className="text-sm text-gray-900 dark:text-gray-100">
                                                {node.used_cpu_cores ?? 0}/{node.total_cpu_capacity}
                                            </span>
                                            <div className="h-1 w-24 rounded-full bg-gray-200 dark:bg-gray-600">
                                                <div
                                                    className="h-full rounded-full bg-blue-600"
                                                    style={{
                                                        width: `${getResourcePercentage(node.used_cpu_cores, node.total_cpu_capacity)}%`,
                                                    }}
                                                ></div>
                                            </div>
                                        </div>
                                    </td>

                                    {/* Memory */}
                                    <td className="px-6 py-4">
                                        <div className="flex flex-col gap-1">
                                            <span className="text-sm text-gray-900 dark:text-gray-100">
                                                {node.used_memory_gb ?? 0}/{node.total_memory_capacity_gb} GB
                                            </span>
                                            <div className="h-1 w-24 rounded-full bg-gray-200 dark:bg-gray-600">
                                                <div
                                                    className="h-full rounded-full bg-purple-600"
                                                    style={{
                                                        width: `${getResourcePercentage(node.used_memory_gb, node.total_memory_capacity_gb)}%`,
                                                    }}
                                                ></div>
                                            </div>
                                        </div>
                                    </td>

                                    {/* Disk */}
                                    <td className="px-6 py-4">
                                        <div className="flex flex-col gap-1">
                                            <span className="text-sm text-gray-900 dark:text-gray-100">
                                                {node.used_disk_gb ?? 0}/{node.total_disk_capacity_gb} GB
                                            </span>
                                            <div className="h-1 w-24 rounded-full bg-gray-200 dark:bg-gray-600">
                                                <div
                                                    className="h-full rounded-full bg-green-600"
                                                    style={{
                                                        width: `${getResourcePercentage(node.used_disk_gb, node.total_disk_capacity_gb)}%`,
                                                    }}
                                                ></div>
                                            </div>
                                        </div>
                                    </td>

                                    {/* Last Heartbeat */}
                                    <td className="px-6 py-4">
                                        {node.last_heartbeat ? (
                                            <span className="text-xs text-gray-600 dark:text-gray-400">
                                                {new Date(node.last_heartbeat).toLocaleString()}
                                            </span>
                                        ) : (
                                            <span className="text-xs text-gray-400 dark:text-gray-500">Never</span>
                                        )}
                                    </td>

                                    {/* Actions */}
                                    <td className="px-6 py-4">
                                        <div className="flex items-center gap-2">
                                            <button
                                                onClick={() => handleEdit(node)}
                                                className="rounded p-1 hover:bg-blue-100 dark:hover:bg-blue-900"
                                                title="Edit node"
                                            >
                                                <Edit2 className="h-4 w-4 text-blue-600 dark:text-blue-400" />
                                            </button>
                                            <button
                                                onClick={() => handleDelete(node)}
                                                className="rounded p-1 hover:bg-red-100 dark:hover:bg-red-900"
                                                title="Delete node"
                                            >
                                                <Trash2 className="h-4 w-4 text-red-600 dark:text-red-400" />
                                            </button>
                                        </div>
                                    </td>
                                </tr>
                            ))
                        )}
                    </tbody>
                </table>
            </div>

            {/* Pagination */}
            {nodesData && (
                <div className="flex items-center justify-between border-t border-gray-300 dark:border-gray-600 pt-4">
                    <div className="text-sm text-gray-600 dark:text-gray-400">
                        Showing {nodesData.nodes.length > 0 ? currentPage * pageSize + 1 : 0} to{' '}
                        {Math.min((currentPage + 1) * pageSize, nodesData.total)} of {nodesData.total}
                    </div>
                    <div className="flex items-center gap-2">
                        <button
                            onClick={() => setCurrentPage(Math.max(0, currentPage - 1))}
                            disabled={currentPage === 0}
                            className="rounded-lg bg-gray-200 dark:bg-gray-700 px-4 py-2 text-gray-900 dark:text-gray-100 hover:bg-gray-300 dark:hover:bg-gray-600 disabled:opacity-50"
                        >
                            Previous
                        </button>
                        <span className="text-sm text-gray-600 dark:text-gray-400">
                            Page {currentPage + 1} of {Math.ceil(nodesData.total / pageSize)}
                        </span>
                        <button
                            onClick={() => setCurrentPage(currentPage + 1)}
                            disabled={!nodesData.has_more}
                            className="rounded-lg bg-gray-200 dark:bg-gray-700 px-4 py-2 text-gray-900 dark:text-gray-100 hover:bg-gray-300 dark:hover:bg-gray-600 disabled:opacity-50"
                        >
                            Next
                        </button>
                    </div>
                </div>
            )}
        </div>
    );
};

export default InfrastructurePanel;
