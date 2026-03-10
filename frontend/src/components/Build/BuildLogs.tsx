import React, { useEffect, useRef, useState } from 'react';
import { BuildClient, LogLevel, LogMessage } from '../../api/buildClient';

interface BuildLogsProps {
    client: BuildClient;
    buildId: string;
}

export const BuildLogs: React.FC<BuildLogsProps> = ({ client, buildId }) => {
    const [logs, setLogs] = useState<LogMessage[]>([]);
    const [isConnected, setIsConnected] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const [autoScroll, setAutoScroll] = useState(true);
    const [filter, setFilter] = useState<LogLevel | 'ALL'>('ALL');
    const [collapsedTaskRuns, setCollapsedTaskRuns] = useState<Record<string, boolean>>({})
    const wsRef = useRef<WebSocket | null>(null);
    const logContainerRef = useRef<HTMLDivElement>(null);

    // Connect to WebSocket
    useEffect(() => {
        const connectWebSocket = async () => {
            try {
                setError(null);
                const ws = await client.streamBuildLogs(buildId);
                wsRef.current = ws;
                setIsConnected(true);

                ws.onmessage = (event) => {
                    try {
                        const message = JSON.parse(event.data) as LogMessage;
                        setLogs((prev) => [...prev, message]);

                        if (autoScroll && logContainerRef.current) {
                            logContainerRef.current.scrollTop = logContainerRef.current.scrollHeight;
                        }
                    } catch (err) {
                        console.error('Failed to parse log message:', err);
                    }
                };

                ws.onerror = () => {
                    setError('WebSocket error occurred');
                    setIsConnected(false);
                };

                ws.onclose = () => {
                    setIsConnected(false);
                };
            } catch (err) {
                setError(err instanceof Error ? err.message : 'Failed to connect to logs');
                setIsConnected(false);
            }
        };

        connectWebSocket();

        return () => {
            if (wsRef.current) {
                wsRef.current.close();
            }
        };
    }, [client, buildId, autoScroll]);

    const logLevelColor: Record<LogLevel, string> = {
        DEBUG: 'text-gray-500',
        INFO: 'text-blue-600',
        WARN: 'text-yellow-600',
        ERROR: 'text-red-600',
    };

    const logLevelBgColor: Record<LogLevel, string> = {
        DEBUG: 'bg-gray-50',
        INFO: 'bg-blue-50',
        WARN: 'bg-yellow-50',
        ERROR: 'bg-red-50',
    };

    const filteredLogs = filter === 'ALL' ? logs : logs.filter((log) => log.level === filter);

    const handleCopyLogs = () => {
        const text = filteredLogs.map((log) => `[${log.level}] ${log.message}`).join('\n');
        navigator.clipboard.writeText(text);
    };

    const handleDownloadLogs = () => {
        const text = filteredLogs.map((log) => `[${log.level}] ${log.message}`).join('\n');
        const element = document.createElement('a');
        element.setAttribute('href', `data:text/plain;charset=utf-8,${encodeURIComponent(text)}`);
        element.setAttribute('download', `build-${buildId}-logs.txt`);
        element.style.display = 'none';
        document.body.appendChild(element);
        element.click();
        document.body.removeChild(element);
    };

    return (
        <div className="space-y-4">
            {/* Controls */}
            <div className="flex justify-between items-center">
                <div className="flex items-center space-x-4">
                    <div>
                        <label className="block text-sm font-medium text-gray-700">Filter by Level</label>
                        <select
                            value={filter}
                            onChange={(e) => setFilter(e.target.value as LogLevel | 'ALL')}
                            className="mt-1 px-3 py-2 border border-gray-300 rounded-md text-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500"
                        >
                            <option value="ALL">All Levels</option>
                            <option value="DEBUG">DEBUG</option>
                            <option value="INFO">INFO</option>
                            <option value="WARN">WARN</option>
                            <option value="ERROR">ERROR</option>
                        </select>
                    </div>

                    <div className="flex items-center space-x-2">
                        <input
                            type="checkbox"
                            id="autoScroll"
                            checked={autoScroll}
                            onChange={(e) => setAutoScroll(e.target.checked)}
                            className="rounded"
                        />
                        <label htmlFor="autoScroll" className="text-sm font-medium text-gray-700">
                            Auto-scroll
                        </label>
                    </div>

                    <div className="flex items-center space-x-2">
                        <div
                            className={`w-3 h-3 rounded-full ${isConnected ? 'bg-green-500' : 'bg-red-500'}`}
                        />
                        <span className="text-sm font-medium text-gray-700">
                            {isConnected ? 'Connected' : 'Disconnected'}
                        </span>
                    </div>
                </div>

                <div className="flex space-x-2">
                    <button
                        onClick={handleCopyLogs}
                        className="text-blue-600 hover:text-blue-800 text-sm font-medium"
                    >
                        Copy
                    </button>
                    <button
                        onClick={handleDownloadLogs}
                        className="text-blue-600 hover:text-blue-800 text-sm font-medium"
                    >
                        Download
                    </button>
                </div>
            </div>

            {/* Error Alert */}
            {error && (
                <div className="bg-red-50 border border-red-200 rounded-lg p-3">
                    <p className="text-red-800 text-sm">{error}</p>
                </div>
            )}

            {/* Logs Container */}
            <div
                ref={logContainerRef}
                className="bg-gray-900 rounded-lg p-4 font-mono text-sm overflow-y-auto max-h-96 border border-gray-200"
            >
                {filteredLogs.length === 0 ? (
                    <div className="text-gray-500">No logs to display</div>
                ) : (() => {
                    const hasTekton = filteredLogs.some(l => l.metadata && String(l.metadata.source) === 'tekton')
                    if (!hasTekton) {
                        return (
                            <div className="space-y-1">
                                {filteredLogs.map((log, index) => (
                                    <div
                                        key={index}
                                        className={`${logLevelBgColor[log.level] || 'bg-gray-50'} p-2 rounded`}
                                    >
                                        <div className="flex items-start gap-3">
                                            <div className="font-semibold mr-2">[{log.level}]</div>
                                            <div className="flex-1">
                                                <div className="text-gray-300">{log.message}</div>
                                            </div>
                                            {log.timestamp && (
                                                <div className="text-gray-400 ml-2 text-xs">{new Date(log.timestamp).toLocaleTimeString()}</div>
                                            )}
                                        </div>
                                    </div>
                                ))}
                            </div>
                        )
                    }

                    // Group by task_run -> step
                    const taskMap: Map<string, Map<string, LogMessage[]>> = new Map()
                    filteredLogs.forEach((entry) => {
                        const task = (entry.metadata && (entry.metadata.task_run || entry.metadata.task)) ? String(entry.metadata.task_run || entry.metadata.task) : '__ungrouped__'
                        const step = (entry.metadata && entry.metadata.step) ? String(entry.metadata.step) : '__other__'
                        if (!taskMap.has(task)) taskMap.set(task, new Map())
                        const stepMap = taskMap.get(task) as Map<string, LogMessage[]>
                        if (!stepMap.has(step)) stepMap.set(step, [])
                            ; (stepMap.get(step) as LogMessage[]).push(entry)
                    })


                    return (
                        <div className="space-y-3">
                            {Array.from(taskMap.entries()).map(([taskRun, stepMap], tIdx) => {
                                const totalLines = Array.from(stepMap.values()).reduce((s, arr) => s + arr.length, 0)
                                const taskLabel = taskRun === '__ungrouped__' ? 'Other' : taskRun
                                const isCollapsed = !!collapsedTaskRuns[taskRun]
                                return (
                                    <div key={"task-" + tIdx} className="mb-2 border border-slate-800 rounded">
                                        <div className="flex items-center justify-between px-3 py-2 bg-slate-800">
                                            <div className="text-sm font-medium text-slate-200">TaskRun: {taskLabel} <span className="text-xs text-slate-400">({totalLines} lines)</span></div>
                                            <div>
                                                <button
                                                    className="text-xs text-slate-300 px-2 py-1 rounded bg-slate-700 hover:bg-slate-600"
                                                    onClick={() => setCollapsedTaskRuns(prev => ({ ...prev, [taskRun]: !prev[taskRun] }))}
                                                >
                                                    {isCollapsed ? 'Expand' : 'Collapse'}
                                                </button>
                                            </div>
                                        </div>
                                        {!isCollapsed && (
                                            <div className="px-3 py-2 space-y-2">
                                                {Array.from(stepMap.entries()).map(([stepName, entries], sIdx) => (
                                                    <div key={"step-" + sIdx}>
                                                        <div className="text-xs text-slate-400 mb-1">Step: {stepName}</div>
                                                        {entries.map((log, i) => (
                                                            <div key={i} className={`${logLevelBgColor[log.level] || 'bg-gray-50'} p-2 rounded mb-1`}>
                                                                <div className="flex items-start gap-3">
                                                                    <div className="font-semibold mr-2">[{log.level}]</div>
                                                                    <div className="flex-1 text-gray-300">{log.message}</div>
                                                                    {log.timestamp && (
                                                                        <div className="text-gray-400 ml-2 text-xs">{new Date(log.timestamp).toLocaleTimeString()}</div>
                                                                    )}
                                                                </div>
                                                            </div>
                                                        ))}
                                                    </div>
                                                ))}
                                            </div>
                                        )}
                                    </div>
                                )
                            })}
                        </div>
                    )
                })()}
            </div>

            {/* Stats */}
            <div className="text-sm text-gray-600">
                Showing {filteredLogs.length} of {logs.length} log entries
            </div>
        </div>
    );
};

export default BuildLogs;
