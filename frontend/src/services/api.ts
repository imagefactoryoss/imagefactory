import { useAuthStore } from '@/store/auth'
import { useTenantStore } from '@/store/tenant'
import { NIL_TENANT_ID } from '@/constants/tenant'
import axios from 'axios'
import toast from 'react-hot-toast'

function resolveApiBaseURL(): string {
    const runtimeBase = window.__APP_CONFIG__?.API_BASE_URL
    const raw = typeof runtimeBase === 'string' ? runtimeBase.trim() : ''
    if (!raw) {
        return '/api/v1'
    }
    const normalized = raw.replace(/\/+$/, '')
    if (/\/api\/v\d+$/i.test(normalized)) {
        return normalized
    }
    if (/\/api$/i.test(normalized)) {
        return `${normalized}/v1`
    }
    return `${normalized}/api/v1`
}

// Create axios instance
const api = axios.create({
    baseURL: resolveApiBaseURL(),
    timeout: 30000,
    headers: {
        'Content-Type': 'application/json',
    },
})

// Request interceptor to add auth token and tenant header
api.interceptors.request.use(
    (config: any) => {
        const { token } = useAuthStore.getState()
        const { selectedTenantId, userTenants } = useTenantStore.getState()

        if (token) {
            config.headers.Authorization = `Bearer ${token}`
        }

        // Tenant context is mandatory only for authenticated API requests.
        // During login/bootstrap flows we must not send stale tenant context.
        const tenantId = selectedTenantId || userTenants?.[0]?.id
        if (token && tenantId && tenantId !== NIL_TENANT_ID) {
            config.headers['X-Tenant-ID'] = tenantId
        } else if (config.headers && 'X-Tenant-ID' in config.headers) {
            delete config.headers['X-Tenant-ID']
        }

        return config
    },
    (error: any) => {
        return Promise.reject(error)
    }
)

// Response interceptor to handle errors and log responses
api.interceptors.response.use(
    (response: any) => {
        return response
    },
    async (error: any) => {
        const errorMessage = getErrorMessage(error)

        // Skip interceptor for login/auth endpoints - let them handle their own errors
        if (error.config?.url?.includes('/auth/login') || error.config?.url?.includes('/auth/sso/callback')) {
            return Promise.reject(error)
        }

        // Check if this is a tenant-related error that indicates stale cache
        const isTenantError = errorMessage.toLowerCase().includes('invalid tenant') ||
            errorMessage.toLowerCase().includes('tenant not found') ||
            errorMessage.toLowerCase().includes('tenant does not exist')

        if (isTenantError && !error.config._tenantRetry) {
            // This looks like a stale tenant cache issue. Try to refresh profile and retry once.
            console.warn('Detected tenant-related error, refreshing profile data:', errorMessage)
            try {
                await useAuthStore.getState().refreshProfile()
                console.log('Profile refreshed successfully, retrying request')
                // Mark this request as retried to avoid infinite loops
                error.config._tenantRetry = true
                // Retry the original request with fresh tenant data
                return api.request(error.config)
            } catch (refreshError) {
                console.error('Profile refresh failed:', refreshError)
                // Logout user and redirect to login for non-permission endpoints
                useAuthStore.getState().logout()
                toast.error('Session expired. Please login again.')
                window.location.href = '/login'
            }
        } else if (error.response?.status === 403) {
            // Forbidden
            toast.error('You do not have permission to perform this action.')
        } else if (error.response?.status === 400) {
            // Bad request - show the error message from backend
            const errorMsg = (error.response?.data as any)?.error || 'Invalid request'
            toast.error(errorMsg)
        } else if (error.response?.status === 409) {
            // Conflict - let the component handle this error to show specific message
            // Don't show toast here to avoid duplicates
        } else if (error.response?.status === 404) {
            // Not found
            toast.error('Resource not found.')
        } else if (error.response?.status && error.response.status >= 500) {
            // Server error
            toast.error('Server error. Please try again later.')
        } else if (error.code === 'ECONNABORTED') {
            // Timeout
            toast.error('Request timeout. Please try again.')
        } else if (!error.response) {
            // Network error
            toast.error('Network error. Please check your connection.')
        }

        return Promise.reject(error)
    }
)

export default api

// Helper function to extract error message
export const getErrorMessage = (error: any): string => {
    if (error.response?.data?.error) {
        return error.response.data.error
    }
    if (error.response?.data?.message) {
        return error.response.data.message
    }
    if (error.message) {
        return error.message
    }
    return 'An unexpected error occurred'
}

// Export the api instance
export { api }

// Helper function to check if error is a specific type
export const isAuthError = (error: any): boolean => {
    return error.response?.status === 401 || error.response?.status === 403
}

export const isNetworkError = (error: any): boolean => {
    return !error.response && error.code !== 'ECONNABORTED'
}

export const isTimeoutError = (error: any): boolean => {
    return error.code === 'ECONNABORTED'
}
