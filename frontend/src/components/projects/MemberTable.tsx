/**
 * Member table component for displaying project members
 */

import { ProjectMember } from '@/types/projectMember';
import React, { useState } from 'react';

interface MemberTableProps {
    members: ProjectMember[];
    loading?: boolean;
    error?: string | null;
    onEditMember?: (member: ProjectMember) => void;
    onDeleteMember?: (member: ProjectMember) => void;
    canManageMembers?: boolean;
}

/**
 * Renders a table of project members with action buttons
 */
export const MemberTable: React.FC<MemberTableProps> = ({
    members,
    loading = false,
    error,
    onEditMember,
    onDeleteMember,
    canManageMembers = true,
}) => {
    const [hoveredRow, setHoveredRow] = useState<string | null>(null);

    if (error) {
        return (
            <div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-md p-4">
                <p className="text-sm text-red-800 dark:text-red-300">{error}</p>
            </div>
        );
    }

    if (loading) {
        return (
            <div className="flex justify-center items-center h-48">
                <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-500"></div>
            </div>
        );
    }

    if (members.length === 0) {
        return (
            <div className="flex flex-col justify-center items-center h-48 bg-slate-50 dark:bg-slate-700 rounded-lg">
                <p className="text-slate-700 dark:text-slate-300 font-medium">No members yet</p>
                <p className="text-slate-600 dark:text-slate-400 text-sm mt-1">
                    Add your first member to get started
                </p>
            </div>
        );
    }

    return (
        <div className="overflow-x-auto">
            <table className="w-full border-collapse">
                <thead>
                    <tr className="border-b border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-700/50">
                        <th className="px-4 py-3 text-left text-sm font-semibold text-slate-900 dark:text-white">
                            User ID
                        </th>
                        <th className="px-4 py-3 text-left text-sm font-semibold text-slate-900 dark:text-white">
                            Role
                        </th>
                        <th className="px-4 py-3 text-left text-sm font-semibold text-slate-900 dark:text-white">
                            Assigned By
                        </th>
                        <th className="px-4 py-3 text-left text-sm font-semibold text-slate-900 dark:text-white">
                            Created At
                        </th>
                        {canManageMembers && (
                            <th className="px-4 py-3 text-left text-sm font-semibold text-slate-900 dark:text-white">
                                Actions
                            </th>
                        )}
                    </tr>
                </thead>
                <tbody>
                    {members.map((member) => (
                        <tr
                            key={member.id}
                            onMouseEnter={() => setHoveredRow(member.id)}
                            onMouseLeave={() => setHoveredRow(null)}
                            className={`border-b border-slate-200 dark:border-slate-700 transition-colors ${hoveredRow === member.id
                                    ? 'bg-slate-50 dark:bg-slate-700/30'
                                    : 'bg-white dark:bg-slate-800'
                                }`}
                        >
                            <td className="px-4 py-3">
                                <code className="text-xs bg-slate-100 dark:bg-slate-700 px-2 py-1 rounded text-slate-900 dark:text-white">
                                    {member.userId.substring(0, 8)}...
                                </code>
                            </td>
                            <td className="px-4 py-3">
                                {member.roleId ? (
                                    <code className="text-xs bg-slate-100 dark:bg-slate-700 px-2 py-1 rounded text-slate-900 dark:text-white">
                                        {member.roleId.substring(0, 8)}...
                                    </code>
                                ) : (
                                    <span className="text-xs text-slate-600 dark:text-slate-400">Default</span>
                                )}
                            </td>
                            <td className="px-4 py-3">
                                {member.assignedByUserId ? (
                                    <code className="text-xs bg-slate-100 dark:bg-slate-700 px-2 py-1 rounded text-slate-900 dark:text-white">
                                        {member.assignedByUserId.substring(0, 8)}...
                                    </code>
                                ) : (
                                    <span className="text-xs text-slate-600 dark:text-slate-400">N/A</span>
                                )}
                            </td>
                            <td className="px-4 py-3 text-xs text-slate-600 dark:text-slate-400">
                                {new Date(member.createdAt).toLocaleDateString()}
                            </td>
                            {canManageMembers && (
                                <td className="px-4 py-3">
                                    <div className="flex gap-2">
                                        <button
                                            onClick={() => onEditMember?.(member)}
                                            title="Edit Member"
                                            className="p-1 text-blue-600 hover:bg-blue-50 dark:hover:bg-blue-900/20 rounded transition-colors"
                                        >
                                            <svg className="h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
                                                <path d="M13.586 3.586a2 2 0 112.828 2.828l-.793.793-2.828-2.828.793-.793zM11.379 5.793L3 14.172V17h2.828l8.38-8.379-2.83-2.828z" />
                                            </svg>
                                        </button>
                                        <button
                                            onClick={() => onDeleteMember?.(member)}
                                            title="Remove Member"
                                            className="p-1 text-red-600 hover:bg-red-50 dark:hover:bg-red-900/20 rounded transition-colors"
                                        >
                                            <svg className="h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
                                                <path fillRule="evenodd" d="M9 2a1 1 0 00-.894.553L7.382 4H4a1 1 0 000 2v10a2 2 0 002 2h8a2 2 0 002-2V6a1 1 0 100-2h-3.382l-.724-1.447A1 1 0 0011 2H9zM7 8a1 1 0 012 0v6a1 1 0 11-2 0V8zm5-1a1 1 0 00-1 1v6a1 1 0 102 0V8a1 1 0 00-1-1z" clipRule="evenodd" />
                                            </svg>
                                        </button>
                                    </div>
                                </td>
                            )}
                        </tr>
                    ))}
                </tbody>
            </table>
        </div>
    );
};
