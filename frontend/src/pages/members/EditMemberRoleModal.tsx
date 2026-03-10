import { roleService } from '@/services/roleService'
import { UserWithRoles } from '@/types'
import { Loader, Shield, X } from 'lucide-react'
import { useEffect, useState } from 'react'

interface EditMemberRoleModalProps {
    isOpen: boolean
    member: UserWithRoles
    tenantId: string
    onClose: () => void
    onSubmit: (roleId: string) => Promise<void>
}

interface Role {
    id: string
    name: string
    description?: string
}

export default function EditMemberRoleModal({
    isOpen,
    member,
    tenantId,
    onClose,
    onSubmit,
}: EditMemberRoleModalProps) {
    const [selectedRoleId, setSelectedRoleId] = useState<string>('')
    const [loading, setLoading] = useState(false)
    const [error, setError] = useState<string | null>(null)
    const [availableRoles, setAvailableRoles] = useState<Role[]>([])
    const [rolesLoading, setRolesLoading] = useState(false)

    // Fetch available roles from API
    useEffect(() => {
        if (!isOpen || !tenantId) return

        const fetchRoles = async () => {
            try {
                setRolesLoading(true)
                const response = await roleService.listRoles(tenantId)
                setAvailableRoles(response.data || [])
            } catch (err) {
                console.error('Failed to fetch roles:', err)
                setError('Failed to load available roles')
                setAvailableRoles([])
            } finally {
                setRolesLoading(false)
            }
        }

        fetchRoles()
    }, [isOpen, tenantId])

    useEffect(() => {
        if (member && isOpen && availableRoles.length > 0) {
            // Get the first role ID for this member, or default to first available role
            const firstRole = member.roles && member.roles.length > 0 ? member.roles[0].id : availableRoles[0]?.id
            setSelectedRoleId(firstRole || '')
            setError(null)
        }
    }, [member, isOpen, availableRoles])

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault()

        if (!selectedRoleId) {
            setError('Please select a role')
            return
        }

        try {
            setLoading(true)
            setError(null)
            await onSubmit(selectedRoleId)
        } catch (err) {
            const message = err instanceof Error ? err.message : 'Failed to update role'
            setError(message)
        } finally {
            setLoading(false)
        }
    }

    const handleClose = () => {
        if (!loading) {
            setError(null)
            onClose()
        }
    }

    if (!isOpen || !member) return null

    const selectedRole = availableRoles.find(r => r.id === selectedRoleId)
    const currentRole = member.roles && member.roles.length > 0 ? member.roles[0].name : 'No role'

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
                            <Shield className="h-6 w-6 text-blue-600 dark:text-blue-400" />
                            <h3 className="text-lg font-medium text-gray-900 dark:text-white">
                                Update Member Role
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
                        {/* Member Info */}
                        <div className="mb-6 p-4 bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-lg">
                            <div className="text-sm">
                                <p className="font-medium text-gray-900 dark:text-white">{member.name}</p>
                                <p className="text-gray-600 dark:text-gray-400">{member.email}</p>
                                <p className="text-xs text-gray-500 dark:text-gray-500 mt-1">
                                    Current role: <span className="font-medium">{currentRole}</span>
                                </p>
                            </div>
                        </div>

                        {/* Error message */}
                        {error && (
                            <div className="mb-4 p-3 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg">
                                <p className="text-sm text-red-800 dark:text-red-400">{error}</p>
                            </div>
                        )}

                        {/* Role Selection */}
                        <div className="mb-6">
                            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-3">
                                Select New Role
                            </label>
                            {rolesLoading ? (
                                <div className="flex items-center justify-center p-4">
                                    <Loader className="h-5 w-5 animate-spin text-blue-600" />
                                    <span className="ml-2 text-sm text-gray-600 dark:text-gray-400">Loading roles...</span>
                                </div>
                            ) : availableRoles.length > 0 ? (
                                <div className="space-y-2">
                                    {availableRoles.map((role) => (
                                        <label key={role.id} className="flex items-start gap-3 p-3 border border-gray-200 dark:border-gray-700 rounded-lg cursor-pointer hover:bg-gray-50 dark:hover:bg-gray-700/50 transition">
                                            <input
                                                type="radio"
                                                name="role"
                                                value={role.id}
                                                checked={selectedRoleId === role.id}
                                                onChange={(e) => setSelectedRoleId(e.target.value)}
                                                disabled={loading}
                                                className="mt-1 h-4 w-4 text-blue-600 focus:ring-blue-500 cursor-pointer"
                                            />
                                            <div className="flex-1">
                                                <div className="text-sm font-medium text-gray-900 dark:text-white">
                                                    {role.name}
                                                </div>
                                                {role.description && (
                                                    <div className="text-sm text-gray-600 dark:text-gray-400">
                                                        {role.description}
                                                    </div>
                                                )}
                                            </div>
                                        </label>
                                    ))}
                                </div>
                            ) : (
                                <p className="text-sm text-gray-600 dark:text-gray-400">No roles available</p>
                            )}
                        </div>

                        {/* Role Summary */}
                        {selectedRole && (
                            <div className="p-3 bg-gray-50 dark:bg-gray-700 rounded-lg">
                                <p className="text-xs text-gray-600 dark:text-gray-400">
                                    <strong>{member.name}</strong> will have <strong>{selectedRole.name}</strong> permissions in this tenant.
                                </p>
                            </div>
                        )}
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
                        <button
                            onClick={handleSubmit}
                            disabled={loading}
                            className="px-4 py-2 text-sm font-medium text-white bg-blue-600 rounded-lg hover:bg-blue-700 disabled:opacity-50 flex items-center gap-2 transition"
                        >
                            {loading && <Loader className="h-4 w-4 animate-spin" />}
                            {loading ? 'Updating...' : 'Update Role'}
                        </button>
                    </div>
                </div>
            </div>
        </div>
    )
}
