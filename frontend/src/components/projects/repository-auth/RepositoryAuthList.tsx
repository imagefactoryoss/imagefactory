import { repositoryAuthClient } from '@/api/repositoryAuthClient'
import Drawer from '@/components/ui/Drawer'
import { RepositoryAuth, RepositoryAuthType } from '@/types/repositoryAuth'
import { Copy } from 'lucide-react'
import React, { useEffect, useState } from 'react'
import toast from 'react-hot-toast'
import RepositoryAuthModal from './RepositoryAuthModal'

interface RepositoryAuthListProps {
    projectId: string
    repoUrl?: string
    refreshKey?: number
}

const RepositoryAuthList: React.FC<RepositoryAuthListProps> = ({ projectId, repoUrl, refreshKey }) => {
    const [auths, setAuths] = useState<RepositoryAuth[]>([])
    const [loading, setLoading] = useState(true)
    const [error, setError] = useState<string | null>(null)
    const [testingConnection, setTestingConnection] = useState<string | null>(null)
    const [showTestDrawer, setShowTestDrawer] = useState(false)
    const [testProgress, setTestProgress] = useState<'initializing' | 'connecting' | 'receiving' | 'completed' | 'failed' | null>(null)
    const [testError, setTestError] = useState<string | null>(null)
    const [testDetails, setTestDetails] = useState<Record<string, any> | null>(null)
    const [selectedAuth, setSelectedAuth] = useState<RepositoryAuth | null>(null)
    const [testMode, setTestMode] = useState<'quick' | 'full'>('quick')
    const [fullTestRepoUrl, setFullTestRepoUrl] = useState('')
    const [fullTestRunning, setFullTestRunning] = useState(false)
    const [editAuth, setEditAuth] = useState<RepositoryAuth | null>(null)

    useEffect(() => {
        loadRepositoryAuths()
    }, [projectId, refreshKey])

    useEffect(() => {
        if (!showTestDrawer || !repoUrl || fullTestRepoUrl.trim().length > 0) return
        setFullTestRepoUrl(repoUrl)
    }, [showTestDrawer, repoUrl, fullTestRepoUrl])

    const loadRepositoryAuths = async () => {
        try {
            setLoading(true)
            setError(null)
            const response = await repositoryAuthClient.getRepositoryAuths(projectId)
            setAuths(response.repository_auths)
        } catch (err) {
            const errorMessage = err instanceof Error ? err.message : 'Failed to load repository authentications'
            setError(errorMessage)
            toast.error(errorMessage)
        } finally {
            setLoading(false)
        }
    }

    // Handle test connection
    const handleTestConnection = async (auth: RepositoryAuth) => {
        setSelectedAuth(auth)
        setTestingConnection(auth.id)
        setTestError(null)
        setTestDetails(null)
        setShowTestDrawer(true)
        setTestMode('quick')
        if (repoUrl && fullTestRepoUrl.trim().length === 0) {
            setFullTestRepoUrl(repoUrl)
        }
        setTestProgress('initializing')

        try {
            // Simulate initialization delay
            await new Promise(resolve => setTimeout(resolve, 500))

            setTestProgress('connecting')

            const result = await repositoryAuthClient.testRepositoryAuth(projectId, auth.id, { full_test: false })

            setTestProgress('receiving')

            if (result.success) {
                setTestProgress('completed')
                setTestError(null)
                setTestDetails(result.details || null)
                toast.success('Authentication validation completed (no repo connection).')
            } else {
                setTestProgress('failed')
                setTestError(result.message || 'Authentication test failed')
                setTestDetails(result.details || null)
            }
        } catch (err) {
            setTestProgress('failed')
            setTestError(err instanceof Error ? err.message : 'Failed to test authentication')
        } finally {
            setTestingConnection(null)
        }
    }

    const handleFullTestConnection = async () => {
        if (!selectedAuth) return
        if (!fullTestRepoUrl.trim()) {
            setTestError('Repository URL is required for a full connection test.')
            return
        }

        setFullTestRunning(true)
        setTestError(null)
        setTestDetails(null)
        setTestMode('full')
        setTestProgress('initializing')

        try {
            await new Promise(resolve => setTimeout(resolve, 300))
            setTestProgress('connecting')

            const result = await repositoryAuthClient.testRepositoryAuth(projectId, selectedAuth.id, {
                full_test: true,
                repo_url: fullTestRepoUrl.trim(),
            })

            setTestProgress('receiving')

            if (result.success) {
                setTestProgress('completed')
                setTestError(null)
                setTestDetails(result.details || null)
                toast.success('Full repository connection test successful!')
            } else {
                setTestProgress('failed')
                setTestError(result.message || 'Full authentication test failed')
                setTestDetails(result.details || null)
            }
        } catch (err) {
            setTestProgress('failed')
            setTestError(err instanceof Error ? err.message : 'Failed to run full connection test')
        } finally {
            setFullTestRunning(false)
        }
    }

    // Handle close test drawer
    const handleCloseTestDrawer = () => {
        setShowTestDrawer(false)
        setTestProgress(null)
        setTestError(null)
        setTestDetails(null)
        setSelectedAuth(null)
        setTestMode('quick')
        setFullTestRepoUrl('')
    }

    const handleOpenEdit = (auth: RepositoryAuth) => {
        setEditAuth(auth)
    }

    const handleCloseEdit = () => {
        setEditAuth(null)
    }

    const getAuthTypeLabel = (authType: RepositoryAuthType): string => {
        switch (authType) {
            case RepositoryAuthType.SSH_KEY:
                return 'SSH Key'
            case RepositoryAuthType.TOKEN:
                return 'Token'
            case RepositoryAuthType.BASIC_AUTH:
                return 'Basic Auth'
            case RepositoryAuthType.OAUTH:
                return 'OAuth'
            default:
                return 'Unknown'
        }
    }

    const formatDetailLabel = (key: string): string => {
        const cleaned = key.replace(/_/g, ' ')
        return cleaned.charAt(0).toUpperCase() + cleaned.slice(1)
    }

    const formatDetailValue = (value: any, key?: string): string => {
        if (value === null || value === undefined) return '-'
        if (typeof value === 'number' && Number.isFinite(value)) {
            if (key === 'duration_ms') return `${value} ms`
            if (key === 'timeout_s') return `${value} s`
            return `${value}`
        }
        if (typeof value === 'object') {
            return JSON.stringify(value)
        }
        return String(value)
    }

    const getAuthTypeColor = (authType: RepositoryAuthType): string => {
        switch (authType) {
            case RepositoryAuthType.SSH_KEY:
                return 'bg-purple-100 text-purple-800 dark:bg-purple-900 dark:text-purple-200'
            case RepositoryAuthType.TOKEN:
                return 'bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200'
            case RepositoryAuthType.BASIC_AUTH:
                return 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200'
            case RepositoryAuthType.OAUTH:
                return 'bg-orange-100 text-orange-800 dark:bg-orange-900 dark:text-orange-200'
            default:
                return 'bg-gray-100 text-gray-800 dark:bg-gray-900 dark:text-gray-200'
        }
    }

    const formatDate = (dateString: string): string => {
        return new Date(dateString).toLocaleDateString()
    }

    const handleCopyId = async (id: string) => {
        try {
            await navigator.clipboard.writeText(id)
            toast.success('Repository auth UUID copied')
        } catch {
            toast.error('Failed to copy UUID')
        }
    }

    if (loading) {
        return (
            <div className="flex justify-center items-center py-8">
                <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-500"></div>
            </div>
        )
    }

    if (error) {
        return (
            <div className="text-center py-8">
                <div className="text-red-600 dark:text-red-400 text-sm">
                    {error}
                </div>
                <button
                    onClick={loadRepositoryAuths}
                    className="mt-2 px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 text-sm"
                >
                    Retry
                </button>
            </div>
        )
    }

    if (auths.length === 0) {
        return (
            <div className="text-center py-12">
                <div className="text-slate-500 dark:text-slate-400">
                    <svg className="mx-auto h-12 w-12 text-slate-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
                    </svg>
                    <h3 className="mt-2 text-sm font-medium text-slate-900 dark:text-white">
                        No Repository Authentications
                    </h3>
                    <p className="mt-1 text-sm text-slate-500 dark:text-slate-400">
                        Get started by creating your first repository authentication method.
                    </p>
                </div>
            </div>
        )
    }

    return (
        <div className="space-y-4">
            {auths.map((auth) => (
                <div
                    key={auth.id}
                    className="bg-white dark:bg-slate-800 rounded-lg border border-slate-200 dark:border-slate-700 p-6 shadow-sm"
                >
                    <div className="flex items-center justify-between">
                        <div className="flex items-center space-x-4">
                            <div>
                                <h3 className="text-lg font-semibold text-slate-900 dark:text-white">
                                    {auth.name}
                                </h3>
                                {auth.description && (
                                    <p className="text-sm text-slate-600 dark:text-slate-400 mt-1">
                                        {auth.description}
                                    </p>
                                )}
                                <div className="text-sm text-slate-600 dark:text-slate-400 mt-1 flex items-center gap-2">
                                    <span>UUID: <span className="font-mono text-xs break-all">{auth.id}</span></span>
                                    <button
                                        type="button"
                                        onClick={() => handleCopyId(auth.id)}
                                        className="inline-flex items-center justify-center rounded border border-slate-300 dark:border-slate-600 p-1 text-slate-600 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-700"
                                        title="Copy UUID"
                                        aria-label="Copy repository auth UUID"
                                    >
                                        <Copy className="h-3.5 w-3.5" />
                                    </button>
                                </div>
                            </div>
                        </div>
                        <div className="flex items-center space-x-3">
                            <span className={`inline-flex items-center px-3 py-1 rounded-full text-sm font-medium ${getAuthTypeColor(auth.auth_type)}`}>
                                {getAuthTypeLabel(auth.auth_type)}
                            </span>
                            <span className={`inline-flex items-center px-3 py-1 rounded-full text-sm font-medium ${auth.is_active
                                ? 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200'
                                : 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200'
                                }`}>
                                {auth.is_active ? 'Active' : 'Inactive'}
                            </span>
                            <button
                                onClick={() => handleTestConnection(auth)}
                                disabled={testingConnection === auth.id}
                                className="inline-flex items-center px-3 py-1 border border-blue-300 dark:border-blue-600 rounded-md text-sm font-medium text-blue-700 dark:text-blue-300 bg-white dark:bg-slate-800 hover:bg-blue-50 dark:hover:bg-blue-900/20 disabled:opacity-50 disabled:cursor-not-allowed"
                                title="Test Connection"
                            >
                                {testingConnection === auth.id ? (
                                    <div className="flex items-center">
                                        <div className="animate-spin rounded-full h-3 w-3 border-b border-blue-600 dark:border-blue-400 mr-1"></div>
                                        Testing...
                                    </div>
                                ) : (
                                    <div className="flex items-center">
                                        <svg className="h-4 w-4 mr-1" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
                                        </svg>
                                        Test
                                    </div>
                                )}
                            </button>
                            <button
                                onClick={() => handleOpenEdit(auth)}
                                className="inline-flex items-center px-3 py-1 border border-slate-300 dark:border-slate-600 rounded-md text-sm font-medium text-slate-700 dark:text-slate-200 bg-white dark:bg-slate-800 hover:bg-slate-50 dark:hover:bg-slate-700"
                                title="Edit Authentication"
                            >
                                <svg className="h-4 w-4 mr-1" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M11 5h2m-1-1v2m-6.5 10.5l8-8 2 2-8 8H4.5v-2.5z" />
                                </svg>
                                Edit
                            </button>
                        </div>
                    </div>

                    <div className="mt-4 grid grid-cols-1 md:grid-cols-3 gap-4 text-sm">
                        <div>
                            <span className="text-slate-500 dark:text-slate-400">Created:</span>
                            <span className="ml-2 text-slate-900 dark:text-white">
                                {formatDate(auth.created_at)}
                            </span>
                        </div>
                        <div>
                            <span className="text-slate-500 dark:text-slate-400">Last Updated:</span>
                            <span className="ml-2 text-slate-900 dark:text-white">
                                {formatDate(auth.updated_at)}
                            </span>
                        </div>
                        <div>
                            <span className="text-slate-500 dark:text-slate-400">Version:</span>
                            <span className="ml-2 text-slate-900 dark:text-white">
                                {auth.version}
                            </span>
                        </div>
                    </div>
                </div>
            ))}

            <RepositoryAuthModal
                projectId={projectId}
                isOpen={Boolean(editAuth)}
                authToEdit={editAuth || undefined}
                onClose={handleCloseEdit}
                onSuccess={() => {
                    handleCloseEdit()
                    loadRepositoryAuths()
                }}
            />

            {/* Test Connection Drawer */}
            <Drawer
                isOpen={showTestDrawer}
                onClose={handleCloseTestDrawer}
                title="Testing Repository Authentication"
                description="Testing connectivity and validation of repository authentication"
            >
                <div className="space-y-6">
                    {/* Progress Steps */}
                    <div className="space-y-4">
                        <h3 className="text-sm font-medium text-gray-900 dark:text-white mb-3">Test Progress</h3>

                        <div className="space-y-3">
                            <div className={`flex items-center space-x-3 ${testProgress === 'initializing' ? 'text-blue-600 dark:text-blue-400' : 'text-gray-400 dark:text-gray-500'}`}>
                                <div className={`w-6 h-6 rounded-full flex items-center justify-center ${testProgress === 'initializing' ? 'bg-blue-100 dark:bg-blue-900' : 'bg-gray-100 dark:bg-gray-800'}`}>
                                    {testProgress === 'initializing' && (
                                        <svg className="h-4 w-4 animate-spin" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
                                        </svg>
                                    )}
                                    {testProgress === 'connecting' && (
                                        <svg className="h-4 w-4 animate-spin" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
                                        </svg>
                                    )}
                                    {testProgress === 'receiving' && (
                                        <svg className="h-4 w-4 animate-spin" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
                                        </svg>
                                    )}
                                    {testProgress === 'completed' && (
                                        <svg className="h-4 w-4 text-green-600 dark:text-green-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
                                        </svg>
                                    )}
                                    {testProgress === 'failed' && (
                                        <svg className="h-4 w-4 text-red-600 dark:text-red-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                                        </svg>
                                    )}
                                </div>
                                <span className="text-sm">
                                    {testProgress === 'initializing' && (testMode === 'full' ? 'Preparing full connection test...' : 'Initializing test...')}
                                    {testProgress === 'connecting' && (testMode === 'full' ? 'Connecting to repository...' : 'Validating authentication...')}
                                    {testProgress === 'receiving' && 'Processing results...'}
                                    {testProgress === 'completed' && (testMode === 'full'
                                        ? 'Full connection test completed'
                                        : 'Validation completed (no repo connection)'
                                    )}
                                    {testProgress === 'failed' && 'Authentication test failed'}
                                </span>
                            </div>
                    </div>

                    {/* Test Details */}
                    {testDetails && Object.keys(testDetails).length > 0 && (
                        <div className="bg-slate-50 dark:bg-slate-900/40 border border-slate-200 dark:border-slate-700 rounded-lg p-4">
                            <h4 className="text-sm font-medium text-slate-900 dark:text-white mb-3">
                                Test Details
                            </h4>
                            <div className="grid grid-cols-1 md:grid-cols-2 gap-2 text-xs">
                                {Object.entries(testDetails).map(([key, value]) => (
                                    <div key={key} className="flex items-center justify-between gap-3">
                                        <span className="text-slate-500 dark:text-slate-400">
                                            {formatDetailLabel(key)}
                                        </span>
                                        <span className="text-slate-900 dark:text-slate-100 text-right break-all">
                                            {formatDetailValue(value, key)}
                                        </span>
                                    </div>
                                ))}
                            </div>
                        </div>
                    )}

                    {/* Error Display */}
                    {testError && (
                        <div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg p-4">
                            <div className="flex">
                                    <svg className="h-5 w-5 text-red-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                                    </svg>
                                    <div className="ml-3">
                                        <h3 className="text-sm font-medium text-red-800 dark:text-red-200">
                                            Test Failed
                                        </h3>
                                        <p className="text-sm text-red-700 dark:text-red-300 mt-1">
                                            {testError}
                                        </p>
                                    </div>
                                </div>
                            </div>
                        )}

                        {/* Auth Info */}
                        {selectedAuth && (
                            <div className="bg-gray-50 dark:bg-gray-800 rounded-lg p-4">
                                <h4 className="text-sm font-medium text-gray-900 dark:text-white mb-2">Authentication Details</h4>
                                <dl className="space-y-2 text-sm">
                                    <div className="flex justify-between">
                                        <dt className="text-gray-500 dark:text-gray-400">Name:</dt>
                                        <dd className="text-gray-900 dark:text-white font-medium">{selectedAuth.name}</dd>
                                    </div>
                                    <div className="flex justify-between">
                                        <dt className="text-gray-500 dark:text-gray-400">Type:</dt>
                                        <dd className="text-gray-900 dark:text-white font-medium">{getAuthTypeLabel(selectedAuth.auth_type)}</dd>
                                    </div>
                                    <div className="flex justify-between gap-3">
                                        <dt className="text-gray-500 dark:text-gray-400">UUID:</dt>
                                        <dd className="text-gray-900 dark:text-white font-mono text-xs break-all text-right flex items-center justify-end gap-2">
                                            <span>{selectedAuth.id}</span>
                                            <button
                                                type="button"
                                                onClick={() => handleCopyId(selectedAuth.id)}
                                                className="inline-flex items-center justify-center rounded border border-slate-300 dark:border-slate-600 p-1 text-slate-600 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-700"
                                                title="Copy UUID"
                                                aria-label="Copy selected repository auth UUID"
                                            >
                                                <Copy className="h-3.5 w-3.5" />
                                            </button>
                                        </dd>
                                    </div>
                                    <div className="flex justify-between">
                                        <dt className="text-gray-500 dark:text-gray-400">Status:</dt>
                                        <dd className="text-gray-900 dark:text-white font-medium">
                                            <span className={`inline-flex items-center px-2 py-1 rounded-full text-xs ${selectedAuth.is_active
                                                ? 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200'
                                                : 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200'
                                                }`}>
                                                {selectedAuth.is_active ? 'Active' : 'Inactive'}
                                            </span>
                                        </dd>
                                    </div>
                                </dl>
                            </div>
                        )}

                        {selectedAuth && (
                            <div className="bg-slate-50 dark:bg-slate-800 rounded-lg p-4 space-y-3">
                                <div>
                                    <h4 className="text-sm font-medium text-slate-900 dark:text-white">Full Connection Test</h4>
                                    <p className="text-xs text-slate-500 dark:text-slate-400">
                                        Runs a real repository connection check (git ls-remote). Requires a repository URL.
                                    </p>
                                </div>
                                <input
                                    type="text"
                                    value={fullTestRepoUrl}
                                    onChange={(e) => setFullTestRepoUrl(e.target.value)}
                                    placeholder="https://github.com/org/repo.git or git@github.com:org/repo.git"
                                    className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md text-sm bg-white dark:bg-slate-900 text-slate-900 dark:text-white"
                                />
                                <div className="flex justify-end">
                                    <button
                                        onClick={handleFullTestConnection}
                                        disabled={fullTestRunning}
                                        className="inline-flex items-center px-3 py-2 rounded-md text-sm font-medium bg-slate-900 text-white hover:bg-slate-800 disabled:opacity-50"
                                    >
                                        {fullTestRunning ? 'Running Full Test...' : 'Run Full Test'}
                                    </button>
                                </div>
                            </div>
                        )}
                    </div>
                </div>
            </Drawer>
        </div>
    )
}

export default RepositoryAuthList
