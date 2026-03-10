import { api } from '@/services/api'
import { useConfirmDialog } from '@/context/ConfirmDialogContext'
import { useTenantStore } from '@/store/tenant'
import { AlertCircle, CheckCircle, Clock, Loader, Mail, RotateCw, Trash2, User } from 'lucide-react'
import { useEffect, useState } from 'react'
import toast from 'react-hot-toast'
import { useSearchParams } from 'react-router-dom'

interface Invitation {
    id: string
    email: string
    role_id: string
    role_name?: string
    status: 'pending' | 'accepted' | 'expired' | 'cancelled'
    created_at: string
    expires_at: string
    invited_by: string
    invited_by_name: string
    accepted_at?: string
    message?: string
}

interface ListInvitationsResponse {
    invitations: Invitation[]
    total: number
}

export default function InvitationsPage() {
    const [searchParams] = useSearchParams()
    const { selectedTenantId } = useTenantStore()
    const tenantId = searchParams.get('tenantId') || selectedTenantId

    const limit = 20  // Items per page

    const [invitations, setInvitations] = useState<Invitation[]>([])
    const [loading, setLoading] = useState(true)
    const [error, setError] = useState<string | null>(null)
    const [page, setPage] = useState(1)
    const [total, setTotal] = useState(0)
    const [resendingId, setResendingId] = useState<string | null>(null)
    const [cancellingId, setCancellingId] = useState<string | null>(null)
    const confirmDialog = useConfirmDialog()

    // Fetch invitations
    const loadInvitations = async () => {
        if (!tenantId) {
            setError('No tenant selected')
            setLoading(false)
            return
        }

        try {
            setLoading(true)
            setError(null)
            const response = await api.get('/invitations', {
                params: {
                    limit,
                    offset: (page - 1) * limit,
                },
            })
            const data: ListInvitationsResponse = response.data
            setInvitations(data.invitations || [])
            setTotal(data.total || 0)
        } catch (err) {
            const message = err instanceof Error ? err.message : 'Failed to load invitations'
            setError(message)
            toast.error(message)
        } finally {
            setLoading(false)
        }
    }

    useEffect(() => {
        if (tenantId) {
            loadInvitations()
        }
    }, [tenantId, page, limit])

    const handleResend = async (invitationId: string) => {
        try {
            setResendingId(invitationId)
            await api.post(`/invitations/${invitationId}/resend`, {
                tenant_id: tenantId,
            })
            toast.success('Invitation resent successfully')
            loadInvitations()
        } catch (err) {
            toast.error(err instanceof Error ? err.message : 'Failed to resend invitation')
        } finally {
            setResendingId(null)
        }
    }

    const handleCancel = async (invitationId: string) => {
        const confirmed = await confirmDialog({
            title: 'Cancel Invitation',
            message: 'Are you sure you want to cancel this invitation?',
            confirmLabel: 'Cancel Invitation',
            destructive: true,
        })
        if (!confirmed) {
            return
        }

        try {
            setCancellingId(invitationId)
            await api.delete(`/invitations/${invitationId}`)
            toast.success('Invitation cancelled')
            loadInvitations()
        } catch (err) {
            toast.error(err instanceof Error ? err.message : 'Failed to cancel invitation')
        } finally {
            setCancellingId(null)
        }
    }

    const getStatusBadge = (status: string, expiresAt: string) => {
        const isExpired = new Date(expiresAt) < new Date()

        if (status === 'accepted') {
            return (
                <span className="inline-flex items-center gap-1 px-3 py-1 rounded-full text-xs font-medium bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400">
                    <CheckCircle className="h-3 w-3" />
                    Accepted
                </span>
            )
        }

        if (status === 'cancelled') {
            return (
                <span className="inline-flex items-center px-3 py-1 rounded-full text-xs font-medium bg-gray-100 text-gray-800 dark:bg-gray-900/30 dark:text-gray-400">
                    Cancelled
                </span>
            )
        }

        if (isExpired || status === 'expired') {
            return (
                <span className="inline-flex items-center gap-1 px-3 py-1 rounded-full text-xs font-medium bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400">
                    <Clock className="h-3 w-3" />
                    Expired
                </span>
            )
        }

        return (
            <span className="inline-flex items-center gap-1 px-3 py-1 rounded-full text-xs font-medium bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-400">
                <Clock className="h-3 w-3" />
                Pending
            </span>
        )
    }

    const formatDate = (dateString: string) => {
        const date = new Date(dateString)
        return date.toLocaleDateString('en-US', {
            month: 'short',
            day: 'numeric',
            year: 'numeric',
            hour: '2-digit',
            minute: '2-digit',
        })
    }

    return (
        <div className="space-y-6">
            {/* Header */}
            <div className="flex items-center justify-between">
                <div>
                    <h1 className="text-3xl font-bold text-gray-900 dark:text-white">Invitations</h1>
                    <p className="mt-2 text-gray-600 dark:text-gray-400">
                        Manage pending invitations and track their status
                    </p>
                </div>
            </div>

            {/* Error Alert */}
            {error && (
                <div className="rounded-lg bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 p-4 flex items-start gap-3">
                    <AlertCircle className="h-5 w-5 text-red-600 dark:text-red-400 flex-shrink-0 mt-0.5" />
                    <div>
                        <h3 className="font-medium text-red-800 dark:text-red-400">Error</h3>
                        <p className="text-sm text-red-700 dark:text-red-300">{error}</p>
                    </div>
                </div>
            )}

            {/* Invitations Table */}
            <div className="rounded-lg border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800 overflow-hidden">
                {loading ? (
                    <div className="flex items-center justify-center py-12">
                        <Loader className="h-6 w-6 animate-spin text-gray-400" />
                    </div>
                ) : invitations.length > 0 ? (
                    <>
                        <div className="overflow-x-auto">
                            <table className="w-full">
                                <thead>
                                    <tr className="border-b border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-900/50">
                                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-700 dark:text-gray-300 uppercase tracking-wider">
                                            Email
                                        </th>
                                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-700 dark:text-gray-300 uppercase tracking-wider">
                                            Status
                                        </th>
                                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-700 dark:text-gray-300 uppercase tracking-wider">
                                            Sent By
                                        </th>
                                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-700 dark:text-gray-300 uppercase tracking-wider">
                                            Sent Date
                                        </th>
                                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-700 dark:text-gray-300 uppercase tracking-wider">
                                            Expires
                                        </th>
                                        <th className="px-6 py-3 text-right text-xs font-medium text-gray-700 dark:text-gray-300 uppercase tracking-wider">
                                            Actions
                                        </th>
                                    </tr>
                                </thead>
                                <tbody className="divide-y divide-gray-200 dark:divide-gray-700">
                                    {invitations.map((invitation) => (
                                        <tr key={invitation.id} className="hover:bg-gray-50 dark:hover:bg-gray-700/50 transition">
                                            <td className="px-6 py-4 whitespace-nowrap">
                                                <div className="flex items-center gap-2">
                                                    <Mail className="h-4 w-4 text-gray-400" />
                                                    <span className="text-sm font-medium text-gray-900 dark:text-white">
                                                        {invitation.email}
                                                    </span>
                                                </div>
                                            </td>
                                            <td className="px-6 py-4 whitespace-nowrap">
                                                {getStatusBadge(invitation.status, invitation.expires_at)}
                                            </td>
                                            <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-600 dark:text-gray-400">
                                                {invitation.invited_by_name || 'Unknown'}
                                            </td>
                                            <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-600 dark:text-gray-400">
                                                {formatDate(invitation.created_at)}
                                            </td>
                                            <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-600 dark:text-gray-400">
                                                {invitation.accepted_at
                                                    ? formatDate(invitation.accepted_at)
                                                    : formatDate(invitation.expires_at)}
                                            </td>
                                            <td className="px-6 py-4 whitespace-nowrap text-right">
                                                {invitation.status === 'pending' && (
                                                    <div className="flex items-center justify-end gap-2">
                                                        <button
                                                            onClick={() => handleResend(invitation.id)}
                                                            disabled={resendingId === invitation.id}
                                                            className="inline-flex items-center gap-1 px-3 py-1.5 text-xs font-medium text-blue-600 dark:text-blue-400 hover:bg-blue-50 dark:hover:bg-blue-900/20 rounded transition disabled:opacity-50"
                                                            title="Resend invitation"
                                                        >
                                                            {resendingId === invitation.id ? (
                                                                <Loader className="h-3 w-3 animate-spin" />
                                                            ) : (
                                                                <RotateCw className="h-3 w-3" />
                                                            )}
                                                            Resend
                                                        </button>
                                                        <button
                                                            onClick={() => handleCancel(invitation.id)}
                                                            disabled={cancellingId === invitation.id}
                                                            className="inline-flex items-center gap-1 px-3 py-1.5 text-xs font-medium text-red-600 dark:text-red-400 hover:bg-red-50 dark:hover:bg-red-900/20 rounded transition disabled:opacity-50"
                                                            title="Cancel invitation"
                                                        >
                                                            {cancellingId === invitation.id ? (
                                                                <Loader className="h-3 w-3 animate-spin" />
                                                            ) : (
                                                                <Trash2 className="h-3 w-3" />
                                                            )}
                                                            Cancel
                                                        </button>
                                                    </div>
                                                )}
                                                {invitation.status === 'accepted' && (
                                                    <span className="text-xs text-gray-500 dark:text-gray-400">
                                                        Accepted {formatDate(invitation.accepted_at || '')}
                                                    </span>
                                                )}
                                                {invitation.status === 'cancelled' && (
                                                    <span className="text-xs text-gray-500 dark:text-gray-400">
                                                        Cancelled
                                                    </span>
                                                )}
                                            </td>
                                        </tr>
                                    ))}
                                </tbody>
                            </table>
                        </div>

                        {/* Pagination */}
                        {total > limit && (
                            <div className="border-t border-gray-200 dark:border-gray-700 px-6 py-4 flex items-center justify-between">
                                <div className="text-sm text-gray-600 dark:text-gray-400">
                                    Showing {(page - 1) * limit + 1} to {Math.min(page * limit, total)} of {total} invitations
                                </div>
                                <div className="flex items-center gap-2">
                                    <button
                                        onClick={() => setPage(p => Math.max(1, p - 1))}
                                        disabled={page === 1}
                                        className="px-3 py-1.5 text-sm font-medium text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded disabled:opacity-50"
                                    >
                                        Previous
                                    </button>
                                    <span className="text-sm text-gray-600 dark:text-gray-400">
                                        Page {page}
                                    </span>
                                    <button
                                        onClick={() => setPage(p => p + 1)}
                                        disabled={page * limit >= total}
                                        className="px-3 py-1.5 text-sm font-medium text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded disabled:opacity-50"
                                    >
                                        Next
                                    </button>
                                </div>
                            </div>
                        )}
                    </>
                ) : (
                    <div className="text-center py-12">
                        <User className="mx-auto h-12 w-12 text-gray-400" />
                        <h3 className="mt-2 text-sm font-medium text-gray-900 dark:text-white">No invitations</h3>
                        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
                            No pending invitations found. Invite members from the Members page.
                        </p>
                    </div>
                )}
            </div>
        </div>
    )
}
