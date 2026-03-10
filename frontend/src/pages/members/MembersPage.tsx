import MembersTable from '@/components/MembersTable'
import { useConfirmDialog } from '@/context/ConfirmDialogContext'
import { adminService } from '@/services/adminService'
import { useTenantStore } from '@/store/tenant'
import { UserWithRoles } from '@/types'
import { canManageMembers } from '@/utils/permissions'
import { AlertCircle, Loader, Mail, Plus, Search, Users } from 'lucide-react'
import { useEffect, useState } from 'react'
import toast from 'react-hot-toast'
import { Link, useSearchParams } from 'react-router-dom'
import EditMemberRoleModal from './EditMemberRoleModal'
import InviteMemberModal from './InviteMemberModal'

export default function MembersPage() {
    const [searchParams] = useSearchParams()
    const { selectedTenantId } = useTenantStore()
    const tenantId = searchParams.get('tenantId') || selectedTenantId

    const limit = 20  // Items per page

    const [members, setMembers] = useState<UserWithRoles[]>([])
    const [loading, setLoading] = useState(true)
    const [error, setError] = useState<string | null>(null)
    const [searchTerm, setSearchTerm] = useState('')
    const [selectedMember, setSelectedMember] = useState<UserWithRoles | null>(null)
    const [isInviteModalOpen, setIsInviteModalOpen] = useState(false)
    const [isEditRoleModalOpen, setIsEditRoleModalOpen] = useState(false)
    const [page, setPage] = useState(1)
    const [total, setTotal] = useState(0)
    const confirmDialog = useConfirmDialog()

    const canManage = canManageMembers()

    // Fetch members
    const loadMembers = async () => {
        if (!tenantId) {
            setError('No tenant selected')
            setLoading(false)
            return
        }

        try {
            setLoading(true)
            setError(null)
            const response = await adminService.getUsers({
                tenantId,
                search: searchTerm,
                page,
                limit,
            })
            setMembers(response.data)
            setTotal(response.pagination.total)
        } catch (err) {
            const message = err instanceof Error ? err.message : 'Failed to load members'
            setError(message)
            toast.error(message)
        } finally {
            setLoading(false)
        }
    }

    useEffect(() => {
        loadMembers()
    }, [page, limit, searchTerm, tenantId])

    // Handle invite member
    const handleInviteMember = async (data: { email: string; roleId: string; message?: string; isLDAP?: boolean }) => {
        if (!tenantId) return

        try {
            await adminService.sendInvitation({
                email: data.email,
                tenantId,
                roleId: data.roleId,
                message: data.message,
                isLDAP: data.isLDAP,
            })
            setIsInviteModalOpen(false)
            toast.success(data.isLDAP ? `User ${data.email} added to tenant` : `Invitation sent to ${data.email}`)
            // Reload members list
            setPage(1)
            loadMembers()
        } catch (err) {
            const message = err instanceof Error ? err.message : 'Failed to send invitation'
            toast.error(message)
        }
    }

    // Handle update member role
    const handleUpdateMemberRole = async (userId: string, roleId: string) => {
        if (!tenantId) return

        try {
            await adminService.updateTenantMemberRole(tenantId, userId, roleId)
            toast.success('Member role updated')
            // Refresh the member list before closing the modal
            await loadMembers()
            setIsEditRoleModalOpen(false)
            setSelectedMember(null)
        } catch (err) {
            const message = err instanceof Error ? err.message : 'Failed to update member role'
            toast.error(message)
        }
    }

    // Handle remove member
    const handleRemoveMember = async (userId: string) => {
        if (!tenantId) return

        const confirmed = await confirmDialog({
            title: 'Remove Member',
            message: 'Are you sure you want to remove this member from the tenant?',
            confirmLabel: 'Remove Member',
            destructive: true,
        })
        if (!confirmed) return

        try {
            await adminService.removeTenantMember(tenantId, userId)
            toast.success('Member removed')
            setMembers(members.filter(m => m.id !== userId))
            loadMembers()
        } catch (err) {
            const message = err instanceof Error ? err.message : 'Failed to remove member'
            toast.error(message)
        }
    }

    if (!tenantId) {
        return (
            <div className="min-h-screen bg-gray-50 dark:bg-gray-900 flex items-center justify-center">
                <div className="text-center">
                    <AlertCircle className="mx-auto h-12 w-12 text-red-500 mb-4" />
                    <h1 className="text-2xl font-bold text-gray-900 dark:text-white mb-2">
                        No Tenant Selected
                    </h1>
                    <p className="text-gray-600 dark:text-gray-400">
                        Please select a tenant from the tenants page first.
                    </p>
                </div>
            </div>
        )
    }

    return (
        <div className="min-h-screen bg-gray-50 dark:bg-gray-900">
            <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
                {/* Navigation Tabs */}
                <div className="mb-6 border-b border-gray-200 dark:border-gray-700">
                    <div className="flex gap-8">
                        <div className="border-b-2 border-blue-600 dark:border-blue-400 px-1">
                            <button className="flex items-center gap-2 py-2 px-0 text-blue-600 dark:text-blue-400 font-medium">
                                <Users className="h-4 w-4" />
                                Members
                            </button>
                        </div>
                        <Link
                            to={`/invitations?tenantId=${tenantId}`}
                            className="flex items-center gap-2 py-2 px-1 text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white font-medium transition"
                        >
                            <Mail className="h-4 w-4" />
                            Invitations
                        </Link>
                    </div>
                </div>

                {/* Header */}
                <div className="flex justify-between items-center mb-8">
                    <div>
                        <div className="flex items-center gap-3 mb-2">
                            <Users className="h-8 w-8 text-blue-600 dark:text-blue-400" />
                            <h1 className="text-3xl font-bold text-gray-900 dark:text-white">
                                Tenant Members
                            </h1>
                        </div>
                        <p className="text-gray-600 dark:text-gray-400">
                            Manage members and their roles in your tenant
                        </p>
                    </div>
                    {canManage && (
                        <button
                            onClick={() => {
                                setSelectedMember(null)
                                setIsInviteModalOpen(true)
                            }}
                            className="flex items-center gap-2 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition"
                        >
                            <Plus className="h-5 w-5" />
                            Invite Member
                        </button>
                    )}
                </div>

                {/* Search */}
                <div className="mb-6">
                    <div className="relative">
                        <Search className="absolute left-3 top-3 h-5 w-5 text-gray-400" />
                        <input
                            type="text"
                            placeholder="Search members by name or email..."
                            value={searchTerm}
                            onChange={(e) => {
                                setSearchTerm(e.target.value)
                                setPage(1)
                            }}
                            className="w-full pl-10 pr-4 py-2 border border-gray-300 dark:border-gray-700 rounded-lg bg-white dark:bg-gray-800 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500"
                        />
                    </div>
                </div>

                {/* Error State */}
                {error && (
                    <div className="mb-6 p-4 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg">
                        <div className="flex gap-3">
                            <AlertCircle className="h-5 w-5 text-red-600 dark:text-red-400 flex-shrink-0 mt-0.5" />
                            <div>
                                <p className="text-sm font-medium text-red-800 dark:text-red-400">
                                    {error}
                                </p>
                            </div>
                        </div>
                    </div>
                )}

                {/* Loading State */}
                {loading && (
                    <div className="flex justify-center items-center py-12">
                        <Loader className="h-8 w-8 text-blue-600 animate-spin" />
                    </div>
                )}

                {/* Members Table */}
                {!loading && members.length === 0 ? (
                    <div className="text-center py-12">
                        <p className="text-gray-600 dark:text-gray-400 mb-4">
                            {searchTerm ? 'No members found matching your search.' : 'No members yet.'}
                        </p>
                        {!searchTerm && (
                            <button
                                onClick={() => {
                                    setSelectedMember(null)
                                    setIsInviteModalOpen(true)
                                }}
                                className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition"
                            >
                                Invite First Member
                            </button>
                        )}
                    </div>
                ) : (
                    <>
                        <MembersTable
                            members={members}
                            tenantId={tenantId}
                            canManage={canManage}
                            onEditRole={(member) => {
                                setSelectedMember(member)
                                setIsEditRoleModalOpen(true)
                            }}
                            onRemove={(member) => handleRemoveMember(member.id)}
                        />

                        {/* Pagination */}
                        {total > limit && (
                            <div className="mt-6 flex justify-between items-center">
                                <p className="text-sm text-gray-600 dark:text-gray-400">
                                    Showing {((page - 1) * limit) + 1} to {Math.min(page * limit, total)} of {total} members
                                </p>
                                <div className="flex gap-2">
                                    <button
                                        onClick={() => setPage(Math.max(1, page - 1))}
                                        disabled={page === 1}
                                        className="px-4 py-2 border border-gray-300 dark:border-gray-700 rounded-lg disabled:opacity-50 disabled:cursor-not-allowed hover:bg-gray-100 dark:hover:bg-gray-800 transition"
                                    >
                                        Previous
                                    </button>
                                    <button
                                        onClick={() => setPage(page + 1)}
                                        disabled={page * limit >= total}
                                        className="px-4 py-2 border border-gray-300 dark:border-gray-700 rounded-lg disabled:opacity-50 disabled:cursor-not-allowed hover:bg-gray-100 dark:hover:bg-gray-800 transition"
                                    >
                                        Next
                                    </button>
                                </div>
                            </div>
                        )}
                    </>
                )}
            </div>

            {/* Modals */}
            <InviteMemberModal
                isOpen={isInviteModalOpen}
                onClose={async () => {
                    setIsInviteModalOpen(false)
                    // Reload members in case an existing user was added
                    if (tenantId) {
                        try {
                            const response = await adminService.getUsers({
                                tenantId,
                                search: searchTerm,
                                page: 1,
                                limit,
                            })
                            setMembers(response.data)
                            setTotal(response.pagination.total)
                            setPage(1)
                        } catch (err) {
                            // Silent reload failure
                        }
                    }
                }}
                onSubmit={handleInviteMember}
                tenantId={tenantId || undefined}
            />

            {selectedMember && (
                <EditMemberRoleModal
                    isOpen={isEditRoleModalOpen}
                    member={selectedMember}
                    tenantId={tenantId!}
                    onClose={() => {
                        setIsEditRoleModalOpen(false)
                        setSelectedMember(null)
                    }}
                    onSubmit={(roleId) => handleUpdateMemberRole(selectedMember.id, roleId)}
                />
            )}
        </div>
    )
}
