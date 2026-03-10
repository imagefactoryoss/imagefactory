import { authService } from '@/services/authService'
import { useAuthStore } from '@/store/auth'
import { useEffect, useRef, useState } from 'react'
import toast from 'react-hot-toast'

interface TokenExpirationState {
    isExpiringSoon: boolean
    timeUntilExpiry: number | null // seconds
    isRefreshing: boolean
}

/**
 * Hook to manage token expiration and auto-refresh
 * - Monitors token expiration time
 * - Auto-refreshes 1 minute before expiration
 * - Shows warning when token will expire in less than 1 minute
 * - Logs user out if refresh fails
 */
export const useTokenExpiration = () => {
    const { token, refreshToken, tokenExpiry, updateTokens, logout } = useAuthStore((state) => ({
        token: state.token,
        refreshToken: state.refreshToken,
        tokenExpiry: state.tokenExpiry,
        updateTokens: state.updateTokens,
        logout: state.logout,
    }))

    const [state, setState] = useState<TokenExpirationState>({
        isExpiringSoon: false,
        timeUntilExpiry: null,
        isRefreshing: false,
    })
    const isRefreshingRef = useRef(false)
    const lastRefreshAttemptRef = useRef<number>(0)
    const lastRefreshFailureToastRef = useRef<number>(0)

    useEffect(() => {
        if (!token || !tokenExpiry) return

        const checkExpiration = () => {
            const now = Math.floor(Date.now() / 1000) // Current time in seconds
            const secondsUntilExpiry = tokenExpiry - now

            // Token already expired
            if (secondsUntilExpiry <= 0) {
                // Final attempt to refresh when already expired.
                if (refreshToken && !isRefreshingRef.current) {
                    refreshAccessToken(true)
                    return
                }
                logout()
                toast.error('Your session has expired. Please login again.')
                return
            }

            // Update state
            const isExpiringSoon = secondsUntilExpiry < 60 // Less than 1 minute
            setState((prev) => ({
                ...prev,
                timeUntilExpiry: Math.max(0, secondsUntilExpiry),
                isExpiringSoon,
            }))

            // Auto-refresh 1 minute before expiration
            if (secondsUntilExpiry <= 60 && refreshToken && !isRefreshingRef.current) {
                const nowMs = Date.now()
                if (nowMs - lastRefreshAttemptRef.current >= 15000) {
                    lastRefreshAttemptRef.current = nowMs
                    refreshAccessToken(false)
                }
            }
        }

        checkExpiration()

        // Check every 10 seconds
        const interval = setInterval(checkExpiration, 10000)
        return () => clearInterval(interval)
    }, [token, tokenExpiry, refreshToken, logout])

    const refreshAccessToken = async (forceLogoutOnFailure: boolean) => {
        if (!refreshToken || isRefreshingRef.current) return

        isRefreshingRef.current = true
        setState((prev) => ({ ...prev, isRefreshing: true }))

        try {
            const result = await authService.refreshToken({ refresh_token: refreshToken })

            if (result && result.access_token && result.refresh_token) {
                // Update auth store with new tokens
                updateTokens(result.access_token, result.refresh_token, result.access_token_expiry)

                // Keep this silent to avoid noisy toasts on background refresh.
            }
        } catch (error) {
            const errMsg = error instanceof Error ? error.message.toLowerCase() : ''
            const terminalRefreshFailure =
                errMsg.includes('invalid refresh token') ||
                errMsg.includes('refresh token expired') ||
                errMsg.includes('token expired') ||
                errMsg.includes('unauthorized')

            if (forceLogoutOnFailure || terminalRefreshFailure) {
                logout()
                toast.error('Session refresh failed. Please login again.')
            } else {
                const nowMs = Date.now()
                if (nowMs - lastRefreshFailureToastRef.current > 60000) {
                    lastRefreshFailureToastRef.current = nowMs
                    toast.error('Session refresh failed. Retrying...')
                }
            }
        } finally {
            isRefreshingRef.current = false
            setState((prev) => ({ ...prev, isRefreshing: false }))
        }
    }

    const manualRefresh = async () => {
        await refreshAccessToken(false)
    }

    return {
        ...state,
        manualRefresh,
        tokenExpiry,
    }
}

/**
 * Hook to show token expiration warning UI
 */
export const useShowTokenExpirationWarning = () => {
    const { isExpiringSoon, timeUntilExpiry } = useTokenExpiration()

    return {
        shouldShowWarning: isExpiringSoon && timeUntilExpiry !== null && timeUntilExpiry > 0,
        timeUntilExpiry,
    }
}
