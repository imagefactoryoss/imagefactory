import { UserWithRoles } from '@/types'
import { Edit, Mail, Trash2 } from 'lucide-react'

interface MembersTableProps {
    members: UserWithRoles[]
    tenantId?: string
    canManage?: boolean
    onEditRole: (member: UserWithRoles) => void
    onRemove: (member: UserWithRoles) => void
}

export default function MembersTable({ members, tenantId, canManage = false, onEditRole, onRemove }: MembersTableProps) {
    const formatDate = (dateString: string) => {
        return new Date(dateString).toLocaleDateString('en-US', {
            year: 'numeric',
            month: 'short',
            day: 'numeric',
        })
    }

    const getRoleColor = (role: string) => {
        switch (role?.toLowerCase()) {
            case 'owner':
                return 'bg-purple-100 text-purple-800 dark:bg-purple-900/20 dark:text-purple-400'
            case 'developer':
                return 'bg-blue-100 text-blue-800 dark:bg-blue-900/20 dark:text-blue-400'
            case 'operator':
                return 'bg-green-100 text-green-800 dark:bg-green-900/20 dark:text-green-400'
            case 'viewer':
                return 'bg-gray-100 text-gray-800 dark:bg-gray-900/20 dark:text-gray-400'
            default:
                return 'bg-gray-100 text-gray-800 dark:bg-gray-900/20 dark:text-gray-400'
        }
    }

    const getStatusColor = (status: string) => {
        switch (status?.toLowerCase()) {
            case 'active':
                return 'bg-green-100 text-green-800 dark:bg-green-900/20 dark:text-green-400'
            case 'pending':
                return 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/20 dark:text-yellow-400'
            case 'suspended':
                return 'bg-red-100 text-red-800 dark:bg-red-900/20 dark:text-red-400'
            default:
                return 'bg-gray-100 text-gray-800 dark:bg-gray-900/20 dark:text-gray-400'
        }
    }

    if (members.length === 0) {
        return null
    }

    // Count owners in the tenant
    const getOwnerCount = () => {
        if (!tenantId) return 0
        return members.filter(member => {
            if (member.rolesByTenant && member.rolesByTenant[tenantId]) {
                const tenantRoles = member.rolesByTenant[tenantId]
                return tenantRoles.some(role => role.name.toLowerCase() === 'owner')
            }
            return member.roles?.some(role => role.name.toLowerCase() === 'owner') ?? false
        }).length
    }

    const ownerCount = getOwnerCount()

    return (
        <div className="overflow-x-auto border border-gray-200 dark:border-gray-700 rounded-lg">
            <table className="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
                <thead className="bg-gray-50 dark:bg-gray-800">
                    <tr>
                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-700 dark:text-gray-300 uppercase tracking-wider">
                            Name
                        </th>
                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-700 dark:text-gray-300 uppercase tracking-wider">
                            Email
                        </th>
                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-700 dark:text-gray-300 uppercase tracking-wider">
                            Role
                        </th>
                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-700 dark:text-gray-300 uppercase tracking-wider">
                            Status
                        </th>
                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-700 dark:text-gray-300 uppercase tracking-wider">
                            Joined
                        </th>
                        <th className="px-6 py-3 text-right text-xs font-medium text-gray-700 dark:text-gray-300 uppercase tracking-wider">
                            Actions
                        </th>
                    </tr>
                </thead>
                <tbody className="divide-y divide-gray-200 dark:divide-gray-700 bg-white dark:bg-gray-800">
                    {members.map((member) => {
                        // Get role for this specific tenant, not all roles
                        let memberRole = 'No role'
                        if (tenantId && member.rolesByTenant && member.rolesByTenant[tenantId]) {
                            const tenantRoles = member.rolesByTenant[tenantId]
                            if (tenantRoles.length > 0) {
                                memberRole = tenantRoles[0].name
                            }
                        } else if (member.roles && member.roles.length > 0) {
                            // Fallback to first role if no tenant-specific roles found
                            memberRole = member.roles[0].name
                        }

                        return (
                            <tr key={member.id} className="hover:bg-gray-50 dark:hover:bg-gray-700/50 transition">
                                <td className="px-6 py-4 whitespace-nowrap">
                                    <div className="text-sm font-medium text-gray-900 dark:text-white">
                                        {member.name}
                                    </div>
                                </td>
                                <td className="px-6 py-4 whitespace-nowrap">
                                    <div className="flex items-center gap-2 text-sm text-gray-600 dark:text-gray-400">
                                        <Mail className="h-4 w-4" />
                                        {member.email}
                                    </div>
                                </td>
                                <td className="px-6 py-4 whitespace-nowrap">
                                    <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${getRoleColor(memberRole)}`}>
                                        {memberRole}
                                    </span>
                                </td>
                                <td className="px-6 py-4 whitespace-nowrap">
                                    <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${getStatusColor(member.status || '')}`}>
                                        {member.status ? (member.status.charAt(0).toUpperCase() + member.status.slice(1)) : 'Unknown'}
                                    </span>
                                </td>
                                <td className="px-6 py-4 whitespace-nowrap">
                                    <div className="text-sm text-gray-600 dark:text-gray-400">
                                        {formatDate(member.createdAt)}
                                    </div>
                                </td>
                                <td className="px-6 py-4 whitespace-nowrap text-right">
                                    {canManage ? (
                                        <div className="flex items-center justify-end gap-2">
                                            <button
                                                onClick={() => onEditRole(member)}
                                                className="inline-flex items-center gap-1 px-3 py-1.5 text-sm font-medium text-blue-600 dark:text-blue-400 hover:text-blue-700 dark:hover:text-blue-300 transition"
                                            >
                                                <Edit className="h-4 w-4" />
                                                <span className="hidden sm:inline">Edit Role</span>
                                            </button>
                                            <div className="relative group">
                                                <button
                                                    onClick={() => onRemove(member)}
                                                    disabled={memberRole.toLowerCase() === 'owner' && ownerCount <= 1}
                                                    className="inline-flex items-center gap-1 px-3 py-1.5 text-sm font-medium text-red-600 dark:text-red-400 hover:text-red-700 dark:hover:text-red-300 transition disabled:opacity-50 disabled:cursor-not-allowed disabled:hover:text-red-600 dark:disabled:hover:text-red-400"
                                                >
                                                    <Trash2 className="h-4 w-4" />
                                                    <span className="hidden sm:inline">Remove</span>
                                                </button>
                                                {memberRole.toLowerCase() === 'owner' && ownerCount <= 1 && (
                                                    <div className="absolute bottom-full right-0 mb-2 z-50 opacity-0 invisible group-hover:opacity-100 group-hover:visible transition-all duration-200">
                                                        <div className="bg-slate-900 dark:bg-slate-700 text-white text-xs rounded py-2 px-3 whitespace-nowrap">
                                                            Cannot remove the last owner.
                                                            <br />
                                                            Assign another owner first.
                                                            <div className="absolute top-full right-3 w-0 h-0 border-l-4 border-r-4 border-t-4 border-l-transparent border-r-transparent border-t-slate-900 dark:border-t-slate-700"></div>
                                                        </div>
                                                    </div>
                                                )}
                                            </div>
                                        </div>
                                    ) : (
                                        <span className="text-sm text-gray-500 dark:text-gray-400">View only</span>
                                    )}
                                </td>
                            </tr>
                        )
                    })}
                </tbody>
            </table>
        </div>
    )
}
