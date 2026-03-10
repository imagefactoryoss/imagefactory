import { useShowTokenExpirationWarning, useTokenExpiration } from '@/hooks/useTokenExpiration'
import { Clock } from 'lucide-react'
import React from 'react'

/**
 * Component to display token expiration warning
 * Shows when token will expire in less than 1 minute
 */
export const TokenExpirationWarning: React.FC = () => {
    const { shouldShowWarning, timeUntilExpiry } = useShowTokenExpirationWarning()
    const { manualRefresh, isRefreshing } = useTokenExpiration()

    if (!shouldShowWarning || timeUntilExpiry === null) return null

    const formatTime = (seconds: number) => {
        if (seconds <= 0) return '0s'
        if (seconds < 60) return `${Math.ceil(seconds)}s`
        const minutes = Math.floor(seconds / 60)
        const secs = seconds % 60
        return `${minutes}m ${secs}s`
    }

    return (
        <div className="fixed bottom-4 right-4 max-w-sm z-50 animate-in slide-in-from-bottom-5">
            <div className="bg-amber-50 dark:bg-amber-950 border-2 border-amber-200 dark:border-amber-800 rounded-lg shadow-lg p-4">
                <div className="flex items-start gap-3">
                    <div className="flex-shrink-0 mt-0.5">
                        <Clock className="w-5 h-5 text-amber-600 dark:text-amber-400" />
                    </div>
                    <div className="flex-1 min-w-0">
                        <h3 className="font-semibold text-amber-900 dark:text-amber-100">
                            Session Expiring Soon
                        </h3>
                        <p className="text-sm text-amber-700 dark:text-amber-200 mt-1">
                            Your session will expire in {formatTime(timeUntilExpiry)}
                        </p>
                    </div>
                    <button
                        onClick={manualRefresh}
                        disabled={isRefreshing}
                        className="flex-shrink-0 ml-2 px-3 py-1 bg-amber-600 hover:bg-amber-700 disabled:bg-amber-500 dark:bg-amber-700 dark:hover:bg-amber-600 text-white text-sm font-medium rounded transition-colors"
                    >
                        {isRefreshing ? 'Refreshing...' : 'Refresh'}
                    </button>
                </div>
            </div>
        </div>
    )
}
