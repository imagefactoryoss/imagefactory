/**
 * Main component for project member management
 */

import { useMemberManagement, useProjectMembers } from '@/hooks/useMemberManagement';
import { ProjectMember } from '@/types/projectMember';
import React, { useState } from 'react';
import { AddMemberModal } from './AddMemberModal';
import { EditMemberRoleModal } from './EditMemberRoleModal';
import { MemberTable } from './MemberTable';

interface ProjectMembersUIProps {
    projectId: string;
    canManageMembers?: boolean;
}

/**
 * Main component for managing project members
 */
export const ProjectMembersUI: React.FC<ProjectMembersUIProps> = ({
    projectId,
    canManageMembers = true,
}) => {
    const [page, setPage] = useState(1);
    const [addMemberOpen, setAddMemberOpen] = useState(false);
    const [editMemberOpen, setEditMemberOpen] = useState(false);
    const [selectedMember, setSelectedMember] = useState<ProjectMember | null>(null);
    const [deleteConfirmOpen, setDeleteConfirmOpen] = useState(false);
    const [memberToDelete, setMemberToDelete] = useState<ProjectMember | null>(null);

    const {
        members,
        loading,
        error,
        totalCount,
        limit,
        offset,
        refetch,
        loadMore,
    } = useProjectMembers({
        projectId,
        initialLimit: 10,
    });

    const {
        addMember,
        removeMember,
        updateMemberRole,
        loading: managementLoading,
        error: managementError,
        success: managementSuccess,
    } = useMemberManagement({
        projectId,
        onSuccess: () => {
            refetch();
            setAddMemberOpen(false);
            setEditMemberOpen(false);
        },
    });

    const handleAddMember = async (userId: string) => {
        await addMember(userId);
    };

    const handleEditMember = (member: ProjectMember) => {
        setSelectedMember(member);
        setEditMemberOpen(true);
    };

    const handleUpdateRole = async (userId: string, roleId?: string) => {
        await updateMemberRole(userId, roleId);
    };

    const handleDeleteMember = (member: ProjectMember) => {
        setMemberToDelete(member);
        setDeleteConfirmOpen(true);
    };

    const handleConfirmDelete = async () => {
        if (memberToDelete) {
            try {
                await removeMember(memberToDelete.userId);
                setDeleteConfirmOpen(false);
                setMemberToDelete(null);
            } catch (err) {
                // Error is handled by the hook
            }
        }
    };

    const totalPages = Math.ceil(totalCount / limit);

    return (
        <div className="flex flex-col gap-4">
            <div className="bg-white dark:bg-slate-800 rounded-lg shadow border border-slate-200 dark:border-slate-700 overflow-hidden">
                {/* Header */}
                <div className="px-6 py-4 border-b border-slate-200 dark:border-slate-700 flex items-center justify-between">
                    <div>
                        <h3 className="text-lg font-semibold text-slate-900 dark:text-white">
                            Project Members
                        </h3>
                        <p className="text-sm text-slate-600 dark:text-slate-400 mt-1">
                            Total: {totalCount} members
                        </p>
                    </div>
                    {canManageMembers && (
                        <button
                            onClick={() => setAddMemberOpen(true)}
                            disabled={loading}
                            className="inline-flex items-center px-4 py-2 border border-transparent text-sm font-medium rounded-md shadow-sm text-white bg-blue-600 hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
                        >
                            <svg className="-ml-1 mr-2 h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
                                <path fillRule="evenodd" d="M10 3a1 1 0 011 1v5h5a1 1 0 110 2h-5v5a1 1 0 11-2 0v-5H4a1 1 0 110-2h5V4a1 1 0 011-1z" clipRule="evenodd" />
                            </svg>
                            Add Member
                        </button>
                    )}
                </div>

                {/* Alerts */}
                <div className="px-6 py-4 space-y-3">
                    {managementSuccess && (
                        <div className="bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800 rounded-md p-4">
                            <p className="text-sm text-green-800 dark:text-green-300">
                                {managementSuccess}
                            </p>
                        </div>
                    )}
                    {error && (
                        <div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-md p-4">
                            <p className="text-sm text-red-800 dark:text-red-300">
                                {error}
                            </p>
                        </div>
                    )}
                    {managementError && (
                        <div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-md p-4">
                            <p className="text-sm text-red-800 dark:text-red-300">
                                {managementError}
                            </p>
                        </div>
                    )}
                </div>

                {/* Table */}
                <div className="px-6 py-4">
                    <MemberTable
                        members={members}
                        loading={loading}
                        error={error}
                        onEditMember={handleEditMember}
                        onDeleteMember={handleDeleteMember}
                        canManageMembers={canManageMembers}
                    />
                </div>

                {/* Pagination */}
                {totalPages > 1 && (
                    <div className="px-6 py-4 border-t border-slate-200 dark:border-slate-700 flex items-center justify-between">
                        <div className="text-sm text-slate-600 dark:text-slate-400">
                            Page {Math.ceil((offset + 1) / limit)} of {totalPages}
                        </div>
                        <div className="flex gap-2">
                            <button
                                onClick={() => setPage(Math.max(1, page - 1))}
                                disabled={page === 1 || loading}
                                className="px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md text-sm font-medium text-slate-700 dark:text-slate-300 hover:bg-slate-50 dark:hover:bg-slate-700 disabled:opacity-50 disabled:cursor-not-allowed"
                            >
                                Previous
                            </button>
                            <button
                                onClick={() => {
                                    if (page < totalPages) {
                                        setPage(page + 1);
                                        loadMore();
                                    }
                                }}
                                disabled={page >= totalPages || loading}
                                className="px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md text-sm font-medium text-slate-700 dark:text-slate-300 hover:bg-slate-50 dark:hover:bg-slate-700 disabled:opacity-50 disabled:cursor-not-allowed"
                            >
                                Next
                            </button>
                        </div>
                    </div>
                )}
            </div>

            {/* Modals */}
            <AddMemberModal
                open={addMemberOpen}
                loading={managementLoading}
                error={managementError}
                onSubmit={handleAddMember}
                onClose={() => setAddMemberOpen(false)}
            />

            <EditMemberRoleModal
                open={editMemberOpen}
                member={selectedMember}
                loading={managementLoading}
                error={managementError}
                onSubmit={handleUpdateRole}
                onClose={() => {
                    setEditMemberOpen(false);
                    setSelectedMember(null);
                }}
            />

            {/* Delete Confirmation Dialog */}
            {deleteConfirmOpen && (
                <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
                    <div className="bg-white dark:bg-slate-800 rounded-lg shadow-lg border border-slate-200 dark:border-slate-700 p-6 max-w-sm">
                        <h3 className="text-lg font-semibold text-slate-900 dark:text-white mb-2">
                            Remove Member
                        </h3>
                        <p className="text-slate-600 dark:text-slate-400 mb-6">
                            Are you sure you want to remove this member from the project? This action cannot be undone.
                        </p>
                        <div className="flex gap-3 justify-end">
                            <button
                                onClick={() => setDeleteConfirmOpen(false)}
                                disabled={managementLoading}
                                className="px-4 py-2 border border-slate-300 dark:border-slate-600 rounded-md text-sm font-medium text-slate-700 dark:text-slate-300 hover:bg-slate-50 dark:hover:bg-slate-700 disabled:opacity-50 disabled:cursor-not-allowed"
                            >
                                Cancel
                            </button>
                            <button
                                onClick={handleConfirmDelete}
                                disabled={managementLoading}
                                className="px-4 py-2 bg-red-600 hover:bg-red-700 text-white rounded-md text-sm font-medium disabled:opacity-50 disabled:cursor-not-allowed"
                            >
                                Remove
                            </button>
                        </div>
                    </div>
                </div>
            )}
        </div>
    );
};
