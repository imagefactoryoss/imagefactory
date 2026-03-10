import api from '@/services/api'
import { Permission } from '@/types'
import { useCallback, useEffect, useState } from 'react'
import toast from 'react-hot-toast'

export interface RolePermission {
    resource: string
    action: string
}

// Cache for permissions to prevent duplicate requests
let permissionsCache: Permission[] | null = null
let permissionsCachePromise: Promise<Permission[]> | null = null

/**
 * Hook to fetch all available permissions (with caching to prevent duplicate API calls)
 */
export const usePermissions = () => {
    const [permissions, setPermissions] = useState<Permission[]>(permissionsCache || [])
    const [loading, setLoading] = useState(!permissionsCache)
    const [error, setError] = useState<string | null>(null)

    useEffect(() => {
        // If we have cached data, use it immediately
        if (permissionsCache) {
            setPermissions(permissionsCache)
            setLoading(false)
            return
        }

        // If a request is already in flight, wait for it
        if (permissionsCachePromise) {
            permissionsCachePromise
                .then((data) => {
                    setPermissions(data)
                    setLoading(false)
                })
                .catch(() => {
                    setLoading(false)
                })
            return
        }

        // Make new request
        const fetchData = async () => {
            try {
                setLoading(true)
                setError(null)

                // Fetch all permissions by getting all pages
                const allPermissions: Permission[] = []
                let page = 1
                const pageSize = 100
                let hasMorePages = true

                while (hasMorePages) {
                    const url = `/permissions?page=${page}&page_size=${pageSize}`
                    const response = await api.get(url)
                    const data = response.data
                    const pagePermissions = data.data || []

                    allPermissions.push(...pagePermissions)

                    // Check if there are more pages
                    const total = data.total || 0
                    hasMorePages = allPermissions.length < total
                    page++
                }

                // Cache the result
                permissionsCache = allPermissions
                permissionsCachePromise = null

                setPermissions(allPermissions)
                setLoading(false)
            } catch (err) {
                const message = err instanceof Error ? err.message : 'Failed to fetch permissions'
                setError(message)
                setLoading(false)
                permissionsCachePromise = null
                // Don't show toast error for permissions fetch - just log it
                // toast.error(message)
            }
        }

        // Create promise for concurrent requests
        const promise = (async () => {
            await fetchData()
            return permissionsCache || []
        })()

        permissionsCachePromise = promise
    }, [])

    const refetch = async (skipCache = false) => {
        if (!skipCache && permissionsCache) {
            setPermissions(permissionsCache)
            return permissionsCache
        }

        try {
            setLoading(true)
            setError(null)

            // Fetch all permissions by getting all pages
            const allPermissions: Permission[] = []
            let page = 1
            const pageSize = 100
            let hasMorePages = true

            while (hasMorePages) {
                const response = await api.get(`/permissions?page=${page}&page_size=${pageSize}`)
                const data = response.data
                const pagePermissions = data.data || []

                allPermissions.push(...pagePermissions)

                // Check if there are more pages
                const total = data.total || 0
                hasMorePages = allPermissions.length < total
                page++
            }

            permissionsCache = allPermissions
            setPermissions(allPermissions)
            setLoading(false)
            return allPermissions
        } catch (err) {
            const message = err instanceof Error ? err.message : 'Failed to fetch permissions'
            setError(message)
            setLoading(false)
            // Don't show toast error for permissions fetch - just log it
            // toast.error(message)
            throw err
        }
    }

    return { permissions, loading, error, refetch }
}

/**
 * Hook to fetch permissions for a specific role
 */
export const useRolePermissions = (roleId: string | null) => {
    const [rolePermissions, setRolePermissions] = useState<Permission[]>([])
    const [loading, setLoading] = useState(false)
    const [error, setError] = useState<string | null>(null)

    const fetchRolePermissions = useCallback(async () => {
        if (!roleId) return

        try {
            setLoading(true)
            setError(null)
            const response = await api.get(`/roles/${roleId}/permissions`)
            const data = response.data
            setRolePermissions(data.data || [])
        } catch (err) {
            const message = err instanceof Error ? err.message : 'Failed to fetch role permissions'
            setError(message)
            toast.error(message)
        } finally {
            setLoading(false)
        }
    }, [roleId])

    useEffect(() => {
        fetchRolePermissions()
    }, [fetchRolePermissions])

    return { rolePermissions, loading, error, refetch: fetchRolePermissions }
}

/**
 * Hook to add a permission to a role
 */
export const useAddPermissionToRole = () => {
    const [loading, setLoading] = useState(false)
    const [error, setError] = useState<string | null>(null)

    const addPermission = useCallback(async (roleId: string, permissionId: string) => {
        try {
            setLoading(true)
            setError(null)
            await api.post(`/roles/${roleId}/permissions/${permissionId}`)
            toast.success('Permission added to role')
            return true
        } catch (err: any) {
            let message = 'Failed to add permission'
            if (err.response?.status === 401) {
                message = 'Unauthorized: Please log in'
            } else if (err.response?.status === 409) {
                message = 'Permission already assigned to this role'
            }
            setError(message)
            toast.error(message)
            return false
        } finally {
            setLoading(false)
        }
    }, [])

    return { addPermission, loading, error }
}

/**
 * Hook to remove a permission from a role
 */
export const useRemovePermissionFromRole = () => {
    const [loading, setLoading] = useState(false)
    const [error, setError] = useState<string | null>(null)

    const removePermission = useCallback(async (roleId: string, permissionId: string) => {
        try {
            setLoading(true)
            setError(null)
            await api.delete(`/roles/${roleId}/permissions/${permissionId}`)
            toast.success('Permission removed from role')
            return true
        } catch (err: any) {
            let message = 'Failed to remove permission'
            if (err.response?.status === 401) {
                message = 'Unauthorized: Please log in'
            } else if (err.response?.status === 404) {
                message = 'Permission not assigned to this role'
            }
            setError(message)
            toast.error(message)
            return false
        } finally {
            setLoading(false)
        }
    }, [])

    return { removePermission, loading, error }
}
