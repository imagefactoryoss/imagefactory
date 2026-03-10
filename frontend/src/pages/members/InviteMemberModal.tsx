import { api } from '@/services/api'
import { Loader, Mail, X } from 'lucide-react'
import { useCallback, useEffect, useRef, useState } from 'react'

interface InviteMemberModalProps {
    isOpen: boolean
    onClose: () => void
    onSubmit: (data: InviteMemberFormData) => Promise<void>
    tenantId?: string
}

export interface InviteMemberFormData {
    email: string
    roleId: string
    message?: string
    isLDAP?: boolean
}

interface LDAPUser {
    username: string
    email: string
    fullName: string
}

interface Role {
    id: string
    name: string
    description: string
}

const DEFAULT_ROLES: Role[] = [
    { id: 'Owner', name: 'Owner', description: 'Full access to tenant resources' },
    { id: 'Developer', name: 'Developer', description: 'Can create and manage projects and builds' },
    { id: 'Operator', name: 'Operator', description: 'Can view and monitor builds' },
    { id: 'Viewer', name: 'Viewer', description: 'Read-only access' },
]

export default function InviteMemberModal({ isOpen, onClose, onSubmit, tenantId }: InviteMemberModalProps) {
    const [formData, setFormData] = useState<InviteMemberFormData>({
        email: '',
        roleId: 'Developer',
        message: '',
        isLDAP: false,
    })
    const [errors, setErrors] = useState<Record<string, string>>({})
    const [loading, setLoading] = useState(false)
    const [ldapUsers, setLdapUsers] = useState<LDAPUser[]>([])
    const [showLdapDropdown, setShowLdapDropdown] = useState(false)
    const [ldapSearching, setLdapSearching] = useState(false)
    const [userExists, setUserExists] = useState(false)
    const [existingUserId, setExistingUserId] = useState<string | null>(null)
    const [availableRoles, setAvailableRoles] = useState<Role[]>([])
    const [rolesLoading, setRolesLoading] = useState(false)
    const searchTimeoutRef = useRef<NodeJS.Timeout>()

    // Fetch available roles for the tenant
    useEffect(() => {
        if (!isOpen || !tenantId) return

        const fetchRoles = async () => {
            try {
                setRolesLoading(true)
                const response = await api.get(`/tenants/${tenantId}/roles`)
                if (response.status >= 200 && response.status < 300) {
                    const data = response.data
                    const roles = (data.data || []).map((role: any) => ({
                        id: role.id,
                        name: role.name,
                        description: role.description || '',
                    }))
                    setAvailableRoles(roles)
                    // Set default role to first role if current default is not in the list
                    if (roles.length > 0 && !roles.some((r: typeof roles[0]) => r.id === formData.roleId)) {
                        setFormData(prev => ({ ...prev, roleId: roles[0].id }))
                    }
                } else {
                    setAvailableRoles(DEFAULT_ROLES)
                }
            } catch (error) {
                setAvailableRoles(DEFAULT_ROLES)
            } finally {
                setRolesLoading(false)
            }
        }

        fetchRoles()
    }, [isOpen, tenantId])

    // Update formData to use available roles

    // Search LDAP users
    const searchLDAPUsers = useCallback(async (query: string) => {
        if (!query || query.length < 2) {
            setLdapUsers([])
            setShowLdapDropdown(false)
            return
        }

        try {
            setLdapSearching(true)
            const response = await api.post('/auth/ldap/search-users', { query, limit: 10 })
            const data = response.data
            setLdapUsers(data.users || [])
            setShowLdapDropdown((data.users || []).length > 0)
        } catch (error) {
            setLdapUsers([])
            setShowLdapDropdown(false)
        } finally {
            setLdapSearching(false)
        }
    }, [])

    // Check if user already exists
    const checkUserExists = useCallback(async (email: string) => {
        try {
            const response = await api.get(`/admin/users/check-email`, {
                params: { email },
            })
            const data = response.data
            setUserExists(data.exists === true)
            if (data.exists && data.id) {
                setExistingUserId(data.id)
            } else {
                setExistingUserId(null)
            }
        } catch (error) {
        }
    }, [])

    const handleEmailChange = (value: string) => {
        setFormData({ ...formData, email: value, isLDAP: false })
        setUserExists(false)
        setExistingUserId(null)

        if (searchTimeoutRef.current) {
            clearTimeout(searchTimeoutRef.current)
        }

        if (value.length >= 2) {
            setLdapSearching(true)
            searchTimeoutRef.current = setTimeout(() => {
                searchLDAPUsers(value)
            }, 300)
        } else {
            setLdapUsers([])
            setShowLdapDropdown(false)
        }
    }

    const selectLDAPUser = (user: LDAPUser) => {
        setFormData({ ...formData, email: user.email, isLDAP: true })
        setLdapUsers([])
        setShowLdapDropdown(false)
        if (errors.email) setErrors({ ...errors, email: '' })
        // Check if this user already exists in the system
        checkUserExists(user.email)
    }

    const validateForm = (): boolean => {
        const newErrors: Record<string, string> = {}

        if (!formData.email.trim()) {
            newErrors.email = 'Email is required'
        } else if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(formData.email)) {
            newErrors.email = 'Please enter a valid email address'
        }

        if (!formData.roleId) {
            newErrors.roleId = 'Role is required'
        }

        setErrors(newErrors)
        return Object.keys(newErrors).length === 0
    }

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault()

        if (!validateForm()) {
            return
        }

        try {
            setLoading(true)
            await onSubmit(formData)
            setFormData({ email: '', roleId: 'developer', message: '', isLDAP: false })
            setErrors({})
            setExistingUserId(null)
        } catch (error) {
            const message = error instanceof Error ? error.message : 'Failed to send invitation'
            setErrors({ submit: message })
        } finally {
            setLoading(false)
        }
    }

    const handleAddExistingUser = async (e: React.FormEvent) => {
        e.preventDefault()

        if (!validateForm() || !existingUserId || !tenantId) {
            return
        }

        try {
            setLoading(true)
            const { adminService } = await import('@/services/adminService')
            const toast = (await import('react-hot-toast')).default

            await adminService.addExistingUserToTenant({
                userId: existingUserId,
                tenantId,
                roleIds: [formData.roleId],
            })

            // Show success message
            toast.success(`User ${formData.email} added to tenant`)

            // Reset form
            setFormData({ email: '', roleId: 'developer', message: '', isLDAP: false })
            setErrors({})
            setExistingUserId(null)

            // Close the modal - parent will handle member list refresh
            handleClose()
        } catch (error) {
            const message = error instanceof Error ? error.message : 'Failed to add user to tenant'
            setErrors({ submit: message })
        } finally {
            setLoading(false)
        }
    }

    const handleClose = () => {
        if (!loading) {
            setFormData({ email: '', roleId: 'developer', message: '', isLDAP: false })
            setErrors({})
            onClose()
        }
    }

    if (!isOpen) return null

    return (
        <div className="fixed inset-0 z-50 overflow-y-auto">
            <div className="flex items-center justify-center min-h-screen px-4 pt-4 pb-20 text-center sm:block sm:p-0">
                {/* Background overlay */}
                <div
                    className="fixed inset-0 bg-gray-500 bg-opacity-75 dark:bg-gray-900 dark:bg-opacity-75 transition-opacity"
                    onClick={handleClose}
                />

                {/* Modal */}
                <div className="inline-block align-bottom bg-white dark:bg-gray-800 rounded-lg text-left overflow-hidden shadow-xl transform transition-all sm:my-8 sm:align-middle sm:max-w-lg sm:w-full">
                    {/* Header */}
                    <div className="flex items-center justify-between px-6 py-4 border-b border-gray-200 dark:border-gray-700">
                        <div className="flex items-center gap-3">
                            <Mail className="h-6 w-6 text-blue-600 dark:text-blue-400" />
                            <h3 className="text-lg font-medium text-gray-900 dark:text-white">
                                Invite Member
                            </h3>
                        </div>
                        <button
                            onClick={handleClose}
                            disabled={loading}
                            className="text-gray-400 hover:text-gray-500 dark:hover:text-gray-300 disabled:opacity-50"
                        >
                            <X className="h-5 w-5" />
                        </button>
                    </div>

                    {/* Body */}
                    <form onSubmit={handleSubmit} className="px-6 py-4">
                        {/* Error message */}
                        {errors.submit && (
                            <div className="mb-4 p-3 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg">
                                <p className="text-sm text-red-800 dark:text-red-400">{errors.submit}</p>
                            </div>
                        )}

                        {/* Email with LDAP Search */}
                        <div className="mb-4">
                            <label htmlFor="email" className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                                Email Address *
                            </label>
                            <div className="relative">
                                <input
                                    type="email"
                                    id="email"
                                    value={formData.email}
                                    onChange={(e) => {
                                        handleEmailChange(e.target.value)
                                        if (errors.email) setErrors({ ...errors, email: '' })
                                    }}
                                    onBlur={() => {
                                        if (formData.email && !userExists) {
                                            checkUserExists(formData.email)
                                        }
                                        setTimeout(() => setShowLdapDropdown(false), 200)
                                    }}
                                    onFocus={() => {
                                        if (ldapUsers.length > 0) {
                                            setShowLdapDropdown(true)
                                        }
                                    }}
                                    placeholder="user@example.com"
                                    disabled={loading}
                                    className={`w-full px-3 py-2 border rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white placeholder-gray-400 dark:placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:opacity-50 ${errors.email ? 'border-red-500' : 'border-gray-300 dark:border-gray-600'
                                        }`}
                                />
                                {ldapSearching && (
                                    <Loader className="absolute right-3 top-2.5 h-5 w-5 animate-spin text-gray-400" />
                                )}
                                {showLdapDropdown && ldapUsers.length > 0 && (
                                    <div className="absolute top-full left-0 right-0 mt-1 bg-white dark:bg-gray-700 border border-gray-300 dark:border-gray-600 rounded-lg shadow-lg z-10 max-h-48 overflow-y-auto">
                                        <div className="p-2">
                                            <div className="text-xs text-gray-500 dark:text-gray-400 px-2 py-1 font-medium">
                                                Found {ldapUsers.length} user{ldapUsers.length !== 1 ? 's' : ''} in LDAP
                                            </div>
                                            {ldapUsers.map((user, index) => (
                                                <button
                                                    key={`${user.email}-${index}`}
                                                    type="button"
                                                    onClick={() => selectLDAPUser(user)}
                                                    className="w-full text-left px-2 py-2 hover:bg-gray-100 dark:hover:bg-gray-600 rounded transition text-sm text-gray-900 dark:text-white"
                                                >
                                                    <div className="font-medium">{user.fullName}</div>
                                                    <div className="text-xs text-gray-500 dark:text-gray-400">{user.email}</div>
                                                </button>
                                            ))}
                                        </div>
                                    </div>
                                )}
                            </div>
                            {errors.email && <p className="mt-1 text-sm text-red-600 dark:text-red-400">{errors.email}</p>}
                            {userExists && formData.email && (
                                <p className="mt-1 text-sm text-amber-600 dark:text-amber-400">
                                    ✓ This user already exists in the system. Select a role and click "Add Existing User" to add them to your tenant.
                                </p>
                            )}
                            {!userExists && !errors.email && (
                                <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
                                    Type to search LDAP directory or enter an email address
                                </p>
                            )}
                        </div>

                        {/* Role Selection */}
                        <div className="mb-6">
                            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-3">
                                Role *
                            </label>
                            {rolesLoading ? (
                                <div className="flex items-center justify-center py-4">
                                    <Loader className="h-5 w-5 animate-spin text-gray-400" />
                                    <span className="ml-2 text-sm text-gray-500">Loading roles...</span>
                                </div>
                            ) : availableRoles.length > 0 ? (
                                <div className="space-y-2">
                                    {availableRoles.map((role) => (
                                        <label key={role.id} className="flex items-start gap-3 p-3 border border-gray-200 dark:border-gray-700 rounded-lg cursor-pointer hover:bg-gray-50 dark:hover:bg-gray-700/50 transition">
                                            <input
                                                type="radio"
                                                name="role"
                                                value={role.id}
                                                checked={formData.roleId === role.id}
                                                onChange={(e) => {
                                                    setFormData({ ...formData, roleId: e.target.value })
                                                    if (errors.roleId) setErrors({ ...errors, roleId: '' })
                                                }}
                                                disabled={loading}
                                                className="mt-1 h-4 w-4 text-blue-600 focus:ring-blue-500 cursor-pointer"
                                            />
                                            <div className="flex-1">
                                                <div className="text-sm font-medium text-gray-900 dark:text-white">
                                                    {role.name}
                                                </div>
                                                <div className="text-sm text-gray-600 dark:text-gray-400">
                                                    {role.description}
                                                </div>
                                            </div>
                                        </label>
                                    ))}
                                </div>
                            ) : (
                                <p className="text-sm text-red-600 dark:text-red-400">No roles available</p>
                            )}
                            {errors.roleId && <p className="mt-1 text-sm text-red-600 dark:text-red-400">{errors.roleId}</p>}
                        </div>

                        {/* Message (Optional) */}
                        <div className="mb-6">
                            <label htmlFor="message" className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                                Personal Message (Optional)
                            </label>
                            <textarea
                                id="message"
                                value={formData.message}
                                onChange={(e) => setFormData({ ...formData, message: e.target.value })}
                                placeholder="Add a personal message to include in the invitation email..."
                                disabled={loading}
                                rows={3}
                                className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white placeholder-gray-400 dark:placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:opacity-50"
                            />
                            <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
                                {formData.isLDAP
                                    ? 'This user will be added immediately and notified by email.'
                                    : `This message will be included in the invitation email sent to ${formData.email || 'the new member'}`}
                            </p>
                        </div>
                    </form>

                    {/* Footer */}
                    <div className="bg-gray-50 dark:bg-gray-700 px-6 py-4 flex justify-end gap-3">
                        <button
                            onClick={handleClose}
                            disabled={loading}
                            className="px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 border border-gray-300 dark:border-gray-600 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-600 disabled:opacity-50 transition"
                        >
                            Cancel
                        </button>
                        {userExists ? (
                            <button
                                onClick={handleAddExistingUser}
                                disabled={loading}
                                className="px-4 py-2 text-sm font-medium text-white bg-green-600 rounded-lg hover:bg-green-700 disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2 transition"
                            >
                                {loading && <Loader className="h-4 w-4 animate-spin" />}
                                {loading ? 'Adding...' : 'Add Existing User'}
                            </button>
                        ) : (
                            <button
                                onClick={handleSubmit}
                                disabled={loading}
                                className="px-4 py-2 text-sm font-medium text-white bg-blue-600 rounded-lg hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2 transition"
                            >
                                {loading && <Loader className="h-4 w-4 animate-spin" />}
                                {loading ? 'Processing...' : formData.isLDAP ? 'Add LDAP User' : 'Send Invitation'}
                            </button>
                        )}
                    </div>
                </div>
            </div>
        </div>
    )
}
