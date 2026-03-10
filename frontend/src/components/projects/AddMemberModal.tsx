/**
 * Modal for adding a new member to a project
 */

import React, { useState } from 'react';

interface AddMemberModalProps {
    open: boolean;
    loading?: boolean;
    error?: string | null;
    onSubmit: (userId: string) => Promise<void>;
    onClose: () => void;
    availableUsers?: Array<{ id: string; name: string; email: string }>;
}

/**
 * Modal dialog for adding a member to the project
 */
export const AddMemberModal: React.FC<AddMemberModalProps> = ({
    open,
    loading = false,
    error,
    onSubmit,
    onClose,
    availableUsers = [],
}) => {
    const [selectedUserId, setSelectedUserId] = useState<string>('');
    const [inputValue, setInputValue] = useState('');
    const [submitting, setSubmitting] = useState(false);
    const [submitError, setSubmitError] = useState<string | null>(null);
    const [filteredUsers, setFilteredUsers] = useState(availableUsers);

    const handleSubmit = async () => {
        if (!selectedUserId.trim()) {
            setSubmitError('Please select a user');
            return;
        }

        try {
            setSubmitting(true);
            setSubmitError(null);
            await onSubmit(selectedUserId);
            // Reset form
            setSelectedUserId('');
            setInputValue('');
        } catch (err) {
            const errorMessage = err instanceof Error ? err.message : 'Failed to add member';
            setSubmitError(errorMessage);
        } finally {
            setSubmitting(false);
        }
    };

    const handleClose = () => {
        setSelectedUserId('');
        setInputValue('');
        setSubmitError(null);
        onClose();
    };

    const handleInputChange = (value: string) => {
        setInputValue(value);
        // Filter users based on input
        if (value.trim()) {
            const filtered = availableUsers.filter(
                (u) =>
                    u.name.toLowerCase().includes(value.toLowerCase()) ||
                    u.email.toLowerCase().includes(value.toLowerCase())
            );
            setFilteredUsers(filtered);
        } else {
            setFilteredUsers(availableUsers);
        }
    };

    if (!open) return null;

    return (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
            <div className="bg-white dark:bg-slate-800 rounded-lg shadow-lg border border-slate-200 dark:border-slate-700 p-6 max-w-md w-full max-h-[90vh] overflow-y-auto">
                {/* Header */}
                <h2 className="text-xl font-semibold text-slate-900 dark:text-white mb-4">
                    Add Member to Project
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
                    {/* User Search/Select */}
                    <div>
                        <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                            Select User
                        </label>
                        <input
                            type="text"
                            placeholder="Search by name or email..."
                            value={inputValue}
                            onChange={(e) => handleInputChange(e.target.value)}
                            disabled={loading || submitting}
                            className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white disabled:opacity-50 disabled:cursor-not-allowed"
                        />

                        {/* Dropdown */}
                        {inputValue && filteredUsers.length > 0 && (
                            <div className="mt-2 border border-slate-300 dark:border-slate-600 rounded-md bg-white dark:bg-slate-700 shadow-lg max-h-48 overflow-y-auto">
                                {filteredUsers.map((user) => (
                                    <button
                                        key={user.id}
                                        onClick={() => {
                                            setSelectedUserId(user.id);
                                            setInputValue(`${user.name} (${user.email})`);
                                            setFilteredUsers([]);
                                        }}
                                        disabled={loading || submitting}
                                        className="w-full text-left px-4 py-2 hover:bg-blue-50 dark:hover:bg-slate-600 border-b border-slate-200 dark:border-slate-600 last:border-b-0 disabled:opacity-50 disabled:cursor-not-allowed"
                                    >
                                        <div className="font-medium text-slate-900 dark:text-white">
                                            {user.name}
                                        </div>
                                        <div className="text-sm text-slate-600 dark:text-slate-400">
                                            {user.email}
                                        </div>
                                    </button>
                                ))}
                            </div>
                        )}

                        {inputValue && filteredUsers.length === 0 && (
                            <div className="mt-2 p-3 text-sm text-slate-600 dark:text-slate-400">
                                No users found
                            </div>
                        )}
                    </div>

                    {/* User ID Display */}
                    {selectedUserId && (
                        <div>
                            <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                                User ID
                            </label>
                            <div className="px-3 py-2 bg-slate-100 dark:bg-slate-700 border border-slate-300 dark:border-slate-600 rounded-md font-mono text-sm text-slate-900 dark:text-white break-all">
                                {selectedUserId}
                            </div>
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
                        disabled={!selectedUserId || submitting || loading}
                        className="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-md text-sm font-medium disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2"
                    >
                        {submitting && (
                            <svg className="animate-spin h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor">
                                <circle className="opacity-25" cx="12" cy="12" r="10" strokeWidth="4" />
                                <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
                            </svg>
                        )}
                        Add Member
                    </button>
                </div>
            </div>
        </div>
    );
};
