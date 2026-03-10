/**
 * Modal for updating a member's role
 */

import { ProjectMember } from '@/types/projectMember';
import React, { useEffect, useState } from 'react';

interface EditMemberRoleModalProps {
    open: boolean;
    member: ProjectMember | null;
    loading?: boolean;
    error?: string | null;
    onSubmit: (userId: string, roleId?: string) => Promise<void>;
    onClose: () => void;
    availableRoles?: Array<{ id: string; name: string }>;
}

/**
 * Modal dialog for updating a member's role override
 */
export const EditMemberRoleModal: React.FC<EditMemberRoleModalProps> = ({
    open,
    member,
    loading = false,
    error,
    onSubmit,
    onClose,
    availableRoles = [],
}) => {
    const [selectedRoleId, setSelectedRoleId] = useState<string>('');
    const [submitting, setSubmitting] = useState(false);
    const [submitError, setSubmitError] = useState<string | null>(null);

    useEffect(() => {
        if (member) {
            setSelectedRoleId(member.roleId || '');
        }
    }, [member]);

    const handleSubmit = async () => {
        if (!member) return;

        try {
            setSubmitting(true);
            setSubmitError(null);
            await onSubmit(member.userId, selectedRoleId || undefined);
            handleClose();
        } catch (err) {
            const errorMessage = err instanceof Error ? err.message : 'Failed to update member role';
            setSubmitError(errorMessage);
        } finally {
            setSubmitting(false);
        }
    };

    const handleClose = () => {
        setSelectedRoleId('');
        setSubmitError(null);
        onClose();
    };

    if (!member || !open) return null;

    return (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
            <div className="bg-white dark:bg-slate-800 rounded-lg shadow-lg border border-slate-200 dark:border-slate-700 p-6 max-w-md w-full max-h-[90vh] overflow-y-auto">
                {/* Header */}
                <h2 className="text-xl font-semibold text-slate-900 dark:text-white mb-4">
                    Update Member Role
                </h2>

                {/* Alerts */}
                {error && (
                    <div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-md p-3 mb-4">
                        <p className="text-sm text-red-800 dark:text-red-300">{error}</p>
                    </div>
                )}
                {submitError && (
                    <div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-md p-3 mb-4">
                        <p className="text-sm text-red-800 dark:text-red-300">{submitError}</p>
                    </div>
                )}

                {/* Content */}
                <div className="space-y-4 mb-6">
                    {/* User ID Display */}
                    <div>
                        <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                            User ID
                        </label>
                        <div className="px-3 py-2 bg-slate-100 dark:bg-slate-700 border border-slate-300 dark:border-slate-600 rounded-md font-mono text-sm text-slate-900 dark:text-white break-all">
                            {member.userId}
                        </div>
                    </div>

                    {/* Role Select */}
                    <div>
                        <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                            Role Override
                        </label>
                        <select
                            value={selectedRoleId}
                            onChange={(e) => setSelectedRoleId(e.target.value)}
                            disabled={loading || submitting}
                            className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white disabled:opacity-50 disabled:cursor-not-allowed"
                        >
                            <option value="">Use Default Role</option>
                            {availableRoles.map((role) => (
                                <option key={role.id} value={role.id}>
                                    {role.name}
                                </option>
                            ))}
                        </select>
                        <p className="text-sm text-slate-600 dark:text-slate-400 mt-2">
                            Leave empty to use the member's default tenant role
                        </p>
                    </div>

                    {/* Info Alert */}
                    {member.roleId && selectedRoleId === '' && (
                        <div className="bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-md p-3">
                            <p className="text-sm text-blue-800 dark:text-blue-300">
                                Clearing the role override will revert to the member's default tenant role
                            </p>
                        </div>
                    )}
                </div>

                {/* Actions */}
                <div className="flex gap-3 justify-end">
                    <button
                        onClick={handleClose}
                        disabled={submitting}
                        className="px-4 py-2 border border-slate-300 dark:border-slate-600 rounded-md text-sm font-medium text-slate-700 dark:text-slate-300 hover:bg-slate-50 dark:hover:bg-slate-700 disabled:opacity-50 disabled:cursor-not-allowed"
                    >
                        Cancel
                    </button>
                    <button
                        onClick={handleSubmit}
                        disabled={submitting || loading}
                        className="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-md text-sm font-medium disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2"
                    >
                        {submitting && (
                            <svg className="animate-spin h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor">
                                <circle className="opacity-25" cx="12" cy="12" r="10" strokeWidth="4" />
                                <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
                            </svg>
                        )}
                        Update Role
                    </button>
                </div>
            </div>
        </div>
    );
};
