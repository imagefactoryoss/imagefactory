import Drawer from '@/components/ui/Drawer'
import { useRefresh } from '@/context/RefreshContext'
import { auditService } from '@/services/auditService'
import { AuditEvent } from '@/types/audit'
import { format } from 'date-fns'
import { useCallback, useEffect, useState } from 'react'

const AuditLogsPage = () => {
    const [events, setEvents] = useState<AuditEvent[]>([])
    const [loading, setLoading] = useState(true)
    const [total, setTotal] = useState(0)
    const [page, setPage] = useState(0)
    const [limit] = useState(50)
    const [selectedEvent, setSelectedEvent] = useState<AuditEvent | null>(null)

    // Filters
    const [searchTerm, setSearchTerm] = useState('')
    const [eventTypeFilter, setEventTypeFilter] = useState('')
    const [severityFilter, setSeverityFilter] = useState('')
    const [userFilter, setUserFilter] = useState('')
    const [resourceFilter, setResourceFilter] = useState('')
    const [actionFilter, setActionFilter] = useState('')

    const { registerRefreshCallback, unregisterRefreshCallback } = useRefresh()

    // Debounced search
    const [debouncedSearchTerm, setDebouncedSearchTerm] = useState('')

    useEffect(() => {
        const timer = setTimeout(() => {
            setDebouncedSearchTerm(searchTerm)
        }, 300)
        return () => clearTimeout(timer)
    }, [searchTerm])

    const loadAuditEvents = useCallback(async () => {
        try {
            setLoading(true)
            const filters: any = {}

            if (debouncedSearchTerm) filters.search = debouncedSearchTerm
            if (eventTypeFilter) filters.event_type = eventTypeFilter
            if (severityFilter) filters.severity = severityFilter
            if (userFilter) filters.user_id = userFilter
            if (resourceFilter) filters.resource = resourceFilter
            if (actionFilter) filters.action = actionFilter

            const response = await auditService.getAuditEvents({
                ...filters,
                limit,
                offset: page * limit
            })

            setEvents(response.events)
            setTotal(response.total)
        } catch (error) {
        } finally {
            setLoading(false)
        }
    }, [debouncedSearchTerm, eventTypeFilter, severityFilter, userFilter, resourceFilter, actionFilter, page, limit])

    const refreshCallback = useCallback(async () => {
        await loadAuditEvents()
    }, [loadAuditEvents])

    useEffect(() => {
        loadAuditEvents()
    }, [loadAuditEvents])

    // Register refresh callback
    useEffect(() => {
        registerRefreshCallback(refreshCallback)
        return () => unregisterRefreshCallback(refreshCallback)
    }, [registerRefreshCallback, unregisterRefreshCallback, refreshCallback])

    const clearFilters = () => {
        setSearchTerm('')
        setEventTypeFilter('')
        setSeverityFilter('')
        setUserFilter('')
        setResourceFilter('')
        setActionFilter('')
        setPage(0)
    }

    const getSeverityBadgeClass = (severity: string) => {
        switch (severity) {
            case 'critical':
            case 'error':
                return 'bg-red-100 dark:bg-red-900 text-red-800 dark:text-red-200'
            case 'warning':
                return 'bg-yellow-100 dark:bg-yellow-900 text-yellow-800 dark:text-yellow-200'
            case 'info':
            default:
                return 'bg-blue-100 dark:bg-blue-900 text-blue-800 dark:text-blue-200'
        }
    }

    const getEventTypeBadgeClass = (eventType: string) => {
        switch (eventType) {
            case 'login_success':
            case 'login_failure':
                return 'bg-slate-100 dark:bg-slate-900 text-slate-800 dark:text-slate-200'
            case 'user_create':
            case 'user_update':
            case 'user_delete':
                return 'bg-green-100 dark:bg-green-900 text-green-800 dark:text-green-200'
            case 'config_change':
                return 'bg-purple-100 dark:bg-purple-900 text-purple-800 dark:text-purple-200'
            default:
                return 'bg-slate-100 dark:bg-slate-900 text-slate-800 dark:text-slate-200'
        }
    }

    return (
        <div className="space-y-6 p-6">
            <div className="flex items-center justify-between">
                <div>
                    <h1 className="text-3xl font-bold tracking-tight text-slate-900 dark:text-white">Audit Logs</h1>
                    <p className="text-slate-600 dark:text-slate-400">
                        Review system activity and security events
                    </p>
                </div>
                <button
                    onClick={loadAuditEvents}
                    className="inline-flex items-center gap-2 px-4 py-2 bg-blue-600 text-white rounded-md text-sm font-medium hover:bg-blue-700 transition-colors"
                    title="Refresh audit logs"
                >
                    <svg
                        className="w-4 h-4"
                        fill="none"
                        stroke="currentColor"
                        viewBox="0 0 24 24"
                    >
                        <path
                            strokeLinecap="round"
                            strokeLinejoin="round"
                            strokeWidth={2}
                            d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"
                        />
                    </svg>
                    Refresh
                </button>
            </div>

            {/* Filters */}
            <div className="bg-white dark:bg-slate-950 rounded-lg shadow-md">
                <div className="px-4 py-5 sm:p-6">
                    <h3 className="text-lg leading-6 font-medium text-slate-900 dark:text-white mb-4">
                        🔍 Filters
                    </h3>
                    <p className="text-sm text-slate-600 dark:text-slate-400 mb-4">
                        Filter audit events by various criteria
                    </p>

                    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
                        <div>
                            <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">Search</label>
                            <input
                                type="text"
                                placeholder="Search messages..."
                                value={searchTerm}
                                onChange={(e) => setSearchTerm(e.target.value)}
                                className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm bg-white dark:bg-slate-900 text-slate-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                            />
                        </div>

                        <div>
                            <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">Event Type</label>
                            <select
                                value={eventTypeFilter}
                                onChange={(e) => setEventTypeFilter(e.target.value)}
                                className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm bg-white dark:bg-slate-900 text-slate-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                            >
                                <option value="">All types</option>
                                <option value="login_success">Login Success</option>
                                <option value="login_failure">Login Failure</option>
                                <option value="logout">Logout</option>
                                <option value="user_create">User Create</option>
                                <option value="user_update">User Update</option>
                                <option value="user_delete">User Delete</option>
                                <option value="config_change">Config Change</option>
                                <option value="api_call">API Call</option>
                            </select>
                        </div>

                        <div>
                            <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">Severity</label>
                            <select
                                value={severityFilter}
                                onChange={(e) => setSeverityFilter(e.target.value)}
                                className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm bg-white dark:bg-slate-900 text-slate-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                            >
                                <option value="">All severities</option>
                                <option value="info">Info</option>
                                <option value="warning">Warning</option>
                                <option value="error">Error</option>
                                <option value="critical">Critical</option>
                            </select>
                        </div>

                        <div>
                            <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">Resource</label>
                            <select
                                value={resourceFilter}
                                onChange={(e) => setResourceFilter(e.target.value)}
                                className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm bg-white dark:bg-slate-900 text-slate-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                            >
                                <option value="">All resources</option>
                                <option value="auth">Authentication</option>
                                <option value="users">Users</option>
                                <option value="system_config">System Config</option>
                                <option value="api">API</option>
                            </select>
                        </div>

                        <div>
                            <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">Action</label>
                            <select
                                value={actionFilter}
                                onChange={(e) => setActionFilter(e.target.value)}
                                className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm bg-white dark:bg-slate-900 text-slate-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                            >
                                <option value="">All actions</option>
                                <option value="create">Create</option>
                                <option value="update">Update</option>
                                <option value="delete">Delete</option>
                                <option value="login">Login</option>
                                <option value="logout">Logout</option>
                                <option value="view">View</option>
                            </select>
                        </div>

                        <div>
                            <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">User</label>
                            <input
                                type="text"
                                placeholder="User ID or name..."
                                value={userFilter}
                                onChange={(e) => setUserFilter(e.target.value)}
                                className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm bg-white dark:bg-slate-900 text-slate-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                            />
                        </div>
                    </div>

                    <div className="flex gap-2 mt-4">
                        <button
                            onClick={clearFilters}
                            className="px-4 py-2 border border-slate-300 dark:border-slate-600 rounded-md text-sm font-medium text-slate-700 dark:text-slate-300 hover:bg-slate-50 dark:hover:bg-slate-700"
                        >
                            Clear Filters
                        </button>
                    </div>
                </div>
            </div>

            {/* Results */}
            <div className="bg-white dark:bg-slate-950 rounded-lg shadow-md overflow-hidden">
                <div className="px-4 py-5 sm:p-6">
                    <h3 className="text-lg leading-6 font-medium text-slate-900 dark:text-white mb-2">Audit Events</h3>
                    <p className="text-sm text-slate-600 dark:text-slate-400 mb-4">
                        {total} total events found
                    </p>

                    {/* Top Pagination */}
                    <div className="flex items-center justify-between mb-4 px-4 py-3 bg-slate-50 dark:bg-slate-900 rounded-md">
                        <div className="text-sm text-slate-700 dark:text-slate-300">
                            Showing {page * limit + 1} to {Math.min((page + 1) * limit, total)} of {total} events
                        </div>
                        <div className="flex gap-2">
                            <button
                                onClick={() => setPage(Math.max(0, page - 1))}
                                disabled={page === 0}
                                className="px-3 py-1 border border-slate-300 dark:border-slate-600 rounded text-sm disabled:opacity-50 disabled:cursor-not-allowed hover:bg-slate-50 dark:hover:bg-slate-600 text-slate-900 dark:text-white"
                            >
                                Previous
                            </button>
                            <button
                                onClick={() => setPage(page + 1)}
                                disabled={(page + 1) * limit >= total}
                                className="px-3 py-1 border border-slate-300 dark:border-slate-600 rounded text-sm disabled:opacity-50 disabled:cursor-not-allowed hover:bg-slate-50 dark:hover:bg-slate-600 text-slate-900 dark:text-white"
                            >
                                Next
                            </button>
                        </div>
                    </div>

                    {loading ? (
                        <div className="flex items-center justify-center py-8">
                            <div className="flex justify-center items-center space-x-2">
                                <div className="animate-spin rounded-full h-6 w-6 border-b-2 border-blue-600"></div>
                                <span className="text-slate-600 dark:text-slate-400">Loading audit events...</span>
                            </div>
                        </div>
                    ) : events.length === 0 ? (
                        <div className="text-center py-8 text-slate-600 dark:text-slate-400">
                            No audit events found matching the current filters.
                        </div>
                    ) : (
                        <>
                            <div className="overflow-x-auto">
                                <table className="min-w-full divide-y divide-slate-200 dark:divide-slate-700">
                                    <thead className="bg-slate-50 dark:bg-slate-900">
                                        <tr>
                                            <th className="px-6 py-3 text-left text-xs font-medium text-slate-700 dark:text-slate-300 uppercase tracking-wide">
                                                Timestamp
                                            </th>
                                            <th className="px-6 py-3 text-left text-xs font-medium text-slate-700 dark:text-slate-300 uppercase tracking-wide">
                                                User
                                            </th>
                                            <th className="px-6 py-3 text-left text-xs font-medium text-slate-700 dark:text-slate-300 uppercase tracking-wide">
                                                Event Type
                                            </th>
                                            <th className="px-6 py-3 text-left text-xs font-medium text-slate-700 dark:text-slate-300 uppercase tracking-wide">
                                                Severity
                                            </th>
                                            <th className="px-6 py-3 text-left text-xs font-medium text-slate-700 dark:text-slate-300 uppercase tracking-wide">
                                                Resource
                                            </th>
                                            <th className="px-6 py-3 text-left text-xs font-medium text-slate-700 dark:text-slate-300 uppercase tracking-wide">
                                                Action
                                            </th>
                                            <th className="px-6 py-3 text-left text-xs font-medium text-slate-700 dark:text-slate-300 uppercase tracking-wide">
                                                Message
                                            </th>
                                        </tr>
                                    </thead>
                                    <tbody className="bg-white dark:bg-slate-950 divide-y divide-slate-200 dark:divide-slate-700">
                                        {events.map((event) => (
                                            <tr
                                                key={event.id}
                                                onClick={() => setSelectedEvent(event)}
                                                className="hover:bg-slate-50 dark:hover:bg-slate-900 transition-colors cursor-pointer"
                                            >
                                                <td className="px-6 py-4 whitespace-nowrap text-sm font-mono text-slate-900 dark:text-white">
                                                    {format(new Date(event.timestamp), 'yyyy-MM-dd HH:mm:ss')}
                                                </td>
                                                <td className="px-6 py-4 whitespace-nowrap text-sm text-slate-900 dark:text-white">
                                                    {event.user_name || (event.user_id ? (
                                                        <code className="text-xs bg-slate-100 dark:bg-slate-700 px-2 py-1 rounded text-slate-900 dark:text-white">
                                                            {event.user_id.substring(0, 8)}...
                                                        </code>
                                                    ) : (
                                                        <span className="text-slate-500 dark:text-slate-400">System</span>
                                                    ))}
                                                </td>
                                                <td className="px-6 py-4 whitespace-nowrap">
                                                    <span className={`inline-flex px-2 py-1 text-xs font-semibold rounded-full ${getEventTypeBadgeClass(event.event_type)}`}>
                                                        {event.event_type.replace('_', ' ')}
                                                    </span>
                                                </td>
                                                <td className="px-6 py-4 whitespace-nowrap">
                                                    <span className={`inline-flex px-2 py-1 text-xs font-semibold rounded-full ${getSeverityBadgeClass(event.severity)}`}>
                                                        {event.severity}
                                                    </span>
                                                </td>
                                                <td className="px-6 py-4 whitespace-nowrap text-sm font-mono text-slate-900 dark:text-white">
                                                    {event.resource}
                                                </td>
                                                <td className="px-6 py-4 whitespace-nowrap text-sm font-mono text-slate-900 dark:text-white">
                                                    {event.action}
                                                </td>
                                                <td className="px-6 py-4 text-sm text-slate-900 dark:text-white max-w-xs">
                                                    <div className="truncate" title={event.message}>
                                                        {event.message}
                                                    </div>
                                                </td>
                                            </tr>
                                        ))}
                                    </tbody>
                                </table>
                            </div>

                            {/* Pagination */}
                            <div className="flex items-center justify-center mt-4 px-6 py-3 bg-slate-50 dark:bg-slate-900">
                                <div className="flex gap-2">
                                    <button
                                        onClick={() => setPage(Math.max(0, page - 1))}
                                        disabled={page === 0}
                                        className="px-3 py-1 border border-slate-300 dark:border-slate-600 rounded text-sm disabled:opacity-50 disabled:cursor-not-allowed hover:bg-slate-50 dark:hover:bg-slate-600 text-slate-900 dark:text-white"
                                    >
                                        Previous
                                    </button>
                                    <button
                                        onClick={() => setPage(page + 1)}
                                        disabled={(page + 1) * limit >= total}
                                        className="px-3 py-1 border border-slate-300 dark:border-slate-600 rounded text-sm disabled:opacity-50 disabled:cursor-not-allowed hover:bg-slate-50 dark:hover:bg-slate-600 text-slate-900 dark:text-white"
                                    >
                                        Next
                                    </button>
                                </div>
                            </div>
                        </>
                    )}
                </div>
            </div>

            {/* Event Details Drawer */}
            <Drawer
                isOpen={!!selectedEvent}
                title="Audit Event Details"
                description={selectedEvent ? `Event ID: ${selectedEvent.id}` : ''}
                onClose={() => setSelectedEvent(null)}
                width="lg"
            >
                {selectedEvent && (
                    <div className="space-y-6">
                        <div className="grid grid-cols-1 gap-6 sm:grid-cols-2">
                            <div>
                                <dt className="text-sm font-medium text-slate-500 dark:text-slate-400">Timestamp</dt>
                                <dd className="mt-1 text-sm text-slate-900 dark:text-white">{format(new Date(selectedEvent.timestamp), 'yyyy-MM-dd HH:mm:ss')}</dd>
                            </div>
                            <div>
                                <dt className="text-sm font-medium text-slate-500 dark:text-slate-400">User</dt>
                                <dd className="mt-1 text-sm text-slate-900 dark:text-white">
                                    {selectedEvent.user_name || (selectedEvent.user_id ? (
                                        <code className="text-xs bg-slate-100 dark:bg-slate-700 px-2 py-1 rounded">
                                            {selectedEvent.user_id}
                                        </code>
                                    ) : (
                                        'System'
                                    ))}
                                </dd>
                            </div>
                            <div>
                                <dt className="text-sm font-medium text-slate-500 dark:text-slate-400">Event Type</dt>
                                <dd className="mt-1 text-sm text-slate-900 dark:text-white">{selectedEvent.event_type.replace('_', ' ')}</dd>
                            </div>
                            <div>
                                <dt className="text-sm font-medium text-slate-500 dark:text-slate-400">Severity</dt>
                                <dd className="mt-1 text-sm text-slate-900 dark:text-white">{selectedEvent.severity}</dd>
                            </div>
                            <div>
                                <dt className="text-sm font-medium text-slate-500 dark:text-slate-400">Resource</dt>
                                <dd className="mt-1 text-sm text-slate-900 dark:text-white font-mono">{selectedEvent.resource}</dd>
                            </div>
                            <div>
                                <dt className="text-sm font-medium text-slate-500 dark:text-slate-400">Action</dt>
                                <dd className="mt-1 text-sm text-slate-900 dark:text-white font-mono">{selectedEvent.action}</dd>
                            </div>
                            <div className="sm:col-span-2">
                                <dt className="text-sm font-medium text-slate-500 dark:text-slate-400">Message</dt>
                                <dd className="mt-1 text-sm text-slate-900 dark:text-white">{selectedEvent.message}</dd>
                            </div>
                            <div>
                                <dt className="text-sm font-medium text-slate-500 dark:text-slate-400">User Agent</dt>
                                <dd className="mt-1 text-sm text-slate-900 dark:text-white break-all">{selectedEvent.user_agent || 'N/A'}</dd>
                            </div>
                        </div>
                        {selectedEvent.details && Object.keys(selectedEvent.details).length > 0 && (
                            <div>
                                <dt className="text-sm font-medium text-slate-500 dark:text-slate-400">Details</dt>
                                <dd className="mt-1">
                                    <pre className="bg-slate-100 dark:bg-slate-700 p-3 rounded text-xs overflow-x-auto whitespace-pre-wrap">
                                        {JSON.stringify(selectedEvent.details, null, 2)}
                                    </pre>
                                </dd>
                            </div>
                        )}

                        {/* Close Button */}
                        <div className="flex justify-end pt-4 border-t border-slate-200 dark:border-slate-700">
                            <button
                                onClick={() => setSelectedEvent(null)}
                                className="px-4 py-2 bg-slate-600 text-white rounded-md text-sm font-medium hover:bg-slate-700 focus:outline-none focus:ring-2 focus:ring-slate-500 focus:ring-offset-2"
                            >
                                Close
                            </button>
                        </div>
                    </div>
                )}
            </Drawer>
        </div>
    )
}

export default AuditLogsPage